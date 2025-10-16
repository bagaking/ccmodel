package execcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func isInteractive() bool {
	if fi, err := os.Stdin.Stat(); err == nil {
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func promptSelect(prompt string, reader *bufio.Reader) (string, error) {
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}
	fmt.Fprint(os.Stdout, prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
