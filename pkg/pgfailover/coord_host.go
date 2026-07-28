package pgfailover

import (
	"fmt"
	"sync/atomic"

	"espx/pkg/broker/log"
	"espx/pkg/broker/protocol"
	bserver "espx/pkg/broker/server"
)

// CoordTopic is the Redis HA coordination key for global Postgres failover.
const CoordTopic = "global-pg"

// coordPartition stores the local fencing floor for the coordinator host.
type coordPartition struct {
	fencingEpoch atomic.Uint64
	nextOffset   atomic.Uint64
}

func (p *coordPartition) NextOffset() uint64 {
	return p.nextOffset.Load()
}

func (p *coordPartition) AppendReplicatedAt(expectedOffset uint64, payload []byte) (uint64, error) {
	next := p.nextOffset.Load()
	if expectedOffset < next {
		return expectedOffset, nil
	}
	if expectedOffset > next {
		return 0, log.ErrReplicationGap
	}
	p.nextOffset.Add(1)
	return expectedOffset, nil
}

func (p *coordPartition) AdvanceFencingEpoch(epoch uint64) error {
	for {
		cur := p.fencingEpoch.Load()
		if epoch <= cur {
			return nil
		}
		if p.fencingEpoch.CompareAndSwap(cur, epoch) {
			return nil
		}
	}
}

// CoordHost implements bserver.CoordHost for global Postgres failover.
type CoordHost struct {
	topicKey  string
	partition *coordPartition
}

// NewCoordHost wires the single global-pg topic into broker coordination.
func NewCoordHost() *CoordHost {
	return &CoordHost{
		topicKey:  protocol.TopicPartitionID(CoordTopic, 0),
		partition: &coordPartition{},
	}
}

// CoordGetOrCreatePartition implements bserver.CoordHost.
func (h *CoordHost) CoordGetOrCreatePartition(topic string) (bserver.CoordPartition, error) {
	if topic != h.topicKey {
		return nil, fmt.Errorf("unknown topic %q", topic)
	}
	return h.partition, nil
}

// CoordRangeTopics implements bserver.CoordHost.
func (h *CoordHost) CoordRangeTopics(fn func(topic string) bool) {
	fn(h.topicKey)
}

// TopicKey returns the coordination topic key used by the broker coordinator.
func (h *CoordHost) TopicKey() string {
	return h.topicKey
}
