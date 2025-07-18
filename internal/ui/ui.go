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
	Primary   = color.New(color.FgHiCyan, color.Bold)
	Secondary = color.New(color.FgHiBlue)
	Success   = color.New(color.FgHiGreen, color.Bold)
	Warning   = color.New(color.FgHiYellow)
	Error     = color.New(color.FgHiRed, color.Bold)
	Muted     = color.New(color.FgHiBlack)
	
	// Accent colors
	Accent1   = color.New(color.FgHiMagenta)
	Accent2   = color.New(color.FgHiCyan)
	Accent3   = color.New(color.FgHiWhite)
	
	// Text styles
	Bold      = color.New(color.Bold)
	Italic    = color.New(color.Italic)
	Underline = color.New(color.Underline)
	Faint     = color.New(color.Faint)
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
	BoxElbow       = "└"
	
	// Modern bullets and separators
	Bullet         = "●"
	Arrow          = "▸"
	CheckMark      = "✓"
	CrossMark      = "✗"
	Lightning      = "⚡"
	Gear           = "⚙"
	Rocket         = "🚀"
	Diamond        = "◆"
	Circle         = "●"
	Star           = "★"
	Heart          = "♥"
	
	// Spacing
	Indent         = "  "
	DoubleIndent   = "    "
	TripleIndent   = "      "
)

// Header prints a stylized header
func Header(title, subtitle string) {
	width := 80
	
	// Gradient-like border effect
	fmt.Print(Primary.Sprint(BoxTopLeft))
	fmt.Print(Primary.Sprint(strings.Repeat(BoxHorizontal, width-2)))
	fmt.Println(Primary.Sprint(BoxTopRight))
	
	// Title line with enhanced styling
	titlePadding := (width - len(title) - 2) / 2
	fmt.Print(Primary.Sprint(BoxVertical))
	fmt.Print(strings.Repeat(" ", titlePadding))
	fmt.Print(Accent3.Sprint(strings.Repeat("▪", 2)))
	fmt.Print(" ")
	fmt.Print(Bold.Sprint(title))
	fmt.Print(" ")
	fmt.Print(Accent3.Sprint(strings.Repeat("▪", 2)))
	fmt.Print(strings.Repeat(" ", width-len(title)-titlePadding-6))
	fmt.Println(Primary.Sprint(BoxVertical))
	
	// Subtitle line with subtle styling
	if subtitle != "" {
		subtitlePadding := (width - len(subtitle) - 2) / 2
		fmt.Print(Primary.Sprint(BoxVertical))
		fmt.Print(strings.Repeat(" ", subtitlePadding))
		fmt.Print(Faint.Sprint(subtitle))
		fmt.Print(strings.Repeat(" ", width-len(subtitle)-subtitlePadding-2))
		fmt.Println(Primary.Sprint(BoxVertical))
	}
	
	// Bottom border with enhanced styling
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

// ModelEntry prints a formatted model entry with consistent alignment
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
	
	// Status indicator - use simple characters for better alignment
	var indicator string
	if isActive {
		indicator = "★"
	} else {
		indicator = "○"
	}
	
	// Ensure consistent width for name field (18 chars max)
	nameWidth := 18
	displayName := name
	if len([]rune(displayName)) > nameWidth {
		displayName = string([]rune(displayName)[:nameWidth-1]) + "…"
	}
	
	// Date formatting - ensure consistent width (10 chars)
	dateStr := formatDate(modified)
	if len(dateStr) > 10 {
		dateStr = dateStr[:10]
	}
	
	// Build the line with consistent spacing
	var line string
	if isActive {
		line = fmt.Sprintf("│ %s │ %s%s%-*s │ %6s │ %s │ ACTIVE",
			indexStr,
			statusIcon.Sprint(indicator),
			" ",
			nameWidth-1,
			nameColor.Sprint(displayName),
			Muted.Sprint(size),
			Muted.Sprint(dateStr))
	} else {
		line = fmt.Sprintf("│ %s │ %s%s%-*s │ %6s │ %s │",
			indexStr,
			statusIcon.Sprint(indicator),
			" ",
			nameWidth-1,
			nameColor.Sprint(displayName),
			Muted.Sprint(size),
			Muted.Sprint(dateStr))
	}
	
	fmt.Println(line)
}

// formatDate formats the date string for better readability
func formatDate(dateStr string) string {
	// Convert "2006-01-02 15:04" to "Jan 02 15:04"
	t, err := time.Parse("2006-01-02 15:04", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Jan 02 15:04")
}

// InfoBox prints an information box with enhanced styling
func InfoBox(title string, items []string) {
	fmt.Printf("%s %s\n", Accent1.Sprint(Diamond), Bold.Sprint(title))
	for _, item := range items {
		// Add emoji support and better formatting
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			fmt.Printf("%s%s %s%s\n", 
				Indent, 
				Muted.Sprint(Bullet), 
				Accent3.Sprint(parts[0]+":"), 
				parts[1])
		} else {
			fmt.Printf("%s%s %s\n", Indent, Muted.Sprint(Bullet), item)
		}
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