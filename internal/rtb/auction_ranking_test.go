package rtb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffectiveScore_defaultsCTR(t *testing.T) {
	assert.Equal(t, int64(100), effectiveScore(100, 0))
}
