package output

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"name":   "test",
		"count":  42,
		"active": true,
	}

	err := PrintJSON(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"name": "test"`)
	assert.Contains(t, out, `"count": 42`)
	assert.Contains(t, out, `"active": true`)
}

func TestPrintJSON_Pipe(t *testing.T) {
	// Writing to a pipe (not a TTY) should produce plain JSON
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	data := map[string]string{"key": "value"}
	err = PrintJSON(w, data)
	require.NoError(t, err)
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	assert.Contains(t, out, `"key": "value"`)
	// Should not contain ANSI escape codes
	assert.NotContains(t, out, "\033[")
}

func TestPrintJSON_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJSON(&buf, nil)
	require.NoError(t, err)
	assert.Equal(t, "null\n", buf.String())
}

func TestPrintJSON_Slice(t *testing.T) {
	var buf bytes.Buffer
	data := []string{"a", "b", "c"}
	err := PrintJSON(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"a"`)
	assert.Contains(t, out, `"b"`)
	assert.Contains(t, out, `"c"`)
}

func TestPrintJSON_NestedObject(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"person": map[string]any{
			"name":  "John",
			"age":   30,
			"admin": false,
		},
		"tags": nil,
	}
	err := PrintJSON(&buf, data)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, `"name": "John"`)
	assert.Contains(t, out, `"age": 30`)
	assert.Contains(t, out, "null")
}

func TestPrintJSON_EmptyObject(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJSON(&buf, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "{}\n", buf.String())
}

func TestPrintJSON_ToFile(t *testing.T) {
	// Create a temp file (not a TTY) and write to it
	f, err := os.CreateTemp(t.TempDir(), "json-test")
	require.NoError(t, err)
	defer f.Close()

	data := map[string]string{"hello": "world"}
	err = PrintJSON(f, data)
	require.NoError(t, err)

	// Read back
	f.Seek(0, 0)
	var buf bytes.Buffer
	buf.ReadFrom(f)
	assert.Contains(t, buf.String(), `"hello": "world"`)
	assert.NotContains(t, buf.String(), "\033[") // no ANSI codes
}

func TestPrintJSON_TTYColorized(t *testing.T) {
	// Override IsTTY to simulate terminal output
	orig := IsTTY
	IsTTY = func(w io.Writer) bool { return true }
	t.Cleanup(func() { IsTTY = orig })

	var buf bytes.Buffer
	data := map[string]any{
		"name":   "test",
		"count":  42,
		"active": true,
		"tags":   nil,
	}

	err := PrintJSON(&buf, data)
	require.NoError(t, err)

	out := buf.String()
	// Should contain ANSI escape codes from colorization
	assert.Contains(t, out, "\033[")
	// Should still contain the actual data
	assert.Contains(t, out, "test")
	assert.Contains(t, out, "42")
}

func TestPrintJSON_TTYEmptyObject(t *testing.T) {
	orig := IsTTY
	IsTTY = func(w io.Writer) bool { return true }
	t.Cleanup(func() { IsTTY = orig })

	var buf bytes.Buffer
	err := PrintJSON(&buf, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "{")
}

func TestIsTTY_DefaultWithBuffer(t *testing.T) {
	var buf bytes.Buffer
	assert.False(t, IsTTY(&buf))
}

func TestIsTTY_DefaultWithPipe(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	// Pipes are not TTYs
	assert.False(t, IsTTY(w))
}

func TestDefaultColors(t *testing.T) {
	colors := defaultColors()
	assert.NotEmpty(t, string(colors.Null))
	assert.NotEmpty(t, string(colors.Bool))
	assert.NotEmpty(t, string(colors.Number))
	assert.NotEmpty(t, string(colors.String))
	assert.NotEmpty(t, string(colors.Key))
	assert.NotEmpty(t, string(colors.Punc))
}
