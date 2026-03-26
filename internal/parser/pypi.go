package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/depscount/depscount/pkg/types"
)

// PyPI parses requirements.txt, pyproject.toml, or uses Pipfile.lock in resolver.
type PyPI struct{}

func NewPyPI() *PyPI { return &PyPI{} }

func (PyPI) Ecosystem() string { return "pypi" }

func (PyPI) Detect(root string) bool {
	for _, f := range []string{
		"pyproject.toml",
		"requirements.txt",
		"Pipfile",
	} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

var reqLine = regexp.MustCompile(`^([a-zA-Z0-9_.\-]+)\s*(==|>=|<=|~=|!=|>|<)?\s*([^\s;#]+)?`)

func (PyPI) Parse(root string) ([]types.Dependency, string, error) {
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
		return parsePyproject(root)
	}
	if _, err := os.Stat(filepath.Join(root, "requirements.txt")); err == nil {
		return parseRequirementsTxt(root)
	}
	if _, err := os.Stat(filepath.Join(root, "Pipfile")); err == nil {
		return parsePipfile(root)
	}
	return nil, "", os.ErrNotExist
}

func parsePipfile(root string) ([]types.Dependency, string, error) {
	data, err := os.ReadFile(filepath.Join(root, "Pipfile"))
	if err != nil {
		return nil, "", err
	}
	var doc struct {
		Packages map[string]any `toml:"packages"`
		Dev      map[string]any `toml:"dev-packages"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, "", err
	}
	var deps []types.Dependency
	add := func(m map[string]any, dev bool) {
		for name, v := range m {
			raw := "*"
			switch t := v.(type) {
			case string:
				raw = t
			}
			deps = append(deps, types.Dependency{
				Name:      normalizePyPIName(name),
				Ecosystem: "pypi",
				Version:   types.VersionConstraint{Raw: raw, Ecosystem: "pypi"},
				IsDev:     dev,
			})
		}
	}
	add(doc.Packages, false)
	add(doc.Dev, true)
	return deps, filepath.Base(root), nil
}

func parsePyproject(root string) ([]types.Dependency, string, error) {
	data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return nil, "", err
	}
	var doc struct {
		Project struct {
			Name         string   `toml:"name"`
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Name         string            `toml:"name"`
				Dependencies map[string]string `toml:"dependencies"`
				Dev          map[string]string `toml:"dev-dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, "", err
	}
	var deps []types.Dependency
	proj := doc.Project.Name
	if proj == "" {
		proj = doc.Tool.Poetry.Name
	}
	for _, line := range doc.Project.Dependencies {
		d := parsePEP508Line(line, false)
		if d.Name != "" {
			deps = append(deps, d)
		}
	}
	for name, ver := range doc.Tool.Poetry.Dependencies {
		deps = append(deps, types.Dependency{
			Name:      normalizePyPIName(name),
			Ecosystem: "pypi",
			Version:   types.VersionConstraint{Raw: ver, Ecosystem: "pypi"},
			IsDev:     false,
		})
	}
	for name, ver := range doc.Tool.Poetry.Dev {
		deps = append(deps, types.Dependency{
			Name:      normalizePyPIName(name),
			Ecosystem: "pypi",
			Version:   types.VersionConstraint{Raw: ver, Ecosystem: "pypi"},
			IsDev:     true,
		})
	}
	if proj == "" {
		proj = filepath.Base(root)
	}
	return deps, proj, nil
}

func parseRequirementsTxt(root string) ([]types.Dependency, string, error) {
	f, err := os.Open(filepath.Join(root, "requirements.txt"))
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	var deps []types.Dependency
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		m := reqLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := normalizePyPIName(m[1])
		raw := line
		deps = append(deps, types.Dependency{
			Name:      name,
			Ecosystem: "pypi",
			Version:   types.VersionConstraint{Raw: raw, Ecosystem: "pypi"},
		})
	}
	return deps, filepath.Base(root), sc.Err()
}

func parsePEP508Line(line string, dev bool) types.Dependency {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return types.Dependency{}
	}
	// strip extras [foo]
	if i := strings.Index(line, "["); i > 0 {
		if j := strings.Index(line, "]"); j > i {
			line = line[:i] + line[j+1:]
		}
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return types.Dependency{}
	}
	name := normalizePyPIName(parts[0])
	return types.Dependency{
		Name:      name,
		Ecosystem: "pypi",
		Version:   types.VersionConstraint{Raw: line, Ecosystem: "pypi"},
		IsDev:     dev,
	}
}

func normalizePyPIName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// PipfileLockEntry is one locked package.
type PipfileLockEntry struct {
	Version string `json:"version"`
}

// ReadPipfileLock reads Pipfile.lock default packages.
func ReadPipfileLock(root string) (map[string]string, error) {
	p := filepath.Join(root, "Pipfile.lock")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Default map[string]PipfileLockEntry `json:"default"`
		Develop map[string]PipfileLockEntry `json:"develop"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for n, e := range doc.Default {
		if e.Version != "" {
			out[normalizePyPIName(n)] = stripPipenvVersion(e.Version)
		}
	}
	for n, e := range doc.Develop {
		if e.Version != "" {
			out[normalizePyPIName(n)] = stripPipenvVersion(e.Version)
		}
	}
	return out, nil
}

func stripPipenvVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "==")
	return v
}
