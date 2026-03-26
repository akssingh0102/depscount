package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/depscount/depscount/internal/engine"
	"github.com/depscount/depscount/internal/engine/abandonment"
	"github.com/depscount/depscount/internal/engine/license"
	"github.com/depscount/depscount/internal/engine/security"
	"github.com/depscount/depscount/internal/httpclient"
	"github.com/depscount/depscount/internal/parser"
	"github.com/depscount/depscount/internal/policy"
	"github.com/depscount/depscount/internal/regmeta"
	"github.com/depscount/depscount/internal/resolver"
	"github.com/depscount/depscount/internal/scorer"
	"github.com/depscount/depscount/pkg/types"
)

// Options configures a scan run.
type Options struct {
	Path   string
	Config *policy.Config
	Offline bool
}

// Scan runs the full pipeline for a project directory or file path.
func Scan(ctx context.Context, opts Options) (*types.ScanReport, error) {
	start := time.Now()
	root, p, err := parser.Discover(opts.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no supported manifest (npm or python) found under %s", opts.Path)
		}
		return nil, fmt.Errorf("discover project: %w", err)
	}
	deps, proj, err := p.Parse(root)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if opts.Config != nil && opts.Config.Project.Name != "" {
		proj = opts.Config.Project.Name
	}
	graph, err := resolver.Resolve(root, p, deps, proj)
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}
	if len(graph.Nodes) == 0 {
		return nil, fmt.Errorf("no dependencies resolved (add a lockfile or pin versions)")
	}
	hc := httpclient.New()
	if !opts.Offline {
		regmeta.Enrich(ctx, hc, graph)
	}
	cfg := opts.Config
	if cfg == nil {
		cfg = policy.Default()
	}
	secW, abanW, licW, scW := cfg.NormalizedWeights()
	secW, abanW, licW = renorm3(secW, abanW, licW, scW)
	scW = 0

	var engines []engine.Engine
	if opts.Offline {
		engines = []engine.Engine{abandonment.New(), license.New()}
	} else {
		engines = []engine.Engine{security.New(hc), abandonment.New(), license.New()}
	}
	raw := runEngines(ctx, engines, graph)
	meta := types.ScanMetadata{
		Project:   root,
		ScannedAt: time.Now(),
		ScanMs:    time.Since(start).Milliseconds(),
		Ecosystem: p.Ecosystem(),
		DepCount:  len(graph.Nodes),
	}
	if opts.Offline {
		meta.Warnings = append(meta.Warnings, "Offline mode: skipping OSV and registry enrichment")
	}
	rep := scorer.BuildReport(meta, graph, raw, secW, abanW, licW, scW)
	rep.PolicyViolations = policy.Evaluate(cfg, rep)
	rep.Summary.PolicyHits = len(rep.PolicyViolations)
	return rep, nil
}

func renorm3(sec, aban, lic, sc float64) (float64, float64, float64) {
	sum := sec + aban + lic
	if sum <= 0 {
		return 0.5, 0.3, 0.2
	}
	extra := sc
	if extra <= 0 {
		return sec / sum, aban / sum, lic / sum
	}
	// add supply weight proportionally
	return (sec + extra*sec/sum) / (sum + extra),
		(aban + extra*aban/sum) / (sum + extra),
		(lic + extra*lic/sum) / (sum + extra)
}

func runEngines(ctx context.Context, engines []engine.Engine, graph *types.DependencyGraph) map[string][]types.RiskResult {
	out := make(map[string][]types.RiskResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, e := range engines {
		e := e
		wg.Add(1)
		go func() {
			defer wg.Done()
			rs, err := e.Run(ctx, graph)
			if err != nil {
				rs = []types.RiskResult{{
					Package:   "scan",
					Dimension: types.DimensionSecurity,
					Title:     e.Name() + " engine error",
					Detail:    err.Error(),
				}}
			}
			mu.Lock()
			out[e.Name()] = rs
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}
