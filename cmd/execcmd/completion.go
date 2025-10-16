package execcmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func execCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	candidates := listTargets()
	matches := make([]string, 0, len(candidates))
	lower := strings.ToLower(toComplete)
	for _, c := range candidates {
		if strings.HasPrefix(c, lower) {
			matches = append(matches, c)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}
