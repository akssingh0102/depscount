package types

import (
	"time"
)

// Dependency represents a declared dependency from a manifest.
type Dependency struct {
	Name        string
	Ecosystem   string
	Version     VersionConstraint
	IsDev       bool
	IsOptional  bool
}

// VersionConstraint holds manifest version info.
type VersionConstraint struct {
	Raw       string
	Resolved  string
	Ecosystem string
}

// Package is a node in the resolved dependency graph.
type Package struct {
	ID              string // ecosystem/name@version
	Name            string
	Version         string
	Ecosystem       string
	Depth           int
	IsDirect        bool
	Parents         []string
	Children        []string
	License         string
	RepoURL              string
	PublishedAt          time.Time
	RegistryModifiedAt   time.Time // e.g. npm package last publish activity
	MetadataPartial      bool
}

// DependencyGraph is the resolved DAG (cycles reported separately).
type DependencyGraph struct {
	Root    *Package
	Nodes   map[string]*Package
	Edges   map[string][]string
	Cycles  [][]string
	Project string
}

// RiskDimension categorizes risk.
type RiskDimension string

const (
	DimensionSecurity    RiskDimension = "security"
	DimensionAbandonment RiskDimension = "abandonment"
	DimensionLicense     RiskDimension = "license"
	DimensionSupplyChain RiskDimension = "supply_chain"
)

// Severity maps composite score to a label.
type Severity string

const (
	SeverityBlocker  Severity = "blocker"
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Evidence links a finding to a source.
type Evidence struct {
	Type   string
	ID     string
	URL    string
	Detail string
}

// Remediation describes how to fix a finding.
type Remediation struct {
	Type          string
	TargetVersion string
	Alternatives  []string
	Instructions  string
}

// RiskResult is one engine finding for a package.
type RiskResult struct {
	Package     string
	Dimension   RiskDimension
	Score       float64
	Severity    Severity
	Title       string
	Detail      string
	Evidence    []Evidence
	Remediation *Remediation
}

// ScanMetadata describes the scan run.
type ScanMetadata struct {
	Project   string
	ScannedAt time.Time
	ScanMs    int64
	Ecosystem string
	DepCount  int
	Warnings  []string
}

// ReportSummary aggregates counts.
type ReportSummary struct {
	Blockers   int
	Critical   int
	High       int
	Medium     int
	Low        int
	Info       int
	PolicyHits int
}

// ScoredPackage is a package with composite score and per-dimension scores.
type ScoredPackage struct {
	ID              string
	Name            string
	Version         string
	Ecosystem       string
	Depth           int
	IsDirect        bool
	CompositeScore  float64
	Severity        Severity
	Security        float64
	Abandonment     float64
	License         float64
	SupplyChain     float64
	Results         []RiskResult
}

// PolicyViolation is a policy rule failure.
type PolicyViolation struct {
	Rule     string
	Package  string
	Detail   string
	Severity Severity
	Blocker  bool
}

// UpgradeSuggestion is a suggested version bump.
type UpgradeSuggestion struct {
	Package       string
	FromVersion   string
	ToVersion     string
	Reason        string
}

// ScanReport is the full scan output.
type ScanReport struct {
	Metadata         ScanMetadata
	Summary          ReportSummary
	Packages         []ScoredPackage
	PolicyViolations []PolicyViolation
	UpgradePaths     []UpgradeSuggestion
	RawResults       map[string][]RiskResult
}

// SeverityFromScore maps 0–1 composite to severity (architecture §4.5).
func SeverityFromScore(s float64) Severity {
	switch {
	case s >= 0.9:
		return SeverityBlocker
	case s >= 0.7:
		return SeverityCritical
	case s >= 0.4:
		return SeverityHigh
	case s >= 0.2:
		return SeverityMedium
	case s > 0:
		return SeverityLow
	default:
		return SeverityInfo
	}
}
