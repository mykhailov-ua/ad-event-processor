package ingestion

import (
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) h2BodyIdleDuration() time.Duration {
	return h.http1BodyIdleDuration()
}

func (h *AdsPacketHandler) h2ResetIncompleteIdle(st *h2ConnState, c gnet.Conn) {
	if st == nil {
		return
	}
	st.incompleteIdleArmed = false
	st.incompleteIdleDeadline = 0
	if c != nil {
		_ = c.SetReadDeadline(time.Time{})
	}
}

func (h *AdsPacketHandler) h2ArmIncompleteIdle(c gnet.Conn, st *h2ConnState) {
	if st == nil || c == nil || st.incompleteIdleArmed {
		return
	}
	idle := h.h2BodyIdleDuration()
	if idle <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(idle))
	st.incompleteIdleDeadline = monotonicNano() + idle.Nanoseconds()
	st.incompleteIdleArmed = true
}

func (h *AdsPacketHandler) h2CheckConnDeadlines(c gnet.Conn, ctx *connContext) gnet.Action {
	if ctx == nil {
		return gnet.None
	}
	maxLife := h.http1MaxConnLifetimeDuration()
	if maxLife > 0 && ctx.http1ConnOpenedMono > 0 &&
		monotonicNano()-ctx.http1ConnOpenedMono >= maxLife.Nanoseconds() {
		metrics.H2HostileDisconnectTotal.Inc()
		h.h2ResetIncompleteIdle(&ctx.h2, c)
		return gnet.Close
	}
	st := &ctx.h2
	if st.incompleteIdleDeadline == 0 {
		return gnet.None
	}
	if monotonicNano() < st.incompleteIdleDeadline {
		return gnet.None
	}
	metrics.H2HostileDisconnectTotal.Inc()
	h.h2ResetIncompleteIdle(st, c)
	return gnet.Close
}
