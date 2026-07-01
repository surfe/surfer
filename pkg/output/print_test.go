package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrint_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := Print(&buf, map[string]string{"key": "value"}, FormatJSON)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"key": "value"`)
}

func TestPrint_CSV(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"items": []any{
			map[string]any{"name": "test"},
		},
	}
	err := Print(&buf, data, FormatCSV)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name")
	assert.Contains(t, buf.String(), "test")
}

func TestPrint_EmptyDefault(t *testing.T) {
	var buf bytes.Buffer
	err := Print(&buf, map[string]string{"a": "b"}, "")
	require.NoError(t, err)
	// Empty string defaults to JSON
	assert.Contains(t, buf.String(), `"a": "b"`)
}

func TestPrint_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Print(&buf, nil, "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}
