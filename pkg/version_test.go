package pkg

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintVersion(t *testing.T) {
	BuildVersion = "1.2.3"
	BuildCommit = "abc1234"
	BuildTime = "1700000000"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintVersion()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	assert.Contains(t, out, "Version: 1.2.3")
	assert.Contains(t, out, "Commit: abc1234")
	assert.Contains(t, out, "Build Time:")
	// Should have parsed the unix timestamp into a human-readable string
	assert.NotContains(t, out, "Build Time: 1700000000")
}

func TestPrintVersion_InvalidTimestamp(t *testing.T) {
	BuildVersion = "dev"
	BuildCommit = ""
	BuildTime = "not-a-number"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintVersion()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	assert.Contains(t, out, "Version: dev")
	assert.Contains(t, out, "Build Time: not-a-number")
}
