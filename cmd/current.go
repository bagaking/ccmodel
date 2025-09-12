package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bagaking/cmdux/ui"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Short:   "Show current model configuration",
	Long:    "Display detailed information about the currently active AI model configuration",
	Aliases: []string{"status", "whoami"},
	RunE:    runCurrent,
}

// ModelStatus represents the current model status for JSON output
type ModelStatus struct {
	Model        string            `json:"model"`
	ConfigPath   string            `json:"config_path"`
	FileSize     int64             `json:"file_size,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
	IsActive     bool              `json:"is_active"`
	IsCustom     bool              `json:"is_custom"`
	Quota        *QuotaStatusJSON  `json:"quota,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// QuotaStatusJSON represents quota info optimized for JSON output
type QuotaStatusJSON struct {
	Total   float64 `json:"total"`
	Used    float64 `json:"used"`
	Percent float64 `json:"percent"`
}

// ModelStatusCollector 职责单一：收集模型状态信息
type ModelStatusCollector struct {
	configDir string
}

func NewModelStatusCollector(configDir string) *ModelStatusCollector {
	return &ModelStatusCollector{configDir: configDir}
}

func (msc *ModelStatusCollector) CollectStatus() (*ModelStatus, error) {
	currentModel, err := getCurrentModel()
	if err != nil {
		return &ModelStatus{
			Model:    "none",
			IsActive: false,
			Error:    err.Error(),
		}, nil // 不返回错误，因为这是有效状态
	}

	configFile := filepath.Join(msc.configDir, "settings.json")
	status := &ModelStatus{
		Model:      currentModel,
		ConfigPath: configFile,
		IsActive:   currentModel != "none",
		IsCustom:   currentModel == "custom",
	}

	// 收集文件信息
	if err := msc.collectFileInfo(status); err != nil {
		status.Error = fmt.Sprintf("file info error: %v", err)
	}

	// 收集配额信息
	if err := msc.collectQuotaInfo(status); err != nil {
		// 配额错误不影响主要功能，记录但不阻止
		if status.Error != "" {
			status.Error = fmt.Sprintf("%s; quota error: %v", status.Error, err)
		} else {
			status.Error = fmt.Sprintf("quota error: %v", err)
		}
	}

	return status, nil
}

func (msc *ModelStatusCollector) collectFileInfo(status *ModelStatus) error {
	if status.Model == "none" {
		return nil
	}

	info, err := os.Stat(status.ConfigPath)
	if err != nil {
		return err
	}

	status.FileSize = info.Size()
	status.LastModified = info.ModTime().Format("2006-01-02T15:04:05Z")
	return nil
}

func (msc *ModelStatusCollector) collectQuotaInfo(status *ModelStatus) error {
	if status.Model == "none" || status.Model == "custom" {
		return nil // 这些情况下没有配额信息是正常的
	}

	quotaInfo, err := getQuotaInfoForModel(status.Model, 8*time.Second)
	if err != nil {
		return err
	}

	if quotaInfo != nil {
		if quotaInfo.Error != nil {
			return quotaInfo.Error
		}
		
		percent := 0.0
		if quotaInfo.Total > 0 {
			percent = (quotaInfo.Used / quotaInfo.Total) * 100
		}

		status.Quota = &QuotaStatusJSON{
			Total:   quotaInfo.Total,
			Used:    quotaInfo.Used,
			Percent: percent,
		}
	}

	return nil
}

func init() {
	currentCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status as JSON")
}

func runCurrent(cmd *cobra.Command, args []string) error {
	collector := NewModelStatusCollector(configDir)
	
	if jsonOutput {
		return handleJSONOutput(collector)
	}
	
	return handleUIOutput(collector)
}

// handleJSONOutput 处理 JSON 输出逻辑
func handleJSONOutput(collector *ModelStatusCollector) error {
	status, err := collector.CollectStatus()
	if err != nil {
		return fmt.Errorf("failed to collect status: %v", err)
	}
	
	return outputJSON(status)
}

// handleUIOutput 处理传统 UI 输出逻辑  
func handleUIOutput(collector *ModelStatusCollector) error {
	status, err := collector.CollectStatus()
	if err != nil {
		return fmt.Errorf("failed to collect status: %v", err)
	}

	return renderUIOutput(status)
}

// renderUIOutput 渲染 UI 输出（保持原有逻辑）
func renderUIOutput(status *ModelStatus) error {
	headerBox := ui.NewBox().
		Title("CURRENT MODEL STATUS").
		Content("Active AI configuration").
		Width(60).
		TitleStyle(app.Theme().Primary).
		ContentStyle(app.Theme().Secondary).
		BorderStyle(app.Theme().Primary)
	app.Render(headerBox)
	app.Println("")

	if !status.IsActive {
		errorBox := ui.NewBox().
			Title("❌ No Active Configuration").
			Content("No active configuration found").
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
		app.Render(errorBox)
		app.Println("")

		nextStepsContent := `Next Steps:

• Create model configurations in ~/.claude/
• Use 'ccmodel list' to see available models  
• Use 'ccmodel switch <model>' to activate one

Quick Commands:

ccmodel list         → List all available models
ccmodel current      → Show currently active model
ccmodel switch <name> → Switch to a different model
ccmodel --help       → Show detailed help`

		infoBox := ui.NewBox().
			Title("📚 Next Steps").
			Content(nextStepsContent).
			Width(55).
			TitleStyle(app.Theme().Accent2).
			ContentStyle(app.Theme().Primary).
			BorderStyle(app.Theme().Accent2)
		app.Render(infoBox)
		return nil
	}

	// Model status display
	if status.IsCustom {
		app.Println("⚙  " + app.Theme().Accent1.Sprint("Custom Configuration"))
		app.Println("   " + app.Theme().Accent1.Sprint("Not managed by ccmodel templates"))
	} else {
		app.Println("●  " + app.Theme().Success.Sprint(status.Model))
	}
	app.Println("")

	// File details
	if status.FileSize > 0 {
		detailsContent := fmt.Sprintf(`Model: %s
Config File: %s
File Size: %s
Last Modified: %s`,
			status.Model,
			status.ConfigPath,
			formatFileSize(status.FileSize),
			parseLastModified(status.LastModified))

		detailsBox := ui.NewBox().
			Title("📋 Configuration Details").
			Content(detailsContent).
			TitleStyle(app.Theme().Accent1).
			ContentStyle(app.Theme().Primary).
			BorderStyle(app.Theme().Accent1)
		app.Render(detailsBox)

		if status.IsCustom {
			app.Println("")
			app.Println("⚠  " + app.Theme().Warning.Sprint("This configuration is not managed by ccmodel templates"))
		}

		// Quota display
		if status.Quota != nil {
			app.Println("")
			renderQuotaInfoFromJSON(status.Quota)
		} else if status.Error != "" && verbose {
			app.Println("")
			app.Println("⚠  " + app.Theme().Warning.Sprint("Quota info: ") + status.Error)
		}
	} else {
		app.Println("❌  " + app.Theme().Error.Sprint("Configuration file not found"))
	}

	return nil
}

// parseLastModified 解析 ISO 时间格式为显示格式
func parseLastModified(isoTime string) string {
	if isoTime == "" {
		return ""
	}
	
	if t, err := time.Parse("2006-01-02T15:04:05Z", isoTime); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	
	return isoTime // fallback
}

// renderQuotaInfoFromJSON 从 JSON 结构渲染配额信息
func renderQuotaInfoFromJSON(quota *QuotaStatusJSON) {
	quotaInfo := &QuotaInfo{
		Total: quota.Total,
		Used:  quota.Used,
		Error: nil,
	}
	renderQuotaInfo(quotaInfo)
}

// outputJSON outputs the status as JSON
func outputJSON(status *ModelStatus) error {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
