package gnet

import (
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

func (s *Server) h2BodyIdleDuration() time.Duration {
	return s.http1BodyIdleDuration()
}

func (s *Server) h2ResetIncompleteIdle(st *httpingress.H2ConnState, c pkgnet.Conn) {
	if st == nil {
		return
	}
	st.IncompleteIdleArmed = false
	st.IncompleteIdleDeadline = 0
	if c != nil {
		_ = c.SetReadDeadline(time.Time{})
	}
}

func (s *Server) h2ArmIncompleteIdle(c pkgnet.Conn, st *httpingress.H2ConnState) {
	if st == nil || c == nil || st.IncompleteIdleArmed {
		return
	}
	idle := s.h2BodyIdleDuration()
	if idle <= 0 {
		return
	}
	_ = c.SetReadDeadline(time.Now().Add(idle))
	st.IncompleteIdleDeadline = filter.MonotonicNano() + idle.Nanoseconds()
	st.IncompleteIdleArmed = true
}

func (s *Server) h2CheckConnDeadlines(c pkgnet.Conn, ctx *ConnContext) pkgnet.Action {
	if ctx == nil {
		return pkgnet.None
	}
	maxLife := s.http1MaxConnLifetimeDuration()
	if maxLife > 0 && ctx.HTTP1ConnOpenedMono > 0 &&
		filter.MonotonicNano()-ctx.HTTP1ConnOpenedMono >= maxLife.Nanoseconds() {
		metrics.H2HostileDisconnectTotal.Inc()
		s.h2ResetIncompleteIdle(&ctx.H2, c)
		return pkgnet.Close
	}
	st := &ctx.H2
	if st.IncompleteIdleDeadline == 0 {
		return pkgnet.None
	}
	if filter.MonotonicNano() < st.IncompleteIdleDeadline {
		return pkgnet.None
	}
	metrics.H2HostileDisconnectTotal.Inc()
	s.h2ResetIncompleteIdle(st, c)
	return pkgnet.Close
}
