package server

// CoordPartition is the minimal log surface required for Redis HA coordination.
type CoordPartition interface {
	NextOffset() uint64
	AppendReplicatedAt(expectedOffset uint64, payload []byte) (uint64, error)
	AdvanceFencingEpoch(epoch uint64) error
}

// CoordHost exposes topic partitions to the Redis coordinator.
type CoordHost interface {
	CoordGetOrCreatePartition(topic string) (CoordPartition, error)
	CoordRangeTopics(fn func(topic string) bool)
}

// CoordGetOrCreatePartition opens a broker partition for coordination and replication.
func (s *Server) CoordGetOrCreatePartition(topic string) (CoordPartition, error) {
	return s.getOrCreatePartition(topic)
}

// CoordRangeTopics lists topic keys known to this broker node.
func (s *Server) CoordRangeTopics(fn func(topic string) bool) {
	s.topics.Range(func(key, _ any) bool {
		return fn(key.(string))
	})
}
