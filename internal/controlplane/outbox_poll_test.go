package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOutboxPollBackoff_ActiveThenIdle(t *testing.T) {
	b := newOutboxPollBackoff()

	assert.Equal(t, time.Duration(0), b.next(5), "work found resets to immediate repoll")
	assert.Equal(t, 40*time.Millisecond, b.next(0))
	assert.Equal(t, 80*time.Millisecond, b.next(0))
	assert.Equal(t, 160*time.Millisecond, b.next(0))
	assert.Equal(t, outboxPollIdleMax, b.next(0))
	assert.Equal(t, outboxPollIdleMax, b.next(0), "caps at idle max")
}

func TestOutboxPollBackoff_IdleMedianAboveDoD(t *testing.T) {
	b := newOutboxPollBackoff()
	var samples []time.Duration
	for range 8 {
		samples = append(samples, b.next(0))
	}
	assert.Greater(t, samples[3], 50*time.Millisecond)
	assert.Equal(t, outboxPollIdleMax, samples[len(samples)-1])
}
