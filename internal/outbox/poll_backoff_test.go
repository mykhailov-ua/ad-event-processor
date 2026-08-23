package outbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPollBackoff_ActiveThenIdle(t *testing.T) {
	backoff := NewPollBackoff()

	assert.Equal(t, time.Duration(0), backoff.Next(5), "work found resets to immediate repoll")
	assert.Equal(t, 40*time.Millisecond, backoff.Next(0))
	assert.Equal(t, 80*time.Millisecond, backoff.Next(0))
	assert.Equal(t, 160*time.Millisecond, backoff.Next(0))
	assert.Equal(t, PollIdleMax, backoff.Next(0))
	assert.Equal(t, PollIdleMax, backoff.Next(0), "caps at idle max")
}

func TestPollBackoff_IdleMedianAboveDoD(t *testing.T) {
	backoff := NewPollBackoff()
	var samples []time.Duration
	for range 8 {
		samples = append(samples, backoff.Next(0))
	}
	assert.Greater(t, samples[3], 50*time.Millisecond)
	assert.Equal(t, PollIdleMax, samples[len(samples)-1])
}
