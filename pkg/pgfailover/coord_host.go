package pgfailover

import (
	"fmt"
	"sync/atomic"

	"ad-event-processor/pkg/broker/log"
	"ad-event-processor/pkg/broker/protocol"
	bserver "ad-event-processor/pkg/broker/server"
)

const CoordTopic = "global-pg"

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

type CoordHost struct {
	topicKey  string
	partition *coordPartition
}

func NewCoordHost() *CoordHost {
	return &CoordHost{
		topicKey:  protocol.TopicPartitionID(CoordTopic, 0),
		partition: &coordPartition{},
	}
}

func (h *CoordHost) CoordGetOrCreatePartition(topic string) (bserver.CoordPartition, error) {
	if topic != h.topicKey {
		return nil, fmt.Errorf("unknown topic %q", topic)
	}
	return h.partition, nil
}

func (h *CoordHost) CoordRangeTopics(fn func(topic string) bool) {
	fn(h.topicKey)
}

func (h *CoordHost) TopicKey() string {
	return h.topicKey
}
