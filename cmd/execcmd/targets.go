package execcmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// execTargetConfig describes how to resolve a proxy target.
type execTargetConfig struct {
	DefaultBinary string
	EnvVar        string
}

// execTarget represents a resolved proxy target.
type execTarget struct {
	Name        string
	Binary      string
	DisplayName string
	EnvVar      string
}

var (
	targetAliases  = map[string]string{}
	targetDefaults = map[string]execTargetConfig{}
	targetOrder    = []string{}
)

func registerTarget(name string, cfg execTargetConfig, aliases ...string) {
	canonical := strings.ToLower(strings.TrimSpace(name))
	if canonical == "" {
		panic("execcmd: registerTarget requires a name")
	}

	targetDefaults[canonical] = cfg
	if !contains(targetOrder, canonical) {
		targetOrder = append(targetOrder, canonical)
	}

	if len(aliases) == 0 {
		aliases = []string{name}
	}
	aliases = append(aliases, canonical)

	for _, alias := range aliases {
		normalized := strings.ToLower(strings.TrimSpace(alias))
		if normalized == "" {
			continue
		}
		targetAliases[normalized] = canonical
	}
}

func listTargets() []string {
	names := append([]string(nil), targetOrder...)
	sort.Strings(names)
	return names
}

func resolveExecTarget(name string, lookPath func(string) (string, error)) (*execTarget, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return nil, errors.New("empty target")
	}

	key, ok := targetAliases[normalized]
	if !ok {
		return nil, fmt.Errorf("unknown target %q (expected %s)", name, strings.Join(listTargets(), " or "))
	}

	cfg, ok := targetDefaults[key]
	if !ok {
		return nil, fmt.Errorf("no configuration for target %q", key)
	}

	commandName := cfg.DefaultBinary
	if envValue := strings.TrimSpace(os.Getenv(cfg.EnvVar)); envValue != "" {
		commandName = envValue
	}

	resolvedPath, err := lookPath(commandName)
	if err != nil {
		return nil, fmt.Errorf("executable for %s not found (looked for %q). Set %s to override", key, commandName, cfg.EnvVar)
	}

	return &execTarget{
		Name:        key,
		Binary:      resolvedPath,
		DisplayName: commandName,
		EnvVar:      cfg.EnvVar,
	}, nil
}

func contains(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}
