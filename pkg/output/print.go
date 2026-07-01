package output

import (
	"fmt"
	"io"
)

const (
	FormatJSON = "json"
	FormatCSV  = "csv"
)

// Print writes v in the specified format (json or csv).
func Print(w io.Writer, v any, format string) error {
	switch format {
	case FormatCSV:
		return PrintCSV(w, v)
	case FormatJSON, "":
		return PrintJSON(w, v)
	default:
		return fmt.Errorf("unsupported output format: %s (use json or csv)", format)
	}
}
