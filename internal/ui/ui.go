package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Color scheme for the CLI
var (
	// Primary colors
	Primary   = color.New(color.FgCyan, color.Bold)
	Secondary = color.New(color.FgBlue)
	Success   = color.New(color.FgGreen, color.Bold)
	Warning   = color.New(color.FgYellow)
	Error     = color.New(color.FgRed, color.Bold)
	Muted     = color.New(color.FgHiBlack)
	
	// Accent colors
	Accent1   = color.New(color.FgMagenta)
	Accent2   = color.New(color.FgHiCyan)
	
	// Text styles
	Bold      = color.New(color.Bold)
	Italic    = color.New(color.Italic)
	Underline = color.New(color.Underline)
)

// Box drawing characters for modern terminals
const (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
	BoxTee         = "├"
	BoxCross       = "┼"
	
	// Modern bullets and separators
	Bullet         = "•"
	Arrow          = "→"
	CheckMark      = "✓"
	CrossMark      = "✗"
	Lightning      = "⚡"
	Gear           = "⚙"
	Rocket         = "🚀"
	Diamond        = "◆"
	Circle         = "●"
	
	// Spacing
	Indent         = "  "
	DoubleIndent   = "    "
)

// Header prints a stylized header
func Header(title, subtitle string) {
	width := 60
	
	// Top border
	fmt.Print(Primary.Sprint(BoxTopLeft))
	fmt.Print(Primary.Sprint(strings.Repeat(BoxHorizontal, width-2)))
	fmt.Println(Primary.Sprint(BoxTopRight))
	
	// Title line
	titlePadding := (width - len(title) - 2) / 2
	fmt.Print(Primary.Sprint(BoxVertical))
	fmt.Print(strings.Repeat(" ", titlePadding))
	fmt.Print(Bold.Sprint(title))
	fmt.Print(strings.Repeat(" ", width-len(title)-titlePadding-2))
	fmt.Println(Primary.Sprint(BoxVertical))
	
	// Subtitle line if provided
	if subtitle != "" {
		subtitlePadding := (width - len(subtitle) - 2) / 2
		fmt.Print(Primary.Sprint(BoxVertical))
		fmt.Print(strings.Repeat(" ", subtitlePadding))
		fmt.Print(Muted.Sprint(subtitle))
		fmt.Print(strings.Repeat(" ", width-len(subtitle)-subtitlePadding-2))
		fmt.Println(Primary.Sprint(BoxVertical))
	}
	
	// Bottom border
	fmt.Print(Primary.Sprint(BoxBottomLeft))
	fmt.Print(Primary.Sprint(strings.Repeat(BoxHorizontal, width-2)))
	fmt.Println(Primary.Sprint(BoxBottomRight))
	fmt.Println()
}

// StatusLine prints a status with icon and colors
func StatusLine(icon, status, detail string, colorFunc *color.Color) {
	fmt.Printf("%s %s", 
		colorFunc.Sprint(icon), 
		colorFunc.Sprint(status))
	if detail != "" {
		fmt.Printf(" %s", Muted.Sprint(detail))
	}
	fmt.Println()
}

// ModelEntry prints a formatted model entry
func ModelEntry(index int, name, size, modified string, isActive bool) {
	var statusIcon, nameColor *color.Color
	
	if isActive {
		statusIcon = Success
		nameColor = Success
	} else {
		statusIcon = Muted
		nameColor = Primary
	}
	
	// Index with proper padding
	indexStr := fmt.Sprintf("%2d", index)
	
	// Status indicator
	var indicator string
	if isActive {
		indicator = Circle + " "
	} else {
		indicator = "  "
	}
	
	fmt.Printf("%s %s%s%-15s %s%8s %s%s%s\n",
		Muted.Sprint(indexStr+"."),
		statusIcon.Sprint(indicator),
		nameColor.Sprint(name),
		"", // padding
		Muted.Sprint(size),
		"",
		Muted.Sprint(modified),
		"",
		func() string {
			if isActive {
				return Success.Sprint(" " + Lightning + " ACTIVE")
			}
			return ""
		}())
}

// InfoBox prints an information box
func InfoBox(title string, items []string) {
	fmt.Printf("%s %s\n", Accent1.Sprint(Diamond), Bold.Sprint(title))
	for _, item := range items {
		fmt.Printf("%s%s %s\n", Indent, Muted.Sprint(Bullet), item)
	}
	fmt.Println()
}

// SuccessBox prints a success message
func SuccessBox(message string) {
	fmt.Printf("%s %s\n", Success.Sprint(CheckMark), Success.Sprint(message))
}

// ErrorBox prints an error message  
func ErrorBox(message string) {
	fmt.Printf("%s %s\n", Error.Sprint(CrossMark), Error.Sprint(message))
}

// WarningBox prints a warning message
func WarningBox(message string) {
	fmt.Printf("%s %s\n", Warning.Sprint("⚠"), Warning.Sprint(message))
}

// QuickStartBox prints quick start instructions
func QuickStartBox() {
	fmt.Printf("%s %s\n", Accent2.Sprint(Rocket), Bold.Sprint("Quick Start"))
	
	commands := []struct {
		cmd  string
		desc string
	}{
		{"ccmodel list", "List all available AI models"},
		{"ccmodel current", "Show currently active model"}, 
		{"ccmodel switch <name>", "Switch to a different model"},
		{"ccmodel --help", "Show detailed help"},
	}
	
	for _, cmd := range commands {
		fmt.Printf("%s%s %-20s %s %s\n", 
			Indent, 
			Primary.Sprint(cmd.cmd),
			"",
			Muted.Sprint(Arrow),
			Muted.Sprint(cmd.desc))
	}
	fmt.Println()
}

// Separator prints a visual separator
func Separator() {
	fmt.Println(Muted.Sprint(strings.Repeat("─", 60)))
}

// Timestamp returns a formatted timestamp
func Timestamp() string {
	return Muted.Sprintf("[%s]", time.Now().Format("15:04:05"))
}