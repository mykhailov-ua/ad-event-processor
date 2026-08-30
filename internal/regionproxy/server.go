package regionproxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bserver "ad-event-processor/internal/broker"
	"ad-event-processor/pkg/broker/log"
	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/gnetutil"
	"ad-event-processor/pkg/iogate"
	"ad-event-processor/pkg/lifecycle"
	"ad-event-processor/pkg/netaddr"
	"ad-event-processor/pkg/regionproxy/keygen"
	"ad-event-processor/pkg/regionproxy/opkey"
	"ad-event-processor/pkg/regionproxy/uplink"
	"ad-event-processor/pkg/regionproxy/wal"

	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DefaultIngressTopic = "region-proxy-ingress"
	proxyBackpressure   = byte(7)
)

var bytePool = sync.Pool{
	New: func() any {
		b := make([]byte, 32)
		return &b
	},
}

var fetchRespPool = sync.Pool{
	New: func() any {
		b := make([]byte, 1024*1024)
		return &b
	},
}

type ReadyProbe func(ctx context.Context) error

// Server: region ingress over gnet; local WAL segment + uplink worker to home region broker.
type Server struct {
	*gnet.BuiltinEventEngine
	addr            string
	healthAddr      string
	dataDir         string
	gate            *iogate.DiskWriteGate
	segment         *wal.WAL
	partition       *wal.Partition
	topicKey        string
	registry        *protocol.TopicRegistry
	coord           *bserver.Coordinator
	readyProbe      ReadyProbe
	closeChan       chan struct{}
	closeOnce       sync.Once
	wg              sync.WaitGroup
	engMu           sync.Mutex
	eng             gnet.Engine
	httpSrv         *http.Server
	keygen          *keygen.KeyGen
	opkey           *opkey.Pool
	uplink          *uplink.Worker
	active          atomic.Bool
	shutdownTimeout time.Duration
	connReadIdle    time.Duration
	connMaxLifetime time.Duration
	connCount       atomic.Int64
}

func NewServer(addr, dataDir string, gate *iogate.DiskWriteGate) (*Server, error) {
	if gate == nil {
		gate = iogate.NewDiskWriteGate(iogate.DefaultConfig())
	}
	segment, err := wal.Open(dataDir, gate)
	if err != nil {
		return nil, err
	}
	topicKey := protocol.TopicPartitionID(DefaultIngressTopic, 0)
	s := &Server{
		BuiltinEventEngine: &gnet.BuiltinEventEngine{},
		addr:               addr,
		healthAddr:         "127.0.0.1:0",
		dataDir:            dataDir,
		gate:               gate,
		segment:            segment,
		partition:          wal.NewPartition(segment),
		topicKey:           topicKey,
		registry:           protocol.NewTopicRegistry(),
		closeChan:          make(chan struct{}),
	}
	s.active.Store(true)
	if _, err := s.registry.Register(DefaultIngressTopic); err != nil {
		_ = segment.Close()
		return nil, err
	}
	return s, nil
}

func (s *Server) SetKeyGen(cfg keygen.Config) {
	s.keygen = keygen.New(s.segment, cfg)
}

func (s *Server) KeyGen() *keygen.KeyGen {
	return s.keygen
}

func (s *Server) SetOpKey(cfg opkey.Config) {
	s.opkey = opkey.New(s.segment, cfg)
}

func (s *Server) OpKey() *opkey.Pool {
	return s.opkey
}

func (s *Server) SetUplink(cfg uplink.Config) {
	if s.opkey == nil {
		s.opkey = opkey.New(s.segment, opkey.Config{NodeID: cfg.NodeID})
	}
	s.uplink = uplink.New(s.segment, s.opkey, cfg)
}

func (s *Server) Uplink() *uplink.Worker {
	return s.uplink
}

func (s *Server) SetHealthAddr(addr string) {
	s.healthAddr = addr
}

func (s *Server) SetReadyProbe(probe ReadyProbe) {
	s.readyProbe = probe
}

func (s *Server) SetShutdownTimeout(d time.Duration) {
	if d > 0 {
		s.shutdownTimeout = d
	}
}

func (s *Server) SetConnReadIdle(d time.Duration) {
	s.connReadIdle = d
}

func (s *Server) SetConnMaxLifetime(d time.Duration) {
	s.connMaxLifetime = d
}

func (s *Server) SetCoordinator(coord *bserver.Coordinator) {
	s.coord = coord
}

func (s *Server) CoordGetOrCreatePartition(topic string) (bserver.CoordPartition, error) {
	if topic != s.topicKey {
		return nil, fmt.Errorf("unknown topic %q", topic)
	}
	return s.partition, nil
}

func (s *Server) CoordRangeTopics(fn func(topic string) bool) {
	fn(s.topicKey)
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) HealthAddr() string {
	return s.healthAddr
}

func (s *Server) WAL() *wal.WAL {
	return s.segment
}

func (s *Server) Gate() *iogate.DiskWriteGate {
	return s.gate
}

func (s *Server) Start() error {
	if strings.HasSuffix(s.addr, ":0") && !netaddr.IsUnixSocketPath(s.addr) {
		l, err := net.Listen("tcp", s.addr)
		if err != nil {
			return err
		}
		s.addr = l.Addr().String()
		_ = l.Close()
	}
	if strings.HasSuffix(s.healthAddr, ":0") && !netaddr.IsUnixSocketPath(s.healthAddr) {
		l, err := net.Listen("tcp", s.healthAddr)
		if err != nil {
			return err
		}
		s.healthAddr = l.Addr().String()
		_ = l.Close()
	}
	if netaddr.IsUnixSocketPath(s.addr) {
		if err := netaddr.PrepareUnixSocket(s.addr); err != nil {
			return err
		}
	}
	if netaddr.IsUnixSocketPath(s.healthAddr) {
		if err := netaddr.PrepareUnixSocket(s.healthAddr); err != nil {
			return err
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.Handle("/metrics", promhttp.Handler())
	s.httpSrv = &http.Server{Addr: s.healthAddr, Handler: mux}
	lifecycle.ApplySidecarHTTPServerTimeouts(s.httpSrv)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if netaddr.IsUnixSocketPath(s.healthAddr) {
			ln, err := netaddr.ListenUnix(s.healthAddr)
			if err != nil {
				slog.Error("region-proxy health listen failed", "addr", s.healthAddr, "error", err)
				return
			}
			_ = s.httpSrv.Serve(ln)
			return
		}
		_ = s.httpSrv.ListenAndServe()
	}()

	s.wg.Add(1)
	errChan := make(chan error, 1)
	go func() {
		defer s.wg.Done()
		err := gnet.Run(s, netaddr.GnetListenURI(s.addr), gnet.WithMulticore(true))
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(100 * time.Millisecond):
		if s.keygen != nil {
			s.keygen.Start()
		}
		if s.opkey != nil {
			s.opkey.Start()
		}
		if s.uplink != nil {
			s.uplink.Start()
		}
		return nil
	}
}

func (s *Server) Stop() {
	s.closeOnce.Do(func() {
		s.active.Store(false)
		close(s.closeChan)
		timeout := s.shutdownTimeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if s.httpSrv != nil {
			_ = s.httpSrv.Shutdown(shutdownCtx)
		}
		s.engMu.Lock()
		eng := s.eng
		s.engMu.Unlock()
		_ = eng.Stop(shutdownCtx)
		if s.keygen != nil {
			s.keygen.Stop()
		}
		if s.opkey != nil {
			s.opkey.Stop()
		}
		if s.uplink != nil {
			s.uplink.Stop()
		}
		if s.segment != nil {
			_ = s.segment.Close()
			s.segment = nil
		}
	})
	s.wg.Wait()
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	s.engMu.Lock()
	s.eng = eng
	s.engMu.Unlock()
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	s.connCount.Add(1)
	ctx := s.newConnState()
	gnetutil.OpenConn(c, s.connPolicy(), ctx)
	return nil, gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	s.connCount.Add(-1)
	return gnet.None
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if !s.active.Load() {
		http.Error(w, "not active", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if !s.active.Load() {
		http.Error(w, "not active", http.StatusServiceUnavailable)
		return
	}
	if s.gate.Degraded() {
		http.Error(w, "proxy_backpressure", http.StatusServiceUnavailable)
		return
	}
	if s.readyProbe != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := s.readyProbe(ctx); err != nil {
			http.Error(w, "deps unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	ctx := s.ensureConnState(c)
	if s.connMaxLifetimeExceeded(ctx) {
		return s.closeConnIdle(c, "max_lifetime")
	}

	const maxFrame = uint32(1 << 20)

	for {
		lenBuf, err := c.Peek(4)
		if err != nil {
			if act := s.waitIncomplete(c, ctx); act != gnet.None {
				return act
			}
			return gnet.None
		}
		length := binary.BigEndian.Uint32(lenBuf)

		if length < 14 || length > maxFrame {
			need := int(4 + length)
			if length < 14 {
				if c.InboundBuffered() < need {
					if act := s.waitIncomplete(c, ctx); act != gnet.None {
						return act
					}
					return gnet.None
				}
				if _, err := c.Discard(need); err != nil {
					return gnet.Close
				}
				return gnet.Close
			}
			if _, err := c.Discard(4); err != nil {
				return gnet.Close
			}
			return gnet.Close
		}

		need := int(4 + length)
		if c.InboundBuffered() < need {
			if act := s.waitIncomplete(c, ctx); act != gnet.None {
				return act
			}
			return gnet.None
		}

		payloadBuf, err := c.Peek(need)
		if err != nil {
			if act := s.waitIncomplete(c, ctx); act != gnet.None {
				return act
			}
			return gnet.None
		}
		if len(payloadBuf) < need {
			if act := s.waitIncomplete(c, ctx); act != gnet.None {
				return act
			}
			return gnet.None
		}

		framePayload := payloadBuf[4 : 4+length]
		cmd := binary.BigEndian.Uint16(framePayload[0:2])
		seq := binary.BigEndian.Uint64(framePayload[2:10])
		reqPayload := framePayload[10 : length-4]
		expected := binary.BigEndian.Uint32(framePayload[length-4:])
		if crc32.ChecksumIEEE(reqPayload) != expected {
			if _, err := c.Discard(int(4 + length)); err != nil {
				return gnet.Close
			}
			return gnet.Close
		}
		switch cmd {
		case protocol.CmdProduceBatch:
			s.handleProduceBatch(c, seq, reqPayload)
		case protocol.CmdRegisterTopic:
			s.handleRegisterTopic(c, seq, reqPayload)
		case protocol.CmdFetch:
			s.handleFetch(c, seq, reqPayload)
		default:
			if _, err := c.Discard(int(4 + length)); err != nil {
				return gnet.Close
			}
			return gnet.Close
		}
		if _, err := c.Discard(int(4 + length)); err != nil {
			return gnet.Close
		}
		s.onFrameProgress(c, ctx)
	}
}

func (s *Server) handleRegisterTopic(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	name, err := protocol.DecodeRegisterTopicRequest(payload)
	if err != nil {
		_, _ = c.Write(protocol.EncodeRegisterTopicResponse(buf, seq, 1, 0))
		return
	}
	id, err := s.registry.Register(name)
	if err != nil {
		_, _ = c.Write(protocol.EncodeRegisterTopicResponse(buf, seq, 2, 0))
		return
	}
	_, _ = c.Write(protocol.EncodeRegisterTopicResponse(buf, seq, 0, id))
}

func (s *Server) handleProduceBatch(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	if s.gate.Degraded() {
		resp := protocol.EncodeProduceBatchResponse(buf, seq, proxyBackpressure, 0, 0)
		_, _ = c.Write(resp)
		return
	}
	if s.opkey != nil && s.opkey.ShouldShed() {
		resp := protocol.EncodeProduceBatchResponse(buf, seq, proxyBackpressure, 0, 0)
		_, _ = c.Write(resp)
		return
	}

	if s.coord != nil && !s.coord.IsLeader(s.topicKey) {
		hasLeader, _ := s.coord.HasLeader(s.topicKey)
		if hasLeader {
			resp := protocol.EncodeProduceBatchResponse(buf, seq, 4, 0, 0)
			_, _ = c.Write(resp)
			return
		}
	}
	if s.coord != nil && s.coord.IsLeader(s.topicKey) && !s.coord.IsLeaderReady(s.topicKey) {
		resp := protocol.EncodeProduceBatchResponse(buf, seq, 6, 0, 0)
		_, _ = c.Write(resp)
		return
	}

	it := protocol.NewBatchIterator(payload)
	var lastOffset uint64
	var committed uint32
	var status byte
	var payloads [][]byte

	for it.Next() {
		if _, exists := s.registry.Lookup(it.TopicID); !exists {
			status = 2
			break
		}
		payloads = append(payloads, append([]byte(nil), it.Payload...))
	}

	if status == 0 && len(payloads) > 0 {
		var epoch uint64
		if s.coord != nil {
			if ep, ok := s.coord.LeaderEpoch(s.topicKey); ok {
				epoch = ep
			}
		}
		for i, p := range payloads {
			offset, err := s.partition.AppendLeader(epoch, p)
			if errors.Is(err, log.ErrStaleFencingEpoch) {
				status = 5
				committed = uint32(i)
				break
			}
			if err != nil {
				status = 3
				committed = uint32(i)
				break
			}
			lastOffset = offset
			committed++
		}
		if s.coord != nil && s.coord.IsLeader(s.topicKey) && committed > 0 {
			s.coord.PublishLogHWM(context.Background(), s.topicKey, s.partition.NextOffset())
		}
	}

	resp := protocol.EncodeProduceBatchResponse(buf, seq, status, lastOffset, committed)
	_, _ = c.Write(resp)
}

func (s *Server) handleFetch(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	_, _, startOffset, maxBytes, err := protocol.DecodeFetchRequest(payload)
	if err != nil {
		s.writeFetchResponse(c, buf, seq, 1, 0, 0, 0, nil)
		return
	}
	hwm := s.partition.NextOffset()
	data, dataBuf, err := s.partition.ReadRawMessages(startOffset, maxBytes)
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.writeFetchResponse(c, buf, seq, 0, 0, 0, hwm, nil)
			return
		}
		s.writeFetchResponse(c, buf, seq, 3, 0, 0, hwm, nil)
		return
	}
	if dataBuf != nil {
		defer fetchRespPool.Put(dataBuf)
	}
	msgCount, msgBytes := countMessages(data)
	s.writeFetchResponse(c, buf, seq, 0, msgCount, msgBytes, hwm, data)
}

func (s *Server) writeFetchResponse(c gnet.Conn, buf []byte, seq uint64, status byte, msgCount, msgBytes uint32, hwm uint64, data []byte) {
	_ = buf
	frameLen := 4 + 2 + 8 + protocol.FetchRespMetaLen + len(data) + 4
	if frameLen <= 128 {
		var scratch [128]byte
		frame := protocol.EncodeFetchResponse(scratch[:], seq, status, msgCount, msgBytes, hwm, data)
		_, _ = c.Write(frame)
		return
	}
	framePtr := fetchRespPool.Get().(*[]byte)
	poolBuf := *framePtr
	if len(poolBuf) < frameLen {
		poolBuf = make([]byte, frameLen)
		*framePtr = poolBuf
	}
	frame := protocol.EncodeFetchResponse(poolBuf[:frameLen], seq, status, msgCount, msgBytes, hwm, data)
	_, _ = c.Write(frame)
	fetchRespPool.Put(framePtr)
}

func countMessages(buf []byte) (count, total uint32) {
	pos := 0
	for pos+12 <= len(buf) {
		length := binary.BigEndian.Uint32(buf[pos : pos+4])
		recordLen := int(12 + int(length) - 8)
		if pos+recordLen > len(buf) {
			break
		}
		count++
		total += uint32(recordLen)
		pos += recordLen
	}
	return count, total
}

func OpenDataDir(base string) string {
	return filepath.Join(base, "wal")
}

func ProbeDiskWritable(dataDir string) bool {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return false
	}
	testFile := filepath.Join(dataDir, ".healthcheck")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return true
}

func (s *Server) LogStart() {
	slog.Info("region-proxy ingress running", "addr", s.addr, "health_addr", s.healthAddr, "wal_seq", s.segment.NextSeq())
}
