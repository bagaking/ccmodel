package execcmd

func init() {
	registerTarget("claude", execTargetConfig{
		DefaultBinary: "claude",
		EnvVar:        "CCMODEL_EXEC_CLAUDE",
	})
}
