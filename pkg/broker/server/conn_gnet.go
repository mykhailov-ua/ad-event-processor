// Package server implements broker server helpers.
package server

import (
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/gnetutil"

	"github.com/panjf2000/gnet/v2"
)

type brokerConnState = gnetutil.ConnState

func (s *Server) connPolicy() gnetutil.ConnPolicy {
	return gnetutil.ConnPolicy{
		ReadIdle:    s.connReadIdle,
		MaxLifetime: s.connMaxLifetime,
	}
}

func (s *Server) newConnState() *brokerConnState {
	return gnetutil.NewConnState()
}

func (s *Server) ensureConnState(c gnet.Conn) *brokerConnState {
	return gnetutil.EnsureConnState(c)
}

func (s *Server) connMaxLifetimeExceeded(ctx *brokerConnState) bool {
	return gnetutil.MaxLifetimeExceeded(s.connPolicy(), ctx)
}

func (s *Server) closeConnIdle(c gnet.Conn, reason string) gnet.Action {
	metrics.BrokerConnIdleCloseTotal.WithLabelValues(reason).Inc()
	gnetutil.ClearReadDeadline(c)
	return gnet.Close
}

func (s *Server) onFrameProgress(c gnet.Conn, ctx *brokerConnState) {
	gnetutil.OnFrameProgress(c, s.connPolicy(), ctx)
}

func (s *Server) waitIncomplete(c gnet.Conn, ctx *brokerConnState) gnet.Action {
	if reason := gnetutil.WaitIncomplete(c, s.connPolicy(), ctx); reason != "" {
		return s.closeConnIdle(c, reason)
	}
	return gnet.None
}
