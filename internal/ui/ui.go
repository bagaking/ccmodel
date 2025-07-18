package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
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
	Accent1 = color.New(color.FgHiMagenta)
	Accent2 = color.New(color.FgHiCyan)
	Accent3 = color.New(color.FgHiWhite)

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
	Bullet    = "●"
	Arrow     = "▸"
	CheckMark = "✓"
	CrossMark = "✗"
	Lightning = "⚡"
	Gear      = "⚙"
	Rocket    = "🚀"
	Diamond   = "◆"
	Circle    = "●"
	Star      = "★"
	Heart     = "♥"

	// Spacing
	Indent       = "  "
	DoubleIndent = "    "
	TripleIndent = "      "
)

const TableWidth = 2 + 2 + 18 + 6 + 10 + 7 + 6*3 + 7 // 字段宽度+分隔符

// Header prints a stylized header
func Header(title, subtitle string) {
	width := TableWidth

	// 顶部边框
	fmt.Print(Primary.Sprint(BoxTopLeft))
	fmt.Print(Primary.Sprint(strings.Repeat(BoxHorizontal, width-2)))
	fmt.Println(Primary.Sprint(BoxTopRight))

	// Title line with enhanced styling
	titleWidth := runewidth.StringWidth(title)
	titlePadding := (width - titleWidth - 6) / 2 // 6 for ▪▪ + spaces
	fmt.Print(Primary.Sprint(BoxVertical))
	fmt.Print(strings.Repeat(" ", titlePadding))
	fmt.Print(Accent3.Sprint(strings.Repeat("▪", 2)))
	fmt.Print(" ")
	fmt.Print(Bold.Sprint(title))
	fmt.Print(" ")
	fmt.Print(Accent3.Sprint(strings.Repeat("▪", 2)))
	fmt.Print(strings.Repeat(" ", width-titleWidth-titlePadding-6))
	fmt.Println(Primary.Sprint(BoxVertical))

	// Subtitle line with subtle styling
	if subtitle != "" {
		subtitleWidth := runewidth.StringWidth(subtitle)
		subtitlePadding := (width - subtitleWidth - 2) / 2
		fmt.Print(Primary.Sprint(BoxVertical))
		fmt.Print(strings.Repeat(" ", subtitlePadding))
		fmt.Print(Faint.Sprint(subtitle))
		fmt.Print(strings.Repeat(" ", width-subtitleWidth-subtitlePadding-2))
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
	// 字段宽度定义
	indexWidth := 2
	indicatorWidth := 2
	nameWidth := 18
	sizeWidth := 6
	dateWidth := 10
	statusWidth := 7

	// Index
	indexStr := fmt.Sprintf("%d", index)
	indexStr = padOrTruncate(indexStr, indexWidth)

	// Indicator
	indicator := "○"
	if isActive {
		indicator = "★"
	}
	indicatorStr := padOrTruncate(indicator, indicatorWidth)

	// Name
	displayName := name
	if runewidth.StringWidth(displayName) > nameWidth {
		runes := []rune(displayName)
		w := 0
		for i, r := range runes {
			w += runewidth.RuneWidth(r)
			if w > nameWidth-1 {
				displayName = string(runes[:i]) + "…"
				break
			}
		}
	}
	displayName = padOrTruncate(displayName, nameWidth)

	// Size
	sizeStr := padOrTruncate(size, sizeWidth)

	// Date
	dateStr := formatDate(modified)
	if runewidth.StringWidth(dateStr) > dateWidth {
		dateStr = runewidth.Truncate(dateStr, dateWidth, "…")
	}
	dateStr = padOrTruncate(dateStr, dateWidth)

	// Status
	statusStr := ""
	if isActive {
		statusStr = "ACTIVE"
	}
	statusStr = padOrTruncate(statusStr, statusWidth)

	fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
		indexWidth, indexStr,
		indicatorWidth, indicatorStr,
		nameWidth, displayName,
		sizeWidth, sizeStr,
		dateWidth, dateStr,
		statusWidth, statusStr,
	)
}

// padOrTruncate 用 runewidth 计算宽度，左对齐补空格，超出截断
func padOrTruncate(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w == width {
		return s
	} else if w < width {
		return s + strings.Repeat(" ", width-w)
	} else {
		return runewidth.Truncate(s, width, "…")
	}
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
