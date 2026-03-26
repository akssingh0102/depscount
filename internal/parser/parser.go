package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/depscount/depscount/pkg/types"
)

// Parser extracts declared dependencies from a manifest.
type Parser interface {
	Detect(root string) bool
	Parse(root string) ([]types.Dependency, string, error)
	Ecosystem() string
}

// Discover finds a supported project root under path (file or directory).
func Discover(path string) (root string, p Parser, err error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if !st.IsDir() {
		path = filepath.Dir(path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	candidates := []Parser{
		NewNPM(),
		NewPyPI(),
	}
	for _, dir := range walkUp(abs) {
		for _, c := range candidates {
			if c.Detect(dir) {
				return dir, c, nil
			}
		}
	}
	return "", nil, os.ErrNotExist
}

func walkUp(start string) []string {
	var out []string
	cur := start
	for {
		out = append(out, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return out
}

// NormalizeName trims and lowercases for comparison where appropriate.
func NormalizeName(ecosystem, name string) string {
	switch ecosystem {
	case "npm":
		return strings.TrimSpace(name)
	default:
		return strings.TrimSpace(strings.ToLower(name))
	}
}
