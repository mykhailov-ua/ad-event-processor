package gnet

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/track"
	"ad-event-processor/pkg/logger"

	pkgnet "github.com/panjf2000/gnet/v2"
)

type Logger = logger.Logger

type Server struct {
	pkgnet.BuiltinEventEngine
	eng                *pkgnet.Engine
	cfg                *config.Config
	reactor            Reactor
	logger             *Logger
	loggerShardCounter atomic.Uint64
	contextPool        sync.Pool
	workerPool         *PinnedWorkerPool
	connWorkerAssign   atomic.Uint64
	onTrackStatus      func(status int)
	monoElapsed        func(start int64) float64
}

type ServerConfig struct {
	Cfg               *config.Config
	Reactor           Reactor
	Logger            *Logger
	RecordTrackStatus func(status int)
	MonoElapsed       func(start int64) float64
}

func NewServer(sc ServerConfig) *Server {
	s := &Server{
		cfg:           sc.Cfg,
		reactor:       sc.Reactor,
		logger:        sc.Logger,
		onTrackStatus: sc.RecordTrackStatus,
		monoElapsed:   sc.MonoElapsed,
	}
	s.contextPool = sync.Pool{New: func() any { return newConnContext() }}
	return s
}

func newConnContext() *ConnContext {
	return &ConnContext{
		PBReq:          pb.AdEvent{Metadata: &pb.EventMetadata{}},
		TrackReq:       track.Request{Payload: make([]byte, 0, 512)},
		Evt:            domain.Event{Payload: make([]byte, 0, 1024)},
		ValSlice:       make([]any, 18),
		BufSlice:       make([]byte, 4096),
		ExtraBuf:       make([]byte, 0, 4096),
		OffloadHTTPPin: make([]byte, 0, 2048),
		ChunkScratch:   make([]byte, 0, 4096),
		H2:             httpingress.NewH2ConnState(),
		WReqID:         filter.BufWrapper{Buf: make([]byte, 0, 128)},
		WCamp:          filter.BufWrapper{Buf: make([]byte, 0, 128)},
		WTime:          filter.BufWrapper{Buf: make([]byte, 0, 128)},
		WorkerID:       -1,
	}
}

func (s *Server) SetReactor(r Reactor) { s.reactor = r }
func (s *Server) SetLogger(l *Logger)  { s.logger = l }
func (s *Server) SetWorkerPool(wp *PinnedWorkerPool) {
	s.workerPool = wp
	if wp != nil {
		wp.handler = s
	}
}

func (s *Server) React(req Request, c pkgnet.Conn) pkgnet.Action {
	if s.reactor == nil {
		return pkgnet.None
	}
	return s.reactor.React(req, c)
}

func (s *Server) HasWorkerPool() bool {
	return s != nil && s.workerPool != nil
}

func (s *Server) ReleaseOffloadBuffers(ctx *ConnContext) {
	s.releaseOffloadBuffers(ctx)
}

func (s *Server) RetireOffloadContext(ctx *ConnContext) {
	s.retireOffloadContext(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	if s.eng != nil {
		return s.eng.Stop(ctx)
	}
	return nil
}

func (s *Server) Write(c pkgnet.Conn, data []byte, ctx *ConnContext)      { s.write(c, data, ctx) }
func (s *Server) WriteClose(c pkgnet.Conn, data []byte, ctx *ConnContext) { s.writeClose(c, data, ctx) }
func (s *Server) WriteFilterReject(c pkgnet.Conn, data []byte, ctx *ConnContext, closeDup bool) {
	if closeDup {
		s.writeClose(c, data, ctx)
		return
	}
	s.write(c, data, ctx)
}
func (s *Server) AllocConnContext(c pkgnet.Conn) *ConnContext { return s.allocConnContext(c) }

func (s *Server) recordTrackStatus(status int) {
	if s.onTrackStatus != nil {
		s.onTrackStatus(status)
	}
}

func (s *Server) releaseOffloadBuffers(ctx *ConnContext) {
	if ctx == nil {
		return
	}
	if ctx.OffloadRelease != nil {
		ctx.OffloadRelease()
		ctx.OffloadRelease = nil
		ctx.OffloadReqSlice = nil
	} else if ctx.OffloadReqBuf != nil {
		putRequestBuffer(ctx.OffloadReqBuf)
		ctx.OffloadReqBuf = nil
	}
}

func (s *Server) retireOffloadContext(ctx *ConnContext) {
	if ctx == nil || !ctx.OffloadRetired.CompareAndSwap(false, true) {
		return
	}
	s.resetConnContextForReuse(ctx)
	s.contextPool.Put(ctx)
}

func (s *Server) write(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	s.writeMaybeClose(c, data, ctx, false)
}

func (s *Server) writeClose(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	s.writeMaybeClose(c, data, ctx, true)
}

func (s *Server) writeFilterReject(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	if s != nil && s.workerPool != nil && bytes.Equal(data, respDuplicate) {
		s.writeClose(c, data, ctx)
		return
	}
	s.write(c, data, ctx)
}

func (s *Server) writeMaybeClose(c pkgnet.Conn, data []byte, ctx *ConnContext, closeAfter bool) {
	if closeAfter && ctx != nil {
		ctx.OffloadCloseAfterWrite.Store(true)
	}
	if ctx != nil && ctx.ProtoH2 && ctx.H2StreamID != 0 {
		buf := ctx.BufSlice
		if cap(buf) < len(data)+512 {
			buf = make([]byte, len(data)+512)
			ctx.BufSlice = buf
		}
		if n, err := httpingress.H2WrapH1Response(buf, ctx.H2StreamID, data); err == nil {
			data = buf[:n]
		}
	}
	if s.workerPool != nil && ctx != nil {
		connCtx := http1ConnContextForWrite(ctx)
		writeLease := cloneAsyncWriteBytes(data)
		s.releaseOffloadBuffers(ctx)
		ctx.OffloadAsyncWrite.Store(true)
		if connCtx != nil {
			connCtx.HTTP1PendingOffloadWrites.Add(1)
		}
		_ = c.AsyncWrite(writeLease.buf, func(c pkgnet.Conn, err error) error {
			s.retireOffloadContext(ctx)
			s.http1OffloadAsyncWriteDone(c, ctx, connCtx, writeLease)
			return nil
		})
	} else {
		_, _ = c.Write(data)
		if closeAfter {
			_ = c.Close()
		}
	}
}

func (s *Server) OnBoot(eng pkgnet.Engine) (action pkgnet.Action) {
	slog.Info("gnet server is booting")
	s.eng = &eng
	return pkgnet.None
}

func (s *Server) OnOpen(c pkgnet.Conn) (out []byte, action pkgnet.Action) {
	metrics.GnetActiveConnections.Inc()
	return nil, pkgnet.None
}

func (s *Server) OnClose(c pkgnet.Conn, err error) (action pkgnet.Action) {
	metrics.GnetActiveConnections.Dec()
	if rawCtx, ok := c.Context().(*ConnContext); ok && rawCtx != nil && rawCtx.HTTP1ConnCtx != nil {
		s.releaseOffloadBuffers(rawCtx)
		s.retireOffloadContext(rawCtx)
	}
	if connCtx := http1ConnContext(c); connCtx != nil {
		connCtx.HTTP1OffloadBusy.Store(false)
		connCtx.HTTP1PendingOffloadWrites.Store(0)
		httpingress.ResetChunkScratch(&connCtx.ChunkScratch)
		s.retireConnContext(connCtx)
	}
	return pkgnet.None
}

func (s *Server) OnTraffic(c pkgnet.Conn) (action pkgnet.Action) {
	loopStart := filter.MonotonicNano()
	defer func() {
		metrics.GnetEventLoopWorkDuration.Add(filter.MonoElapsedSeconds(loopStart))
	}()

	for {
		inboundBuffered := c.InboundBuffered()
		if inboundBuffered == 0 {
			break
		}
		buf, err := c.Peek(inboundBuffered)
		if err != nil {
			return pkgnet.Close
		}

		metrics.GnetBytesReceived.Add(float64(len(buf)))
		metrics.GnetPacketsReceived.Inc()

		if httpingress.IsH2ClientPreface(buf) {
			if act := s.onTrafficH2(c, buf); act != pkgnet.None {
				return act
			}
			continue
		}
		if ctx, ok := c.Context().(*ConnContext); ok && ctx != nil && ctx.ProtoH2 {
			if act := s.onTrafficH2(c, buf); act != pkgnet.None {
				return act
			}
			continue
		}

		var scratchPtr *[]byte
		connCtx := s.http1EnsureConnContext(c)
		// Tier A invariant: one in-flight offload per HTTP/1 conn; epoll stops parsing until Tier B clears busy.
		if connCtx.HTTP1OffloadBusy.Load() {
			break
		}
		if act := s.http1CheckBodyIdle(c, connCtx); act != pkgnet.None {
			return act
		}
		scratchPtr = &connCtx.ChunkScratch

		reqLen, req, err := s.parseHTTP(buf, scratchPtr)
		if err != nil {
			if errors.Is(err, httpingress.ErrIncomplete) {
				if act := s.http1HandleIncomplete(c, connCtx, buf, reqLen); act != pkgnet.None {
					return act
				}
				break
			}
			if errors.Is(err, httpingress.ErrPayloadTooLarge) {
				metrics.HTTPParseErrors.WithLabelValues("payload_too_large").Inc()
				_, _ = c.Write(respPayloadTooLarge)
				s.recordTrackStatus(http.StatusRequestEntityTooLarge)
				return pkgnet.Close
			}
			metrics.HTTPParseErrors.WithLabelValues("invalid").Inc()
			_, _ = c.Write(respBadRequestClose)
			return pkgnet.Close
		}

		s.http1ResetIncompleteState(connCtx, c)

		if s.workerPool != nil {
			offloadCtx := s.contextPool.Get().(*ConnContext)
			offloadCtx.OffloadAsyncWrite.Store(false)
			offloadCtx.OffloadCloseAfterWrite.Store(false)
			offloadCtx.OffloadRetired.Store(false)
			if connCtx.WorkerID < 0 {
				connCtx.WorkerID = int(s.connWorkerAssign.Add(1) % uint64(len(s.workerPool.workers)))
			}
			if s.logger != nil {
				offloadCtx.ShardID = int(s.loggerShardCounter.Add(1) % uint64(len(s.logger.Shards())))
			}
			offloadCtx.OffloadConn = c
			offloadCtx.HTTP1ConnCtx = connCtx
			offloadCtx.OffloadReqBuf = nil
			offloadCtx.OffloadReqSlice = nil
			offloadCtx.OffloadRelease = nil
			offloadCtx.OffloadReqLen = reqLen
			offloadCtx.OffloadReq = PinParsedHTTPRequest(offloadCtx, req)
			offloadCtx.OffloadReqPin = true
			offloadCtx.OffloadOnEnter = nil
			offloadCtx.OffloadBlock = nil
			offloadCtx.OffloadWG = nil

			// Tier A -> B: enqueue pinned ConnContext, then Discard peek frame on epoll thread only.
			connCtx.HTTP1OffloadBusy.Store(true)
			submitted := s.workerPool.SubmitOffloadToWorker(connCtx.WorkerID, offloadCtx, buf[:reqLen])
			if _, err := c.Discard(reqLen); err != nil {
				if !submitted {
					connCtx.HTTP1OffloadBusy.Store(false)
					s.retireOffloadContext(offloadCtx)
				}
				return pkgnet.Close
			}
			if !submitted {
				connCtx.HTTP1OffloadBusy.Store(false)
				s.retireOffloadContext(offloadCtx)
				metrics.WorkerPoolRejectTotal.Inc()
				s.write(c, respWorkerPoolOverload, nil)
				s.recordTrackStatus(http.StatusServiceUnavailable)
			}
			break
		} else {
			act := s.React(req, c)
			if _, err := c.Discard(reqLen); err != nil {
				return pkgnet.Close
			}

			if act != pkgnet.None {
				return act
			}
		}
	}
	return pkgnet.None
}

// runOffloadedRequest runs Tier B on a pinned worker: React, sync FilterEngine.Check (incl. EVALSHA), response write.
func (s *Server) runOffloadedRequest(WorkerID int, ctx *ConnContext) {
	if ctx == nil {
		return
	}
	if ctx.OffloadReqSlice == nil && ctx.OffloadReqBuf == nil && !ctx.OffloadReqPin {
		finishOffloadCtx(ctx)
		s.retireOffloadContext(ctx)
		return
	}
	// AsyncWrite path: defer skips arena retire until callback; cloneAsyncWriteBytes owns response bytes.
	defer func() {
		if !ctx.OffloadAsyncWrite.Load() {
			s.releaseOffloadBuffers(ctx)
			s.http1OffloadWriteDone(ctx.OffloadConn, ctx)
			s.retireOffloadContext(ctx)
		}
	}()

	ctx.WorkerID = WorkerID
	c := ctx.OffloadConn
	if c == nil {
		return
	}
	c.SetContext(ctx)
	var reqParsed Request
	if ctx.OffloadReqPin {
		reqParsed = ctx.OffloadReq
		ctx.OffloadReqPin = false
	} else {
		var reqBytes []byte
		if len(ctx.OffloadReqSlice) > 0 {
			reqBytes = ctx.OffloadReqSlice[:ctx.OffloadReqLen:ctx.OffloadReqLen]
		} else {
			reqBytes = (*ctx.OffloadReqBuf)[:ctx.OffloadReqLen:ctx.OffloadReqLen]
		}
		var err error
		_, reqParsed, err = s.parseHTTP(reqBytes, &ctx.ChunkScratch)
		if err != nil {
			s.writeClose(c, respBadRequestClose, ctx)
			return
		}
	}
	act := s.React(reqParsed, c)
	if act == pkgnet.Close && !ctx.OffloadCloseAfterWrite.Load() {
		ctx.OffloadCloseAfterWrite.Store(true)
		if !ctx.OffloadAsyncWrite.Load() {
			_ = c.Close()
		}
	}
}

func (s *Server) ParseHTTP(data []byte, scratchPtr *[]byte) (int, Request, error) {
	return s.parseHTTP(data, scratchPtr)
}

func (s *Server) parseHTTP(data []byte, scratchPtr *[]byte) (int, Request, error) {
	maxBody := int64(1 << 20)
	if s != nil && s.cfg != nil {
		maxBody = s.cfg.MaxRequestBodySize
	}
	return httpingress.ParseHTTP1(data, maxBody, scratchPtr)
}
