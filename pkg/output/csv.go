package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// PrintCSV writes data as CSV. It expects either:
//   - a map with a top-level key containing a slice (e.g. {"people": [...]})
//   - a slice of maps directly
//   - a single map (written as one row)
func PrintCSV(w io.Writer, v any) error {
	rows, err := extractRows(v)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	// Collect all unique headers across all rows (preserves first-seen order)
	headers := collectHeaders(rows)
	if len(headers) == 0 {
		return nil
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			record[i] = formatValue(row[h])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func extractRows(v any) ([]map[string]any, error) {
	switch data := v.(type) {
	case map[string]any:
		// Look for the first slice value (e.g. "people", "companies")
		for _, val := range data {
			if slice, ok := val.([]any); ok {
				return toMapSlice(slice)
			}
		}
		// Single object — treat as one row
		return []map[string]any{data}, nil

	case []any:
		return toMapSlice(data)

	default:
		return nil, fmt.Errorf("CSV output requires a JSON object or array, got %T", v)
	}
}

func toMapSlice(slice []any) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		if m, ok := item.(map[string]any); ok {
			rows = append(rows, m)
		}
	}
	return rows, nil
}

func collectHeaders(rows []map[string]any) []string {
	seen := map[string]bool{}
	var headers []string
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	return headers
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			parts = append(parts, formatValue(item))
		}
		return strings.Join(parts, "; ")
	case map[string]any:
		// Flatten nested objects as key=value pairs
		parts := make([]string, 0, len(val))
		for k, v := range val {
			parts = append(parts, fmt.Sprintf("%s=%s", k, formatValue(v)))
		}
		return strings.Join(parts, "; ")
	default:
		return fmt.Sprintf("%v", val)
	}
}
