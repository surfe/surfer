package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintCSV_PeopleResults(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"people": []any{
			map[string]any{
				"firstName":   "John",
				"lastName":    "Doe",
				"jobTitle":    "CTO",
				"companyName": "Acme",
				"country":     "US",
			},
			map[string]any{
				"firstName":   "Jane",
				"lastName":    "Smith",
				"jobTitle":    "VP Engineering",
				"companyName": "Globex",
				"country":     "UK",
			},
		},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "firstName")
	assert.Contains(t, out, "John")
	assert.Contains(t, out, "Jane")
	assert.Contains(t, out, "CTO")
	assert.Contains(t, out, "VP Engineering")
}

func TestPrintCSV_ArrayValues(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"people": []any{
			map[string]any{
				"name":        "John",
				"departments": []any{"Engineering", "Management"},
				"seniorities": []any{"C-Level"},
			},
		},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Engineering; Management")
	assert.Contains(t, out, "C-Level")
}

func TestPrintCSV_SingleObject(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"emailCredits":  float64(500),
		"mobileCredits": float64(100),
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "100")
}

func TestPrintCSV_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"people": []any{},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestPrintCSV_NilValues(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"items": []any{
			map[string]any{
				"name":  "Test",
				"email": nil,
			},
		},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Test")
}

func TestPrintCSV_BoolValues(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"items": []any{
			map[string]any{
				"name":   "Test",
				"active": true,
				"admin":  false,
			},
		},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "false")
}

func TestPrintCSV_NestedObject(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"items": []any{
			map[string]any{
				"name":   "Stripe",
				"nested": map[string]any{"key": "val"},
			},
		},
	}

	err := PrintCSV(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "key=val")
}

func TestPrintCSV_UnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	err := PrintCSV(&buf, "not a map or slice")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CSV output requires")
}
