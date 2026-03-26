package license

import (
	"context"
	"strings"

	"github.com/depscount/depscount/pkg/types"
)

// Engine classifies declared package licenses (metadata only, Phase 1).
type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "license" }

func (e *Engine) Run(ctx context.Context, graph *types.DependencyGraph) ([]types.RiskResult, error) {
	_ = ctx
	var out []types.RiskResult
	for _, p := range graph.Nodes {
		raw := strings.TrimSpace(p.License)
		if raw == "" {
			out = append(out, types.RiskResult{
				Package:   p.ID,
				Dimension: types.DimensionLicense,
				Score:     TierScore("UNKNOWN"),
				Severity:  types.SeverityHigh,
				Title:     "UNKNOWN",
				Detail:    "No license metadata available for package",
			})
			continue
		}
		id := NormalizeID(raw)
		score := TierScore(id)
		out = append(out, types.RiskResult{
			Package:   p.ID,
			Dimension: types.DimensionLicense,
			Score:     score,
			Severity:  types.SeverityFromScore(score),
			Title:     id,
			Detail:    "Declared license: " + raw,
		})
	}
	return out, nil
}
