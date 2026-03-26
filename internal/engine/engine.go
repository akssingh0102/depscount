package engine

import (
	"context"

	"github.com/depscount/depscount/pkg/types"
)

// Engine analyzes the dependency graph.
type Engine interface {
	Name() string
	Run(ctx context.Context, graph *types.DependencyGraph) ([]types.RiskResult, error)
}
