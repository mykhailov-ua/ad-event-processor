package regionproxy

import (
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/gnetutil"

	"github.com/panjf2000/gnet/v2"
)

type proxyConnState = gnetutil.ConnState

func (s *Server) connPolicy() gnetutil.ConnPolicy {
	return gnetutil.ConnPolicy{
		ReadIdle:    s.connReadIdle,
		MaxLifetime: s.connMaxLifetime,
	}
}

func (s *Server) newConnState() *proxyConnState {
	return gnetutil.NewConnState()
}

func (s *Server) ensureConnState(c gnet.Conn) *proxyConnState {
	return gnetutil.EnsureConnState(c)
}

func (s *Server) connMaxLifetimeExceeded(ctx *proxyConnState) bool {
	return gnetutil.MaxLifetimeExceeded(s.connPolicy(), ctx)
}

func (s *Server) closeConnIdle(c gnet.Conn, reason string) gnet.Action {
	metrics.RegionProxyConnIdleCloseTotal.WithLabelValues(reason).Inc()
	gnetutil.ClearReadDeadline(c)
	return gnet.Close
}

func (s *Server) onFrameProgress(c gnet.Conn, ctx *proxyConnState) {
	gnetutil.OnFrameProgress(c, s.connPolicy(), ctx)
}

func (s *Server) waitIncomplete(c gnet.Conn, ctx *proxyConnState) gnet.Action {
	if reason := gnetutil.WaitIncomplete(c, s.connPolicy(), ctx); reason != "" {
		return s.closeConnIdle(c, reason)
	}
	return gnet.None
}
