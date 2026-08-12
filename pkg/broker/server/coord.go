package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/broker/client"
	"github.com/bidshard/ad-event-processor/pkg/broker/log"
	"github.com/bidshard/ad-event-processor/pkg/broker/protocol"
	"github.com/redis/go-redis/v9"
)

// topicLeaderState carries the local view of a topic lease. leaseExpiresAtNano
// bounds authority in local monotonic-ish wall time: once it passes, the node
// must stop acting as leader even when Redis is unreachable, otherwise a
// partitioned node keeps accepting writes that the next leader never sees.
type topicLeaderState struct {
	isLeader           bool
	epoch              uint64
	ready              bool
	leaseExpiresAtNano int64
}

const claimQueueCapacity = 64

type Coordinator struct {
	nodeID        string
	tcpAddr       string
	rdb           redis.UniversalClient
	host          CoordHost
	cfg           CoordConfig
	closeChan     chan struct{}
	closeOnce     sync.Once
	wg            sync.WaitGroup
	leaders       atomic.Pointer[map[string]topicLeaderState]
	renewFailures map[string]int
	renewMu       sync.Mutex
	claimCh       chan string
}

func NewCoordinator(nodeID string, tcpAddr string, redisURL string, host CoordHost) (*Coordinator, error) {
	return NewCoordinatorWithConfig(nodeID, tcpAddr, redisURL, host, DefaultCoordConfig())
}

func NewCoordinatorWithConfig(nodeID string, tcpAddr string, redisURL string, host CoordHost, cfg CoordConfig) (*Coordinator, error) {
	rdb, err := openCoordRedis(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open redis: %w", err)
	}

	c := &Coordinator{
		nodeID:        nodeID,
		tcpAddr:       tcpAddr,
		rdb:           rdb,
		host:          host,
		cfg:           cfg.normalized(),
		closeChan:     make(chan struct{}),
		renewFailures: make(map[string]int),
		claimCh:       make(chan string, claimQueueCapacity),
	}
	initMap := make(map[string]topicLeaderState)
	c.leaders.Store(&initMap)
	return c, nil
}

func openCoordRedis(redisURL string) (redis.UniversalClient, error) {
	master := os.Getenv("BROKER_REDIS_SENTINEL_MASTER")
	if master != "" {
		addrs := strings.Split(os.Getenv("BROKER_REDIS_SENTINEL_ADDRS"), ",")
		trimmed := make([]string, 0, len(addrs))
		for _, a := range addrs {
			a = strings.TrimSpace(a)
			if a != "" {
				trimmed = append(trimmed, a)
			}
		}
		if len(trimmed) == 0 {
			return nil, fmt.Errorf("BROKER_REDIS_SENTINEL_ADDRS is empty")
		}
		var pwd string
		if opts, err := redis.ParseURL(redisURL); err == nil {
			pwd = opts.Password
		}
		if pwd == "" {
			pwd = os.Getenv("BROKER_REDIS_PASSWORD")
		}
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       master,
			SentinelAddrs:    trimmed,
			Password:         pwd,
			SentinelPassword: pwd,
		}), nil
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

func (c *Coordinator) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runHeartbeatLoop()
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runCoordinationLoop()
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runClaimLoop()
	}()
}

func (c *Coordinator) Redis() redis.UniversalClient {
	return c.rdb
}

func (c *Coordinator) Stop() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
	c.wg.Wait()
	_ = c.rdb.Close()
}

func leaseHeld(st topicLeaderState) bool {
	return st.isLeader && st.leaseExpiresAtNano > time.Now().UnixNano()
}

func (c *Coordinator) leaderStateSnapshot(topic string) topicLeaderState {
	m := c.leaders.Load()
	if m == nil {
		return topicLeaderState{}
	}
	return (*m)[topic]
}

func (c *Coordinator) IsLeader(topic string) bool {
	return leaseHeld(c.leaderStateSnapshot(topic))
}

func (c *Coordinator) LeaderEpoch(topic string) (uint64, bool) {
	st := c.leaderStateSnapshot(topic)
	if !leaseHeld(st) || st.epoch == 0 {
		return 0, false
	}
	return st.epoch, true
}

func (c *Coordinator) IsLeaderReady(topic string) bool {
	st := c.leaderStateSnapshot(topic)
	return leaseHeld(st) && st.ready
}

// publishHWMScript keeps the cluster high watermark monotonic. A leader that
// takes over while still behind must not lower the recorded tail, otherwise the
// catch-up target for the next failover is silently truncated.
var publishHWMScript = redis.NewScript(`
local cur = redis.call('GET', KEYS[1])
if cur and tonumber(cur) >= tonumber(ARGV[1]) then
  return cur
end
redis.call('SET', KEYS[1], ARGV[1])
return ARGV[1]
`)

func (c *Coordinator) PublishLogHWM(topic string, hwm uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = publishHWMScript.Run(ctx, c.rdb, []string{logHWMKey(topic)}, strconv.FormatUint(hwm, 10)).Err()
}

// RequestClaim asks the claim loop to try acquiring leadership for a topic that
// received a write while unowned. Callers run on the gnet event loop, so the
// enqueue never blocks and drops the hint when the queue is saturated.
func (c *Coordinator) RequestClaim(topic string) {
	if c == nil {
		return
	}
	select {
	case c.claimCh <- strings.Clone(topic):
	default:
	}
}

func (c *Coordinator) runClaimLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		case topic := <-c.claimCh:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			c.claimTopic(ctx, topic)
			cancel()
		}
	}
}

func (c *Coordinator) claimTopic(ctx context.Context, topic string) {
	if c.IsLeader(topic) {
		return
	}
	lKey := leaderKey(topic)
	ok, err := c.rdb.SetNX(ctx, lKey, c.nodeID, c.cfg.LeaseTTL).Result()
	if err != nil || !ok {
		return
	}
	epoch, bumped, err := c.acquireEpoch(ctx, topic, leaderEpochKey(topic))
	if err != nil {
		_ = c.rdb.Del(ctx, lKey).Err()
		return
	}
	c.clearRenewFailures(topic)
	c.onAcquiredLeadership(ctx, topic, epoch, bumped)
}

func (c *Coordinator) HasLeader(topic string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	exists, err := c.rdb.Exists(ctx, leaderKey(topic)).Result()
	return exists > 0, err
}

func leaderKey(topic string) string {
	return "ad_event_processor:topics:" + topic + ":leader"
}

func leaderEpochKey(topic string) string {
	return "ad_event_processor:topics:" + topic + ":leader_epoch"
}

func logHWMKey(topic string) string {
	return "ad_event_processor:topics:" + topic + ":log_hwm"
}

func (c *Coordinator) runHeartbeatLoop() {
	interval := c.cfg.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lease := c.cfg.LeaseTTL

	for {
		select {
		case <-c.closeChan:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = c.rdb.Del(ctx, "ad_event_processor:brokers:"+c.nodeID).Err()
			cancel()
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = c.rdb.Set(ctx, "ad_event_processor:brokers:"+c.nodeID, c.tcpAddr, lease).Err()
			cancel()
		}
	}
}

func (c *Coordinator) runCoordinationLoop() {
	interval := c.cfg.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	replications := make(map[string]chan struct{})

	for {
		select {
		case <-c.closeChan:
			for _, stopCh := range replications {
				close(stopCh)
			}
			return
		case <-ticker.C:
			var topics []string
			c.host.CoordRangeTopics(func(topic string) bool {
				topics = append(topics, topic)
				return true
			})

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			for _, topic := range topics {
				c.coordTopic(ctx, topic, replications)
			}
			cancel()
		}
	}
}

func (c *Coordinator) coordTopic(ctx context.Context, topic string, replications map[string]chan struct{}) {
	lKey := leaderKey(topic)
	eKey := leaderEpochKey(topic)
	lease := c.cfg.LeaseTTL

	ok, err := c.rdb.SetNX(ctx, lKey, c.nodeID, lease).Result()
	if err != nil {
		return
	}

	if ok {
		epoch, bumped, err := c.acquireEpoch(ctx, topic, eKey)
		if err != nil {
			_ = c.rdb.Del(ctx, lKey).Err()
			return
		}
		if stopCh, exists := replications[topic]; exists {
			close(stopCh)
			delete(replications, topic)
		}
		_ = c.rdb.Expire(ctx, lKey, lease).Err()
		c.clearRenewFailures(topic)
		c.onAcquiredLeadership(ctx, topic, epoch, bumped)
		return
	}

	currentLeader, err := c.rdb.Get(ctx, lKey).Result()
	if err == nil && currentLeader == c.nodeID {
		epoch := c.readEpoch(ctx, topic)
		if !c.renewLease(ctx, topic, lKey) {
			clusterEpoch := c.readEpoch(ctx, topic)
			c.demoteTopic(topic, clusterEpoch)
			c.updateTopicMetrics(ctx, topic)
			return
		}
		prev := c.leaderStateSnapshot(topic)
		if !prev.isLeader {
			c.onAcquiredLeadership(ctx, topic, epoch, false)
			c.updateTopicMetrics(ctx, topic)
			return
		}
		if prev.ready {
			if pl, plErr := c.host.CoordGetOrCreatePartition(topic); plErr == nil {
				c.PublishLogHWM(topic, pl.NextOffset())
			}
		}
		c.setLeaderState(topic, true, epoch, prev.ready)
		c.updateTopicMetrics(ctx, topic)
		return
	}

	clusterEpoch := c.readEpoch(ctx, topic)
	if c.IsLeader(topic) {
		c.demoteTopic(topic, clusterEpoch)
	}
	if _, exists := replications[topic]; exists {
		return
	}
	stopCh := make(chan struct{})
	replications[topic] = stopCh
	c.wg.Add(1)
	go func(t string, leaderID string, sCh chan struct{}) {
		defer c.wg.Done()
		c.replicate(t, leaderID, sCh)
	}(topic, currentLeader, stopCh)

	c.updateTopicMetrics(ctx, topic)
}

func (c *Coordinator) acquireEpoch(ctx context.Context, topic, eKey string) (uint64, bool, error) {
	lastWinner, _ := c.rdb.Get(ctx, leaderLastWinnerKey(topic)).Result()
	lastSince, _ := c.rdb.Get(ctx, leaderSinceKey(topic)).Result()
	if lastWinner == c.nodeID && lastSince != "" {
		if sinceUnix, err := strconv.ParseInt(lastSince, 10, 64); err == nil {
			elapsed := time.Since(time.Unix(sinceUnix, 0))
			if elapsed < c.cfg.DebounceWindow {
				epoch := c.readEpoch(ctx, topic)
				if epoch > 0 {
					now := strconv.FormatInt(time.Now().Unix(), 10)
					_ = c.rdb.Set(ctx, leaderSinceKey(topic), now, 0).Err()
					_ = c.rdb.Set(ctx, leaderLastWinnerKey(topic), c.nodeID, 0).Err()
					return epoch, false, nil
				}
			}
		}
	}
	epoch, err := c.rdb.Incr(ctx, eKey).Result()
	if err != nil {
		return 0, false, err
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	_ = c.rdb.Set(ctx, leaderSinceKey(topic), now, 0).Err()
	_ = c.rdb.Set(ctx, leaderLastWinnerKey(topic), c.nodeID, 0).Err()
	return uint64(epoch), true, nil
}

func (c *Coordinator) renewLease(ctx context.Context, topic, lKey string) bool {
	lease := c.cfg.LeaseTTL
	ok, err := c.rdb.Expire(ctx, lKey, lease).Result()
	if err != nil || !ok {
		c.recordRenewFailure(topic)
		return false
	}
	currentLeader, err := c.rdb.Get(ctx, lKey).Result()
	if err != nil || currentLeader != c.nodeID {
		c.recordRenewFailure(topic)
		return false
	}
	c.clearRenewFailures(topic)
	return true
}

func (c *Coordinator) recordRenewFailure(topic string) {
	c.renewMu.Lock()
	defer c.renewMu.Unlock()
	c.renewFailures[topic]++
	if c.renewFailures[topic] >= c.cfg.RenewFailThreshold {
		slog.Warn("Leader lease renew failed repeatedly; stepping down proactively",
			"topic", topic, "node_id", c.nodeID, "failures", c.renewFailures[topic])
	}
}

func (c *Coordinator) clearRenewFailures(topic string) {
	c.renewMu.Lock()
	defer c.renewMu.Unlock()
	delete(c.renewFailures, topic)
}

func (c *Coordinator) updateTopicMetrics(ctx context.Context, topic string) {
	hwm := c.readLogHWM(ctx, topic)
	local := uint64(0)
	pl, err := c.host.CoordGetOrCreatePartition(topic)
	if err == nil {
		local = pl.NextOffset()
	}
	lag := float64(0)
	if hwm > local {
		lag = float64(hwm - local)
	}
	metrics.BrokerReplicationLag.WithLabelValues(topic).Set(lag)

	st := c.leaderStateSnapshot(topic)
	leader := float64(0)
	ready := float64(0)
	epoch := float64(0)
	if leaseHeld(st) {
		leader = 1
		if st.ready {
			ready = 1
		}
		epoch = float64(st.epoch)
	}
	metrics.BrokerActiveLeader.WithLabelValues(topic, c.nodeID).Set(leader)
	metrics.BrokerLeaderReady.WithLabelValues(topic, c.nodeID).Set(ready)
	metrics.BrokerLeaderEpoch.WithLabelValues(topic, c.nodeID).Set(epoch)
}

func (c *Coordinator) readEpoch(ctx context.Context, topic string) uint64 {
	val, err := c.rdb.Get(ctx, leaderEpochKey(topic)).Result()
	if err != nil {
		return 0
	}
	epoch, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0
	}
	return epoch
}

func (c *Coordinator) readLogHWM(ctx context.Context, topic string) uint64 {
	val, err := c.rdb.Get(ctx, logHWMKey(topic)).Result()
	if err != nil {
		return 0
	}
	hwm, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0
	}
	return hwm
}

func (c *Coordinator) onAcquiredLeadership(ctx context.Context, topic string, epoch uint64, bumped bool) {
	if bumped {
		metrics.BrokerLeaderElectionTotal.WithLabelValues(topic).Inc()
	}

	pl, err := c.host.CoordGetOrCreatePartition(topic)
	if err != nil {
		c.setLeaderState(topic, true, epoch, false)
		slog.Error("Acquired topic leadership without partition log; leader stays not ready",
			"topic", topic, "epoch", epoch, "error", err)
		return
	}

	local := pl.NextOffset()
	hwm := c.readLogHWM(ctx, topic)
	ready := local >= hwm
	c.setLeaderState(topic, true, epoch, ready)
	if ready {
		c.PublishLogHWM(topic, local)
	}
	slog.Info("Acquired topic leadership", "topic", topic, "epoch", epoch, "local", local, "hwm", hwm, "ready", ready)

	if !ready {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.recoverLeaderReadiness(topic, hwm, time.Now())
		}()
	}
	c.updateTopicMetrics(ctx, topic)
}

func (c *Coordinator) recoverLeaderReadiness(topic string, targetHWM uint64, started time.Time) {
	const timeout = 5 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pl, err := c.host.CoordGetOrCreatePartition(topic)
		if err == nil && pl.NextOffset() >= targetHWM {
			c.setLeaderReady(topic, true)
			metrics.BrokerReplicationCatchupSeconds.WithLabelValues(topic).Observe(time.Since(started).Seconds())
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			c.updateTopicMetrics(ctx, topic)
			cancel()
			slog.Info("Leader ready after catch-up", "topic", topic, "offset", pl.NextOffset())
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	pl, err := c.host.CoordGetOrCreatePartition(topic)
	if err != nil {
		c.setLeaderReady(topic, true)
		metrics.BrokerReplicationCatchupSeconds.WithLabelValues(topic).Observe(time.Since(started).Seconds())
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		c.updateTopicMetrics(ctx, topic)
		cancel()
		return
	}
	// Availability wins after the catch-up budget, but the cluster HWM stays at
	// the known tail: lowering it here would erase the catch-up target and make
	// the truncation invisible to the next failover and to alerting.
	local := pl.NextOffset()
	recordReplicationError(topic, "catchup_gap")
	c.setLeaderReady(topic, true)
	metrics.BrokerReplicationCatchupSeconds.WithLabelValues(topic).Observe(time.Since(started).Seconds())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	c.updateTopicMetrics(ctx, topic)
	cancel()
	slog.Warn("Leader readiness accepted with replication gap",
		"topic", topic, "local", local, "target_hwm", targetHWM)
}

func (c *Coordinator) demoteTopic(topic string, clusterEpoch uint64) {
	c.setLeaderState(topic, false, 0, false)
	if clusterEpoch == 0 {
		return
	}
	pl, err := c.host.CoordGetOrCreatePartition(topic)
	if err != nil {
		return
	}
	if err := pl.AdvanceFencingEpoch(clusterEpoch); err != nil {
		slog.Warn("Failed to advance fencing epoch on demotion", "topic", topic, "epoch", clusterEpoch, "error", err)
	}
}

func (c *Coordinator) setLeaderState(topic string, isLeader bool, epoch uint64, ready bool) {
	for {
		old := c.leaders.Load()
		newMap := make(map[string]topicLeaderState, len(*old)+1)
		for k, v := range *old {
			newMap[k] = v
		}
		if !isLeader {
			ready = false
		}
		var expiresAt int64
		if isLeader {
			expiresAt = time.Now().Add(c.cfg.LeaseTTL).UnixNano()
		}
		newMap[topic] = topicLeaderState{
			isLeader:           isLeader,
			epoch:              epoch,
			ready:              ready,
			leaseExpiresAtNano: expiresAt,
		}
		if c.leaders.CompareAndSwap(old, &newMap) {
			return
		}
	}
}

func (c *Coordinator) setLeaderReady(topic string, ready bool) {
	for {
		old := c.leaders.Load()
		st := (*old)[topic]
		if !st.isLeader {
			return
		}
		newMap := make(map[string]topicLeaderState, len(*old))
		for k, v := range *old {
			newMap[k] = v
		}
		st.ready = ready
		newMap[topic] = st
		if c.leaders.CompareAndSwap(old, &newMap) {
			return
		}
	}
}

func (c *Coordinator) replicate(topic string, leaderID string, stopCh chan struct{}) {
	slog.Info("Starting replication", "topic", topic, "leader", leaderID)
	defer slog.Info("Stopped replication", "topic", topic)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var cli *client.Client
	var currentAddr string

	defer func() {
		if cli != nil {
			_ = cli.Close()
		}
	}()

	for {
		select {
		case <-stopCh:
			return
		case <-c.closeChan:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			leaderAddr, err := c.rdb.Get(ctx, "ad_event_processor:brokers:"+leaderID).Result()
			cancel()
			if err != nil {
				if cli != nil {
					_ = cli.Close()
					cli = nil
				}
				continue
			}

			if cli == nil || leaderAddr != currentAddr {
				if cli != nil {
					_ = cli.Close()
				}
				cli = client.NewClient(leaderAddr, time.Second)
				currentAddr = leaderAddr
				if err := cli.Connect(); err != nil {
					_ = cli.Close()
					cli = nil
					continue
				}
			}

			pl, err := c.host.CoordGetOrCreatePartition(topic)
			if err != nil {
				continue
			}

			nextOffset := pl.NextOffset()

			topicName, part := protocol.ParseTopicPartitionID(topic)
			iter, fetchErr := cli.Fetch(topicName, part, nextOffset, 65536)
			if fetchErr == nil {
				for iter.Next() {
					if _, err = pl.AppendReplicatedAt(iter.Offset, iter.Payload); err != nil {
						if errors.Is(err, log.ErrReplicationGap) {
							slog.Warn("Replication gap detected, halting batch",
								"topic", topic,
								"expected", pl.NextOffset(),
								"got", iter.Offset,
							)
						}
						fetchErr = err
						break
					}
				}
			}

			if fetchErr != nil {
				_ = cli.Close()
				cli = nil
				if errors.Is(fetchErr, log.ErrReplicationGap) {
					recordReplicationError(topic, "gap")
				} else if !errors.Is(fetchErr, io.EOF) {
					recordReplicationError(topic, "fetch")
				}
				if !errors.Is(fetchErr, io.EOF) {
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
	}
}
