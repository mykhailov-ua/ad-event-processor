package ingest

import (
	"ad-event-processor/internal/config"
	fw "ad-event-processor/internal/ingest/filterwire"
	"ad-event-processor/internal/ingest/pb"

	"github.com/prometheus/client_golang/prometheus"
)

const ProtoMaxFields = fw.ProtoMaxFields

func configureProtoMaxFields(cfg *config.Config) {
	fw.ConfigureProtoMaxFields(cfg)
}

func unmarshalAdEventVT(evt *pb.AdEvent, wire []byte) error {
	return fw.UnmarshalAdEventVT(evt, wire)
}

func chaosProtoWireFieldFlood(n int) []byte {
	return fw.ChaosProtoWireFieldFlood(n)
}

func newRedisLuaObservers(numShards int) []prometheus.Observer {
	return fw.NewRedisLuaObservers(numShards)
}

func observeRedisLua(observers []prometheus.Observer, shard int, seconds float64) {
	fw.ObserveRedisLua(observers, shard, seconds)
}

var errProtoFieldBudget = fw.ErrProtoFieldBudget
