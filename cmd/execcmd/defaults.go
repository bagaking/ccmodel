package execcmd

import (
	"os"
	"os/exec"
)

var (
	execLookPath = exec.LookPath
	exitFunc     = os.Exit
)
