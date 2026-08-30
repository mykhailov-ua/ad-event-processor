package stream

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/shard"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"
)

type (
	UDPControlLimits        = domain.UDPControlLimits
	UDPHeader               = domain.UDPHeader
	UDPConfigRequestPayload = domain.UDPConfigRequestPayload
	TCPControlHeader        = domain.TCPControlHeader
	TCPAckPayload           = domain.TCPAckPayload
)

const (
	UDPHeaderSize         = shard.UDPHeaderSize
	TCPControlHeaderSize  = domain.TCPControlHeaderSize
	TCPMsgSnapshot        = domain.TCPMsgSnapshot
	TCPMsgSnapshotRequest = domain.TCPMsgSnapshotRequest
	TCPMsgAck             = domain.TCPMsgAck
)

var (
	EncodeTCPControlFrame  = domain.EncodeTCPControlFrame
	DecodeTCPControlFrame  = domain.DecodeTCPControlFrame
	EncodeTCPLimitsPayload = domain.EncodeTCPLimitsPayload
	EncodeTCPAckPayload    = domain.EncodeTCPAckPayload
	DecodeTCPAckPayload    = domain.DecodeTCPAckPayload
	ErrTCPControlCorrupt   = domain.ErrTCPControlCorrupt
	ErrTCPControlHMAC      = domain.ErrTCPControlHMAC
	EffectiveNodeWeights   = shard.EffectiveNodeWeights
	ComputeUDPConfigHash   = shard.ComputeUDPConfigHash
	UDPMaxControlShards    = shard.UDPMaxControlShards
	UDPMsgQuotaEpoch       = shard.UDPMsgQuotaEpoch
	UDPMsgConfigSnapshot   = shard.UDPMsgConfigSnapshot
	UDPMsgConfigRequest    = shard.UDPMsgConfigRequest
	UDPMsgMigrationBarrier = shard.UDPMsgMigrationBarrier
	UDPFlagSnapshot        = shard.UDPFlagSnapshot
	UDPFlagNodeWeights     = shard.UDPFlagNodeWeights
	NodeWeightsDrainFrozen = shard.NodeWeightsDrainFrozen
)

const (
	ingressCacheLine  = 64
	maxIngressWorkers = 64
)

type IngressQuotaCell struct {
	maxAllowed uint64
	_          [ingressCacheLine - 8]byte
	currentOps atomic.Uint64
	_          [ingressCacheLine - 8]byte
}

type IngressQuotaMap struct {
	epoch      int64
	numShards  uint8
	numWorkers uint8
	cells      []IngressQuotaCell
}

var ingressQuotaMapPool = sync.Pool{
	New: func() any {
		return &IngressQuotaMap{}
	},
}

func BuildIngressQuotaMap(epoch int64, limits *UDPControlLimits, numWorkers int) *IngressQuotaMap {
	if limits == nil || limits.NumShards == 0 || numWorkers <= 0 {
		return nil
	}
	if numWorkers > maxIngressWorkers {
		numWorkers = maxIngressWorkers
	}
	n := int(limits.NumShards) * numWorkers
	m := ingressQuotaMapPool.Get().(*IngressQuotaMap)
	if cap(m.cells) < n {
		m.cells = make([]IngressQuotaCell, n)
	} else {
		m.cells = m.cells[:n]
		for i := range m.cells {
			m.cells[i].maxAllowed = 0
			m.cells[i].currentOps.Store(0)
		}
	}
	m.epoch = epoch
	m.numShards = limits.NumShards
	m.numWorkers = uint8(numWorkers)
	for shard := range int(limits.NumShards) {
		limit := limits.Limits[shard]
		perWorker := limit / uint64(numWorkers)
		if perWorker == 0 && limit > 0 {
			perWorker = 1
		}
		base := shard * numWorkers
		for w := range numWorkers {
			m.cells[base+w].maxAllowed = perWorker
			m.cells[base+w].currentOps.Store(0)
		}
	}
	return m
}

func (m *IngressQuotaMap) TryAcquire(shard, worker int) bool {
	if m == nil {
		return true
	}
	if shard < 0 || worker < 0 || shard >= int(m.numShards) {
		return true
	}
	if worker >= int(m.numWorkers) {
		worker %= int(m.numWorkers)
	}
	idx := shard*int(m.numWorkers) + worker
	if idx >= len(m.cells) {
		return true
	}
	cell := &m.cells[idx]
	if cell.maxAllowed == 0 {
		return true
	}
	ops := cell.currentOps.Add(1)
	if ops > cell.maxAllowed {
		cell.currentOps.Add(^uint64(0))
		return false
	}
	return true
}

type UnpaddedIngressCounters struct {
	counters [maxIngressWorkers]atomic.Uint64
	max      uint64
}

func NewUnpaddedIngressCountersForTest(max uint64) *UnpaddedIngressCounters {
	return &UnpaddedIngressCounters{max: max}
}

func (m *UnpaddedIngressCounters) TryAcquire(worker int) bool {
	if worker < 0 || worker >= maxIngressWorkers {
		return true
	}
	ops := m.counters[worker].Add(1)
	if ops > m.max {
		m.counters[worker].Add(^uint64(0))
		return false
	}
	return true
}

type TCPControlClient struct {
	enabled     bool
	secret      []byte
	trackerID   uint32
	controlAddr string
	dialTO      time.Duration
	sharder     *domain.StaticSlotSharder
	udpControl  *UDPControl
	lastEpoch   atomic.Int64
}

type TCPControlClientConfig struct {
	Enabled     bool
	Secret      []byte
	TrackerID   uint32
	ControlAddr string
	DialTO      time.Duration
	Sharder     *domain.StaticSlotSharder
	UDP         *UDPControl
}

func NewTCPControlClient(cfg TCPControlClientConfig) *TCPControlClient {
	if cfg.DialTO <= 0 {
		cfg.DialTO = 3 * time.Second
	}
	return &TCPControlClient{
		enabled:     cfg.Enabled,
		secret:      cfg.Secret,
		trackerID:   cfg.TrackerID,
		controlAddr: cfg.ControlAddr,
		dialTO:      cfg.DialTO,
		sharder:     cfg.Sharder,
		udpControl:  cfg.UDP,
	}
}

func (c *TCPControlClient) RequestSnapshot(ctx context.Context) error {
	if c == nil || !c.enabled || c.controlAddr == "" {
		return nil
	}
	dialer := net.Dialer{Timeout: c.dialTO}
	conn, err := dialer.DialContext(ctx, "tcp", c.controlAddr)
	if err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(c.dialTO))

	var reqHdr TCPControlHeader
	reqHdr.MsgType = TCPMsgSnapshotRequest
	reqHdr.TrackerID = c.trackerID
	if c.sharder != nil {
		reqHdr.SlotMapVersion = c.sharder.ActiveVersion()
		reqHdr.RoutingEpoch = c.sharder.Snapshot().MigrationGen
	}
	var reqBuf [TCPControlHeaderSize]byte
	if _, err := EncodeTCPControlFrame(reqBuf[:], c.secret, &reqHdr, nil); err != nil {
		return err
	}
	if _, err := conn.Write(reqBuf[:]); err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return err
	}

	var frame [4096]byte
	n, err := io.ReadAtLeast(conn, frame[:], TCPControlHeaderSize)
	if err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return err
	}
	for n < cap(frame) {
		m, rerr := conn.Read(frame[n:])
		n += m
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			metrics.TCPControlSnapshotErrorsTotal.Inc()
			return rerr
		}
		if m == 0 {
			break
		}
	}
	var hdr TCPControlHeader
	payload, err := DecodeTCPControlFrame(frame[:n], c.secret, &hdr)
	if err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return err
	}
	if hdr.MsgType != TCPMsgSnapshot {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return ErrTCPControlCorrupt
	}

	var limits UDPControlLimits
	if hdr.NumShards > 0 && len(payload) > 0 {
		ver := uint8(0)
		if len(payload) > udpShardPayloadLen(hdr.NumShards)+8 {
			ver = udpProtocolVersion2
		}
		if !udpDecodeShardLimits(payload, hdr.NumShards, ver, &limits) {
			metrics.TCPControlSnapshotErrorsTotal.Inc()
			return ErrTCPControlCorrupt
		}
	}
	c.applySnapshot(&hdr, &limits)
	if err := c.sendACK(conn, &hdr); err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return err
	}
	metrics.TCPControlSnapshotAppliedTotal.Inc()
	slog.Info("tcp routing snapshot applied",
		"routing_epoch", hdr.RoutingEpoch,
		"slot_version", hdr.SlotMapVersion,
		"tracker_id", c.trackerID,
	)
	return nil
}

func (c *TCPControlClient) applySnapshot(hdr *TCPControlHeader, limits *UDPControlLimits) {
	if c.sharder != nil {
		prev := c.sharder.Snapshot()
		c.sharder.SwapSnapshot(hdr.SlotMapVersion, &prev.Table, hdr.RoutingEpoch)
	}
	if c.udpControl != nil && limits != nil && limits.NumShards > 0 {
		udpHdr := UDPHeader{
			EpochID:        hdr.RoutingEpoch,
			SlotMapVersion: hdr.SlotMapVersion,
			NumShards:      limits.NumShards,
		}
		c.udpControl.commitSnapshot(&udpHdr, limits, nil)
		c.udpControl.currentEpoch.Store(hdr.RoutingEpoch)
		c.udpControl.markFresh()
	}
	c.lastEpoch.Store(hdr.RoutingEpoch)
}

func (c *TCPControlClient) sendACK(conn net.Conn, snap *TCPControlHeader) error {
	ack := TCPAckPayload{
		TrackerID:      c.trackerID,
		AppliedEpoch:   snap.RoutingEpoch,
		AppliedSlotVer: snap.SlotMapVersion,
	}
	var body [16]byte
	if EncodeTCPAckPayload(body[:], &ack) == 0 {
		return ErrTCPControlCorrupt
	}
	var hdr TCPControlHeader
	hdr.MsgType = TCPMsgAck
	hdr.TrackerID = c.trackerID
	hdr.RoutingEpoch = snap.RoutingEpoch
	hdr.SlotMapVersion = snap.SlotMapVersion
	var frame [80]byte
	n, err := EncodeTCPControlFrame(frame[:], c.secret, &hdr, body[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(frame[:n])
	if err == nil {
		metrics.TCPControlAckSentTotal.Inc()
	}
	return err
}

func udpShardPayloadLen(numShards uint8) int {
	return shard.UDPShardPayloadLen(numShards)
}

func udpDecodeHeader(src []byte, hdr *UDPHeader) bool {
	return shard.UDPDecodeHeader(src, hdr)
}

func udpDecodeShardLimits(payload []byte, numShards uint8, version uint8, out *UDPControlLimits) bool {
	return shard.UDPDecodeShardLimits(payload, numShards, version, out)
}

func udpDecodeNodeWeights(payload []byte, out *[]UDPNodeWeight) bool {
	return shard.UDPDecodeNodeWeights(payload, out)
}

func udpEncodeHeader(dst []byte, hdr *UDPHeader) int {
	return shard.UDPEncodeHeader(dst, hdr)
}

func udpEncodeConfigRequest(dst []byte, req *UDPConfigRequestPayload) int {
	return shard.UDPEncodeConfigRequest(dst, req)
}

func udpEncodeShardLimits(dst []byte, limits *UDPControlLimits) int {
	return shard.UDPEncodeShardLimits(dst, limits)
}

func udpEncodeNodeWeights(dst []byte, weights []UDPNodeWeight) int {
	return shard.UDPEncodeNodeWeights(dst, weights)
}

func udpApplyCanaryFloor(limits *UDPControlLimits) {
	shard.UDPApplyCanaryFloor(limits)
}

func udpLimitsTightening(prev, next *UDPControlLimits) bool {
	return shard.UDPLimitsTightening(prev, next)
}

const (
	udpMagic            = shard.UDPMagic
	udpProtocolVersion  = shard.UDPProtocolVersion
	udpProtocolVersion2 = shard.UDPProtocolVersion2
	udpProtocolVersion3 = shard.UDPProtocolVersion3
)

type UDPChannelState uint32

const (
	UDPChannelOK UDPChannelState = iota
	UDPChannelStale
)

type ingressSnapshot struct {
	epoch          int64
	configHash     [16]byte
	slotMapVersion int32
	limits         UDPControlLimits
	nodeWeights    []UDPNodeWeight
}

var ingressSnapshotPool = sync.Pool{
	New: func() any {
		return &ingressSnapshot{}
	},
}

// UDPControl: recvLoop on :8191 applies quota epoch from control plane; failClosed blocks ingest on stale epoch.
type UDPControl struct {
	enabled            bool
	failClosed         bool
	trackerID          uint32
	bindAddr           string
	syncInterval       time.Duration
	controlAddr        *net.UDPAddr
	snapshot           atomic.Pointer[ingressSnapshot]
	quotaMap           atomic.Pointer[IngressQuotaMap]
	numWorkers         int
	channelState       atomic.Uint32
	currentEpoch       atomic.Int64
	lastPublisherEpoch atomic.Int64
	lastPacketMono     atomic.Int64
	knownHash          [16]byte
	conn               *net.UDPConn
	requestConn        *net.UDPConn
}

type UDPControlConfig struct {
	Enabled      bool
	FailClosed   bool
	TrackerID    uint32
	BindAddr     string
	ControlAddr  string
	SyncInterval time.Duration
	NumShards    int
	NumWorkers   int
	InitialRPS   uint64
}

func NewUDPControl(cfg UDPControlConfig) *UDPControl {
	c := &UDPControl{
		enabled:      cfg.Enabled,
		failClosed:   cfg.FailClosed,
		trackerID:    cfg.TrackerID,
		bindAddr:     cfg.BindAddr,
		syncInterval: cfg.SyncInterval,
		numWorkers:   cfg.NumWorkers,
	}
	if c.numWorkers <= 0 {
		c.numWorkers = 1
	}
	if c.numWorkers > maxIngressWorkers {
		c.numWorkers = maxIngressWorkers
	}
	if cfg.SyncInterval <= 0 {
		c.syncInterval = 10 * time.Second
	}
	if cfg.ControlAddr != "" {
		if addr, err := net.ResolveUDPAddr("udp", cfg.ControlAddr); err == nil {
			c.controlAddr = addr
		}
	}
	if cfg.Enabled && cfg.NumShards > 0 {
		seed := ingressSnapshotPool.Get().(*ingressSnapshot)
		seed.epoch = 0
		if cfg.NumShards > UDPMaxControlShards {
			cfg.NumShards = UDPMaxControlShards
		}
		seed.limits.NumShards = uint8(cfg.NumShards)
		rps := cfg.InitialRPS
		if rps == 0 {
			rps = 50_000
		}
		for i := range cfg.NumShards {
			seed.limits.Limits[i] = rps
		}
		seed.configHash = ComputeUDPConfigHash(0, 0, &seed.limits)
		c.knownHash = seed.configHash
		c.snapshot.Store(seed)
		if qm := BuildIngressQuotaMap(0, &seed.limits, c.numWorkers); qm != nil {
			c.quotaMap.Store(qm)
		}
	}
	return c
}

func NewUDPControlFromConfig(cfg *config.Config, numShards int) *UDPControl {
	if cfg == nil || !cfg.UDPControlEnabled {
		return nil
	}
	return NewUDPControl(UDPControlConfig{
		Enabled:      true,
		FailClosed:   cfg.UDPFailClosed,
		TrackerID:    cfg.UDPTrackerID,
		BindAddr:     cfg.UDPTrackerBindAddr,
		ControlAddr:  cfg.UDPControlAddr,
		SyncInterval: time.Duration(cfg.UDPSyncIntervalMs) * time.Millisecond,
		NumShards:    numShards,
		NumWorkers:   cfg.MaxWorkers,
		InitialRPS:   cfg.UDPDefaultShardRPS,
	})
}

func (c *UDPControl) Start(ctx context.Context) error {
	if c == nil || !c.enabled {
		return nil
	}
	bind := ":8191"
	if c.bindAddr != "" {
		bind = c.bindAddr
	}
	addr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	_ = conn.SetReadBuffer(1 << 20)
	c.conn = conn

	reqConn, err := net.DialUDP("udp", nil, c.controlAddr)
	if err == nil {
		c.requestConn = reqConn
	}
	go c.recvLoop(ctx)
	go c.staleLoop(ctx)
	slog.Info("udp control plane started", "bind", addr.String(), "control_addr", c.controlAddr)
	return nil
}

func (c *UDPControl) Close() error {
	if c == nil {
		return nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.requestConn != nil {
		_ = c.requestConn.Close()
	}
	return nil
}

func (c *UDPControl) ChannelState() UDPChannelState {
	if c == nil {
		return UDPChannelOK
	}
	return UDPChannelState(c.channelState.Load())
}

func (c *UDPControl) CurrentEpoch() int64 {
	if c == nil {
		return 0
	}
	return c.currentEpoch.Load()
}

func (c *UDPControl) PublisherEpoch() int64 {
	if c == nil {
		return 0
	}
	if ep := c.lastPublisherEpoch.Load(); ep > 0 {
		return ep
	}
	return c.currentEpoch.Load()
}

func (c *UDPControl) DrainFrozen() bool {
	if c == nil || !c.enabled {
		return false
	}
	return NodeWeightsDrainFrozen(c.ChannelState() == UDPChannelStale, c.PublisherEpoch(), c.CurrentEpoch())
}

func (c *UDPControl) NodeWeights() []UDPNodeWeight {
	if c == nil {
		return nil
	}
	snap := c.snapshot.Load()
	if snap == nil || len(snap.nodeWeights) == 0 {
		return nil
	}
	stale := c.ChannelState() == UDPChannelStale
	return EffectiveNodeWeights(snap.nodeWeights, stale, c.PublisherEpoch(), c.CurrentEpoch())
}

func (c *UDPControl) TryIngress(shard, workerID int) bool {
	if c == nil || !c.enabled {
		return true
	}
	m := c.quotaMap.Load()
	if m == nil {
		return true
	}
	if m.TryAcquire(shard, workerID) {
		metrics.UDPIngressAcquireTotal.Inc()
		return true
	}
	metrics.UDPIngressRejectTotal.Inc()
	return false
}

func (c *UDPControl) ShardLimitRPS(shard int) uint64 {
	if c == nil || shard < 0 || shard >= UDPMaxControlShards {
		return 0
	}
	snap := c.snapshot.Load()
	if snap == nil {
		return 0
	}
	if int(shard) >= int(snap.limits.NumShards) {
		return 0
	}
	return snap.limits.Limits[shard]
}

func (c *UDPControl) ApplyPacket(buf []byte) bool {
	if c == nil || len(buf) < UDPHeaderSize {
		metrics.UDPControlCorruptTotal.Inc()
		return false
	}
	var hdr UDPHeader
	if !udpDecodeHeader(buf, &hdr) {
		metrics.UDPControlCorruptTotal.Inc()
		return false
	}
	payload := buf[UDPHeaderSize:]
	if int(hdr.PayloadLen) > len(payload) {
		metrics.UDPControlCorruptTotal.Inc()
		return false
	}
	payload = payload[:hdr.PayloadLen]
	filter.ApplyUDPCoarseTime(hdr.CoarseTimeNs)
	c.lastPublisherEpoch.Store(hdr.EpochID)
	metrics.UDPControlEpochLag.Set(float64(hdr.EpochID - c.currentEpoch.Load()))

	switch hdr.MsgType {
	case UDPMsgQuotaEpoch, UDPMsgConfigSnapshot:
		var limits UDPControlLimits
		if !udpDecodeShardLimits(payload, hdr.NumShards, hdr.Version, &limits) {
			metrics.UDPControlCorruptTotal.Inc()
			return false
		}
		shardLen := udpShardPayloadLen(hdr.NumShards)
		if hdr.Version == udpProtocolVersion2 {
			shardLen += 8
		}
		var nodeWeights []UDPNodeWeight
		if hdr.Version >= udpProtocolVersion3 || hdr.Flags&UDPFlagNodeWeights != 0 {
			if len(payload) > shardLen {
				if !udpDecodeNodeWeights(payload[shardLen:], &nodeWeights) {
					metrics.UDPControlCorruptTotal.Inc()
					return false
				}
			}
		}
		isSnapshot := hdr.MsgType == UDPMsgConfigSnapshot || hdr.Flags&UDPFlagSnapshot != 0
		return c.applyLimits(&hdr, &limits, nodeWeights, isSnapshot)
	case UDPMsgMigrationBarrier:
		c.markFresh()
		return true
	default:
		metrics.UDPControlCorruptTotal.Inc()
		return false
	}
}

func (c *UDPControl) applyLimits(hdr *UDPHeader, limits *UDPControlLimits, nodeWeights []UDPNodeWeight, isSnapshot bool) bool {
	cur := c.currentEpoch.Load()
	epoch := hdr.EpochID

	if epoch <= cur {
		metrics.UDPControlStaleDropTotal.Inc()
		return false
	}

	if epoch == cur+1 || cur == 0 {
		c.commitSnapshot(hdr, limits, nodeWeights)
		c.currentEpoch.Store(epoch)
		c.markFresh()
		if isSnapshot {
			metrics.UDPControlSnapshotAppliedTotal.Inc()
		}
		return true
	}

	prev := c.snapshot.Load()
	tightening := udpLimitsTightening(&prev.limits, limits)
	if tightening {
		c.commitSnapshot(hdr, limits, nodeWeights)
		c.currentEpoch.Store(epoch)
		c.markFresh()
		metrics.UDPControlEpochTightenTotal.Inc()
		return true
	}

	if isSnapshot {
		c.commitSnapshot(hdr, limits, nodeWeights)
		c.currentEpoch.Store(epoch)
		c.markFresh()
		c.knownHash = hdr.ConfigHash
		metrics.UDPControlSnapshotAppliedTotal.Inc()
		return true
	}

	metrics.UDPControlLoosenBlockedTotal.Inc()
	return false
}

func (c *UDPControl) commitSnapshot(hdr *UDPHeader, limits *UDPControlLimits, nodeWeights []UDPNodeWeight) {
	next := ingressSnapshotPool.Get().(*ingressSnapshot)
	next.epoch = hdr.EpochID
	next.configHash = hdr.ConfigHash
	next.slotMapVersion = hdr.SlotMapVersion
	next.limits = *limits
	if len(nodeWeights) == 0 {
		next.nodeWeights = nil
	} else {
		next.nodeWeights = append(next.nodeWeights[:0], nodeWeights...)
	}
	old := c.snapshot.Swap(next)
	if old != nil {
		old.nodeWeights = nil
		ingressSnapshotPool.Put(old)
	}
	if qm := BuildIngressQuotaMap(hdr.EpochID, limits, c.numWorkers); qm != nil {
		oldMap := c.quotaMap.Swap(qm)
		if oldMap != nil {
			oldMap.cells = oldMap.cells[:0]
			ingressQuotaMapPool.Put(oldMap)
		}
	}
}

func (c *UDPControl) markFresh() {
	c.lastPacketMono.Store(filter.MonotonicNano())
	if c.channelState.Swap(uint32(UDPChannelOK)) == uint32(UDPChannelStale) {
		metrics.UDPControlRecoveredTotal.Inc()
	}
}

func (c *UDPControl) recvLoop(ctx context.Context) {
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if n > 0 {
			metrics.UDPControlPacketsReceivedTotal.Inc()
			if c.ApplyPacket(buf[:n]) {
				metrics.UDPControlPacketsAppliedTotal.Inc()
			}
		}
	}
}

func (c *UDPControl) staleLoop(ctx context.Context) {
	ticker := time.NewTicker(c.syncInterval / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkStale()
		}
	}
}

func (c *UDPControl) checkStale() {
	last := c.lastPacketMono.Load()
	if last == 0 {
		return
	}
	threshold := c.syncInterval.Nanoseconds() * 2
	if filter.MonotonicNano()-last <= threshold {
		return
	}
	if c.channelState.Swap(uint32(UDPChannelStale)) != uint32(UDPChannelStale) {
		metrics.UDPControlStaleTotal.Inc()
		c.tightenCanaryFloor()
		c.sendConfigRequest()
	}
}

func (c *UDPControl) tightenCanaryFloor() {
	snap := c.snapshot.Load()
	if snap == nil {
		return
	}
	var limits UDPControlLimits
	limits.NumShards = snap.limits.NumShards
	limits.Limits = snap.limits.Limits
	udpApplyCanaryFloor(&limits)
	hdr := UDPHeader{
		Magic:          udpMagic,
		Version:        udpProtocolVersion,
		MsgType:        UDPMsgConfigSnapshot,
		Flags:          UDPFlagSnapshot,
		EpochID:        snap.epoch,
		ConfigHash:     snap.configHash,
		SlotMapVersion: snap.slotMapVersion,
		NumShards:      limits.NumShards,
	}
	c.commitSnapshot(&hdr, &limits, snap.nodeWeights)
}

func (c *UDPControl) sendConfigRequest() {
	if c.requestConn == nil || c.controlAddr == nil {
		return
	}
	snap := c.snapshot.Load()
	var hash [16]byte
	epoch := c.currentEpoch.Load()
	if snap != nil {
		hash = snap.configHash
	}
	var buf [UDPHeaderSize + 28]byte
	hdr := UDPHeader{
		Magic:      udpMagic,
		Version:    udpProtocolVersion,
		MsgType:    UDPMsgConfigRequest,
		EpochID:    epoch,
		ConfigHash: hash,
		PayloadLen: 28,
	}
	udpEncodeHeader(buf[:], &hdr)
	req := UDPConfigRequestPayload{TrackerID: c.trackerID, LastEpoch: epoch, Hash: hash}
	udpEncodeConfigRequest(buf[UDPHeaderSize:], &req)
	_, _ = c.requestConn.Write(buf[:])
	metrics.UDPControlConfigRequestTotal.Inc()
}

func EncodeQuotaEpochDatagram(dst []byte, msgType uint8, hdr *UDPHeader, limits *UDPControlLimits) int {
	return domain.EncodeQuotaEpochDatagramWithWeights(dst, msgType, hdr, limits, nil)
}
