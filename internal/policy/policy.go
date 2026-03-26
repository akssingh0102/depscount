package policy

import (
	"strings"

	"github.com/depscount/depscount/internal/engine/license"
	"github.com/depscount/depscount/pkg/types"
)

// Evaluate applies license allow/block lists from config to scored packages.
func Evaluate(cfg *Config, rep *types.ScanReport) []types.PolicyViolation {
	var out []types.PolicyViolation
	if cfg == nil {
		return out
	}
	allowed := make(map[string]struct{})
	for _, a := range cfg.Licenses.Allowed {
		allowed[license.NormalizeID(a)] = struct{}{}
	}
	blocked := make(map[string]struct{})
	for _, b := range cfg.Licenses.Blocked {
		blocked[license.NormalizeID(b)] = struct{}{}
	}
	for _, sp := range rep.Packages {
		for _, r := range sp.Results {
			if r.Dimension != types.DimensionLicense {
				continue
			}
			id := license.NormalizeID(r.Title)
			if id == "" {
				id = "UNKNOWN"
			}
			if _, ok := blocked[id]; ok {
				out = append(out, types.PolicyViolation{
					Rule:     "licenses.blocked",
					Package:  sp.Name + "@" + sp.Version,
					Detail:   "License " + id + " is blocked by policy",
					Severity: types.SeverityBlocker,
					Blocker:  true,
				})
				continue
			}
			if len(allowed) > 0 && id != "UNKNOWN" && id != "UNLICENSED" {
				if _, ok := allowed[id]; !ok {
					out = append(out, types.PolicyViolation{
						Rule:     "licenses.allowed",
						Package:  sp.Name + "@" + sp.Version,
						Detail:   "License " + id + " is not in the allow list",
						Severity: types.SeverityHigh,
						Blocker:  false,
					})
				}
			}
			if id == "UNKNOWN" || id == "UNLICENSED" {
				switch strings.ToLower(strings.TrimSpace(cfg.Licenses.TreatUnknownAs)) {
				case "fail":
					out = append(out, types.PolicyViolation{
						Rule:     "licenses.treat_unknown_as",
						Package:  sp.Name + "@" + sp.Version,
						Detail:   "Unknown or missing license is not permitted",
						Severity: types.SeverityBlocker,
						Blocker:  true,
					})
				}
			}
		}
	}
	return out
}

// ExitCode returns CI exit status from report and fail_on threshold.
func ExitCode(rep *types.ScanReport, failOn string) int {
	var softPolicy bool
	for _, v := range rep.PolicyViolations {
		if v.Blocker {
			return 3
		}
		softPolicy = true
	}
	if softPolicy {
		return 2
	}
	th := failOnRank(failOn)
	for _, p := range rep.Packages {
		if severityRank(p.Severity) >= th {
			return 2
		}
	}
	return 0
}

func failOnRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none":
		return 999
	case "any":
		return 0
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	case "blocker":
		return 5
	default:
		return 3
	}
}

func severityRank(s types.Severity) int {
	switch s {
	case types.SeverityInfo:
		return 0
	case types.SeverityLow:
		return 1
	case types.SeverityMedium:
		return 2
	case types.SeverityHigh:
		return 3
	case types.SeverityCritical:
		return 4
	case types.SeverityBlocker:
		return 5
	default:
		return 0
	}
}
