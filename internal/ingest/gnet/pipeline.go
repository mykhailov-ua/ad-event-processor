package gnet

import (
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

const defaultHTTP1MaxPipelineDepth = 4

func (s *Server) http1MaxPipelineDepth() int {
	if s != nil && s.cfg != nil && s.cfg.HTTP1MaxPipelineDepth > 0 {
		return s.cfg.HTTP1MaxPipelineDepth
	}
	return defaultHTTP1MaxPipelineDepth
}

func (s *Server) http1MaxPipelineBusyBytes() int64 {
	if s != nil && s.cfg != nil && s.cfg.HTTP1MaxPipelineBusyBytes > 0 {
		return s.cfg.HTTP1MaxPipelineBusyBytes
	}
	return 256 << 10
}

func (s *Server) http1MaxBodyBytes() int64 {
	maxBody := int64(1 << 20)
	if s != nil && s.cfg != nil {
		maxBody = s.cfg.MaxRequestBodySize
	}
	return maxBody
}

func (s *Server) http1ParseLimits() httpingress.ParseLimits {
	limits := httpingress.ParseLimits{}
	if s != nil && s.cfg != nil {
		limits.MinChunkedDataBytes = s.cfg.HTTP1MinChunkedDataBytes
	}
	return limits
}

// http1CheckPipelineBackpressure closes hostile keep-alive pipelining (depth and byte caps).
func (s *Server) http1CheckPipelineBackpressure(buf []byte, scratchPtr *[]byte) pkgnet.Action {
	if int64(len(buf)) > s.http1MaxPipelineBusyBytes() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("pipeline_buffer").Inc()
		return pkgnet.Close
	}
	depth := httpingress.CountCompleteHTTP1Messages(buf, s.http1MaxBodyBytes(), scratchPtr, s.http1ParseLimits())
	if depth > s.http1MaxPipelineDepth() {
		metrics.HTTP1IncompleteCloseTotal.WithLabelValues("pipeline_depth").Inc()
		return pkgnet.Close
	}
	return pkgnet.None
}

// h2CheckPipelineBackpressure applies the same conn ingress caps as HTTP/1 (HTTP1_MAX_PIPELINE_*).
func (s *Server) h2CheckPipelineBackpressure(ctx *ConnContext, buf []byte) pkgnet.Action {
	if int64(len(buf)) > s.http1MaxPipelineBusyBytes() {
		metrics.H2HostileDisconnectTotal.Inc()
		return pkgnet.Close
	}
	var st *httpingress.H2ConnState
	if ctx != nil {
		st = &ctx.H2
	}
	if httpingress.CountCompleteH2Messages(buf, s.http1MaxBodyBytes(), st) > s.http1MaxPipelineDepth() {
		metrics.H2HostileDisconnectTotal.Inc()
		return pkgnet.Close
	}
	return pkgnet.None
}
