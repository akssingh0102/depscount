package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/depscount/depscount/internal/output"
	"github.com/depscount/depscount/internal/policy"
	"github.com/depscount/depscount/internal/scanner"
	"github.com/spf13/cobra"
)

func ciCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "ci [path]",
		Short: "CI mode: scan and exit with policy gate codes",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			p := path
			if len(args) > 0 {
				p = args[0]
			}
			if p == "" {
				var err error
				p, err = os.Getwd()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
			ctx := context.Background()
			rep, err := scanner.Scan(ctx, scanner.Options{Path: p, Config: cfg, Offline: offline})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if jsonOut {
				_ = output.JSON(os.Stdout, rep)
			} else if !quiet {
				output.TerminalStdout(rep, noColor)
			}
			os.Exit(policy.ExitCode(rep, cfg.Thresholds.FailOn))
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Project root (default: cwd)")
	return cmd
}
