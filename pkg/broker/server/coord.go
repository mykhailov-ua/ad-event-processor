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

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/broker/client"
	"ad-event-processor/pkg/broker/log"
	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/netaddr"
	"github.com/redis/go-redis/v9"
)

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
	redisClient           redis.UniversalClient
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
	redisClient, err := openCoordRedis(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open redis: %w", err)
	}

	c := &Coordinator{
		nodeID:        nodeID,
		tcpAddr:       tcpAddr,
		redisClient:           redisClient,
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

	pwd := os.Getenv("BROKER_REDIS_PASSWORD")
	if pwd == "" {
		pwd = os.Getenv("REDIS_PASSWORD")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		if strings.HasPrefix(redisURL, "unix://") || netaddr.IsUnixSocketPath(redisURL) {
			redisClient, parseErr := netaddr.ParseRedisURL(redisURL, pwd)
			if parseErr != nil {
				return nil, parseErr
			}
			return redisClient, nil
		}
		return nil, err
	}
	return redis.NewClient(opts), nil
}

func (c *Coordinator) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runHeartbeatLoop(ctx)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runCoordinationLoop(ctx)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runClaimLoop(ctx)
	}()
}

func (c *Coordinator) runClaimLoop(ctx context.Context) {
	for {
		select {
		case <-c.closeChan:
			return
		case <-ctx.Done():
			return
		case topic := <-c.claimCh:
			claimCtx, cancel := context.WithTimeout(ctx, time.Second)
			c.claimTopic(claimCtx, topic)
			cancel()
		}
	}
}

func (c *Coordinator) Redis() redis.UniversalClient {
	return c.redisClient
}

func (c *Coordinator) Stop() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
	c.wg.Wait()
	_ = c.redisClient.Close()
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

var publishHWMScript = redis.NewScript(`
local cur = redis.call('GET', KEYS[1])
if cur and tonumber(cur) >= tonumber(ARGV[1]) then
 return cur
end
redis.call('SET', KEYS[1], ARGV[1])
return ARGV[1]
`)

func (c *Coordinator) PublishLogHWM(ctx context.Context, topic string, hwm uint64) {
	pubCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = publishHWMScript.Run(pubCtx, c.redisClient, []string{logHWMKey(topic)}, strconv.FormatUint(hwm, 10)).Err()
}

func (c *Coordinator) RequestClaim(topic string) {
	if c == nil {
		return
	}
	select {
	case c.claimCh <- strings.Clone(topic):
	default:
	}
}

func (c *Coordinator) claimTopic(ctx context.Context, topic string) {
	if c.IsLeader(topic) {
		return
	}
	lKey := leaderKey(topic)
	ok, err := c.redisClient.SetNX(ctx, lKey, c.nodeID, c.cfg.LeaseTTL).Result()
	if err != nil || !ok {
		return
	}
	epoch, bumped, err := c.acquireEpoch(ctx, topic, leaderEpochKey(topic))
	if err != nil {
		_ = c.redisClient.Del(ctx, lKey).Err()
		return
	}
	c.clearRenewFailures(topic)
	c.onAcquiredLeadership(ctx, topic, epoch, bumped)
}

func (c *Coordinator) HasLeader(topic string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	exists, err := c.redisClient.Exists(ctx, leaderKey(topic)).Result()
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

func (c *Coordinator) runHeartbeatLoop(ctx context.Context) {
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
			shutdownCtx, cancel := context.WithTimeout(ctx, time.Second)
			_ = c.redisClient.Del(shutdownCtx, "ad_event_processor:brokers:"+c.nodeID).Err()
			cancel()
			return
		case <-ticker.C:
			beatCtx, cancel := context.WithTimeout(ctx, time.Second)
			_ = c.redisClient.Set(beatCtx, "ad_event_processor:brokers:"+c.nodeID, c.tcpAddr, lease).Err()
			cancel()
		}
	}
}

func (c *Coordinator) runCoordinationLoop(ctx context.Context) {
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

			loopCtx, cancel := context.WithTimeout(ctx, time.Second)
			for _, topic := range topics {
				c.coordTopic(loopCtx, topic, replications)
			}
			cancel()
		}
	}
}

func (c *Coordinator) coordTopic(ctx context.Context, topic string, replications map[string]chan struct{}) {
	lKey := leaderKey(topic)
	eKey := leaderEpochKey(topic)
	lease := c.cfg.LeaseTTL

	ok, err := c.redisClient.SetNX(ctx, lKey, c.nodeID, lease).Result()
	if err != nil {
		return
	}

	if ok {
		epoch, bumped, err := c.acquireEpoch(ctx, topic, eKey)
		if err != nil {
			_ = c.redisClient.Del(ctx, lKey).Err()
			return
		}
		if stopCh, exists := replications[topic]; exists {
			close(stopCh)
			delete(replications, topic)
		}
		_ = c.redisClient.Expire(ctx, lKey, lease).Err()
		c.clearRenewFailures(topic)
		c.onAcquiredLeadership(ctx, topic, epoch, bumped)
		return
	}

	currentLeader, err := c.redisClient.Get(ctx, lKey).Result()
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
				c.PublishLogHWM(ctx, topic, pl.NextOffset())
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
	go func(parent context.Context, t string, leaderID string, sCh chan struct{}) {
		defer c.wg.Done()
		c.replicate(parent, t, leaderID, sCh)
	}(ctx, topic, currentLeader, stopCh)

	c.updateTopicMetrics(ctx, topic)
}

func (c *Coordinator) acquireEpoch(ctx context.Context, topic, eKey string) (uint64, bool, error) {
	lastWinner, _ := c.redisClient.Get(ctx, leaderLastWinnerKey(topic)).Result()
	lastSince, _ := c.redisClient.Get(ctx, leaderSinceKey(topic)).Result()
	if lastWinner == c.nodeID && lastSince != "" {
		if sinceUnix, err := strconv.ParseInt(lastSince, 10, 64); err == nil {
			elapsed := time.Since(time.Unix(sinceUnix, 0))
			if elapsed < c.cfg.DebounceWindow {
				epoch := c.readEpoch(ctx, topic)
				if epoch > 0 {
					now := strconv.FormatInt(time.Now().Unix(), 10)
					_ = c.redisClient.Set(ctx, leaderSinceKey(topic), now, 0).Err()
					_ = c.redisClient.Set(ctx, leaderLastWinnerKey(topic), c.nodeID, 0).Err()
					return epoch, false, nil
				}
			}
		}
	}
	epoch, err := c.redisClient.Incr(ctx, eKey).Result()
	if err != nil {
		return 0, false, err
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	_ = c.redisClient.Set(ctx, leaderSinceKey(topic), now, 0).Err()
	_ = c.redisClient.Set(ctx, leaderLastWinnerKey(topic), c.nodeID, 0).Err()
	return uint64(epoch), true, nil
}

func (c *Coordinator) renewLease(ctx context.Context, topic, lKey string) bool {
	lease := c.cfg.LeaseTTL
	ok, err := c.redisClient.Expire(ctx, lKey, lease).Result()
	if err != nil || !ok {
		c.recordRenewFailure(topic)
		return false
	}
	currentLeader, err := c.redisClient.Get(ctx, lKey).Result()
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
	val, err := c.redisClient.Get(ctx, leaderEpochKey(topic)).Result()
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
	val, err := c.redisClient.Get(ctx, logHWMKey(topic)).Result()
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
		c.PublishLogHWM(ctx, topic, local)
	}
	slog.Info("Acquired topic leadership", "topic", topic, "epoch", epoch, "local", local, "hwm", hwm, "ready", ready)

	if !ready {
		c.wg.Add(1)
		go func(parent context.Context) {
			defer c.wg.Done()
			c.recoverLeaderReadiness(parent, topic, hwm, time.Now())
		}(ctx)
	}
	c.updateTopicMetrics(ctx, topic)
}

func (c *Coordinator) recoverLeaderReadiness(ctx context.Context, topic string, targetHWM uint64, started time.Time) {
	const timeout = 5 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pl, err := c.host.CoordGetOrCreatePartition(topic)
		if err == nil && pl.NextOffset() >= targetHWM {
			c.setLeaderReady(topic, true)
			metrics.BrokerReplicationCatchupSeconds.WithLabelValues(topic).Observe(time.Since(started).Seconds())
			metricsCtx, cancel := context.WithTimeout(ctx, time.Second)
			c.updateTopicMetrics(metricsCtx, topic)
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
		metricsCtx, cancel := context.WithTimeout(ctx, time.Second)
		c.updateTopicMetrics(metricsCtx, topic)
		cancel()
		return
	}

	local := pl.NextOffset()
	recordReplicationError(topic, "catchup_gap")
	c.setLeaderReady(topic, true)
	metrics.BrokerReplicationCatchupSeconds.WithLabelValues(topic).Observe(time.Since(started).Seconds())
	metricsCtx, cancel := context.WithTimeout(ctx, time.Second)
	c.updateTopicMetrics(metricsCtx, topic)
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

func (c *Coordinator) replicate(ctx context.Context, topic string, leaderID string, stopCh chan struct{}) {
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
			func() {
				repCtx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				leaderAddr, err := c.redisClient.Get(repCtx, "ad_event_processor:brokers:"+leaderID).Result()
				if err != nil {
					if cli != nil {
						_ = cli.Close()
						cli = nil
					}
					return
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
						return
					}
				}

				pl, err := c.host.CoordGetOrCreatePartition(topic)
				if err != nil {
					return
				}

				nextOffset := pl.NextOffset()

				topicName, part := protocol.ParseTopicPartitionID(topic)
				iter, fetchErr := cli.Fetch(ctx, topicName, part, nextOffset, 65536)
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
			}()
		}
	}
}
