package cmd

import (
	"fmt"
	"time"

	"github.com/bagaking/cmdux/ui"
	"github.com/bagaking/cmdux/ux"
)

// CmduxUIRenderer 基于cmdux的UI渲染实现
type CmduxUIRenderer struct {
	verbose bool
}

// NewCmduxUIRenderer 创建cmdux UI渲染器
func NewCmduxUIRenderer(verbose bool) UIRenderer {
	return &CmduxUIRenderer{
		verbose: verbose,
	}
}

// ShowSuccess 显示成功消息
func (renderer *CmduxUIRenderer) ShowSuccess(title, message string) {
	successBox := ui.NewBox().
		Title(fmt.Sprintf("✅ %s", title)).
		Content(message).
		TitleStyle(app.Theme().Success).
		ContentStyle(app.Theme().Success).
		BorderStyle(app.Theme().Success)

	app.Render(successBox)
}

// ShowError 显示错误消息
func (renderer *CmduxUIRenderer) ShowError(title, message string) {
	errorBox := ui.NewBox().
		Title(fmt.Sprintf("❌ %s", title)).
		Content(message).
		TitleStyle(app.Theme().Error).
		ContentStyle(app.Theme().Error).
		BorderStyle(app.Theme().Error)

	app.Render(errorBox)
}

// ShowSpinner 显示进度指示器
func (renderer *CmduxUIRenderer) ShowSpinner(message string) SpinnerController {
	spinner := ux.NewSpinner(ux.SpinnerDots).Color(app.Theme().Primary)
	spinner.Start(message)

	return &CmduxSpinnerController{
		spinner: spinner,
		verbose: renderer.verbose,
	}
}

// ShowInfo 显示信息消息
func (renderer *CmduxUIRenderer) ShowInfo(message string) {
	if renderer.verbose {
		app.Println(message)
	}
}

// CmduxSpinnerController cmdux进度指示器控制器实现
type CmduxSpinnerController struct {
	spinner *ux.Spinner
	verbose bool
}

// Success 显示成功并停止
func (controller *CmduxSpinnerController) Success(message string) {
	// 显示加载动画一段时间，提升用户体验
	time.Sleep(1 * time.Second)
	controller.spinner.Success(message)

	// 输出空行，改善视觉效果
	if controller.verbose {
		app.Println("")
	}
}

// Error 显示错误并停止
func (controller *CmduxSpinnerController) Error(message string) {
	controller.spinner.Error(message)
}

// Stop 停止指示器
func (controller *CmduxSpinnerController) Stop() {
	controller.spinner.Stop()
}

// ConsoleUIRenderer 简单的控制台UI渲染实现（用于测试）
type ConsoleUIRenderer struct {
	verbose bool
}

// NewConsoleUIRenderer 创建控制台UI渲染器
func NewConsoleUIRenderer(verbose bool) UIRenderer {
	return &ConsoleUIRenderer{
		verbose: verbose,
	}
}

// ShowSuccess 显示成功消息
func (renderer *ConsoleUIRenderer) ShowSuccess(title, message string) {
	fmt.Printf("✅ %s: %s\n", title, message)
}

// ShowError 显示错误消息
func (renderer *ConsoleUIRenderer) ShowError(title, message string) {
	fmt.Printf("❌ %s: %s\n", title, message)
}

// ShowSpinner 显示进度指示器
func (renderer *ConsoleUIRenderer) ShowSpinner(message string) SpinnerController {
	if renderer.verbose {
		fmt.Printf("🔄 %s\n", message)
	}
	return &ConsoleSpinnerController{
		verbose: renderer.verbose,
	}
}

// ShowInfo 显示信息消息
func (renderer *ConsoleUIRenderer) ShowInfo(message string) {
	if renderer.verbose {
		fmt.Printf("ℹ️  %s\n", message)
	}
}

// ConsoleSpinnerController 控制台进度指示器控制器实现
type ConsoleSpinnerController struct {
	verbose bool
}

// Success 显示成功并停止
func (controller *ConsoleSpinnerController) Success(message string) {
	if controller.verbose {
		fmt.Printf("✅ %s\n", message)
	}
}

// Error 显示错误并停止
func (controller *ConsoleSpinnerController) Error(message string) {
	fmt.Printf("❌ %s\n", message)
}

// Stop 停止指示器
func (controller *ConsoleSpinnerController) Stop() {
	// 控制台实现无需特殊停止操作
}
