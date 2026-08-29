package gnet

import (
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

const http1MaxBufferedOverhead = 8192

func http1HeadersComplete(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return true
		}
	}
	return false
}

func (h *Server) http1IncompleteMax() uint8 {
	limit := uint8(3)
	if h == nil || h.cfg == nil {
		return limit
	}
	if h.cfg.HTTP1IncompleteMax <= 0 {
		return limit
	}
	if h.cfg.HTTP1IncompleteMax > 255 {
		return 255
	}
	return uint8(h.cfg.HTTP1IncompleteMax)
}

func (h *Server) http1BodyIdleDuration() time.Duration {
	ms := 5000
	if h != nil && h.cfg != nil {
		if h.cfg.HTTP1BodyIdleMs > 0 {
			ms = h.cfg.HTTP1BodyIdleMs
		} else if h.cfg.Env != "production" {
			ms = 500
		}
	}
	return time.Duration(ms) * time.Millisecond
}

func (h *Server) http1MaxConnLifetimeDuration() time.Duration {
	if h == nil || h.cfg == nil || h.cfg.HTTP1MaxConnLifetimeMs <= 0 {
		return 0
	}
	return time.Duration(h.cfg.HTTP1MaxConnLifetimeMs) * time.Millisecond
}

func (h *Server) http1MaxBufferedBytes() int64 {
	maxBody := int64(1 << 20)
	if h != nil && h.cfg != nil {
		maxBody = h.cfg.MaxRequestBodySize
	}
	return maxBody + http1MaxBufferedOverhead
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

func (h *Server) http1OffloadAsyncWriteDone(c pkgnet.Conn, offloadCtx, connCtx *ConnContext, lease asyncWriteLease) {
	putAsyncWriteLease(lease)
	if connCtx != nil && connCtx.HTTP1PendingOffloadWrites.Add(-1) != 0 {
		return
	}
	h.http1OffloadWriteDone(c, offloadCtx)
}

func (h *Server) http1EnsureConnContext(c pkgnet.Conn) *ConnContext {
	if connCtx := http1ConnContext(c); connCtx != nil {
		return connCtx
	}
	ctx := h.allocConnContext(c)
	c.SetContext(ctx)
	return ctx
}

func (h *Server) http1ResetIncompleteState(ctx *ConnContext, c pkgnet.Conn) {
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

func (h *Server) http1ArmBodyIdle(c pkgnet.Conn, ctx *ConnContext) {
	if ctx == nil || c == nil || ctx.HTTP1BodyIdleArmed {
		return
	}
	idle := h.http1BodyIdleDuration()
	if idle <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(idle))
	ctx.HTTP1BodyIdleDeadline = filter.MonotonicNano() + idle.Nanoseconds()
	ctx.HTTP1BodyIdleArmed = true
}

func (h *Server) http1CheckBodyIdle(c pkgnet.Conn, ctx *ConnContext) pkgnet.Action {
	if ctx == nil {
		return pkgnet.None
	}
	maxLife := h.http1MaxConnLifetimeDuration()
	if maxLife > 0 && ctx.HTTP1ConnOpenedMono > 0 &&
		filter.MonotonicNano()-ctx.HTTP1ConnOpenedMono >= maxLife.Nanoseconds() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}
	if ctx.HTTP1BodyIdleDeadline == 0 {
		return pkgnet.None
	}
	if filter.MonotonicNano() < ctx.HTTP1BodyIdleDeadline {
		return pkgnet.None
	}
	metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
	h.http1ResetIncompleteState(ctx, c)
	return pkgnet.Close
}

func (h *Server) http1OffloadWriteDone(c pkgnet.Conn, offloadCtx *ConnContext) {
	if h == nil || h.workerPool == nil || c == nil {
		return
	}
	var connCtx *ConnContext
	if offloadCtx != nil {
		connCtx = offloadCtx.HTTP1ConnCtx
	}
	if connCtx == nil {
		connCtx = http1ConnContext(c)
	}
	if connCtx != nil {
		c.SetContext(connCtx)
		connCtx.HTTP1OffloadBusy.Store(false)
	}
	if offloadCtx != nil && offloadCtx.OffloadCloseAfterWrite.Load() {
		h.http1ResetIncompleteState(connCtx, c)
		_ = c.Close()
		return
	}
	if c.InboundBuffered() > 0 {
		_ = c.Wake(nil)
	}
}

func (h *Server) http1HandleIncomplete(c pkgnet.Conn, ctx *ConnContext, buf []byte, consumed int) pkgnet.Action {
	metrics.HTTPParseErrors.WithLabelValues("incomplete").Inc()

	if consumed > 0 {
		return pkgnet.None
	}

	if int64(len(buf)) > h.http1MaxBufferedBytes() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("buffer").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}

	if http1HeadersComplete(buf) {
		h.http1ArmBodyIdle(c, ctx)
	}

	ctx.HTTP1IncompleteSpin++
	if ctx.HTTP1IncompleteSpin >= h.http1IncompleteMax() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("spin").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return pkgnet.Close
	}
	return pkgnet.None
}
