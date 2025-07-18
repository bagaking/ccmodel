package ux

import (\t
	"fmt"
	"time"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

// Animation styles for different operations
var (
	LoadingDots    = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	LoadingCircle  = []string{"◐", "◓", "◑", "◒"}
	LoadingArrows  = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	LoadingBounce  = []string{"⠁", "⠂", "⠄", "⠂"}
	LoadingPulse   = []string{"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"}
	LoadingBlocks  = []string{"▖", "▘", "▝", "▗"}
	LoadingWaves   = []string{"▂", "▄", "▅", "▆", "▇", "▆", "▅", "▄"}
	
	// Colors for different states
	Cyan    = color.New(color.FgCyan)
	Green   = color.New(color.FgGreen)
	Yellow  = color.New(color.FgYellow)
	Red     = color.New(color.FgRed)
	Blue    = color.New(color.FgBlue)
	Magenta = color.New(color.FgMagenta)
)

// Spinner represents an animated spinner
type Spinner struct {
	frames []string
	color  *color.Color
	stop   chan bool
	text   string
}

// NewSpinner creates a new spinner with specified style
func NewSpinner(style string) *Spinner {
	var frames []string
	switch style {
	case "dots":
		frames = LoadingDots
	case "circle":
		frames = LoadingCircle
	case "arrows":
		frames = LoadingArrows
	case "bounce":
		frames = LoadingBounce
	case "pulse":
		frames = LoadingPulse
	case "blocks":
		frames = LoadingBlocks
	case "waves":
		frames = LoadingWaves
	default:
		frames = LoadingDots
	}
	
	return &Spinner{
		frames: frames,
		color:  Cyan,
		stop:   make(chan bool),
	}
}

// Start starts the spinner animation
func (s *Spinner) Start(text string) {
	s.text = text
	go func() {
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				frame := s.frames[i%len(s.frames)]
				fmt.Printf("\r%s %s", s.color.Sprint(frame), s.text)
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	close(s.stop)
	fmt.Print("\r")
	fmt.Print(strings.Repeat(" ", utf8.RuneCountInString(s.text)+2))
	fmt.Print("\r")
}

// Success finishes spinner with success symbol
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Printf("\r%s %s\n", Green.Sprint("✓"), message)
}

// Error finishes spinner with error symbol
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Printf("\r%s %s\n", Red.Sprint("✗"), message)
}

// Warning finishes spinner with warning symbol
func (s *Spinner) Warning(message string) {
	s.Stop()
	fmt.Printf("\r%s %s\n", Yellow.Sprint("⚠"), message)
}

// ProgressBar represents a progress bar
type ProgressBar struct {
	width     int
	current   int
	total     int
	prefix    string
	suffix    string
	completed bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		width:  width,
		prefix: "Progress",
		suffix: "",
	}
}

// SetTotal sets the total value
func (pb *ProgressBar) SetTotal(total int) {
	pb.total = total
}

// SetPrefix sets the prefix text
func (pb *ProgressBar) SetPrefix(prefix string) {
	pb.prefix = prefix
}

// SetSuffix sets the suffix text
func (pb *ProgressBar) SetSuffix(suffix string) {
	pb.suffix = suffix
}

// Update updates the progress
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pb.render()
}

// Complete marks the progress as complete
func (pb *ProgressBar) Complete(message string) {
	pb.current = pb.total
	pb.completed = true
	pb.render()
	fmt.Printf("\n%s %s\n", Green.Sprint("✓"), message)
}

// render displays the progress bar
func (pb *ProgressBar) render() {
	if pb.total == 0 {
		return
	}
	
	percentage := float64(pb.current) / float64(pb.total) * 100
	filledWidth := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", pb.width-filledWidth)
	
	fmt.Printf("\r%s [%s] %.1f%% %s (%d/%d)", 
		pb.prefix, 
		Blue.Sprint(bar), 
		percentage, 
		pb.suffix,
		pb.current,
		pb.total,
	)
}

// LoadingAnimation shows a loading sequence
func LoadingAnimation(duration time.Duration, message string) {
	spinner := NewSpinner("dots")
	spinner.Start(message)
	time.Sleep(duration)
	spinner.Success("Completed")
}

// MatrixEffect displays a matrix-like effect
func MatrixEffect(text string) {
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < 3; i++ {
		for j := 0; j < len(text); j++ {
			if j <= i {
				fmt.Print(Green.Sprint(string(text[j])))
			} else {
				fmt.Print(string(chars[time.Now().UnixNano()%int64(len(chars))]))
			}
		}
		fmt.Print("\r")
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
}

// TypewriterEffect types text character by character
func TypewriterEffect(text string, delay time.Duration) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(delay)
	}
	fmt.Println()
}

// GlitchEffect creates a glitch effect
func GlitchEffect(text string) {
	glitchChars := "▓▒░█▄▀▌▐▬▖▗▘▙▚▛▜▝▞▟"
	for i := 0; i < 5; i++ {
		for j := 0; j < len(text); j++ {
			if time.Now().UnixNano()%3 == 0 {
				fmt.Print(Red.Sprint(string(glitchChars[time.Now().UnixNano()%int64(len(glitchChars))])))
			} else {
				fmt.Print(string(text[j]))
			}
		}
		fmt.Print("\r")
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println(text)
}

// ASCIIArt displays ASCII art
func ASCIIArt() {
	art := `
    ╭─────────────────────────────────────────────────────────────────╮
    │                                                                 │
    │   ██████╗ ██████╗ ███╗   ███╗ ██████╗ ██████╗ ███████╗        │
    │  ██╔════╝██╔═══██╗████╗ ████║██╔═══██╗██╔══██╗██╔════╝        │
    │  ██║     ██║   ██║██╔████╔██║██║   ██║██████╔╝█████╗          │
    │  ██║     ██║   ██║██║╚██╔╝██║██║   ██║██╔══██╗██╔══╝          │
    │  ╚██████╗╚██████╔╝██║ ╚═╝ ██║╚██████╔╝██████╔╝███████╗        │
    │   ╚═════╝ ╚═════╝ ╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝        │
    │                                                                 │
    ╰─────────────────────────────────────────────────────────────────╯
	`
	Cyan.Println(art)
}

// RainbowText displays rainbow colored text
func RainbowText(text string) {
	colors := []*color.Color{
		color.New(color.FgRed),
		color.New(color.FgYellow),
		color.New(color.FgGreen),
		color.New(color.FgCyan),
		color.New(color.FgBlue),
		color.New(color.FgMagenta),
	}
	
	for i, char := range text {
		if char != ' ' {
			colors[i%len(colors)].Print(string(char))
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Println()
}