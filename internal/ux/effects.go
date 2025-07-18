package ux

import (
	"fmt"
	"math"

	"math/rand"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Terminal effects for a premium CLI experience

// WaveEffect creates a wave animation across the terminal
func WaveEffect(text string, duration time.Duration) {
	width := 80
	height := 5
	startTime := time.Now()

	for time.Since(startTime) < duration {
		frame := make([]string, height)
		for i := range frame {
			frame[i] = strings.Repeat(" ", width)
		}

		// Create wave pattern
		for x := 0; x < len(text) && x < width; x++ {
			y := int(2 + 1.5*math.Sin(float64(x)*0.5+float64(time.Since(startTime).Milliseconds())*0.01))
			if y >= 0 && y < height {
				row := []rune(frame[y])
				if x < len(row) {
					row[x] = rune(text[x%len(text)])
					frame[y] = string(row)
				}
			}
		}

		// Clear screen and print frame
		fmt.Print("\033[2J\033[H")
		for _, line := range frame {
			Green.Println(line)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Reset cursor position
	fmt.Print("\033[H")
}

// ParticleSystem creates floating particles
func ParticleSystem(duration time.Duration) {
	width := 80
	height := 10
	particles := make([]struct{ x, y, vx, vy int }, 15)

	// Initialize particles
	for i := range particles {
		particles[i] = struct{ x, y, vx, vy int }{
			x:  rand.Intn(width),
			y:  rand.Intn(height),
			vx: rand.Intn(3) - 1,
			vy: rand.Intn(3) - 1,
		}
	}

	startTime := time.Now()
	for time.Since(startTime) < duration {
		frame := make([][]rune, height)
		for i := range frame {
			frame[i] = []rune(strings.Repeat(" ", width))
		}

		// Update and draw particles
		for i, p := range particles {
			p.x += p.vx
			p.y += p.vy

			// Boundary checking
			if p.x < 0 || p.x >= width {
				p.vx = -p.vx
				p.x = max(0, min(width-1, p.x))
			}
			if p.y < 0 || p.y >= height {
				p.vy = -p.vy
				p.y = max(0, min(height-1, p.y))
			}

			particles[i] = p

			// Draw particle
			if p.y >= 0 && p.y < height && p.x >= 0 && p.x < width {
				chars := []rune{'•', '◦', '·', '∙'}
				frame[p.y][p.x] = chars[rand.Intn(len(chars))]
			}
		}

		// Clear and print
		fmt.Print("\033[2J\033[H")
		for _, line := range frame {
			Cyan.Println(string(line))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// MatrixRain creates a matrix-style rain effect
func MatrixRain(duration time.Duration) {
	width, height := 80, 15
	chars := "アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン"

	drops := make([]struct{ x, y, speed int }, width)
	for i := range drops {
		drops[i] = struct{ x, y, speed int }{
			x:     i,
			y:     rand.Intn(height),
			speed: 1 + rand.Intn(3),
		}
	}

	startTime := time.Now()
	for time.Since(startTime) < duration {
		frame := make([][]rune, height)
		for i := range frame {
			frame[i] = []rune(strings.Repeat(" ", width))
		}

		// Update and draw drops
		for i, drop := range drops {
			drop.y += drop.speed
			if drop.y >= height {
				drop.y = 0
				drop.x = rand.Intn(width)
			}
			drops[i] = drop

			for y := 0; y < height; y++ {
				if y >= drop.y {
					char := rune(chars[rand.Intn(len(chars))])
					frame[y][drop.x] = char
				}
			}
		}

		fmt.Print("\033[2J\033[H")
		for _, line := range frame {
			Green.Println(string(line))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// BreathingEffect creates a breathing pulse
func BreathingEffect(text string, duration time.Duration) {
	startTime := time.Now()
	for time.Since(startTime) < duration {
		// Create breathing effect with colors
		c := color.New(color.FgGreen)

		fmt.Print("\033[2K\r")
		c.Printf("%s", text)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println()
}

// GlitchMatrix creates a glitchy matrix effect
func GlitchMatrix(text string, duration time.Duration) {
	glitchChars := "$#@!%^*&*()_+-=[]{}|;:,.<>?"
	startTime := time.Now()

	for time.Since(startTime) < duration {
		fmt.Print("\033[2K\r")

		glitched := ""
		for _, char := range text {
			if rand.Float32() < 0.1 {
				glitched += string(glitchChars[rand.Intn(len(glitchChars))])
			} else {
				glitched += string(char)
			}
		}

		if rand.Float32() < 0.3 {
			Red.Printf("%s", glitched)
		} else {
			Green.Printf("%s", glitched)
		}

		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
}

// ScanlineEffect creates a CRT scanline effect
func ScanlineEffect(text string) {
	lines := strings.Split(text, "\n")
	faint := color.New(color.Faint)
	for i, line := range lines {
		// Alternate scanline brightness
		if i%2 == 0 {
			faint.Printf("%s\n", line)
		} else {
			fmt.Println(line)
		}
	}
}

// ASCIIProgressBar creates a retro ASCII progress bar
func ASCIIProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	empty := width - filled

	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
	percentage := fmt.Sprintf("%.1f%%", progress*100)

	blue := color.New(color.FgBlue)
	green := color.New(color.FgGreen)
	return fmt.Sprintf("%s %s", blue.Sprint(bar), green.Sprint(percentage))
}

// RetroTerminal creates a retro terminal effect
func RetroTerminal(text string) {
	// CRT-style flicker
	green := color.New(color.FgGreen)
	faint := color.New(color.Faint)
	for i := 0; i < 3; i++ {
		fmt.Print("\033[2K\r")
		green.Printf("%s", text)
		time.Sleep(50 * time.Millisecond)

		fmt.Print("\033[2K\r")
		faint.Printf("%s", text)
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println()
}

// HexDumpEffect displays data like a hex dump
func HexDumpEffect(data []byte) {
	for i := 0; i < len(data); i += 16 {
		fmt.Print("\033[2K\r")
		fmt.Printf("%08x  ", i)

		// Hex bytes
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Print("   ")
			}
		}

		// ASCII representation
		fmt.Print(" |")
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Print("|")

		time.Sleep(100 * time.Millisecond)
		fmt.Println()
	}
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
