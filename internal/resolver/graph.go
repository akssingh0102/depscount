package resolver

import (
	"fmt"
	"strings"

	"github.com/depscount/depscount/pkg/types"
)

// GraphKey returns canonical node id.
func GraphKey(ecosystem, name, version string) string {
	return fmt.Sprintf("%s/%s@%s", ecosystem, strings.TrimSpace(name), strings.TrimSpace(version))
}

// NewGraph builds an empty graph.
func NewGraph(project string) *types.DependencyGraph {
	return &types.DependencyGraph{
		Nodes:   make(map[string]*types.Package),
		Edges:   make(map[string][]string),
		Project: project,
	}
}

// AddNode inserts or updates a package node.
func AddNode(g *types.DependencyGraph, pkg *types.Package) {
	g.Nodes[pkg.ID] = pkg
}

// AddEdge records parent -> child.
func AddEdge(g *types.DependencyGraph, parentID, childID string) {
	if parentID == "" || childID == "" || parentID == childID {
		return
	}
	g.Edges[parentID] = append(g.Edges[parentID], childID)
	if p, ok := g.Nodes[parentID]; ok {
		p.Children = appendUnique(p.Children, childID)
	}
	if c, ok := g.Nodes[childID]; ok {
		c.Parents = appendUnique(c.Parents, parentID)
	}
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
