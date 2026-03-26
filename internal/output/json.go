package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/depscount/depscount/pkg/types"
)

// JSON writes a scan report as JSON.
func JSON(w io.Writer, rep *types.ScanReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// JSONToStdout writes the report to stdout.
func JSONToStdout(rep *types.ScanReport) error {
	return JSON(os.Stdout, rep)
}
