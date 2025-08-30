package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// QuotaHistoryEntry represents a single quota history record
type QuotaHistoryEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	ModelName    string    `json:"model_name"`
	Total        float64   `json:"total"`
	Used         float64   `json:"used"`
	Percentage   float64   `json:"percentage"`
	EventType    string    `json:"event_type"` // "change", "heartbeat", "error"
	ErrorMessage string    `json:"error_message,omitempty"`
	ProcessID    int       `json:"process_id"`
	ProcessName  string    `json:"process_name,omitempty"`
}

// QuotaHistoryManager manages quota history logging
type QuotaHistoryManager struct {
	historyDir      string
	lastValues      map[string]*QuotaInfo // Track last known values per model
	lastHeartbeat   map[string]time.Time  // Track last heartbeat per model
	heartbeatInterval time.Duration        // 3 minutes for heartbeat
	mutex           sync.RWMutex
}

// NewQuotaHistoryManager creates a new quota history manager
func NewQuotaHistoryManager() *QuotaHistoryManager {
	claudeDir := filepath.Join(os.Getenv("HOME"), ".claude")
	historyDir := filepath.Join(claudeDir, "ccmodel", "quota_history")
	
	return &QuotaHistoryManager{
		historyDir:        historyDir,
		lastValues:        make(map[string]*QuotaInfo),
		lastHeartbeat:     make(map[string]time.Time),
		heartbeatInterval: 3 * time.Minute,
	}
}

// Initialize creates the history directory structure
func (qh *QuotaHistoryManager) Initialize() error {
	qh.mutex.Lock()
	defer qh.mutex.Unlock()
	
	if err := os.MkdirAll(qh.historyDir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %v", err)
	}
	
	return nil
}

// RecordQuota records quota information with change detection and heartbeat logic
func (qh *QuotaHistoryManager) RecordQuota(modelName string, quotaInfo *QuotaInfo) error {
	qh.mutex.Lock()
	defer qh.mutex.Unlock()
	
	now := time.Now()
	
	// Determine event type
	eventType := "heartbeat"
	shouldRecord := false
	
	lastValue := qh.lastValues[modelName]
	lastHeartbeatTime, hasHeartbeat := qh.lastHeartbeat[modelName]
	
	if quotaInfo.Error != nil {
		// Always record errors
		eventType = "error"
		shouldRecord = true
	} else if lastValue == nil {
		// First time seeing this model
		eventType = "change"
		shouldRecord = true
	} else if hasQuotaChanged(lastValue, quotaInfo) {
		// Quota values changed
		eventType = "change"
		shouldRecord = true
	} else if !hasHeartbeat || now.Sub(lastHeartbeatTime) >= qh.heartbeatInterval {
		// No change, but it's time for heartbeat
		eventType = "heartbeat"
		shouldRecord = true
	}
	
	if !shouldRecord {
		return nil
	}
	
	// Create history entry
	entry := &QuotaHistoryEntry{
		Timestamp:   now,
		ModelName:   modelName,
		EventType:   eventType,
		ProcessID:   os.Getpid(),
		ProcessName: "ccmodel",
	}
	
	if quotaInfo.Error != nil {
		entry.ErrorMessage = quotaInfo.Error.Error()
	} else {
		entry.Total = quotaInfo.Total
		entry.Used = quotaInfo.Used
		if quotaInfo.Total > 0 {
			entry.Percentage = (quotaInfo.Used / quotaInfo.Total) * 100
		}
	}
	
	// Write to daily log file
	if err := qh.writeToLogFile(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %v", err)
	}
	
	// Update tracking data
	if quotaInfo.Error == nil {
		qh.lastValues[modelName] = quotaInfo
	}
	qh.lastHeartbeat[modelName] = now
	
	return nil
}

// hasQuotaChanged checks if quota values have changed significantly
func hasQuotaChanged(old, new *QuotaInfo) bool {
	if old == nil || new == nil {
		return true
	}
	
	// Check if either had error before
	if (old.Error != nil) != (new.Error != nil) {
		return true
	}
	
	// If both have errors, compare error messages
	if old.Error != nil && new.Error != nil {
		return old.Error.Error() != new.Error.Error()
	}
	
	// Compare values with small tolerance for floating point comparison
	const tolerance = 0.001
	return abs(old.Total-new.Total) > tolerance || abs(old.Used-new.Used) > tolerance
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// writeToLogFile writes the entry to the appropriate daily log file with file locking
func (qh *QuotaHistoryManager) writeToLogFile(entry *QuotaHistoryEntry) error {
	// Generate filename based on date: YYYY-MM-DD.jsonl
	dateStr := entry.Timestamp.Format("2006-01-02")
	filename := filepath.Join(qh.historyDir, fmt.Sprintf("%s.jsonl", dateStr))
	lockFile := filename + ".lock"
	
	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %v", err)
	}
	
	// Implement file locking to avoid concurrent write conflicts
	if err := qh.withFileLock(lockFile, func() error {
		// Append to file (create if doesn't exist)
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %v", err)
		}
		defer file.Close()
		
		// Write JSON line
		if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
			return fmt.Errorf("failed to write entry: %v", err)
		}
		
		return nil
	}); err != nil {
		return err
	}
	
	return nil
}

// withFileLock executes a function with file locking to prevent concurrent access
func (qh *QuotaHistoryManager) withFileLock(lockFile string, fn func() error) error {
	// Create lock file with process ID and timestamp for identification
	lockContent := fmt.Sprintf("pid:%d timestamp:%d", os.Getpid(), time.Now().Unix())
	
	// Retry mechanism for acquiring lock
	maxRetries := 10
	retryDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Try to create lock file exclusively
		lock, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// Successfully acquired lock
			defer func() {
				lock.Close()
				os.Remove(lockFile) // Clean up lock file
			}()
			
			// Write lock info
			lock.WriteString(lockContent)
			
			// Execute the protected function
			return fn()
		}
		
		// Failed to acquire lock, check if it's stale
		if qh.isLockStale(lockFile) {
			// Remove stale lock and retry
			os.Remove(lockFile)
			continue
		}
		
		// Wait before retrying
		time.Sleep(retryDelay)
		retryDelay = time.Duration(float64(retryDelay) * 1.2) // Exponential backoff
	}
	
	return fmt.Errorf("failed to acquire file lock after %d attempts", maxRetries)
}

// isLockStale checks if a lock file is stale (older than 30 seconds)
func (qh *QuotaHistoryManager) isLockStale(lockFile string) bool {
	info, err := os.Stat(lockFile)
	if err != nil {
		return true // If we can't stat it, consider it stale
	}
	
	// Consider locks older than 30 seconds as stale
	return time.Since(info.ModTime()) > 30*time.Second
}

// CleanupOldLogs removes log files older than specified days
func (qh *QuotaHistoryManager) CleanupOldLogs(retentionDays int) error {
	qh.mutex.RLock()
	defer qh.mutex.RUnlock()
	
	if retentionDays <= 0 {
		return nil // No cleanup needed
	}
	
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	files, err := os.ReadDir(qh.historyDir)
	if err != nil {
		return fmt.Errorf("failed to read history directory: %v", err)
	}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// Parse date from filename (YYYY-MM-DD.jsonl)
		name := file.Name()
		if len(name) < 10 || filepath.Ext(name) != ".jsonl" {
			continue
		}
		
		dateStr := name[:10] // First 10 characters should be YYYY-MM-DD
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // Skip files with invalid date format
		}
		
		if fileDate.Before(cutoffTime) {
			fullPath := filepath.Join(qh.historyDir, name)
			if err := os.Remove(fullPath); err != nil {
				// Log the error but don't fail the entire cleanup
				if verbose {
					fmt.Printf("Warning: Failed to remove old log file %s: %v\n", fullPath, err)
				}
			}
		}
	}
	
	return nil
}

// GetHistoryPath returns the path to the quota history directory
func (qh *QuotaHistoryManager) GetHistoryPath() string {
	return qh.historyDir
}

// Global quota history manager instance
var quotaHistoryManager *QuotaHistoryManager
var quotaHistoryOnce sync.Once

// getQuotaHistoryManager returns the global quota history manager instance
func getQuotaHistoryManager() *QuotaHistoryManager {
	quotaHistoryOnce.Do(func() {
		quotaHistoryManager = NewQuotaHistoryManager()
		if err := quotaHistoryManager.Initialize(); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to initialize quota history: %v\n", err)
			}
		}
		
		// Automatic cleanup: remove logs older than 30 days
		if err := quotaHistoryManager.CleanupOldLogs(30); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to cleanup old quota logs: %v\n", err)
			}
		}
	})
	return quotaHistoryManager
}

// GetRecentQuotaFromHistory attempts to get recent quota data from history logs
// Returns quota info and source process info if found within the specified time window
func (qh *QuotaHistoryManager) GetRecentQuotaFromHistory(modelName string, timeWindow time.Duration) (*QuotaInfo, *QuotaHistoryEntry, error) {
	qh.mutex.RLock()
	defer qh.mutex.RUnlock()
	
	// Look for recent entries within the specified time window
	cutoffTime := time.Now().Add(-timeWindow)
	
	// Read today's log file
	dateStr := time.Now().Format("2006-01-02")
	filename := filepath.Join(qh.historyDir, fmt.Sprintf("%s.jsonl", dateStr))
	lockFile := filename + ".lock"
	
	var recentEntry *QuotaHistoryEntry
	var quotaInfo *QuotaInfo
	
	// Use read lock for reading
	err := qh.withFileReadLock(lockFile, func() error {
		file, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // No log file exists yet
			}
			return err
		}
		defer file.Close()
		
		// Read entries from the end (most recent first)
		// For simplicity, read all lines and check from the end
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		
		// Iterate from the last line backwards
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			
			var entry QuotaHistoryEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue // Skip malformed entries
			}
			
			// Check if this is for our model and within time window
			if entry.ModelName == modelName && entry.Timestamp.After(cutoffTime) {
				// Found a recent entry
				recentEntry = &entry
				
				// Convert to QuotaInfo if it's not an error
				if entry.EventType != "error" {
					quotaInfo = &QuotaInfo{
						Total: entry.Total,
						Used:  entry.Used,
						Error: nil,
					}
				} else {
					quotaInfo = &QuotaInfo{
						Error: fmt.Errorf(entry.ErrorMessage),
					}
				}
				return nil
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, nil, err
	}
	
	return quotaInfo, recentEntry, nil
}

// withFileReadLock executes a function with read access (allows multiple readers)
func (qh *QuotaHistoryManager) withFileReadLock(lockFile string, fn func() error) error {
	// For read operations, we just check if there's a write lock
	// If no write lock exists, we proceed
	maxRetries := 5
	retryDelay := 50 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if write lock exists
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			// No write lock, safe to read
			return fn()
		}
		
		// Check if write lock is stale
		if qh.isLockStale(lockFile) {
			os.Remove(lockFile)
			return fn()
		}
		
		// Wait briefly for write lock to be released
		time.Sleep(retryDelay)
	}
	
	// If we can't get read access, proceed anyway as read operations are generally safe
	return fn()
}