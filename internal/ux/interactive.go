package ux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

const (
	MenuBoxWidth    = 38 // 菜单外框宽度（内容宽度+2，保证边框和内容对齐）
	MenuTitleWidth  = 30 // 标题内容宽度
	MenuOptionWidth = 35 // 菜单项内容宽度
)

// HolographicLoader creates a futuristic holographic loading effect
func HolographicLoader(text string, duration time.Duration) {
	frames := []string{
		"◢◣",
		"◤◥",
		"◢◣",
		"◤◥",
	}

	colors := []*color.Color{
		color.New(color.FgCyan),
		color.New(color.FgMagenta),
		color.New(color.FgYellow),
		color.New(color.FgGreen),
	}

	startTime := time.Now()
	i := 0

	for time.Since(startTime) < duration {
		frame := frames[i%len(frames)]
		c := colors[i%len(colors)]

		fmt.Printf("\r%s %s %s",
			c.Sprint(frame),
			text,
			c.Sprint(frame))

		time.Sleep(200 * time.Millisecond)
		i++
	}

	// Clear line
	fmt.Print("\r" + strings.Repeat(" ", len(text)+10) + "\r")

	// Success with holographic effect
	fmt.Printf("✨ %s %s\n",
		color.New(color.FgHiGreen, color.Bold).Sprint("✓"),
		color.New(color.FgHiCyan).Sprint(text+" Complete!"))
}

// NeuralNetworkLoader simulates a neural network activation
func NeuralNetworkLoader(layers []int, duration time.Duration) {
	startTime := time.Now()

	for time.Since(startTime) < duration {
		fmt.Print("\033[2J\033[H") // Clear screen

		// Draw neural network
		fmt.Println(color.New(color.FgHiCyan).Sprint("🧠 NEURAL NETWORK ACTIVATION"))
		fmt.Println()

		for i, neurons := range layers {
			prefix := fmt.Sprintf("Layer %d", i+1)
			fmt.Printf("%-10s", prefix)

			for j := 0; j < neurons; j++ {
				if time.Now().UnixNano()%3 == 0 {
					fmt.Print(color.New(color.FgHiGreen).Sprint("●"))
				} else if time.Now().UnixNano()%2 == 0 {
					fmt.Print(color.New(color.FgYellow).Sprint("◐"))
				} else {
					fmt.Print(color.New(color.FgHiBlack).Sprint("○"))
				}
				fmt.Print(" ")
			}
			fmt.Println()
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Print("\033[2J\033[H") // Clear screen
	fmt.Println(color.New(color.FgHiGreen).Sprint("🚀 Neural Network Ready!"))
}

// InteractiveMenu creates a cyberpunk-style interactive menu
func InteractiveMenu(title string, options []string) (int, error) {
	selected := 0
	for {
		fmt.Print("\033[2J\033[H") // Clear screen

		// Title with cyberpunk styling
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╔" + strings.Repeat("═", MenuBoxWidth) + "╗"))
		titleStr := padOrTruncate(title, MenuTitleWidth)
		fmt.Printf("║ %s %s%s ║\n", "🔮", titleStr, strings.Repeat(" ", MenuBoxWidth-2-2-runewidth.StringWidth(titleStr)))
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╠" + strings.Repeat("═", MenuBoxWidth) + "╣"))

		// Options
		for i, option := range options {
			prefix := "  "
			optStr := padOrTruncate(option, MenuOptionWidth)
			if i == selected {
				prefix = color.New(color.FgHiMagenta).Sprint("▶ ")
				optStr = color.New(color.FgHiWhite, color.Bold).Sprint(optStr)
			} else {
				optStr = color.New(color.FgWhite).Sprint(optStr)
			}
			// 2 for prefix, MenuOptionWidth for content, 剩余补空格
			pad := MenuBoxWidth - 2 - len([]rune(prefix)) - runewidth.StringWidth(stripColor(optStr))
			if pad < 0 {
				pad = 0
			}
			fmt.Printf("║ %s%s%s ║\n", prefix, optStr, strings.Repeat(" ", pad))
		}

		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╚" + strings.Repeat("═", MenuBoxWidth) + "╝"))
		fmt.Println()
		fmt.Println(color.New(color.FgHiBlack).Sprint("Use w/s/k/j or number, Enter to select, 'q' to quit"))

		// Get input
		var input string
		fmt.Scanln(&input)

		switch input {
		case "w", "k": // Up
			if selected > 0 {
				selected--
			}
		case "s", "j": // Down
			if selected < len(options)-1 {
				selected++
			}
		case "", " ": // Enter
			return selected, nil
		case "q": // Quit
			return -1, fmt.Errorf("cancelled")
		default:
			// Try to parse as number
			if num, err := strconv.Atoi(input); err == nil && num > 0 && num <= len(options) {
				return num - 1, nil
			}
		}
	}
}

// InteractiveMenuWithDesc renders menu with descriptions
func InteractiveMenuWithDesc(title string, options, descs []string) (int, error) {
	selected := 0
	for {
		fmt.Print("\033[2J\033[H") // Clear screen
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╔" + strings.Repeat("═", MenuBoxWidth) + "╗"))
		titleStr := padOrTruncate(title, MenuTitleWidth)
		fmt.Printf("║ %s %s%s ║\n", "🔮", titleStr, strings.Repeat(" ", MenuBoxWidth-2-2-runewidth.StringWidth(titleStr)))
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╠" + strings.Repeat("═", MenuBoxWidth) + "╣"))
		for i, option := range options {
			prefix := "  "
			optStr := padOrTruncate(option, MenuOptionWidth)
			desc := ""
			if i < len(descs) {
				desc = descs[i]
			}
			descStr := padOrTruncate(desc, MenuBoxWidth-2-len([]rune(prefix))-MenuOptionWidth-1)
			if i == selected {
				prefix = color.New(color.FgHiMagenta).Sprint("▶ ")
				optStr = color.New(color.FgHiWhite, color.Bold).Sprint(optStr)
				descStr = color.New(color.FgHiBlack).Sprint(descStr)
			} else {
				optStr = color.New(color.FgWhite).Sprint(optStr)
				descStr = color.New(color.FgHiBlack).Sprint(descStr)
			}
			pad := MenuBoxWidth - 2 - len([]rune(prefix)) - runewidth.StringWidth(stripColor(optStr)) - runewidth.StringWidth(stripColor(descStr))
			if pad < 0 {
				pad = 0
			}
			fmt.Printf("║ %s%s %s%s ║\n", prefix, optStr, descStr, strings.Repeat(" ", pad))
		}
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint("╚" + strings.Repeat("═", MenuBoxWidth) + "╝"))
		fmt.Println()
		fmt.Println(color.New(color.FgHiBlack).Sprint("Use w/s/k/j or number, Enter to select, 'q' to quit"))
		var input string
		fmt.Scanln(&input)
		switch input {
		case "w", "k":
			if selected > 0 {
				selected--
			}
		case "s", "j":
			if selected < len(options)-1 {
				selected++
			}
		case "", " ":
			return selected, nil
		case "q":
			return -1, fmt.Errorf("cancelled")
		default:
			if num, err := strconv.Atoi(input); err == nil && num > 0 && num <= len(options) {
				return num - 1, nil
			}
		}
	}
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

// stripColor removes ANSI color codes for width calculation
func stripColor(s string) string {
	var b strings.Builder
	inSeq := false
	for _, r := range s {
		if r == '\x1b' {
			inSeq = true
			continue
		}
		if inSeq {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inSeq = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// DataStreamEffect simulates data streaming like in The Matrix
func DataStreamEffect(duration time.Duration) {
	width := 80
	height := 20

	// Create streams
	streams := make([]struct {
		x, y, speed int
		chars       []rune
	}, 10)

	dataChars := []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*()_+-=[]{}|;:,.<>?")

	for i := range streams {
		streams[i] = struct {
			x, y, speed int
			chars       []rune
		}{
			x:     i * (width / len(streams)),
			y:     0,
			speed: 1 + (i % 3),
			chars: make([]rune, height),
		}

		// Initialize with random chars
		for j := range streams[i].chars {
			streams[i].chars[j] = dataChars[time.Now().UnixNano()%int64(len(dataChars))]
		}
	}

	startTime := time.Now()
	for time.Since(startTime) < duration {
		fmt.Print("\033[2J\033[H")

		// Update streams
		for i := range streams {
			streams[i].y += streams[i].speed
			if streams[i].y >= height*2 {
				streams[i].y = -height
			}

			// Update characters
			for j := range streams[i].chars {
				if time.Now().UnixNano()%5 == 0 {
					streams[i].chars[j] = dataChars[time.Now().UnixNano()%int64(len(dataChars))]
				}
			}
		}

		// Draw frame
		frame := make([][]rune, height)
		for i := range frame {
			frame[i] = make([]rune, width)
			for j := range frame[i] {
				frame[i][j] = ' '
			}
		}

		// Draw streams
		for _, stream := range streams {
			for i, char := range stream.chars {
				y := stream.y - len(stream.chars) + i
				if y >= 0 && y < height && stream.x >= 0 && stream.x < width {
					frame[y][stream.x] = char
				}
			}
		}

		// Print with colors
		for y, line := range frame {
			for _, char := range line {
				if char != ' ' {
					// Color based on position
					if y < height/4 {
						color.New(color.FgHiGreen, color.Bold).Print(string(char))
					} else if y < height/2 {
						color.New(color.FgGreen).Print(string(char))
					} else {
						color.New(color.FgHiBlack).Print(string(char))
					}
				} else {
					fmt.Print(" ")
				}
			}
			fmt.Println()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// CyberpunkBanner displays an epic cyberpunk banner
func CyberpunkBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════════════════════╗
║                                                                               ║
║   ██████╗ ██████╗███╗   ███╗ ██████╗ ██████╗ ███████╗██╗         ██████╗     ║
║  ██╔════╝██╔════╝████╗ ████║██╔═══██╗██╔══██╗██╔════╝██║        ██╔═████╗    ║
║  ██║     ██║     ██╔████╔██║██║   ██║██║  ██║█████╗  ██║        ██║██╔██║    ║
║  ██║     ██║     ██║╚██╔╝██║██║   ██║██║  ██║██╔══╝  ██║        ████╔╝██║    ║
║  ╚██████╗╚██████╗██║ ╚═╝ ██║╚██████╔╝██████╔╝███████╗███████╗██╗╚██████╔╝    ║
║   ╚═════╝ ╚═════╝╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝╚══════╝╚═╝ ╚═════╝     ║
║                                                                               ║
║                    🔮 NEURAL CONFIGURATION MATRIX 🔮                         ║
║                                                                               ║
╚═══════════════════════════════════════════════════════════════════════════════╝`

	lines := strings.Split(banner, "\n")
	colors := []*color.Color{
		color.New(color.FgHiCyan),
		color.New(color.FgHiMagenta),
		color.New(color.FgHiBlue),
		color.New(color.FgHiGreen),
	}

	for i, line := range lines {
		c := colors[i%len(colors)]
		c.Println(line)
		time.Sleep(50 * time.Millisecond)
	}
}

// QuantumSpinner creates a quantum-style loading effect
func QuantumSpinner(text string, duration time.Duration) {
	states := []string{"◐", "◓", "◑", "◒"}
	colors := []*color.Color{
		color.New(color.FgHiCyan),
		color.New(color.FgHiMagenta),
		color.New(color.FgHiYellow),
		color.New(color.FgHiGreen),
	}

	startTime := time.Now()
	i := 0

	for time.Since(startTime) < duration {
		state := states[i%len(states)]
		c := colors[i%len(colors)]

		// Quantum superposition effect
		superposition := ""
		for j := 0; j < 3; j++ {
			if time.Now().UnixNano()%2 == 0 {
				superposition += "⟨"
			} else {
				superposition += "⟩"
			}
		}

		fmt.Printf("\r%s %s %s %s",
			c.Sprint(superposition),
			c.Sprint(state),
			text,
			c.Sprint(superposition))

		time.Sleep(150 * time.Millisecond)
		i++
	}

	// Quantum collapse
	fmt.Print("\r" + strings.Repeat(" ", len(text)+20) + "\r")
	fmt.Printf("⚡ %s %s\n",
		color.New(color.FgHiYellow, color.Bold).Sprint("⟨ψ|Ready⟩"),
		color.New(color.FgHiCyan).Sprint(text))
}

// PromptUser creates an enhanced prompt with cyberpunk styling
func PromptUser(prompt string) string {
	fmt.Print(color.New(color.FgHiCyan).Sprint("┌─[") +
		color.New(color.FgHiMagenta, color.Bold).Sprint("ccmodel") +
		color.New(color.FgHiCyan).Sprint("]─[") +
		color.New(color.FgHiGreen).Sprint("user") +
		color.New(color.FgHiCyan).Sprint("]\n└─$ ") +
		color.New(color.FgWhite).Sprint(prompt+" "))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
