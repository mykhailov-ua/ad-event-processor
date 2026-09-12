package postback

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShouldFireOutboundPostbackSample_holdout(t *testing.T) {
	t.Parallel()
	postbackID := uuid.MustParse("11111111-1111-7111-8111-111111111111")
	clickID := "click-holdout-1"

	assert.True(t, ShouldFireOutboundPostbackSample(clickID, postbackID, 100))
	assert.False(t, ShouldFireOutboundPostbackSample(clickID, postbackID, 0))
}

func TestShouldFireOutboundPostbackSample_stableAcrossRetries(t *testing.T) {
	t.Parallel()
	postbackID := uuid.MustParse("22222222-2222-7222-8222-222222222222")
	clickID := "click-stable-1"
	first := ShouldFireOutboundPostbackSample(clickID, postbackID, 50)
	second := ShouldFireOutboundPostbackSample(clickID, postbackID, 50)
	assert.Equal(t, first, second)
}
