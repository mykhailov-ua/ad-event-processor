package server

type CoordPartition interface {
	NextOffset() uint64
	AppendReplicatedAt(expectedOffset uint64, payload []byte) (uint64, error)
	AdvanceFencingEpoch(epoch uint64) error
}

type CoordHost interface {
	CoordGetOrCreatePartition(topic string) (CoordPartition, error)
	CoordRangeTopics(fn func(topic string) bool)
}

func (s *Server) CoordGetOrCreatePartition(topic string) (CoordPartition, error) {
	return s.getOrCreatePartition(topic)
}

func (s *Server) CoordRangeTopics(fn func(topic string) bool) {
	s.topics.Range(func(key, _ any) bool {
		return fn(key.(string))
	})
}
