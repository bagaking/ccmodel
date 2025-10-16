package execcmd

import (
	"fmt"
	"os"
)

func (r *runner) logExecWarning(format string, args ...any) {
	if !r.deps.Verbose() {
		return
	}
	fmt.Fprintf(os.Stderr, "ccmodel exec: "+format+"\n", args...)
}
