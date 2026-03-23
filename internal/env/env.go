package env

import (
	"regexp"

	"github.com/joho/godotenv"
)

var varPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Load reads a .env file and returns a map of variables.
// Returns an empty map if path is empty or the file doesn't exist.
func Load(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	vars, err := godotenv.Read(path)
	if err != nil {
		// Gracefully handle missing file
		return map[string]string{}, nil
	}
	return vars, nil
}

// Substitute replaces {{VAR}} placeholders in s with values from vars.
// Unknown variables are left intact.
func Substitute(s string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		key := match[2 : len(match)-2]
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	})
}
