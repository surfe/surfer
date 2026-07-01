package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIcons(t *testing.T) {
	assert.Equal(t, "✔", IconSuccess)
	assert.Equal(t, "✘", IconError)
	assert.Equal(t, "⚠", IconWarning)
	assert.Equal(t, "●", IconInfo)
	assert.Equal(t, "→", IconArrow)
	assert.Equal(t, "🚀", IconRocket)
}
