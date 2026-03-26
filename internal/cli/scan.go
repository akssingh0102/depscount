package cli

import (
	"context"
	"os"

	"github.com/depscount/depscount/internal/output"
	"github.com/depscount/depscount/internal/scanner"
	"github.com/spf13/cobra"
)

func scanCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a project directory for dependency risk",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := path
			if len(args) > 0 {
				p = args[0]
			}
			if p == "" {
				var err error
				p, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			ctx := context.Background()
			rep, err := scanner.Scan(ctx, scanner.Options{Path: p, Config: cfg, Offline: offline})
			if err != nil {
				return err
			}
			if jsonOut {
				return output.JSON(os.Stdout, rep)
			}
			if !quiet {
				output.TerminalStdout(rep, noColor)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Project root (default: cwd)")
	return cmd
}
