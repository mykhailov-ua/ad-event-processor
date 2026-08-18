package gnetutil

import (
	"time"

	"github.com/panjf2000/gnet/v2"
)

const (
	DefaultConnReadIdle    = 30 * time.Second
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
	OpenedAt         time.Time
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
	ctx.readIdleArmed = false
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
		return "max_lifetime"
	}
	if ctx.readIdleArmed && time.Now().After(ctx.readIdleDeadline) {
		return "read_idle"
	}
	if !ctx.readIdleArmed {
		idle := p.ReadIdleDuration()
		deadline := time.Now().Add(idle)
		maxEnd := ctx.OpenedAt.Add(p.MaxLifetimeDuration())
		if deadline.After(maxEnd) {
			deadline = maxEnd
		}
		_ = c.SetReadDeadline(deadline)
		ctx.readIdleDeadline = deadline
		ctx.readIdleArmed = true
	}
	return ""
}

func ClearReadDeadline(c gnet.Conn) {
	_ = c.SetReadDeadline(time.Time{})
}
