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
	// Commands are added in root.go
}

func runDemo(cmd *cobra.Command, args []string) error {
	// Epic cyberpunk banner
	ux.CyberpunkBanner()
	time.Sleep(2 * time.Second)
	
	// Interactive menu for demo selection
	options := []string{
		"🌀 Holographic Loaders",
		"🧠 Neural Network Simulation", 
		"📊 Quantum Progress Bars",
		"🔮 Matrix Data Stream",
		"⚡ Loading Animations Showcase",
		"🎨 All Effects Combined",
		"🚀 Exit to Main System",
	}
	
	for {
		fmt.Print("\033[2J\033[H") // Clear screen
		selected, err := ux.InteractiveMenu("NEURAL DEMO SYSTEM", options)
		if err != nil || selected == 6 {
			fmt.Println("🚀 Exiting neural demo system...")
			return nil
		}
		
		switch selected {
		case 0:
			showcaseHolographicLoaders()
		case 1:
			showcaseNeuralNetwork()
		case 2:
			showcaseQuantumProgress()
		case 3:
			showcaseMatrixStream()
		case 4:
			showcaseLoaders()
		case 5:
			showcaseAllEffects()
		}
		
		fmt.Println("\nPress Enter to return to menu...")
		fmt.Scanln()
	}
}

func showcaseHolographicLoaders() {
	fmt.Print("\033[2J\033[H")
	fmt.Println("🌀 HOLOGRAPHIC LOADING SYSTEMS")
	fmt.Println()
	
	effects := []string{
		"Initializing quantum entanglement...",
		"Calibrating neural pathways...", 
		"Synchronizing reality matrices...",
		"Loading consciousness patterns...",
	}
	
	for _, effect := range effects {
		ux.HolographicLoader(effect, 3*time.Second)
		time.Sleep(500 * time.Millisecond)
	}
}

func showcaseNeuralNetwork() {
	fmt.Print("\033[2J\033[H")
	fmt.Println("🧠 NEURAL NETWORK SIMULATION")
	fmt.Println()
	
	// Different network architectures
	networks := [][]int{
		{8, 16, 8, 4},    // Simple network
		{12, 24, 12, 6},  // Medium network
		{16, 32, 16, 8},  // Complex network
	}
	
	for i, network := range networks {
		fmt.Printf("Activating Network Architecture %d...\n", i+1)
		ux.NeuralNetworkLoader(network, 4*time.Second)
		time.Sleep(1 * time.Second)
	}
}

func showcaseQuantumProgress() {
	fmt.Print("\033[2J\033[H") 
	fmt.Println("📊 QUANTUM PROGRESS SYSTEMS")
	fmt.Println()
	
	tasks := []string{
		"Quantum state preparation",
		"Superposition calibration", 
		"Entanglement verification",
		"Quantum error correction",
	}
	
	for _, task := range tasks {
		ux.QuantumSpinner(task, 3*time.Second)
		time.Sleep(500 * time.Millisecond)
	}
}

func showcaseMatrixStream() {
	fmt.Print("\033[2J\033[H")
	fmt.Println("🔮 MATRIX DATA STREAM")
	fmt.Println("Entering the Matrix...")
	time.Sleep(2 * time.Second)
	
	ux.DataStreamEffect(8 * time.Second)
	
	fmt.Print("\033[2J\033[H")
	fmt.Println("🚀 Welcome to the real world.")
}

func showcaseAllEffects() {
	fmt.Print("\033[2J\033[H")
	fmt.Println("🎨 ULTIMATE EFFECTS SHOWCASE")
	fmt.Println()
	
	// Combined spectacular show
	ux.TypewriterEffect("Initializing ultimate demo sequence...", 30*time.Millisecond)
	time.Sleep(1 * time.Second)
	
	ux.MatrixEffect("LOADING CCMODEL")
	time.Sleep(1 * time.Second)
	
	ux.HolographicLoader("Reality engine startup", 2*time.Second)
	
	ux.QuantumSpinner("Quantum consciousness sync", 2*time.Second)
	
	fmt.Println()
	ux.RainbowText("🌈 CCMODEL NEURAL MATRIX ONLINE 🌈")
	fmt.Println()
	
	ux.BreathingEffect("System fully operational...", 3*time.Second)
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
