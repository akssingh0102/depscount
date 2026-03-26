package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/depscount/depscount/pkg/types"
)

// Terminal renders a human-readable report to w.
func Terminal(w io.Writer, rep *types.ScanReport, color bool) {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	bad := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	if !color {
		title = lipgloss.NewStyle()
		ok = lipgloss.NewStyle()
		warn = lipgloss.NewStyle()
		bad = lipgloss.NewStyle()
		dim = lipgloss.NewStyle()
	}
	fmt.Fprintf(w, "%s\n", title.Render("depscount — dependency risk scan"))
	fmt.Fprintf(w, "%s\n\n", dim.Render(fmt.Sprintf("Project: %s  ·  Ecosystem: %s  ·  Packages: %d  ·  %d ms",
		rep.Metadata.Project, rep.Metadata.Ecosystem, rep.Metadata.DepCount, rep.Metadata.ScanMs)))

	if len(rep.Metadata.Warnings) > 0 {
		for _, m := range rep.Metadata.Warnings {
			fmt.Fprintf(w, "%s %s\n", warn.Render("!"), m)
		}
		fmt.Fprintln(w)
	}

	hasFinding := false
	for _, p := range rep.Packages {
		for _, r := range p.Results {
			if r.Dimension == types.DimensionSecurity && r.Score > 0 {
				hasFinding = true
				fmt.Fprintf(w, "%s %s\n", bad.Render(strings.ToUpper(string(r.Severity))), p.Name+"@"+p.Version)
				fmt.Fprintf(w, "  %s — %s\n", r.Title, truncate(r.Detail, 120))
				if r.Remediation != nil && r.Remediation.Instructions != "" {
					fmt.Fprintf(w, "  %s\n", dim.Render("→ "+r.Remediation.Instructions))
				}
			}
		}
	}
	for _, v := range rep.PolicyViolations {
		hasFinding = true
		fmt.Fprintf(w, "%s %s — %s\n", bad.Render("POLICY"), v.Package, v.Detail)
	}

	if !hasFinding && len(rep.PolicyViolations) == 0 {
		fmt.Fprintf(w, "%s No known security vulnerabilities (OSV) or policy violations in this scan.\n", ok.Render("✓"))
	}

	fmt.Fprintf(w, "\n%s\n", title.Render("Summary"))
	fmt.Fprintf(w, "  Blockers: %d  Critical: %d  High: %d  Medium: %d  Low: %d  Policy: %d\n",
		rep.Summary.Blockers, rep.Summary.Critical, rep.Summary.High, rep.Summary.Medium, rep.Summary.Low, rep.Summary.PolicyHits)
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// TerminalStdout renders to stdout with color if tty.
func TerminalStdout(rep *types.ScanReport, noColor bool) {
	color := !noColor && isTTY()
	Terminal(os.Stdout, rep, color)
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
