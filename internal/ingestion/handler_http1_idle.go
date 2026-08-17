package ingestion

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/panjf2000/gnet/v2"
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

func (h *AdsPacketHandler) http1IncompleteMax() uint8 {
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

func (h *AdsPacketHandler) http1BodyIdleDuration() time.Duration {
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

func (h *AdsPacketHandler) http1MaxConnLifetimeDuration() time.Duration {
	if h == nil || h.cfg == nil || h.cfg.HTTP1MaxConnLifetimeMs <= 0 {
		return 0
	}
	return time.Duration(h.cfg.HTTP1MaxConnLifetimeMs) * time.Millisecond
}

func (h *AdsPacketHandler) http1MaxBufferedBytes() int64 {
	maxBody := int64(1 << 20)
	if h != nil && h.cfg != nil {
		maxBody = h.cfg.MaxRequestBodySize
	}
	return maxBody + http1MaxBufferedOverhead
}

func (h *AdsPacketHandler) http1EnsureConnContext(c gnet.Conn) *connContext {
	ctx, ok := c.Context().(*connContext)
	if ok && ctx != nil {
		return ctx
	}
	ctx = h.allocConnContext(c)
	c.SetContext(ctx)
	return ctx
}

func (h *AdsPacketHandler) http1ResetIncompleteState(ctx *connContext, c gnet.Conn) {
	if ctx == nil {
		return
	}
	ctx.http1IncompleteSpin = 0
	ctx.http1BodyIdleArmed = false
	ctx.http1BodyIdleDeadline = 0
	resetChunkScratch(&ctx.chunkScratch)
	if c != nil {
		_ = c.SetReadDeadline(time.Time{})
	}
}

func (h *AdsPacketHandler) http1ArmBodyIdle(c gnet.Conn, ctx *connContext) {
	if ctx == nil || c == nil || ctx.http1BodyIdleArmed {
		return
	}
	idle := h.http1BodyIdleDuration()
	if idle <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(idle))
	ctx.http1BodyIdleDeadline = monotonicNano() + idle.Nanoseconds()
	ctx.http1BodyIdleArmed = true
}

func (h *AdsPacketHandler) http1CheckBodyIdle(c gnet.Conn, ctx *connContext) gnet.Action {
	if ctx == nil {
		return gnet.None
	}
	maxLife := h.http1MaxConnLifetimeDuration()
	if maxLife > 0 && ctx.http1ConnOpenedMono > 0 &&
		monotonicNano()-ctx.http1ConnOpenedMono >= maxLife.Nanoseconds() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return gnet.Close
	}
	if ctx.http1BodyIdleDeadline == 0 {
		return gnet.None
	}
	if monotonicNano() < ctx.http1BodyIdleDeadline {
		return gnet.None
	}
	metrics.HTTP1IncompleteCloseTotal.WithLabelValues("idle").Inc()
	h.http1ResetIncompleteState(ctx, c)
	return gnet.Close
}

func (h *AdsPacketHandler) http1HandleIncomplete(c gnet.Conn, ctx *connContext, buf []byte, consumed int) gnet.Action {
	metrics.HTTPParseErrors.WithLabelValues("incomplete").Inc()

	if consumed > 0 {
		return gnet.None
	}

	if int64(len(buf)) > h.http1MaxBufferedBytes() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("buffer").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return gnet.Close
	}

	if http1HeadersComplete(buf) {
		h.http1ArmBodyIdle(c, ctx)
	}

	ctx.http1IncompleteSpin++
	if ctx.http1IncompleteSpin >= h.http1IncompleteMax() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("spin").Inc()
		h.http1ResetIncompleteState(ctx, c)
		return gnet.Close
	}
	return gnet.None
}
