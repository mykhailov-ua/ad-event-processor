package gnetutil

import (
	"time"

	"github.com/panjf2000/gnet/v2"
)

const (
	// DefaultConnReadIdle closes incomplete reads with no further bytes (broker framing stall).
	DefaultConnReadIdle = 30 * time.Second
	// DefaultConnMaxLifetime hard cap from accept; independent of per-frame progress.
	DefaultConnMaxLifetime = 120 * time.Second
)

type ConnPolicy struct {
	ReadIdle    time.Duration
	MaxLifetime time.Duration
}

func (p ConnPolicy) ReadIdleDuration() time.Duration {
	if p.ReadIdle > 0 {
		return p.ReadIdle
	}
	return DefaultConnReadIdle
}

func (p ConnPolicy) MaxLifetimeDuration() time.Duration {
	if p.MaxLifetime > 0 {
		return p.MaxLifetime
	}
	return DefaultConnMaxLifetime
}

type ConnState struct {
	OpenedAt         time.Time // accept time; MaxLifetimeExceeded compares wall clock since this
	readIdleArmed    bool
	readIdleDeadline time.Time
}

func NewConnState() *ConnState {
	return &ConnState{OpenedAt: time.Now()}
}

func EnsureConnState(c gnet.Conn) *ConnState {
	if ctx, ok := c.Context().(*ConnState); ok && ctx != nil {
		return ctx
	}
	ctx := NewConnState()
	c.SetContext(ctx)
	return ctx
}

func OpenConn(c gnet.Conn, p ConnPolicy, ctx *ConnState) {
	c.SetContext(ctx)
	// Arms kernel read deadline at max lifetime; OnFrameProgress slides remaining time per frame.
	_ = c.SetReadDeadline(time.Now().Add(p.MaxLifetimeDuration()))
}

func MaxLifetimeExceeded(p ConnPolicy, ctx *ConnState) bool {
	if ctx == nil {
		return false
	}
	return time.Since(ctx.OpenedAt) >= p.MaxLifetimeDuration()
}

func OnFrameProgress(c gnet.Conn, p ConnPolicy, ctx *ConnState) {
	if ctx == nil {
		return
	}
	ctx.readIdleArmed = false // fresh bytes arrived; disarm read-idle until next incomplete wait
	remaining := p.MaxLifetimeDuration() - time.Since(ctx.OpenedAt)
	if remaining <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(remaining))
}

func WaitIncomplete(c gnet.Conn, p ConnPolicy, ctx *ConnState) string {
	if ctx == nil {
		return ""
	}
	if MaxLifetimeExceeded(p, ctx) {
		return "max_lifetime" // hard cap from OpenedAt; checked before read-idle
	}
	if ctx.readIdleArmed && time.Now().After(ctx.readIdleDeadline) {
		return "read_idle"
	}
	if !ctx.readIdleArmed {
		idle := p.ReadIdleDuration()
		deadline := time.Now().Add(idle)
		maxEnd := ctx.OpenedAt.Add(p.MaxLifetimeDuration())
		if deadline.After(maxEnd) {
			deadline = maxEnd // read-idle never extends past max lifetime
		}
		_ = c.SetReadDeadline(deadline) // syscall boundary: gnet maps to SO_RCVTIMEO / poll deadline
		ctx.readIdleDeadline = deadline
		ctx.readIdleArmed = true
	}
	return ""
}

func ClearReadDeadline(c gnet.Conn) {
	// Zero time clears deadline before explicit close (avoids spurious timeout on teardown).
	_ = c.SetReadDeadline(time.Time{})
}
