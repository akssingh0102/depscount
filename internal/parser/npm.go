package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/depscount/depscount/pkg/types"
)

// NPM parses package.json (and prefers package-lock.json in resolver).
type NPM struct{}

func NewNPM() *NPM { return &NPM{} }

func (NPM) Ecosystem() string { return "npm" }

func (NPM) Detect(root string) bool {
	_, err := os.Stat(filepath.Join(root, "package.json"))
	return err == nil
}

func (NPM) Parse(root string) ([]types.Dependency, string, error) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, "", err
	}
	var m struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, "", err
	}
	var deps []types.Dependency
	for name, ver := range m.Dependencies {
		deps = append(deps, types.Dependency{
			Name:      name,
			Ecosystem: "npm",
			Version:   types.VersionConstraint{Raw: ver, Ecosystem: "npm"},
			IsDev:     false,
		})
	}
	for name, ver := range m.DevDependencies {
		deps = append(deps, types.Dependency{
			Name:      name,
			Ecosystem: "npm",
			Version:   types.VersionConstraint{Raw: ver, Ecosystem: "npm"},
			IsDev:     true,
		})
	}
	proj := m.Name
	if proj == "" {
		proj = filepath.Base(root)
	}
	return deps, proj, nil
}

// NpmLockPackage is one entry in package-lock.json "packages".
type NpmLockPackage struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Name         string            `json:"name"`
	Dependencies map[string]string `json:"dependencies"`
}

// NpmLockfile is package-lock.json top level.
type NpmLockfile struct {
	LockfileVersion int                       `json:"lockfileVersion"`
	Packages        map[string]NpmLockPackage `json:"packages"`
	Dependencies    map[string]any            `json:"dependencies"` // v1 tree
}

// ReadNpmLockfile loads package-lock.json if present.
func ReadNpmLockfile(root string) (*NpmLockfile, error) {
	p := filepath.Join(root, "package-lock.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var lock NpmLockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	return &lock, nil
}

// PackageNameFromLockPath derives npm package name from lockfile path key.
func PackageNameFromLockPath(lockPath string, meta NpmLockPackage) string {
	if meta.Name != "" {
		return meta.Name
	}
	if lockPath == "" {
		return ""
	}
	// path like node_modules/foo or node_modules/a/node_modules/b
	idx := strings.LastIndex(lockPath, "node_modules/")
	if idx < 0 {
		return lockPath
	}
	return lockPath[idx+len("node_modules/"):]
}

// FlattenNpmLock returns all resolved packages (path -> id components).
func FlattenNpmLock(lock *NpmLockfile) map[string]struct{ Name, Version string } {
	out := make(map[string]struct{ Name, Version string })
	if lock == nil {
		return out
	}
	if lock.LockfileVersion >= 2 && len(lock.Packages) > 0 {
		for pth, meta := range lock.Packages {
			if meta.Version == "" {
				continue
			}
			name := PackageNameFromLockPath(pth, meta)
			if name == "" {
				continue
			}
			out[pth] = struct{ Name, Version string }{Name: name, Version: meta.Version}
		}
		return out
	}
	walkLegacy(lock.Dependencies, "", out)
	return out
}

func walkLegacy(node map[string]any, prefix string, out map[string]struct{ Name, Version string }) {
	for name, v := range node {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		ver, _ := m["version"].(string)
		if ver != "" {
			key := prefix + "/node_modules/" + name
			if prefix == "" {
				key = "node_modules/" + name
			}
			out[key] = struct{ Name, Version string }{Name: name, Version: ver}
		}
		if children, ok := m["dependencies"].(map[string]any); ok && len(children) > 0 {
			childPrefix := prefix + "/node_modules/" + name
			if prefix == "" {
				childPrefix = "node_modules/" + name
			}
			walkLegacy(children, childPrefix, out)
		}
		if requires, ok := m["requires"].(map[string]any); ok {
			_ = requires
		}
	}
}

// DepthFromNpmPath approximates nesting depth from a lockfile path key.
func DepthFromNpmPath(pth string) int {
	if pth == "" || pth == "node_modules" {
		return 0
	}
	return strings.Count(pth, "node_modules/")
}
