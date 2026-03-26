package security

import (
	"context"
	"strings"

	"github.com/depscount/depscount/internal/httpclient"
	"github.com/depscount/depscount/pkg/types"
)

// Engine queries OSV in batches (Phase 1).
type Engine struct {
	HTTP *httpclient.Client
}

func New(hc *httpclient.Client) *Engine {
	return &Engine{HTTP: hc}
}

func (e *Engine) Name() string { return "security" }

func (e *Engine) Run(ctx context.Context, graph *types.DependencyGraph) ([]types.RiskResult, error) {
	if e.HTTP == nil {
		e.HTTP = httpclient.New()
	}
	type job struct {
		q   batchQuery
		pid string
	}
	var jobs []job
	for _, p := range graph.Nodes {
		if p.MetadataPartial {
			continue
		}
		v := strings.TrimSpace(p.Version)
		if v == "" || v == "0.0.0" {
			continue
		}
		// Skip manifest range strings (unpinned); OSV needs concrete releases.
		if strings.ContainsAny(v, "^~*<>") {
			continue
		}
		var q batchQuery
		q.Package.Name = p.Name
		q.Package.Ecosystem = osvEcosystem(p.Ecosystem)
		q.Version = v
		jobs = append(jobs, job{q: q, pid: p.ID})
	}
	const chunk = 500
	var all []types.RiskResult
	for start := 0; start < len(jobs); start += chunk {
		end := start + chunk
		if end > len(jobs) {
			end = len(jobs)
		}
		part := jobs[start:end]
		queries := make([]batchQuery, len(part))
		for i := range part {
			queries[i] = part[i].q
		}
		res, err := queryBatch(ctx, e.HTTP, queries)
		if err != nil {
			return []types.RiskResult{{
				Package:   "scan",
				Dimension: types.DimensionSecurity,
				Title:     "OSV query failed",
				Detail:    err.Error(),
			}}, nil
		}
		for i, r := range res.Results {
			if i >= len(part) {
				break
			}
			pid := part[i].pid
			for _, v := range r.Vulns {
				base := cvssBaseScore(v)
				if base <= 0 {
					base = 5.0
				}
				norm := base / 10.0
				rem := &types.Remediation{Type: "upgrade", Instructions: "Review OSV advisory and upgrade to a patched release"}
				all = append(all, types.RiskResult{
					Package:     pid,
					Dimension:   types.DimensionSecurity,
					Score:       norm,
					Severity:    types.SeverityFromScore(norm),
					Title:       v.ID,
					Detail:      firstNonEmpty(v.Summary, v.Details, "Security vulnerability"),
					Evidence:    []types.Evidence{{Type: "advisory", ID: v.ID, URL: "https://osv.dev/vulnerability/" + v.ID}},
					Remediation: rem,
				})
			}
		}
	}
	return all, nil
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
