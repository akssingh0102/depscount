package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the subset of .depscount.yml implemented in Phase 1.
type Config struct {
	Version   int    `yaml:"version"`
	Weights   Weights `yaml:"weights"`
	Thresholds struct {
		FailOn string `yaml:"fail_on"`
	} `yaml:"thresholds"`
	Licenses struct {
		Allowed        []string `yaml:"allowed"`
		Blocked        []string `yaml:"blocked"`
		TreatUnknownAs string   `yaml:"treat_unknown_as"` // warn | fail | ignore
	} `yaml:"licenses"`
	Project struct {
		Name      string `yaml:"name"`
		License   string `yaml:"license"`
		Ecosystem string `yaml:"ecosystem"`
	} `yaml:"project"`
}

// Weights for composite score (should sum to 1; renormalized if not).
type Weights struct {
	Security    float64 `yaml:"security"`
	Abandonment float64 `yaml:"abandonment"`
	License     float64 `yaml:"license"`
	SupplyChain float64 `yaml:"supply_chain"`
}

// Default returns baseline policy matching the architecture defaults.
func Default() *Config {
	c := &Config{Version: 1}
	c.Weights.Security = 0.45
	c.Weights.Abandonment = 0.25
	c.Weights.License = 0.20
	c.Weights.SupplyChain = 0.10
	c.Thresholds.FailOn = "high"
	c.Licenses.TreatUnknownAs = "warn"
	c.Licenses.Allowed = []string{
		"MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC", "CC0-1.0",
	}
	return c
}

// LoadFile reads YAML from path.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := Default()
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Discover walks upward from start looking for config files (new name first, then legacy aliases).
func Discover(start string) (string, *Config, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", nil, err
	}
	names := []string{
		".depscount.yml", "depscount.yml",
		".depguard.yml", "depguard.yml",
	}
	for _, dir := range walkUp(abs) {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				c, err := LoadFile(p)
				return p, c, err
			}
		}
	}
	return "", Default(), nil
}

func walkUp(start string) []string {
	var out []string
	cur := start
	st, err := os.Stat(cur)
	if err == nil && !st.IsDir() {
		cur = filepath.Dir(cur)
	}
	for {
		out = append(out, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return out
}

// ExpandEnv replaces ${VAR} in strings (minimal).
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if i := strings.Index(key, ":-"); i >= 0 {
			key = key[:i]
		}
		return os.Getenv(key)
	})
}

// Validate checks basic constraints.
func (c *Config) Validate() error {
	if c.Version != 0 && c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", c.Version)
	}
	return nil
}

// NormalizedWeights returns nonnegative weights renormalized to sum to 1.
func (c *Config) NormalizedWeights() (sec, aban, lic, sc float64) {
	w := c.Weights
	sec, aban, lic, sc = w.Security, w.Abandonment, w.License, w.SupplyChain
	if sec < 0 {
		sec = 0
	}
	if aban < 0 {
		aban = 0
	}
	if lic < 0 {
		lic = 0
	}
	if sc < 0 {
		sc = 0
	}
	sum := sec + aban + lic + sc
	if sum <= 0 {
		return 0.45, 0.25, 0.20, 0.10
	}
	return sec / sum, aban / sum, lic / sum, sc / sum
}
