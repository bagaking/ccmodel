package execcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Dependencies bundles the external collaborators required by exec commands.
type Dependencies struct {
	ConfigDir       func() string
	Verbose         func() bool
	GetCurrentModel func() (string, error)
	FileChecksum    func(string) (string, error)
	Now             func() time.Time
	LookPath        func(string) (string, error)
	Exit            func(int)
}

type runner struct {
	deps   Dependencies
	runCmd *cobra.Command
}

func newRunner(deps Dependencies) *runner {
	if deps.ConfigDir == nil {
		panic("execcmd: ConfigDir provider is required")
	}
	if deps.GetCurrentModel == nil {
		panic("execcmd: GetCurrentModel is required")
	}
	if deps.FileChecksum == nil {
		panic("execcmd: FileChecksum is required")
	}
	if deps.Verbose == nil {
		deps.Verbose = func() bool { return false }
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.LookPath == nil {
		deps.LookPath = execLookPath
	}
	if deps.Exit == nil {
		deps.Exit = exitFunc
	}
	return &runner{deps: deps}
}

// NewCommand returns the root exec command with subcommands.
func NewCommand(deps Dependencies) *cobra.Command {
	r := newRunner(deps)

	root := &cobra.Command{
		Use:   "exec",
		Short: "Proxy execution for Claude/Codex CLIs with session tracking",
		Long:  "Run Claude or Codex CLI instances under ccmodel with session records, tmux integration, and monitoring helpers.",
		RunE:  r.runLegacy,
	}

	runCmd := r.buildRunCommand()
	r.runCmd = runCmd

	root.AddCommand(
		runCmd,
		r.buildResumeCommand(),
		r.buildWatchCommand(),
		r.buildStatusCommand(),
		r.buildAttachCommand(),
	)

	root.ValidArgsFunction = r.completeRoot

	return root
}

func (r *runner) completeRoot(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		names := []string{"run", "resume", "watch", "status", "attach"}
		matches := make([]string, 0, len(names))
		for _, n := range names {
			if strings.HasPrefix(n, strings.ToLower(toComplete)) {
				matches = append(matches, n)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// runLegacy preserves the v1 syntax: `ccmodel exec <target> [-- <args>]`.
func (r *runner) runLegacy(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		_ = cmd.Help()
		return nil
	}

	if args[0] == "-h" || args[0] == "--help" {
		return cmd.Help()
	}

	targetName := args[0]
	if strings.HasPrefix(targetName, "-") {
		_ = cmd.Help()
		return fmt.Errorf("unknown target %q (expected claude or codex)", targetName)
	}

	targetArgs := []string{}
	if len(args) > 1 {
		if args[1] == "--" {
			targetArgs = append(targetArgs, args[2:]...)
		} else {
			targetArgs = append(targetArgs, args[1:]...)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	sessionName, err := deriveSessionName(absDir)
	if err != nil {
		return err
	}
	if err := r.maybeRestoreSession(cmd, sessionName, absDir); err != nil {
		return err
	}

	exitCode, runErr := r.executeRun(targetName, targetArgs, runOptions{
		UseTmux:             true,
		AllowDirectFallback: true,
		AttachBehavior:      attachAuto,
		WorkingDir:          absDir,
		SessionName:         sessionName,
	})
	if runErr != nil {
		return runErr
	}
	if exitCode != 0 {
		r.deps.Exit(exitCode)
	}
	return nil
}

func (r *runner) buildRunCommand() *cobra.Command {
	var (
		useTmux = true
		noTmux  bool
		detach  bool
		name    string
		dirFlag string
		envVars []string
	)

	cmd := &cobra.Command{
		Use:                   "run <claude|codex> [-- <args>]",
		Short:                 "Run a Claude/Codex CLI instance with session tracking",
		Long:                  "Launch the native Claude or Codex CLI while recording session metadata. tmux is used by default to host the process so multiple instances can co-exist.",
		DisableFlagsInUseLine: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("target required: use 'claude' or 'codex'")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			targetName := args[0]
			targetArgs := []string{}
			if len(args) > 1 {
				targetArgs = append(targetArgs, args[1:]...)
			}

			workingDir := strings.TrimSpace(dirFlag)
			if workingDir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				workingDir = wd
			}
			absDir, err := filepath.Abs(workingDir)
			if err != nil {
				return err
			}
			sessionName, err := deriveSessionName(absDir)
			if err != nil {
				return err
			}
			if useTmux && !noTmux {
				if err := r.maybeRestoreSession(cmd, sessionName, absDir); err != nil {
					return err
				}
			}

			extraEnv := make(map[string]string)
			for _, raw := range envVars {
				parts := strings.SplitN(raw, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --env %q (expected KEY=VALUE)", raw)
				}
				key := strings.TrimSpace(parts[0])
				if key == "" {
					return fmt.Errorf("invalid --env %q (missing key)", raw)
				}
				extraEnv[key] = parts[1]
			}

			opts := runOptions{
				UseTmux:             useTmux && !noTmux,
				AllowDirectFallback: true,
				WindowName:          strings.TrimSpace(name),
				WorkingDir:          absDir,
				SessionName:         sessionName,
				ExtraEnv:            extraEnv,
			}
			if detach {
				opts.AttachBehavior = attachNever
			} else {
				opts.AttachBehavior = attachAuto
			}

			exitCode, err := r.executeRun(targetName, targetArgs, opts)
			if err != nil {
				return err
			}
			if exitCode != 0 {
				r.deps.Exit(exitCode)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&useTmux, "tmux", true, "use tmux to manage the spawned CLI (default)")
	cmd.Flags().BoolVar(&noTmux, "no-tmux", false, "disable tmux even if available")
	cmd.Flags().BoolVar(&detach, "detach", false, "run without switching/attaching to the new tmux window")
	cmd.Flags().StringVar(&name, "name", "", "explicit tmux window name (defaults to target and directory)")
	cmd.Flags().StringVar(&dirFlag, "dir", "", "working directory to associate with the session")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "environment variables (KEY=VALUE) to inject into the target process")

	cmd.ValidArgsFunction = execCompletion

	return cmd
}

// Placeholder implementations for the additional subcommands.
func (r *runner) buildResumeCommand() *cobra.Command {
	var (
		dir    string
		target string
		detach bool
	)

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Restore tmux-backed sessions for the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.resumeSessions(cmd, resumeOptions{
				Dir:    dir,
				Target: strings.TrimSpace(target),
				Detach: detach,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "directory to match against recorded sessions")
	cmd.Flags().StringVar(&target, "target", "", "limit to a specific target (claude or codex)")
	cmd.Flags().BoolVar(&detach, "detach", false, "restore sessions without attaching to the tmux session afterwards")

	return cmd
}

func (r *runner) buildWatchCommand() *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch Codex/Claude logs for active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode == "" {
				mode = "auto"
			}
			return r.watchLogs(cmd, strings.ToLower(mode))
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "auto", "watch mode: auto, tail, or multitail")
	cmd.Flags().Bool("simple", false, "alias for --mode tail")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if simple, _ := cmd.Flags().GetBool("simple"); simple {
			mode = "tail"
		}
	}

	return cmd
}

func (r *runner) buildStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [logs]",
		Short: "Show tmux sessions (with window breadcrumbs) or log status",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 && args[0] == "logs" {
				return nil
			}
			return fmt.Errorf("unknown status scope %q (expected logs)", strings.Join(args, " "))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := ""
			if len(args) > 0 {
				scope = args[0]
			}
			return r.printStatus(cmd, scope)
		},
	}
	return cmd
}

func (r *runner) buildAttachCommand() *cobra.Command {
	var windowFlag string
	cmd := &cobra.Command{
		Use:   "attach [window]",
		Short: "Attach to the tmux session or switch to a specific exec-managed window",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetWindow := windowFlag
			if len(args) > 0 {
				targetWindow = args[0]
			}
			return r.attachToManagedSession(cmd, targetWindow)
		},
	}
	cmd.Flags().StringVar(&windowFlag, "window", "", "window name or index to focus after attaching")
	return cmd
}
