package execcmd

func init() {
	registerTarget("codex", execTargetConfig{
		DefaultBinary: "codex",
		EnvVar:        "CCMODEL_EXEC_CODEX",
	}, "code")
}
