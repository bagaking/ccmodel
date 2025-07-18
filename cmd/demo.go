package cmd

import (
	"fmt"
	"time"

	"github.com/bagaking/ccmodel/internal/ux"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Showcase advanced terminal effects and animations",
	Long:  `Demonstrate the advanced CLI animations and effects available in ccmodel`,
	RunE:  runDemo,
}

func init() {
	rootCmd.AddCommand(demoCmd)
}

func runDemo(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 ccmodel Advanced Terminal Effects Demo")
	fmt.Println()

	// 1. Loading Animations
	fmt.Println("1. Loading Animations:")
	showcaseLoaders()
	
	// 2. Progress Bars
	fmt.Println("\n2. Progress Bars:")
	showcaseProgressBars()
	
	// 3. Effects
	fmt.Println("\n3. Terminal Effects:")
	showcaseEffects()
	
	// 4. ASCII Art
	fmt.Println("\n4. ASCII Art:")
	ux.ASCIIArt()
	
	return nil
}

func showcaseLoaders() {
	loaders := []string{"dots", "circle", "arrows", "bounce", "pulse"}
	for _, loader := range loaders {
		spinner := ux.NewSpinner(loader)
		spinner.Start(fmt.Sprintf("Loading with %s style...", loader))
		time.Sleep(2 * time.Second)
		spinner.Success(fmt.Sprintf("Done with %s style!", loader))
	}
}

func showcaseProgressBars() {
	pb := ux.NewProgressBar(30)
	pb.SetTotal(100)
	pb.SetPrefix("Processing")
	pb.SetSuffix("models")
	
	for i := 0; i <= 100; i += 10 {
		pb.Update(i)
		time.Sleep(200 * time.Millisecond)
	}
	pb.Complete("All models processed!")
}

func showcaseEffects() {
	fmt.Print("Typewriter effect: ")
	ux.TypewriterEffect("Welcome to the future of CLI tools!", 50*time.Millisecond)
	
	fmt.Print("Matrix effect: ")
	ux.MatrixEffect("LOADING CC MODEL")
	
	fmt.Print("Glitch effect: ")
	ux.GlitchMatrix("SYSTEM READY", 1*time.Second)
	
	fmt.Print("Breathing effect: ")
	ux.BreathingEffect("ccmodel is alive...", 2*time.Second)
	
	fmt.Print("Rainbow text: ")
	ux.RainbowText("NEURAL NETWORKS LOADED")
}
