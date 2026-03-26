package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/depscount/depscount/internal/policy"
	"github.com/spf13/cobra"
)

var (
	configPath string
	cacheDir   string
	offline    bool
	quiet      bool
	jsonOut    bool
	noColor    bool
	verbose    bool
	cfg        *policy.Config
)

// Execute runs the depscount CLI.
func Execute() error {
	root := &cobra.Command{
		Use:   "depscount",
		Short: "Multi-signal dependency risk scanner",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if configPath != "" {
				c, err := policy.LoadFile(configPath)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				if err := c.Validate(); err != nil {
					return err
				}
				cfg = c
				return nil
			}
			wd, _ := os.Getwd()
			_, c, err := policy.Discover(wd)
			if err != nil {
				return fmt.Errorf("discover config: %w", err)
			}
			if err := c.Validate(); err != nil {
				return err
			}
			cfg = c
			return nil
		},
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "Path to .depscount.yml")
	root.PersistentFlags().StringVar(&cacheDir, "cache-dir", defaultCacheDir(), "Cache directory")
	root.PersistentFlags().BoolVar(&offline, "offline", envOffline(), "No network calls")
	root.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress progress output")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output JSON")
	root.PersistentFlags().BoolVar(&noColor, "no-color", envNoColor(), "Disable color")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose logging")

	root.AddCommand(scanCmd(), ciCmd())
	return root.Execute()
}

func defaultCacheDir() string {
	if d := os.Getenv("DEPSCOUNT_CACHE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("DEPGUARD_CACHE_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".depscount", "cache")
	}
	return filepath.Join(home, ".depscount", "cache")
}

func envOffline() bool {
	return envBool("DEPSCOUNT_OFFLINE") || envBool("DEPGUARD_OFFLINE")
}

func envNoColor() bool {
	return envBool("DEPSCOUNT_NO_COLOR") || envBool("DEPGUARD_NO_COLOR")
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "1" || v == "true" || v == "yes"
}
