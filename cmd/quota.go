package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bagaking/cmdux/ui"
	"github.com/tidwall/gjson"
)

// QuotaCache represents cached quota information
type QuotaCache struct {
	Data      *QuotaInfo
	Timestamp time.Time
}

// Global cache with mutex for thread safety
var (
	quotaCache    = make(map[string]*QuotaCache)
	quotaCacheMux = sync.RWMutex{}
	cacheTimeout  = 30 * time.Second // Cache for 30 seconds
)

// QuotaConfig represents the quota test configuration
type QuotaConfig struct {
	QuotaTest *QuotaTest `json:"quota_test,omitempty"`
}

// QuotaTest represents a quota test configuration
type QuotaTest struct {
	Post *HTTPConfig `json:"post,omitempty"`
	Get  *HTTPConfig `json:"get,omitempty"`
}

// HTTPConfig represents the HTTP configuration for both GET and POST
type HTTPConfig struct {
	URL     string            `json:"url"`
	Header  map[string]string `json:"header,omitempty"`
	Data    map[string]any    `json:"data,omitempty"`
	Query   map[string]any    `json:"query,omitempty"`
	Result  *ResultConfig     `json:"result,omitempty"`
	Timeout *int              `json:"timeout,omitempty"` // Custom timeout in seconds
	Retry   *RetryConfig      `json:"retry,omitempty"`   // Retry configuration
}

// RetryConfig represents retry configuration for failed requests
type RetryConfig struct {
	Count int `json:"count,omitempty"` // Number of retries (default: 1)
	Delay int `json:"delay,omitempty"` // Delay between retries in milliseconds (default: 1000)
}

// PostConfig is an alias for backward compatibility
type PostConfig = HTTPConfig

// ResultConfig represents how to parse the response
type ResultConfig struct {
	Total string `json:"total"`
	Used  string `json:"used"`
}

// QuotaInfo represents the parsed quota information
type QuotaInfo struct {
	Total float64
	Used  float64
	Error error
}

// parseConfig parses the __ccmodel or __cc configuration from settings.json
func parseConfig(configFile string) (*QuotaConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	jsonStr := string(data)

	// Try __ccmodel first, then __cc as shorthand
	var ccmodelData gjson.Result
	if result := gjson.Get(jsonStr, "__ccmodel"); result.Exists() {
		ccmodelData = result
	} else if result := gjson.Get(jsonStr, "__cc"); result.Exists() {
		ccmodelData = result
	} else {
		return nil, nil // No quota config found
	}

	var quotaConfig QuotaConfig
	if err := json.Unmarshal([]byte(ccmodelData.Raw), &quotaConfig); err != nil {
		return nil, err
	}

	return &quotaConfig, nil
}

// expandVariables replaces variables using JSON path lookup from config
func expandVariables(value any, configJSON string) any {
	switch v := value.(type) {
	case string:
		return expandStringVariables(v, configJSON)
	case map[string]any:
		result := make(map[string]any)
		for k, val := range v {
			// Handle special $key syntax where key name starts with $
			if strings.HasPrefix(k, "$") {
				if pathStr, ok := val.(string); ok {
					if pathResult := gjson.Get(configJSON, pathStr); pathResult.Exists() {
						// Replace $xxx with xxx and use the looked up value
						newKey := k[1:] // Remove $ prefix
						result[newKey] = pathResult.String()
						continue
					}
				}
			}
			result[k] = expandVariables(val, configJSON)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = expandVariables(val, configJSON)
		}
		return result
	default:
		return v
	}
}

// expandStringVariables replaces $key with values from config using JSON path
func expandStringVariables(s string, configJSON string) string {
	// Pattern for $key syntax, e.g., $key or $path.to.value
	keyPattern := regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9._]*)`)
	return keyPattern.ReplaceAllStringFunc(s, func(match string) string {
		path := match[1:] // Remove "$" prefix
		result := gjson.Get(configJSON, path)
		if result.Exists() {
			return result.String()
		}
		return match // Return original if path not found
	})
}

// extractValueByPath extracts a value from JSON response using gjson path
func extractValueByPath(jsonData string, path string) (float64, error) {
	if path == "" {
		return 0, fmt.Errorf("empty path")
	}

	// Remove leading dot if present (gjson doesn't need it)
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}

	result := gjson.Get(jsonData, path)
	if !result.Exists() {
		return 0, fmt.Errorf("path not found: %s", path)
	}

	return result.Float(), nil
}

// executeQuotaRequest executes the quota request and returns the result
func executeQuotaRequest(config *HTTPConfig, method string, timeout time.Duration, configJSON string) (*QuotaInfo, error) {
	if config == nil {
		return nil, fmt.Errorf("no http config provided")
	}

	// Set up retry configuration
	retryCount := 1
	retryDelay := 1000 * time.Millisecond
	if config.Retry != nil {
		if config.Retry.Count > 0 {
			retryCount = config.Retry.Count
		}
		if config.Retry.Delay > 0 {
			retryDelay = time.Duration(config.Retry.Delay) * time.Millisecond
		}
	}

	var lastError error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		quotaInfo, err := performQuotaRequest(config, method, timeout, configJSON)
		if err == nil {
			return quotaInfo, nil
		}
		lastError = err
	}

	return nil, lastError
}

// performQuotaRequest performs a single quota request
func performQuotaRequest(config *HTTPConfig, method string, timeout time.Duration, configJSON string) (*QuotaInfo, error) {
	// Build URL with query parameters for GET requests
	requestURL := expandStringVariables(config.URL, configJSON)
	if method == "GET" && config.Query != nil {
		expandedQuery := expandVariables(config.Query, configJSON)
		if queryMap, ok := expandedQuery.(map[string]any); ok {
			queryParams := make([]string, 0, len(queryMap))
			for k, v := range queryMap {
				queryParams = append(queryParams, fmt.Sprintf("%s=%s", k, fmt.Sprintf("%v", v)))
			}
			if len(queryParams) > 0 {
				separator := "?"
				if strings.Contains(requestURL, "?") {
					separator = "&"
				}
				requestURL += separator + strings.Join(queryParams, "&")
			}
		}
	}

	var requestBody io.Reader
	if method == "POST" && config.Data != nil {
		// Expand variables in the request using config JSON
		expandedData := expandVariables(config.Data, configJSON)

		// Marshal the request data
		requestData, err := json.Marshal(expandedData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %v", err)
		}
		requestBody = bytes.NewBuffer(requestData)
	}

	// Create HTTP request
	req, err := http.NewRequest(method, requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	for key, value := range config.Header {
		req.Header.Set(key, expandStringVariables(value, configJSON))
	}

	// Use custom timeout if specified, otherwise use default
	actualTimeout := timeout
	if config.Timeout != nil && *config.Timeout > 0 {
		actualTimeout = time.Duration(*config.Timeout) * time.Second
	}

	// Execute request
	client := &http.Client{Timeout: actualTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(responseData))
	}

	// Convert response to string for gjson
	responseStr := string(responseData)

	// Extract quota information
	quotaInfo := &QuotaInfo{}

	if config.Result != nil {
		if config.Result.Total != "" {
			if total, err := extractValueByPath(responseStr, config.Result.Total); err == nil {
				quotaInfo.Total = total
			} else {
				return nil, fmt.Errorf("failed to extract total: %v", err)
			}
		}

		if config.Result.Used != "" {
			if used, err := extractValueByPath(responseStr, config.Result.Used); err == nil {
				quotaInfo.Used = used
			} else {
				return nil, fmt.Errorf("failed to extract used: %v", err)
			}
		}
	}

	return quotaInfo, nil
}

// getQuotaInfo retrieves quota information for the current model
func getQuotaInfo() (*QuotaInfo, error) {
	return getQuotaInfoWithTimeout(8 * time.Second)
}

// getQuotaInfoWithTimeout retrieves quota information with custom timeout
func getQuotaInfoWithTimeout(timeout time.Duration) (*QuotaInfo, error) {
	// Get current model to find its original config file
	currentModel, err := getCurrentModel()
	if err != nil || currentModel == "none" || currentModel == "custom" {
		return nil, nil
	}

	// Use original model config file for quota info
	return getQuotaInfoForModel(currentModel, timeout)
}

// getQuotaInfoForModel retrieves quota information for a specific model with collaboration
func getQuotaInfoForModel(modelName string, timeout time.Duration) (*QuotaInfo, error) {
	// Try to get recent data from history first (collaboration mechanism)
	historyManager := getQuotaHistoryManager()

	// 对于一般quota查询，使用30秒的合理默认协商窗口
	// 这提供了足够的时间让不同ccmodel实例共享数据，避免重复API调用
	collaborationWindow := NewDefaultCollaborationTimeWindow()
	timeWindow := collaborationWindow.CalculateWindow()

	if recentQuota, sourceEntry, err := historyManager.GetRecentQuotaFromHistory(modelName, timeWindow); err == nil && recentQuota != nil {
		// Found recent data from another process, use it
		if verbose {
			fmt.Printf("Debug: Using recent quota from process %d (%s)\n", sourceEntry.ProcessID, sourceEntry.ProcessName)
		}
		return recentQuota, nil
	}

	// Check cache first
	cacheKey := modelName
	quotaCacheMux.RLock()
	if cached, exists := quotaCache[cacheKey]; exists {
		if time.Since(cached.Timestamp) < cacheTimeout {
			quotaCacheMux.RUnlock()
			return cached.Data, nil
		}
	}
	quotaCacheMux.RUnlock()

	configFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", modelName))

	// Read raw config JSON
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	configJSON := string(configData)

	config, err := parseConfig(configFile)
	if err != nil {
		return nil, err
	}

	if config == nil || config.QuotaTest == nil {
		return nil, nil // No quota config found
	}

	var quotaInfo *QuotaInfo
	// Try POST first, then GET
	if config.QuotaTest.Post != nil {
		quotaInfo, err = executeQuotaRequest(config.QuotaTest.Post, "POST", timeout, configJSON)
	} else if config.QuotaTest.Get != nil {
		quotaInfo, err = executeQuotaRequest(config.QuotaTest.Get, "GET", timeout, configJSON)
	} else {
		return nil, nil // No valid config found
	}

	// Cache the result (even if it's an error, to avoid repeated failed calls)
	quotaCacheMux.Lock()
	quotaCache[cacheKey] = &QuotaCache{
		Data:      quotaInfo,
		Timestamp: time.Now(),
	}
	quotaCacheMux.Unlock()

	// Record to history for collaboration with other ccmodel instances
	if quotaInfo != nil {
		if err := historyManager.RecordQuota(modelName, quotaInfo); err != nil {
			if verbose {
				fmt.Printf("Debug: Failed to record quota history: %v\n", err)
			}
		}
	}

	return quotaInfo, err
}

// renderQuotaInfo displays quota information in the UI
func renderQuotaInfo(quotaInfo *QuotaInfo) {
	if quotaInfo == nil {
		return
	}

	if quotaInfo.Error != nil {
		errorBox := ui.NewBox().
			Title("❌ Quota Query Failed").
			Content(fmt.Sprintf("Error: %v", quotaInfo.Error)).
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
		app.Render(errorBox)
		return
	}

	// Calculate usage percentage
	var usagePercent float64
	if quotaInfo.Total > 0 {
		usagePercent = (quotaInfo.Used / quotaInfo.Total) * 100
	}

	// Format quota values
	totalStr := formatQuotaValue(quotaInfo.Total)
	usedStr := formatQuotaValue(quotaInfo.Used)
	remainingStr := formatQuotaValue(quotaInfo.Total - quotaInfo.Used)

	quotaContent := fmt.Sprintf(`Total Quota: %s
Used: %s (%.1f%%)
Remaining: %s`,
		totalStr,
		usedStr,
		usagePercent,
		remainingStr)

	// Choose style based on usage percentage and create box
	var quotaBox *ui.Box
	if usagePercent >= 90 {
		quotaBox = ui.NewBox().
			Title("📊 API Quota Status").
			Content(quotaContent).
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
	} else if usagePercent >= 70 {
		quotaBox = ui.NewBox().
			Title("📊 API Quota Status").
			Content(quotaContent).
			TitleStyle(app.Theme().Warning).
			ContentStyle(app.Theme().Warning).
			BorderStyle(app.Theme().Warning)
	} else {
		quotaBox = ui.NewBox().
			Title("📊 API Quota Status").
			Content(quotaContent).
			TitleStyle(app.Theme().Success).
			ContentStyle(app.Theme().Success).
			BorderStyle(app.Theme().Success)
	}
	app.Render(quotaBox)
}

// formatQuotaValue formats quota values for display
func formatQuotaValue(value float64) string {
	if value >= 1000000 {
		return fmt.Sprintf("%.3fM", value/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("%.3fK", value/1000)
	} else {
		if value == float64(int(value)) {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.3f", value)
	}
}

// formatQuotaForTable formats quota info for table display (compact format)
func formatQuotaForTable(quotaInfo *QuotaInfo) string {
	if quotaInfo == nil || quotaInfo.Error != nil {
		return "-"
	}

	if quotaInfo.Total <= 0 {
		return "-"
	}

	usagePercent := (quotaInfo.Used / quotaInfo.Total) * 100
	usedStr := formatQuotaValue(quotaInfo.Used)
	totalStr := formatQuotaValue(quotaInfo.Total)

	return fmt.Sprintf("%s/%s (%.2f%%)", usedStr, totalStr, usagePercent)
}

// formatQuotaForTopTable 格式化quota信息用于top命令的表格显示（不包含百分比避免冗余）
func formatQuotaForTopTable(quotaInfo *QuotaInfo) string {
	if quotaInfo == nil || quotaInfo.Error != nil {
		return "-"
	}

	if quotaInfo.Total <= 0 {
		return "-"
	}

	usedStr := formatQuotaValue(quotaInfo.Used)
	totalStr := formatQuotaValue(quotaInfo.Total)

	return fmt.Sprintf("%s/%s", usedStr, totalStr)
}

// getQuotaStatusColor returns the appropriate color for quota status
func getQuotaStatusColor(quotaInfo *QuotaInfo) func(...any) string {
	if quotaInfo == nil || quotaInfo.Error != nil || quotaInfo.Total <= 0 {
		return app.Theme().Primary.Sprint
	}

	usagePercent := (quotaInfo.Used / quotaInfo.Total) * 100
	if usagePercent >= 90 {
		return app.Theme().Error.Sprint
	} else if usagePercent >= 70 {
		return app.Theme().Warning.Sprint
	} else {
		return app.Theme().Success.Sprint
	}
}
