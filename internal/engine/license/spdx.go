package license

import (
	"strings"
)

// NormalizeID uppercases and trims common SPDX aliases.
func NormalizeID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	// take first clause from OR/AND expressions (Phase 1 simplification)
	if i := strings.Index(strings.ToUpper(s), " OR "); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(strings.ToUpper(s), " AND "); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	return strings.ToUpper(s)
}

// TierScore maps SPDX-like ids to base risk 0–1 (architecture §4.4.3).
func TierScore(id string) float64 {
	id = NormalizeID(id)
	switch id {
	case "", "UNKNOWN", "UNLICENSED", "NONE":
		return 0.9
	case "MIT", "ISC", "0BSD", "UNLICENSE", "CC0-1.0", "WTFPL":
		return 0
	case "APACHE-2.0", "APACHE LICENSE 2.0", "APACHE-2":
		return 0
	case "BSD-2-CLAUSE", "BSD-3-CLAUSE", "BSD-2-CLAUSE-FREEBSD", "BSD-3-CLAUSE-CLEAR":
		return 0
	case "LGPL-2.1", "LGPL-2.1-ONLY", "LGPL-3.0", "LGPL-3.0-ONLY", "MPL-2.0", "MPL-1.1":
		return 0.3
	case "GPL-2.0", "GPL-2.0-ONLY", "GPL-3.0", "GPL-3.0-ONLY":
		return 0.7
	case "AGPL-3.0", "AGPL-3.0-ONLY", "EUPL-1.2":
		return 0.8
	default:
		if strings.HasPrefix(id, "GPL-") {
			return 0.7
		}
		if strings.Contains(id, "AGPL") {
			return 0.8
		}
		return 0.4
	}
}
