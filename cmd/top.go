package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	topRefreshInterval = 5 * time.Second
	topRunning        = false
	topMutex          sync.RWMutex
	forceColorScheme  = ""  // Manual override for color scheme
)

// Terminal color constants - compatible with both light and dark backgrounds
const (
	ColorReset     = "\033[0m"
	ColorBold      = "\033[1m"
	ColorDim       = "\033[2m"
	
	// High contrast colors that work on both light and dark backgrounds
	ColorBlue      = "\033[34m"  // Blue - good contrast on both
	ColorCyan      = "\033[36m"  // Cyan - good contrast on both
	ColorGreen     = "\033[32m"  // Green - readable on both
	ColorYellow    = "\033[33m"  // Yellow - readable on both
	ColorRed       = "\033[31m"  // Red - readable on both
	ColorMagenta   = "\033[35m"  // Magenta - good contrast
	
	// Bright variants for better visibility
	ColorBrightBlue    = "\033[94m"
	ColorBrightCyan    = "\033[96m" 
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightRed     = "\033[91m"
	ColorBrightMagenta = "\033[95m"
)

// Terminal background detection
type BackgroundType int

const (
	BackgroundUnknown BackgroundType = iota
	BackgroundLight
	BackgroundDark
)

// Global background detection cache
var (
	detectedBackground BackgroundType = BackgroundUnknown
	backgroundDetected = false
	backgroundMutex    sync.Mutex
)

// detectTerminalBackground attempts to detect if terminal has light or dark background
func detectTerminalBackground() BackgroundType {
	backgroundMutex.Lock()
	defer backgroundMutex.Unlock()
	
	if backgroundDetected {
		return detectedBackground
	}
	
	// Method 1: Try to query terminal background color using OSC 11
	if bg := queryTerminalBackground(); bg != BackgroundUnknown {
		detectedBackground = bg
		backgroundDetected = true
		return bg
	}
	
	// Method 2: Check environment variables for known terminals
	if bg := detectFromEnvironment(); bg != BackgroundUnknown {
		detectedBackground = bg
		backgroundDetected = true
		return bg
	}
	
	// Method 3: Fallback - assume dark background (safer default)
	detectedBackground = BackgroundDark
	backgroundDetected = true
	return detectedBackground
}

// queryTerminalBackground queries terminal background color using escape sequences
func queryTerminalBackground() BackgroundType {
	// Skip OSC 11 query in VS Code terminal as it doesn't support /dev/tty access
	if strings.ToLower(os.Getenv("TERM_PROGRAM")) == "vscode" {
		if verbose {
			fmt.Println("Debug: Skipping OSC 11 in VS Code terminal")
		}
		return BackgroundUnknown
	}
	
	// Try to open /dev/tty for direct terminal communication
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		if verbose {
			fmt.Printf("Debug: Cannot access /dev/tty: %v\n", err)
		}
		return BackgroundUnknown
	}
	defer tty.Close()
	
	// Save terminal state to restore later (to prevent escape sequences from showing)
	// Note: This is a simplified approach - a full implementation would save/restore terminal modes
	
	// Query background color using OSC 11 (Operating System Command)  
	// This asks terminal to report its background color
	query := "\033]11;?\033\\"
	
	// Set a short timeout for the query
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		// Write query
		if _, err := tty.Write([]byte(query)); err != nil {
			errorChan <- err
			return
		}
		
		// Read response with larger buffer to capture full response
		buffer := make([]byte, 200)
		n, err := tty.Read(buffer)
		if err != nil {
			errorChan <- err
			return
		}
		
		responseChan <- string(buffer[:n])
	}()
	
	// Wait for response with 300ms timeout (shorter to avoid blocking)
	select {
	case response := <-responseChan:
		// Debug output if verbose
		if verbose {
			// Clean the response for display (replace control chars)
			cleanResponse := strings.ReplaceAll(response, "\033", "^[")
			cleanResponse = strings.ReplaceAll(cleanResponse, "\007", "^G") 
			fmt.Printf("Debug: OSC 11 response: %q\n", cleanResponse)
		}
		return parseBackgroundResponse(response)
	case <-errorChan:
		if verbose {
			fmt.Println("Debug: OSC 11 query failed")
		}
		return BackgroundUnknown
	case <-time.After(300 * time.Millisecond):
		if verbose {
			fmt.Println("Debug: OSC 11 query timeout")
		}
		return BackgroundUnknown
	}
}

// parseBackgroundResponse parses the terminal's background color response
func parseBackgroundResponse(response string) BackgroundType {
	// OSC 11 response format: \033]11;rgb:RRRR/GGGG/BBBB\033\\
	// or \033]11;rgb:RR/GG/BB\033\\
	// The response you saw: ^[]11;rgb:1818/1818/1818^[\
	
	if !strings.Contains(response, "rgb:") {
		return BackgroundUnknown
	}
	
	// Extract RGB values - handle multiple possible terminators
	parts := strings.Split(response, "rgb:")
	if len(parts) < 2 {
		return BackgroundUnknown
	}
	
	rgbPart := parts[1]
	// Remove various possible terminators: \033\, \007, \033\\, etc.
	rgbPart = strings.Split(rgbPart, "\033")[0]  // Remove ESC sequences
	rgbPart = strings.Split(rgbPart, "\007")[0]  // Remove BEL (^G)
	rgbPart = strings.Split(rgbPart, "\\")[0]    // Remove backslash terminators
	
	rgbComponents := strings.Split(rgbPart, "/")
	
	if len(rgbComponents) < 3 {
		return BackgroundUnknown
	}
	
	// Parse RGB values (can be 2-digit or 4-digit hex)
	r, err1 := parseHexComponent(rgbComponents[0])
	g, err2 := parseHexComponent(rgbComponents[1])  
	b, err3 := parseHexComponent(rgbComponents[2])
	
	if err1 != nil || err2 != nil || err3 != nil {
		// Debug: log the parsing error if verbose mode is on
		if verbose {
			fmt.Printf("Debug: Failed to parse RGB components: %s -> R:%s G:%s B:%s\n", 
				rgbPart, rgbComponents[0], rgbComponents[1], rgbComponents[2])
		}
		return BackgroundUnknown
	}
	
	// Calculate luminance using standard formula
	// Y = 0.299*R + 0.587*G + 0.114*B
	luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	
	// Debug info if verbose
	if verbose {
		fmt.Printf("Debug: Parsed RGB(%d,%d,%d) -> Luminance: %.1f\n", r, g, b, luminance)
	}
	
	// For your case: rgb:1818/1818/1818 
	// This is 24/24/24 in decimal (very dark gray)
	// Threshold: if luminance > 128 (mid-point of 0-255), consider it light  
	if luminance > 128 {
		return BackgroundLight
	} else {
		return BackgroundDark
	}
}

// parseHexComponent parses a hex color component (2 or 4 digits)
func parseHexComponent(hex string) (int, error) {
	if len(hex) == 2 {
		// 2-digit hex (00-FF)
		val, err := strconv.ParseInt(hex, 16, 32)
		return int(val), err
	} else if len(hex) == 4 {
		// 4-digit hex (0000-FFFF), take first 2 digits
		val, err := strconv.ParseInt(hex[:2], 16, 32)
		return int(val), err
	}
	return 0, fmt.Errorf("invalid hex length: %d", len(hex))
}

// detectFromEnvironment detects background from environment variables
func detectFromEnvironment() BackgroundType {
	// Check COLORFGBG environment variable (used by some terminals)
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		// Format: "foreground;background" where colors are 0-15
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			bgColor, err := strconv.Atoi(parts[1])
			if err == nil {
				// Colors 0-7 are typically dark, 8-15 are bright
				// But for background, 0 is black (dark) and 15 is white (light)
				if bgColor >= 8 || bgColor == 7 { // 7 is light gray
					return BackgroundLight
				} else {
					return BackgroundDark
				}
			}
		}
	}
	
	// Check terminal program names
	term := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	termName := strings.ToLower(os.Getenv("TERM"))
	
	// For iTerm2 and other terminals, we can't assume background color
	// since users often customize them. Better to rely on OSC 11 query first.
	// Only use these as last resort fallbacks.
	switch {
	case strings.Contains(term, "terminal") && strings.Contains(os.Getenv("PATH"), "Darwin"):
		// macOS Terminal.app default is white - this is more reliable
		return BackgroundLight
	case strings.Contains(termName, "xterm") && os.Getenv("XDG_SESSION_TYPE") == "x11":
		// Most X11 xterms default to white/light
		return BackgroundLight
	// Note: Removed iTerm2 assumption since users frequently customize it
	}
	
	return BackgroundUnknown
}

// Adaptive color schemes based on background type
type ColorScheme struct {
	Header     string
	SubHeader  string
	TableHead  string
	Border     string
	Success    string
	Warning    string
	Error      string
	Active     string
	Controls   string
	Secondary  string
}

// getColorScheme returns appropriate colors based on detected background or manual override
func getColorScheme() *ColorScheme {
	var bg BackgroundType
	
	// Check for manual override first
	switch strings.ToLower(forceColorScheme) {
	case "light":
		bg = BackgroundLight
	case "dark":
		bg = BackgroundDark
	case "auto", "":
		bg = detectTerminalBackground()
	default:
		// Invalid option, fall back to detection
		bg = detectTerminalBackground()
	}
	
	switch bg {
	case BackgroundLight:
		// Optimized for light backgrounds - use ONLY black for maximum contrast and readability
		return &ColorScheme{
			Header:    "\033[30m" + ColorBold,     // Black, bold - maximum contrast on white
			SubHeader: "\033[30m",                 // Black - high contrast
			TableHead: "\033[30m" + ColorBold,     // Black, bold for table headers  
			Border:    "\033[30m",                 // Black for borders
			Success:   "\033[30m",                 // Black for normal status (most readable)
			Warning:   "\033[30m" + ColorBold,     // Black bold for warnings (readable)
			Error:     "\033[30m" + ColorBold,     // Black bold for errors (readable)
			Active:    "\033[30m" + ColorBold,     // Black bold for active (most contrast)
			Controls:  "\033[30m",                 // Black for controls text
			Secondary: "\033[90m",                 // Dark gray for secondary info
		}
	case BackgroundDark:
		// Optimized for dark backgrounds - use brighter colors
		return &ColorScheme{
			Header:    ColorBrightCyan + ColorBold,
			SubHeader: ColorBrightBlue,
			TableHead: ColorBrightMagenta + ColorBold,
			Border:    ColorDim,
			Success:   ColorBrightGreen,
			Warning:   ColorBrightYellow,
			Error:     ColorBrightRed,
			Active:    ColorBrightYellow + ColorBold,
			Controls:  ColorBrightCyan,
			Secondary: ColorDim,
		}
	default:
		// Fallback - use medium contrast colors that work reasonably on both
		return &ColorScheme{
			Header:    ColorBrightCyan + ColorBold,
			SubHeader: ColorBrightBlue,
			TableHead: ColorBrightMagenta + ColorBold,
			Border:    ColorDim,
			Success:   ColorBrightGreen,
			Warning:   ColorBrightYellow,
			Error:     ColorBrightRed,
			Active:    ColorBrightYellow + ColorBold,
			Controls:  ColorBrightCyan,
			Secondary: ColorDim,
		}
	}
}

// Color helper functions for consistent styling
func colorize(text string, color string) string {
	return color + text + ColorReset
}

func getStatusColor(percent float64) string {
	scheme := getColorScheme()
	if percent >= 90 {
		return scheme.Error
	} else if percent >= 70 {
		return scheme.Warning
	} else {
		return scheme.Success
	}
}

func getRowStatusColor(status string) string {
	scheme := getColorScheme()
	switch status {
	case "ERR":
		return scheme.Error
	case "HIGH":
		return scheme.Error
	case "WARN":
		return scheme.Warning
	default:
		return scheme.Success
	}
}

var topCmd = &cobra.Command{
	Use:     "top",
	Short:   "Interactive real-time monitoring of AI model quota usage",
	Long:    "Display real-time API quota usage for all model configurations with interactive switching capabilities",
	Aliases: []string{"monitor", "watch"},
	RunE:    runTop,
}

type ModelQuotaStatus struct {
	Name      string
	IsActive  bool
	QuotaInfo *QuotaInfo
	Error     error
	LastCheck time.Time
	SourcePID int    // Process ID that provided this data
	IsFromHistory bool // Whether data came from history collaboration
}

type TopMonitor struct {
	models       []string
	currentModel string
	statuses     map[string]*ModelQuotaStatus
	statusMutex  sync.RWMutex
	refreshRate  time.Duration
	quit         chan bool
	inputChan    chan rune
}

func init() {
	rootCmd.AddCommand(topCmd)
	topCmd.Flags().DurationVarP(&topRefreshInterval, "interval", "i", 10*time.Second, "Refresh interval (minimum 10s, e.g., 10s, 15s, 30s)")
	topCmd.Flags().StringVarP(&forceColorScheme, "colors", "c", "", "Force color scheme: 'light', 'dark', or 'auto' (default: auto-detect)")
}

func runTop(cmd *cobra.Command, args []string) error {
	// Prevent multiple instances
	topMutex.Lock()
	if topRunning {
		topMutex.Unlock()
		return fmt.Errorf("top command is already running")
	}
	topRunning = true
	topMutex.Unlock()

	defer func() {
		topMutex.Lock()
		topRunning = false
		topMutex.Unlock()
	}()

	// 验证并调整刷新间隔，确保不小于10秒
	validatedInterval := ValidateMinimumInterval(topRefreshInterval)
	if validatedInterval != topRefreshInterval && verbose {
		fmt.Printf("Warning: Refresh interval adjusted from %v to %v (minimum 10s required)\n", 
			topRefreshInterval, validatedInterval)
	}

	monitor, err := NewTopMonitor(validatedInterval)
	if err != nil {
		return fmt.Errorf("failed to initialize monitor: %v", err)
	}

	return monitor.Run()
}

func NewTopMonitor(refreshRate time.Duration) (*TopMonitor, error) {
	models, err := getAvailableModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get available models: %v", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no model configurations found")
	}

	currentModel, _ := getCurrentModel()

	monitor := &TopMonitor{
		models:       models,
		currentModel: currentModel,
		statuses:     make(map[string]*ModelQuotaStatus),
		refreshRate:  refreshRate,
		quit:         make(chan bool),
		inputChan:    make(chan rune, 10),
	}

	// Initialize model statuses
	for _, model := range models {
		monitor.statuses[model] = &ModelQuotaStatus{
			Name:      model,
			IsActive:  model == currentModel,
			LastCheck: time.Now(),
		}
	}

	return monitor, nil
}

func (m *TopMonitor) Run() error {
	// Clear screen and hide cursor
	fmt.Print("\033[2J\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	// Set up input handling in raw mode
	if err := m.setupRawInput(); err != nil {
		return fmt.Errorf("failed to setup input: %v", err)
	}
	defer m.restoreInput()

	// Start background goroutines
	go m.fetchQuotaData()
	go m.handleInput()

	// Initial render
	m.render()

	// Main event loop
	ticker := time.NewTicker(1 * time.Second) // Render more frequently than data refresh
	defer ticker.Stop()

	for {
		select {
		case <-m.quit:
			return nil
		case <-ticker.C:
			m.render()
		}
	}
}

func (m *TopMonitor) fetchQuotaData() {
	ticker := time.NewTicker(m.refreshRate)
	defer ticker.Stop()

	// Initial fetch
	m.fetchAllQuotas()

	for {
		select {
		case <-m.quit:
			return
		case <-ticker.C:
			m.fetchAllQuotas()
		}
	}
}

func (m *TopMonitor) fetchAllQuotas() {
	var wg sync.WaitGroup
	
	for _, model := range m.models {
		wg.Add(1)
		go func(modelName string) {
			defer wg.Done()
			
			quotaInfo, sourcePID, fromHistory, err := m.getQuotaWithSource(modelName, 8*time.Second)
			
			m.statusMutex.Lock()
			if status := m.statuses[modelName]; status != nil {
				status.QuotaInfo = quotaInfo
				status.Error = err
				status.LastCheck = time.Now()
				status.SourcePID = sourcePID
				status.IsFromHistory = fromHistory
			}
			m.statusMutex.Unlock()
		}(model)
	}
	
	wg.Wait()
}

// getQuotaWithSource gets quota info and returns source information
func (m *TopMonitor) getQuotaWithSource(modelName string, timeout time.Duration) (*QuotaInfo, int, bool, error) {
	// Try to get recent data from history first
	historyManager := getQuotaHistoryManager()
	
	// 使用刷新间隔作为时间窗口，实现基于检测周期的协商机制
	// 确保间隔不小于10秒，满足最小检测间隔要求
	validatedInterval := ValidateMinimumInterval(m.refreshRate)
	collaborationWindow := NewCollaborationTimeWindow(validatedInterval)
	timeWindow := collaborationWindow.CalculateWindow()
	
	if recentQuota, sourceEntry, err := historyManager.GetRecentQuotaFromHistory(modelName, timeWindow); err == nil && recentQuota != nil {
		// Found recent data from another process
		return recentQuota, sourceEntry.ProcessID, true, nil
	}
	
	// Get fresh data ourselves
	quotaInfo, err := getQuotaInfoForModel(modelName, timeout)
	return quotaInfo, os.Getpid(), false, err
}

func (m *TopMonitor) handleInput() {
	for {
		select {
		case <-m.quit:
			return
		case ch := <-m.inputChan:
			m.processInput(ch)
		}
	}
}

func (m *TopMonitor) processInput(ch rune) {
	switch ch {
	case 'q', 'Q', 27: // ESC key
		m.quit <- true
	case 'r', 'R':
		// Force refresh
		go m.fetchAllQuotas()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// Switch to model by number
		modelIndex := int(ch - '0')
		if modelIndex > 0 && modelIndex <= len(m.models) {
			modelName := m.models[modelIndex-1]
			m.switchToModel(modelName)
		}
	case 'h', 'H', '?':
		// Show help (toggle)
		// Implementation can be added later
	}
}

func (m *TopMonitor) switchToModel(modelName string) {
	// Perform switch operation
	if err := switchModel(modelName); err == nil {
		// Update current model tracking
		m.statusMutex.Lock()
		oldCurrent := m.currentModel
		m.currentModel = modelName
		
		// Update active status
		if oldStatus := m.statuses[oldCurrent]; oldStatus != nil {
			oldStatus.IsActive = false
		}
		if newStatus := m.statuses[modelName]; newStatus != nil {
			newStatus.IsActive = true
		}
		m.statusMutex.Unlock()
	}
}

func (m *TopMonitor) render() {
	// Move cursor to top-left
	fmt.Print("\033[H")
	
	m.statusMutex.RLock()
	defer m.statusMutex.RUnlock()
	
	// Render header
	m.renderHeader()
	
	// Render model table
	m.renderModelTable()
	
	// Render footer with controls
	m.renderFooter()
	
	// Clear rest of screen
	fmt.Print("\033[J")
}

func (m *TopMonitor) renderHeader() {
	now := time.Now().Format("15:04:05")
	scheme := getColorScheme()
	
	// Determine color scheme info for display
	colorSchemeInfo := ""
	switch strings.ToLower(forceColorScheme) {
	case "light":
		colorSchemeInfo = " [Colors: Light]"
	case "dark":
		colorSchemeInfo = " [Colors: Dark]"
	default:
		// Show detected background
		bg := detectTerminalBackground()
		switch bg {
		case BackgroundLight:
			colorSchemeInfo = " [Colors: Auto/Light]"
		case BackgroundDark:
			colorSchemeInfo = " [Colors: Auto/Dark]"
		default:
			colorSchemeInfo = " [Colors: Auto/Unknown]"
		}
	}
	
	headerLine1 := fmt.Sprintf("ccmodel top - %s", now)
	headerLine2 := fmt.Sprintf("Refresh: %v | Active: %s | Models: %d%s", 
		m.refreshRate, m.currentModel, len(m.models), colorSchemeInfo)
	
	// Use adaptive colors based on background detection
	if app != nil {
		// Use cmdux if available, but fallback to adaptive colors
		fmt.Print(app.Theme().Header.Sprint(headerLine1))
		fmt.Print(strings.Repeat(" ", 80-len(headerLine1)))
		fmt.Println()
		fmt.Print(app.Theme().Secondary.Sprint(headerLine2))
		fmt.Print(strings.Repeat(" ", 80-len(headerLine2)))
		fmt.Println()
	} else {
		// Use adaptive colors optimized for detected background
		fmt.Printf("%s%s%s\n", scheme.Header, headerLine1, ColorReset)
		fmt.Printf("%s%s%s\n", scheme.SubHeader, headerLine2, ColorReset)
	}
	
	fmt.Println()
}

func (m *TopMonitor) renderModelTable() {
	scheme := getColorScheme()
	
	// Header - added SOURCE column
	header := fmt.Sprintf("%-3s %-2s %-12s %-18s %-8s %-12s %-8s %-8s", 
		"#", "ST", "MODEL", "QUOTA USAGE", "PERCENT", "LAST CHECK", "STATUS", "SOURCE")
	
	if app != nil {
		fmt.Println(app.Theme().Header.Sprint(header))
		fmt.Println(app.Theme().Border.Sprint(strings.Repeat("-", 90)))
	} else {
		// Use adaptive colors for table headers
		fmt.Printf("%s%s%s\n", scheme.TableHead, header, ColorReset)
		fmt.Printf("%s%s%s\n", scheme.Border, strings.Repeat("-", 90), ColorReset)
	}
	
	// Sort models for consistent display
	sortedModels := make([]string, len(m.models))
	copy(sortedModels, m.models)
	sort.Strings(sortedModels)
	
	for i, model := range sortedModels {
		status := m.statuses[model]
		if status == nil {
			continue
		}
		
		m.renderModelRow(i+1, status)
	}
	
	fmt.Println()
}

func (m *TopMonitor) renderModelRow(index int, status *ModelQuotaStatus) {
	scheme := getColorScheme()
	
	// Status indicator
	statusIndicator := "○"
	if status.IsActive {
		statusIndicator = "★"
	}
	
	// Format quota information
	quotaDisplay := "-"
	percentDisplay := "-"
	rowStatus := "OK"
	
	// Determine colors based on status and quota
	var rowColor string = ""  // Default: no special color
	var percent float64 = 0
	
	if status.Error != nil {
		quotaDisplay = "ERROR"
		rowStatus = "ERR"
		rowColor = scheme.Error
	} else if status.QuotaInfo != nil && status.QuotaInfo.Total > 0 {
		quotaDisplay = formatQuotaForTopTable(status.QuotaInfo)
		percent = (status.QuotaInfo.Used / status.QuotaInfo.Total) * 100
		percentDisplay = fmt.Sprintf("%.1f%%", percent)
		
		// Color based on usage percentage using adaptive scheme
		if percent >= 90 {
			rowColor = scheme.Error
			rowStatus = "HIGH"
		} else if percent >= 70 {
			rowColor = scheme.Warning
			rowStatus = "WARN"
		} else {
			rowColor = scheme.Success
		}
	}
	
	// Last check time
	lastCheck := status.LastCheck.Format("15:04:05")
	
	// Source information
	sourceDisplay := "self"
	if status.IsFromHistory {
		if status.SourcePID == os.Getpid() {
			sourceDisplay = "self"
		} else {
			sourceDisplay = fmt.Sprintf("p%d", status.SourcePID)
		}
	}
	
	// Format row with appropriate coloring
	if app != nil {
		// Use cmdux if available
		var colorFunc func(...interface{}) string
		if status.Error != nil {
			colorFunc = app.Theme().Error.Sprint
		} else if percent >= 90 {
			colorFunc = app.Theme().Error.Sprint
		} else if percent >= 70 {
			colorFunc = app.Theme().Warning.Sprint
		} else if percent > 0 {
			colorFunc = app.Theme().Success.Sprint
		} else {
			colorFunc = app.Theme().Primary.Sprint
		}
		
		row := fmt.Sprintf("%-3d %-2s %-12s %-18s %-8s %-12s %-8s %-8s",
			index, statusIndicator, status.Name, quotaDisplay, 
			percentDisplay, lastCheck, rowStatus, sourceDisplay)
		
		if status.IsActive {
			// Highlight active model
			fmt.Printf("%s%s%s\n", ColorBold, colorFunc(row), ColorReset)
		} else {
			fmt.Println(colorFunc(row))
		}
	} else {
		// Use adaptive colors based on background detection
		row := fmt.Sprintf("%-3d %-2s %-12s %-18s %-8s %-12s %-8s %-8s",
			index, statusIndicator, status.Name, quotaDisplay, 
			percentDisplay, lastCheck, rowStatus, sourceDisplay)
		
		if status.IsActive {
			// Bold + adaptive active color for active model
			fmt.Printf("%s%s%s%s\n", ColorBold, scheme.Active, row, ColorReset)
		} else if rowColor != "" {
			// Adaptive color for other models
			fmt.Printf("%s%s%s\n", rowColor, row, ColorReset)
		} else {
			// No special color for normal status
			fmt.Println(row)
		}
	}
}

func (m *TopMonitor) renderFooter() {
	scheme := getColorScheme()
	fmt.Println()
	
	helpText := []string{
		"Controls: [1-9] Switch Model | [r] Refresh | [q/ESC] Quit | [h/?] Help",
		"",
		"Models are automatically refreshed every " + m.refreshRate.String(),
	}
	
	for i, line := range helpText {
		if app != nil {
			fmt.Println(app.Theme().Secondary.Sprint(line))
		} else {
			// Use adaptive colors for help text
			if i == 0 {
				// Make controls more visible using adaptive scheme
				fmt.Printf("%s%s%s\n", scheme.Controls, line, ColorReset)
			} else if line != "" {
				// Use secondary color for additional information
				fmt.Printf("%s%s%s\n", scheme.Secondary, line, ColorReset)
			} else {
				fmt.Println(line)
			}
		}
	}
}

// Platform-specific input handling functions
// These will need to be implemented for raw terminal input
func (m *TopMonitor) setupRawInput() error {
	// Start a goroutine to read from stdin
	go func() {
		buffer := make([]byte, 1)
		for {
			select {
			case <-m.quit:
				return
			default:
				n, err := os.Stdin.Read(buffer)
				if err != nil || n == 0 {
					continue
				}
				select {
				case m.inputChan <- rune(buffer[0]):
				default:
					// Channel full, skip
				}
			}
		}
	}()
	
	return nil
}

func (m *TopMonitor) restoreInput() {
	// Restore terminal to normal mode
	// Implementation depends on platform
}