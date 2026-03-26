package scorer

import (
	"sort"

	"github.com/depscount/depscount/pkg/types"
)

// BuildReport aggregates engine output into a ScanReport.
func BuildReport(
	meta types.ScanMetadata,
	graph *types.DependencyGraph,
	raw map[string][]types.RiskResult,
	secW, abanW, licW, scW float64,
) *types.ScanReport {
	byPkg := make(map[string]map[types.RiskDimension][]types.RiskResult)
	for eng, rs := range raw {
		_ = eng
		for _, r := range rs {
			if r.Package == "scan" {
				continue
			}
			if _, ok := byPkg[r.Package]; !ok {
				byPkg[r.Package] = make(map[types.RiskDimension][]types.RiskResult)
			}
			byPkg[r.Package][r.Dimension] = append(byPkg[r.Package][r.Dimension], r)
		}
	}
	var list []types.ScoredPackage
	for id, p := range graph.Nodes {
		dims := byPkg[id]
		sec := maxDimScore(dims[types.DimensionSecurity])
		aban := maxDimScore(dims[types.DimensionAbandonment])
		lic := maxDimScore(dims[types.DimensionLicense])
		sup := maxDimScore(dims[types.DimensionSupplyChain])
		depthMult := depthMultiplier(p.Depth)
		comp := (sec*secW + aban*abanW + lic*licW + sup*scW) * depthMult
		if comp > 1 {
			comp = 1
		}
		var flat []types.RiskResult
		for _, rs := range dims {
			flat = append(flat, rs...)
		}
		sp := types.ScoredPackage{
			ID:             id,
			Name:           p.Name,
			Version:        p.Version,
			Ecosystem:      p.Ecosystem,
			Depth:          p.Depth,
			IsDirect:       p.IsDirect,
			CompositeScore: comp,
			Severity:       types.SeverityFromScore(comp),
			Security:       sec,
			Abandonment:    aban,
			License:        lic,
			SupplyChain:    sup,
			Results:        flat,
		}
		list = append(list, sp)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CompositeScore > list[j].CompositeScore
	})
	rep := &types.ScanReport{
		Metadata:   meta,
		Packages:   list,
		RawResults: raw,
	}
	rep.Summary = summarize(rep)
	return rep
}

func maxDimScore(rs []types.RiskResult) float64 {
	m := 0.0
	for _, r := range rs {
		if r.Score > m {
			m = r.Score
		}
	}
	return m
}

func depthMultiplier(depth int) float64 {
	switch {
	case depth <= 1:
		return 1.0
	case depth == 2:
		return 0.9
	case depth == 3:
		return 0.75
	default:
		return 0.6
	}
}

func summarize(rep *types.ScanReport) types.ReportSummary {
	var s types.ReportSummary
	for _, p := range rep.Packages {
		switch p.Severity {
		case types.SeverityBlocker:
			s.Blockers++
		case types.SeverityCritical:
			s.Critical++
		case types.SeverityHigh:
			s.High++
		case types.SeverityMedium:
			s.Medium++
		case types.SeverityLow:
			s.Low++
		default:
			s.Info++
		}
	}
	return s
}
