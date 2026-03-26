package abandonment

import (
	"context"
	"time"

	"github.com/depscount/depscount/pkg/types"
)

// Engine scores maintenance signals from registry metadata only (Phase 1).
type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "abandonment" }

func (e *Engine) Run(ctx context.Context, graph *types.DependencyGraph) ([]types.RiskResult, error) {
	_ = ctx
	var out []types.RiskResult
	now := time.Now()
	for _, p := range graph.Nodes {
		ref := p.RegistryModifiedAt
		if ref.IsZero() {
			ref = p.PublishedAt
		}
		if ref.IsZero() {
			out = append(out, types.RiskResult{
				Package:   p.ID,
				Dimension: types.DimensionAbandonment,
				Score:     0.35,
				Severity:  types.SeverityMedium,
				Title:     "Release activity unknown",
				Detail:    "Could not determine last publish time from registry metadata",
			})
			continue
		}
		days := now.Sub(ref).Hours() / 24
		score := scoreFromAgeDays(days)
		out = append(out, types.RiskResult{
			Package:   p.ID,
			Dimension: types.DimensionAbandonment,
			Score:     score,
			Severity:  types.SeverityFromScore(score),
			Title:     "Time since registry activity",
			Detail:    formatAge(ref),
		})
	}
	return out, nil
}

func scoreFromAgeDays(days float64) float64 {
	switch {
	case days < 365:
		return 0
	case days < 730:
		return 0.2
	case days < 1095:
		return 0.5
	case days < 1460:
		return 0.7
	default:
		return 0.9
	}
}

func formatAge(t time.Time) string {
	d := time.Since(t).Round(24 * time.Hour)
	return "Last registry activity approximately " + d.String() + " ago"
}
