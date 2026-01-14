package config

import (
	"os"
	"regexp"
)

// envVarPattern matches ${VAR} or ${VAR:-default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// ExpandEnvVars expands environment variable references in the input string.
// Supports two formats:
//   - ${VAR} - replaced with the value of VAR, or empty string if not set
//   - ${VAR:-default} - replaced with VAR's value, or "default" if not set
func ExpandEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		defaultVal := ""
		if len(submatches) >= 3 {
			defaultVal = submatches[2]
		}

		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return defaultVal
	})
}

// ExpandEnvVarsBytes is a convenience wrapper for byte slices.
func ExpandEnvVarsBytes(input []byte) []byte {
	return []byte(ExpandEnvVars(string(input)))
}
