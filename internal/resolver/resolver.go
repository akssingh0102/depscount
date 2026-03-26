package resolver

import (
	"strings"

	"github.com/depscount/depscount/internal/parser"
	"github.com/depscount/depscount/pkg/types"
)

// Resolve builds a dependency graph from project root using lockfiles when present.
func Resolve(root string, p parser.Parser, declared []types.Dependency, projectName string) (*types.DependencyGraph, error) {
	switch p.Ecosystem() {
	case "npm":
		return resolveNPM(root, declared, projectName)
	case "pypi":
		return resolvePyPI(root, declared, projectName)
	default:
		g := NewGraph(projectName)
		return g, nil
	}
}

func directNameSet(declared []types.Dependency, includeDev bool) map[string]struct{} {
	m := make(map[string]struct{})
	for _, d := range declared {
		if d.IsDev && !includeDev {
			continue
		}
		m[d.Name] = struct{}{}
	}
	return m
}

func resolveNPM(root string, declared []types.Dependency, project string) (*types.DependencyGraph, error) {
	g := NewGraph(project)
	lock, err := parser.ReadNpmLockfile(root)
	if err != nil {
		// manifest-only: direct deps with unresolved semver ranges (limited OSV accuracy)
		direct := directNameSet(declared, true)
		for _, d := range declared {
			if _, ok := direct[d.Name]; !ok {
				continue
			}
			ver := d.Version.Raw
			if ver == "" {
				ver = "0.0.0"
			}
			id := GraphKey("npm", d.Name, ver)
			pkg := &types.Package{
				ID: id, Name: d.Name, Version: ver, Ecosystem: "npm",
				Depth: 0, IsDirect: true, MetadataPartial: true,
			}
			AddNode(g, pkg)
		}
		return g, nil
	}
	flat := parser.FlattenNpmLock(lock)
	direct := directNameSet(declared, true)
	seen := make(map[string]*types.Package)
	for path, nv := range flat {
		id := GraphKey("npm", nv.Name, nv.Version)
		depth := parser.DepthFromNpmPath(path)
		isDirect := false
		if _, ok := direct[nv.Name]; ok && depth <= 1 && path != "" {
			isDirect = true
		}
		pkg := &types.Package{
			ID: id, Name: nv.Name, Version: nv.Version, Ecosystem: "npm",
			Depth: depth, IsDirect: isDirect && depth <= 1,
		}
		if _, ok := seen[id]; !ok {
			seen[id] = pkg
			AddNode(g, pkg)
		}
	}
	// Edges from lockfile packages field (v2+)
	if lock.LockfileVersion >= 2 && len(lock.Packages) > 0 {
		for pth, meta := range lock.Packages {
			if meta.Version == "" {
				continue
			}
			selfName := parser.PackageNameFromLockPath(pth, meta)
			if selfName == "" {
				continue
			}
			parentID := GraphKey("npm", selfName, meta.Version)
			if g.Nodes[parentID] == nil {
				continue
			}
			for depName := range meta.Dependencies {
				childPath := resolveNpmChildPath(lock.Packages, pth, depName)
				if childPath == "" {
					continue
				}
				cm := lock.Packages[childPath]
				if cm.Version == "" {
					continue
				}
				cn := parser.PackageNameFromLockPath(childPath, cm)
				cid := GraphKey("npm", cn, cm.Version)
				if g.Nodes[cid] != nil {
					AddEdge(g, parentID, cid)
				}
			}
		}
	}
	return g, nil
}

func resolveNpmChildPath(packages map[string]parser.NpmLockPackage, parentPath, dep string) string {
	try := func(base string) string {
		if base == "" {
			return "node_modules/" + dep
		}
		return base + "/node_modules/" + dep
	}
	for cur := parentPath; ; {
		cand := try(cur)
		if _, ok := packages[cand]; ok {
			return cand
		}
		if cur == "" {
			break
		}
		next := parentModulesPath(cur)
		if next == cur {
			break
		}
		cur = next
	}
	// search any path ending with /node_modules/dep
	suffix := "/node_modules/" + dep
	var best string
	for k := range packages {
		if len(k) >= len(suffix) && k[len(k)-len(suffix):] == suffix {
			if best == "" || len(k) < len(best) {
				best = k
			}
		}
	}
	return best
}

func parentModulesPath(p string) string {
	if p == "" {
		return ""
	}
	base := strings.TrimPrefix(p, "node_modules/")
	if idx := strings.LastIndex(base, "/node_modules/"); idx >= 0 {
		return "node_modules/" + base[:idx]
	}
	return ""
}

func resolvePyPI(root string, declared []types.Dependency, project string) (*types.DependencyGraph, error) {
	g := NewGraph(project)
	lock, err := parser.ReadPipfileLock(root)
	if err == nil && len(lock) > 0 {
		direct := directNameSet(declared, true)
		for name, ver := range lock {
			_, isDirect := direct[name]
			pkg := &types.Package{
				ID: GraphKey("pypi", name, ver), Name: name, Version: ver, Ecosystem: "pypi",
				Depth: ternary(isDirect, 0, 1), IsDirect: isDirect,
			}
			AddNode(g, pkg)
		}
		return g, nil
	}
	// requirements / pyproject: only pinned direct deps get resolved versions
	direct := directNameSet(declared, true)
	for _, d := range declared {
		ver := extractPinnedVersion(d.Version.Raw)
		if ver == "" {
			continue
		}
		if _, ok := direct[d.Name]; !ok {
			continue
		}
		pkg := &types.Package{
			ID: GraphKey("pypi", d.Name, ver), Name: d.Name, Version: ver, Ecosystem: "pypi",
			Depth: 0, IsDirect: true,
		}
		AddNode(g, pkg)
	}
	if len(g.Nodes) == 0 {
		// unpinned: still list names with placeholder for discovery
		for _, d := range declared {
			if d.IsDev {
				continue
			}
			ver := "0.0.0"
			pkg := &types.Package{
				ID: GraphKey("pypi", d.Name, ver), Name: d.Name, Version: ver, Ecosystem: "pypi",
				Depth: 0, IsDirect: true, MetadataPartial: true,
			}
			AddNode(g, pkg)
		}
	}
	return g, nil
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func extractPinnedVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "=="); i >= 0 {
		v := strings.TrimSpace(raw[i+2:])
		for _, sep := range []string{" ", ";", "#"} {
			if j := strings.IndexAny(v, sep); j >= 0 {
				v = strings.TrimSpace(v[:j])
			}
		}
		return strings.TrimSpace(v)
	}
	return ""
}
