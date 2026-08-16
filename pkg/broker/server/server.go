package server

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

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/broker/log"
	"github.com/bidshard/ad-event-processor/pkg/broker/protocol"
	"github.com/bidshard/ad-event-processor/pkg/gnetutil"
	"github.com/bidshard/ad-event-processor/pkg/iogate"
	"github.com/bidshard/ad-event-processor/pkg/netaddr"

	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var bytePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32)
		return &b
	},
}

var fetchRespPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 1024*1024)
		return &b
	},
}

type Server struct {
	*gnet.BuiltinEventEngine
	addr            string
	healthAddr      string
	dataDir         string
	maxSegSize      int64
	indexInterval   int64
	durability      log.DurabilityConfig
	topics          sync.Map
	initMu          sync.Mutex
	closeChan       chan struct{}
	closeOnce       sync.Once
	wg              sync.WaitGroup
	engMu           sync.Mutex
	eng             gnet.Engine
	active          atomic.Bool
	httpSrv         *http.Server
	coord           *Coordinator
	shutdownTimeout time.Duration
	registry        *protocol.TopicRegistry

	connCount       atomic.Int64
	maxConnections  int64
	connReadIdle    time.Duration
	connMaxLifetime time.Duration

	diskOK atomic.Bool

	diskGate *iogate.DiskWriteGate

	retention              log.RetentionPolicy
	retentionCheckInterval time.Duration

	offsetStore OffsetStore
}

func NewServer(addr string, dataDir string, maxSegSize int64, indexInterval int64) *Server {
	s := &Server{
		BuiltinEventEngine: &gnet.BuiltinEventEngine{},
		addr:               addr,
		healthAddr:         "127.0.0.1:0",
		dataDir:            dataDir,
		maxSegSize:         maxSegSize,
		indexInterval:      indexInterval,
		durability:         log.DefaultDurabilityConfig(),
		closeChan:          make(chan struct{}),
		registry:           protocol.NewTopicRegistry(),
		offsetStore:        NewMemoryOffsetStore(),
	}
	s.diskOK.Store(true)
	gateCfg := iogate.DefaultConfig()
	gateCfg.DiskWritable = s.diskOK.Load
	s.diskGate = iogate.NewDiskWriteGate(gateCfg)
	return s
}

func (s *Server) SetHealthAddr(addr string) {
	s.healthAddr = addr
}

func (s *Server) SetShutdownTimeout(d time.Duration) {
	if d > 0 {
		s.shutdownTimeout = d
	}
}

// SetConnReadIdle bounds how long a peer may drip partial length-prefixed frames (default 30s).
func (s *Server) SetConnReadIdle(d time.Duration) {
	s.connReadIdle = d
}

// SetConnMaxLifetime closes broker TCP connections after wall time regardless of traffic (default 120s).
func (s *Server) SetConnMaxLifetime(d time.Duration) {
	s.connMaxLifetime = d
}

func (s *Server) SetDurability(cfg log.DurabilityConfig) {
	s.durability = cfg
	if s.diskGate != nil {
		gateCfg := iogate.DefaultConfig()
		gateCfg.GroupCommitRecords = cfg.GroupCommitRecords
		gateCfg.GroupCommitInterval = cfg.FlushInterval
		gateCfg.DiskWritable = s.diskOK.Load
		s.diskGate = iogate.NewDiskWriteGate(gateCfg)
	}
}

func (s *Server) Durability() log.DurabilityConfig {
	return s.durability
}

func (s *Server) SetMaxConnections(n int64) {
	if n < 0 {
		n = 0
	}
	s.maxConnections = n
}

func (s *Server) MaxConnections() int64 {
	return s.maxConnections
}

func (s *Server) SetCoordinator(coord *Coordinator) {
	s.coord = coord
	s.wireTopicRegistryFromCoord(coord)
}

func (s *Server) wireTopicRegistryFromCoord(coord *Coordinator) {
	if coord == nil || s.registry == nil {
		return
	}
	store := NewRedisTopicStore(coord.Redis())
	s.registry.SetRedisStore(store)
	s.offsetStore = NewRedisOffsetStore(coord.Redis())

	ctx, cancel := context.WithTimeout(context.Background(), MergeTimeout())
	defer cancel()
	snap, err := store.Load(ctx)
	if err != nil {
		slog.Warn("failed to load topic registry from redis", "error", err)
		return
	}
	if err := s.registry.Merge(snap); err != nil {
		slog.Warn("failed to merge redis topic registry", "error", err)
	}
}

func (s *Server) HealthAddr() string {
	return s.healthAddr
}

func (s *Server) DiskGate() *iogate.DiskWriteGate {
	return s.diskGate
}

func (s *Server) SetDiskGate(g *iogate.DiskWriteGate) {
	if g != nil {
		s.diskGate = g
	}
}

func (s *Server) Start() error {
	store, err := protocol.NewFileRegistryStore(s.dataDir)
	if err != nil {
		return fmt.Errorf("topic registry store: %w", err)
	}
	s.registry.SetFileStore(store)
	if err := s.registry.Load(); err != nil {
		return fmt.Errorf("load topic registry: %w", err)
	}

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

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runDiskHealthWorker()
	}()

	if s.retention.Enabled() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runRetentionWorker()
		}()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/health", s.handleHealthz)
	mux.HandleFunc("/leaderz", s.handleLeaderz)
	mux.Handle("/metrics", promhttp.Handler())
	s.httpSrv = &http.Server{
		Addr:              s.healthAddr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if netaddr.IsUnixSocketPath(s.healthAddr) {
			ln, err := netaddr.ListenUnix(s.healthAddr)
			if err != nil {
				slog.Error("broker health listen failed", "addr", s.healthAddr, "error", err)
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
		addr := netaddr.GnetListenURI(s.addr)
		err := gnet.Run(s, addr,
			gnet.WithMulticore(true),
		)
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) runDiskHealthWorker() {
	const interval = 5 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeChan:
			return
		case <-ticker.C:
			ok := s.probeDisk()
			s.diskOK.Store(ok)
			if ok {
				metrics.BrokerDiskWritable.Set(1)
			} else {
				metrics.BrokerDiskWritable.Set(0)
			}
		}
	}
}

func (s *Server) probeDisk() bool {
	testFile := filepath.Join(s.dataDir, ".healthcheck")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return true
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if !s.active.Load() {
		http.Error(w, "server not active", http.StatusServiceUnavailable)
		return
	}

	if !s.diskOK.Load() {
		http.Error(w, "disk not writable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) handleLeaderz(w http.ResponseWriter, r *http.Request) {
	if !s.active.Load() {
		http.Error(w, "server not active", http.StatusServiceUnavailable)
		return
	}
	if !s.diskOK.Load() {
		http.Error(w, "disk not writable", http.StatusServiceUnavailable)
		return
	}

	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "tracker-logs"
	}

	if s.coord == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	if s.coord.IsLeader(topic) && s.coord.IsLeaderReady(topic) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	http.Error(w, "not leader", http.StatusServiceUnavailable)
}

func (s *Server) Stop() {
	s.closeOnce.Do(func() {
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
		if s.active.Load() {
			s.engMu.Lock()
			eng := s.eng
			s.engMu.Unlock()
			_ = eng.Stop(shutdownCtx)
		}

		s.topics.Range(func(_, val any) bool {
			_ = val.(*log.PartitionLog).Close()
			return true
		})
	})
	s.wg.Wait()
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	s.engMu.Lock()
	s.eng = eng
	s.engMu.Unlock()
	s.active.Store(true)
	s.diskOK.Store(s.probeDisk())
	if s.diskOK.Load() {
		metrics.BrokerDiskWritable.Set(1)
	}
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	new := s.connCount.Add(1)
	if s.maxConnections > 0 && new > s.maxConnections {
		remaining := s.connCount.Add(-1)
		metrics.BrokerActiveConnections.Set(float64(remaining))
		metrics.BrokerConnectionsRejected.Inc()
		return nil, gnet.Close
	}
	metrics.BrokerActiveConnections.Set(float64(new))
	ctx := s.newConnState()
	gnetutil.OpenConn(c, s.connPolicy(), ctx)
	return nil, gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	metrics.BrokerActiveConnections.Set(float64(s.connCount.Add(-1)))
	return gnet.None
}

func (s *Server) isAdmissionShedding() bool {
	max := s.maxConnections
	if max <= 0 {
		return false
	}
	threshold := max * 9 / 10
	if threshold < 1 {
		threshold = 1
	}
	return s.connCount.Load() >= threshold
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	ctx := s.ensureConnState(c)
	if s.connMaxLifetimeExceeded(ctx) {
		return s.closeConnIdle(c, "max_lifetime")
	}

	for {
		lenBuf, err := c.Peek(4)
		if err != nil {
			if act := s.waitIncomplete(c, ctx); act != gnet.None {
				return act
			}
			return gnet.None
		}
		length := binary.BigEndian.Uint32(lenBuf)
		maxFrame := uint32(1 << 20)
		if s.maxSegSize > 0 && s.maxSegSize < int64(maxFrame) {
			maxFrame = uint32(s.maxSegSize)
		}

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
		calculated := crc32.ChecksumIEEE(reqPayload)
		if calculated != expected {
			if _, err := c.Discard(int(4 + length)); err != nil {
				return gnet.Close
			}
			return gnet.Close
		}

		switch cmd {
		case protocol.CmdProduce:
			s.handleProduce(c, seq, reqPayload)
		case protocol.CmdFetch:
			s.handleFetch(c, seq, reqPayload)
		case protocol.CmdProduceBatch:
			s.handleProduceBatch(c, seq, reqPayload)
		case protocol.CmdRegisterTopic:
			s.handleRegisterTopic(c, seq, reqPayload)
		case protocol.CmdCommitOffset:
			s.handleCommitOffset(c, seq, reqPayload)
		case protocol.CmdCommittedOffset:
			s.handleCommittedOffset(c, seq, reqPayload)
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

	topicName, err := protocol.DecodeRegisterTopicRequest(payload)
	if err != nil {
		resp := protocol.EncodeRegisterTopicResponse(buf, seq, 1, 0)
		_, _ = c.Write(resp)
		return
	}

	id, err := s.registry.Register(topicName)
	if err != nil {
		resp := protocol.EncodeRegisterTopicResponse(buf, seq, 2, 0)
		_, _ = c.Write(resp)
		return
	}

	resp := protocol.EncodeRegisterTopicResponse(buf, seq, 0, id)
	_, _ = c.Write(resp)
}

func (s *Server) handleProduceBatch(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	it := protocol.NewBatchIterator(payload)
	var lastOffset uint64
	var status byte = 0

	if s.isAdmissionShedding() {
		resp := protocol.EncodeProduceBatchResponse(buf, seq, 7, 0, 0)
		recordProduce("batch", 7)
		_, _ = c.Write(resp)
		return
	}

	var committed uint32
	for it.Next() {
		meta, exists := s.registry.Lookup(it.TopicID)
		if !exists {
			status = 2
			recordProduce("unknown", status)
			break
		}

		tpKey := protocol.TopicPartitionID(meta.Name, 0)

		if s.coord != nil && !s.coord.IsLeader(tpKey) {
			s.requestTopicClaim(tpKey)
			status = 4
			recordProduce(meta.Name, status)
			break
		}

		pl, err := s.getOrCreatePartition(tpKey)
		if err != nil {
			status = 2
			recordProduce(meta.Name, status)
			break
		}

		offset, st, err := s.appendLeader(tpKey, pl, it.Payload)
		if err != nil {
			status = 3
			recordProduce(meta.Name, status)
			break
		}
		if st != 0 {
			status = st
			recordProduce(meta.Name, status)
			break
		}
		recordProduce(meta.Name, 0)
		lastOffset = offset
		committed++
	}

	resp := protocol.EncodeProduceBatchResponse(buf, seq, status, lastOffset, committed)
	_, _ = c.Write(resp)
}

func (s *Server) handleProduce(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	topic, partition, msgPayload, err := protocol.DecodeProduceRequest(payload)
	if err != nil {
		finishProduce(c, buf, seq, "unknown", time.Time{}, false, 1, 0)
		return
	}
	tpKey := protocol.TopicPartitionID(topic, partition)
	produceStart := time.Now()

	if s.isAdmissionShedding() {
		finishProduce(c, buf, seq, tpKey, produceStart, true, 7, 0)
		return
	}

	if s.coord != nil && !s.coord.IsLeader(tpKey) {
		s.requestTopicClaim(tpKey)
		finishProduce(c, buf, seq, tpKey, produceStart, true, 4, 0)
		return
	}

	pl, err := s.getOrCreatePartition(tpKey)
	if err != nil {
		finishProduce(c, buf, seq, tpKey, produceStart, true, 2, 0)
		return
	}

	offset, status, err := s.appendLeader(tpKey, pl, msgPayload)
	if status != 0 {
		finishProduce(c, buf, seq, tpKey, produceStart, true, status, 0)
		return
	}
	if err != nil {
		finishProduce(c, buf, seq, tpKey, produceStart, true, 3, 0)
		return
	}

	finishProduce(c, buf, seq, tpKey, produceStart, true, 0, offset)
}

// requestTopicClaim makes an unowned topic visible to the coordinator so a
// leader can be elected for it. The partition must exist first: the coordination
// loop only ranges over materialized partitions.
func (s *Server) requestTopicClaim(tpKey string) {
	if s.coord == nil {
		return
	}
	if _, err := s.getOrCreatePartition(tpKey); err != nil {
		return
	}
	s.coord.RequestClaim(tpKey)
}

// appendLeader refuses to write unless this node holds an unexpired lease with a
// nonzero epoch. Appending without an epoch would bypass fencing entirely and
// let a demoted or leaderless node accept writes the next leader never replays.
func (s *Server) appendLeader(topic string, pl *log.PartitionLog, payload []byte) (uint64, byte, error) {
	var epoch uint64
	if s.coord != nil {
		if !s.coord.IsLeader(topic) {
			return 0, 4, nil
		}
		if !s.coord.IsLeaderReady(topic) {
			return 0, 6, nil
		}
		ep, ok := s.coord.LeaderEpoch(topic)
		if !ok {
			return 0, 4, nil
		}
		epoch = ep
	}
	offset, err := pl.AppendFenced(epoch, payload)
	if errors.Is(err, log.ErrStaleFencingEpoch) {
		return 0, 5, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if s.coord != nil && s.coord.IsLeader(topic) {
		s.coord.PublishLogHWM(topic, offset+1)
	}
	return offset, 0, nil
}

func (s *Server) handleFetch(c gnet.Conn, seq uint64, payload []byte) {
	bufPtr := bytePool.Get().(*[]byte)
	defer bytePool.Put(bufPtr)
	buf := (*bufPtr)[:32]

	topic, partition, startOffset, maxBytes, err := protocol.DecodeFetchRequest(payload)
	if err != nil {
		s.writeFetchResponse(c, buf, seq, "", time.Time{}, false, 1, 0, 0, 0, nil)
		return
	}
	tpKey := protocol.TopicPartitionID(topic, partition)
	fetchStart := time.Now()

	pl, err := s.getOrCreatePartition(tpKey)
	if err != nil {
		s.writeFetchResponse(c, buf, seq, tpKey, fetchStart, true, 2, 0, 0, 0, nil)
		return
	}

	highWatermark := pl.NextOffset()

	data, dataBufPtr, err := pl.ReadRawMessages(startOffset, maxBytes)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, log.ErrSegmentNotFound) {
			s.writeFetchResponse(c, buf, seq, tpKey, fetchStart, true, 0, 0, 0, highWatermark, nil)
			return
		}
		s.writeFetchResponse(c, buf, seq, tpKey, fetchStart, true, 3, 0, 0, highWatermark, nil)
		return
	}
	if dataBufPtr != nil {
		defer log.FetchBufPool.Put(dataBufPtr)
	}

	recordFetch(tpKey)
	msgCount, totalBytes := countMessages(data)
	s.writeFetchResponse(c, buf, seq, tpKey, fetchStart, true, 0, msgCount, totalBytes, highWatermark, data)
}

func (s *Server) writeFetchResponse(c gnet.Conn, buf []byte, seq uint64, tpKey string, fetchStart time.Time, timed bool, status byte, msgCount, msgBytes uint32, highWatermark uint64, data []byte) {
	_ = buf
	if timed {
		observeFetchDuration(tpKey, fetchStart)
	}

	frameLen := 4 + 2 + 8 + protocol.FetchRespMetaLen + len(data) + 4
	if frameLen <= 128 {
		var scratch [128]byte
		frame := protocol.EncodeFetchResponse(scratch[:], seq, status, msgCount, msgBytes, highWatermark, data)
		_, _ = c.Write(frame)
		return
	}

	framePtr := fetchRespPool.Get().(*[]byte)
	poolBuf := *framePtr
	if len(poolBuf) < frameLen {
		poolBuf = make([]byte, frameLen)
		*framePtr = poolBuf
	}
	frame := protocol.EncodeFetchResponse(poolBuf[:frameLen], seq, status, msgCount, msgBytes, highWatermark, data)
	_, _ = c.Write(frame)
	fetchRespPool.Put(framePtr)
}

func countMessages(buf []byte) (uint32, uint32) {
	var count, total uint32
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

func (s *Server) getOrCreatePartition(partitionKey string) (*log.PartitionLog, error) {
	if val, ok := s.topics.Load(partitionKey); ok {
		return val.(*log.PartitionLog), nil
	}

	s.initMu.Lock()
	defer s.initMu.Unlock()

	if val, ok := s.topics.Load(partitionKey); ok {
		return val.(*log.PartitionLog), nil
	}

	parts := strings.Split(partitionKey, "/")
	dir := filepath.Join(append([]string{s.dataDir}, parts...)...)
	pl, err := log.NewPartitionLogWithDurabilityAndGate(dir, s.maxSegSize, s.indexInterval, s.durability, s.diskGate)
	if err != nil {
		return nil, err
	}
	s.topics.Store(strings.Clone(partitionKey), pl)
	return pl, nil
}
