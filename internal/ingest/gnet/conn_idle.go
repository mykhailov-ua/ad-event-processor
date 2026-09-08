package gnet

import (
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

const http1MaxBufferedOverhead = 8192

func (s *Server) http1IncompleteMax() uint8 {
	limit := uint8(3)
	if s == nil || s.cfg == nil {
		return limit
	}
	if s.cfg.HTTP1IncompleteMax <= 0 {
		return limit
	}
	if s.cfg.HTTP1IncompleteMax > 255 {
		return 255
	}
	return uint8(s.cfg.HTTP1IncompleteMax)
}

func (s *Server) http1BodyIdleDuration() time.Duration {
	ms := 5000
	if s != nil && s.cfg != nil {
		if s.cfg.HTTP1BodyIdleMs > 0 {
			ms = s.cfg.HTTP1BodyIdleMs
		} else if s.cfg.Env != "production" {
			ms = 500
		}
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *Server) http1MaxConnLifetimeDuration() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.HTTP1MaxConnLifetimeMs <= 0 {
		return 0
	}
	return time.Duration(s.cfg.HTTP1MaxConnLifetimeMs) * time.Millisecond
}

func (s *Server) http1MaxBufferedBytes() int64 {
	maxBody := int64(1 << 20)
	if s != nil && s.cfg != nil {
		maxBody = s.cfg.MaxRequestBodySize
	}
	return maxBody + http1MaxBufferedOverhead
}

func HTTP1ConnContext(c pkgnet.Conn) *ConnContext {
	return http1ConnContext(c)
}

func http1ConnContext(c pkgnet.Conn) *ConnContext {
	if c == nil {
		return nil
	}
	ctx, ok := c.Context().(*ConnContext)
	if !ok || ctx == nil {
		return nil
	}
	if conn := ctx.HTTP1ConnCtx; conn != nil {
		return conn
	}
	return ctx
}

func http1ConnContextForWrite(ctx *ConnContext) *ConnContext {
	if ctx == nil {
		return nil
	}
	if conn := ctx.HTTP1ConnCtx; conn != nil {
		return conn
	}
	return ctx
}

type asyncWriteLease struct {
	buf     []byte
	poolPtr *[]byte
}

func cloneAsyncWriteBytes(src []byte) asyncWriteLease {
	bufPtr := responseBytesPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < len(src) {
		buf = make([]byte, len(src))
		*bufPtr = buf
	}
	buf = buf[:len(src)]
	copy(buf, src)
	return asyncWriteLease{buf: buf, poolPtr: bufPtr}
}

func putAsyncWriteLease(lease asyncWriteLease) {
	if lease.poolPtr == nil {
		return
	}
	responseBytesPool.Put(lease.poolPtr)
}

func (s *Server) http1OffloadAsyncWriteDone(c pkgnet.Conn, offloadCtx, connCtx *ConnContext, lease asyncWriteLease) {
	putAsyncWriteLease(lease)
	if connCtx != nil && connCtx.HTTP1PendingOffloadWrites.Add(-1) != 0 {
		return
	}
	if connCtx == nil {
		connCtx = http1ConnContext(c)
	}
	if connCtx != nil {
		c.SetContext(connCtx)
		connCtx.HTTP1OffloadBusy.Store(false)
	}
	if offloadCtx != nil && offloadCtx.OffloadCloseAfterWrite.Load() {
		s.http1ResetIncompleteState(connCtx, c)
		_ = c.Close()
		return
	}
	if c.InboundBuffered() > 0 {
		_ = c.Wake(nil)
	}
}

func (s *Server) http1EnsureConnContext(c pkgnet.Conn) *ConnContext {
	if connCtx := http1ConnContext(c); connCtx != nil {
		return connCtx
	}
	ctx := s.allocConnContext(c)
	c.SetContext(ctx)
	return ctx
}

func (s *Server) http1ResetIncompleteState(ctx *ConnContext, c pkgnet.Conn) {
	if ctx == nil {
		return
	}
	ctx.HTTP1IncompleteSpin = 0
	ctx.HTTP1BodyIdleArmed = false
	ctx.HTTP1BodyIdleDeadline = 0
	httpingress.ResetChunkScratch(&ctx.ChunkScratch)
	if c != nil {
		_ = c.SetReadDeadline(time.Time{})
	}
}

func (s *Server) http1ArmBodyIdle(c pkgnet.Conn, ctx *ConnContext) {
	if ctx == nil || c == nil || ctx.HTTP1BodyIdleArmed {
		return
	}
	idle := s.http1BodyIdleDuration()
	if idle <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(idle))
	ctx.HTTP1BodyIdleDeadline = filter.MonotonicNano() + idle.Nanoseconds()
	ctx.HTTP1BodyIdleArmed = true
}

func (s *Server) http1CheckBodyIdle(c pkgnet.Conn, ctx *ConnContext) pkgnet.Action {
	if ctx == nil {
		return pkgnet.None
	}
	maxLife := s.http1MaxConnLifetimeDuration()
	if maxLife > 0 && ctx.HTTP1ConnOpenedMono > 0 &&
		filter.MonotonicNano()-ctx.HTTP1ConnOpenedMono >= maxLife.Nanoseconds() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
		s.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}
	if ctx.HTTP1BodyIdleDeadline == 0 {
		return pkgnet.None
	}
	if filter.MonotonicNano() < ctx.HTTP1BodyIdleDeadline {
		return pkgnet.None
	}
	metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
	s.http1ResetIncompleteState(ctx, c)
	return pkgnet.Close
}

func (s *Server) http1OffloadWriteDone(c pkgnet.Conn, ctx *ConnContext) {
	if s == nil || s.workerPool == nil || c == nil {
		return
	}
	offloadCtx := ctx
	connCtx := ctx
	if ctx != nil && ctx.HTTP1ConnCtx != nil {
		connCtx = ctx.HTTP1ConnCtx
	}
	if connCtx == nil {
		connCtx = http1ConnContext(c)
	}
	if connCtx != nil {
		c.SetContext(connCtx)
		connCtx.HTTP1OffloadBusy.Store(false)
	}
	if offloadCtx != nil && offloadCtx.OffloadCloseAfterWrite.Load() {
		s.http1ResetIncompleteState(connCtx, c)
		_ = c.Close()
		return
	}
	if c.InboundBuffered() > 0 {
		_ = c.Wake(nil)
	}
}

func (s *Server) http1HandleIncomplete(c pkgnet.Conn, ctx *ConnContext, buf []byte, consumed int) pkgnet.Action {
	metrics.HTTPParseErrors.WithLabelValues("incomplete").Inc()

	// Arm once for header or body stall; monotonic deadline is not reset on drip bytes (H2 parity).
	s.http1ArmBodyIdle(c, ctx)

	if consumed > 0 {
		return pkgnet.None
	}

	if int64(len(buf)) > s.http1MaxBufferedBytes() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("buffer").Inc()
		s.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}

	ctx.HTTP1IncompleteSpin++
	if ctx.HTTP1IncompleteSpin >= s.http1IncompleteMax() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("spin").Inc()
		s.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}
	return pkgnet.None
}
