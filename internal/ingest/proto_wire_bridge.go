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

func newRedisLuaNoScriptCounters(numShards int) []prometheus.Counter {
	return fw.NewRedisLuaNoScriptCounters(numShards)
}

func incRedisLuaNoScript(counters []prometheus.Counter, shard int) {
	fw.IncRedisLuaNoScript(counters, shard)
}

func observeRedisLua(observers []prometheus.Observer, shard int, seconds float64) {
	fw.ObserveRedisLua(observers, shard, seconds)
}

var errProtoFieldBudget = fw.ErrProtoFieldBudget

func newRedisLuaTierObservers(numShards int) []prometheus.Observer {
	return fw.NewRedisLuaTierObservers(numShards)
}

func newRedisLuaPathCounters(numShards int, fast bool) []prometheus.Counter {
	return fw.NewRedisLuaPathCounters(numShards, fast)
}

func incRedisLuaTier(counters []prometheus.Counter, shard int) {
	fw.IncRedisLuaTier(counters, shard)
}

func observeRedisLuaTier(observers []prometheus.Observer, shard int, seconds float64) {
	fw.ObserveRedisLuaTier(observers, shard, seconds)
}

func newRedisOpsCounters(numShards int) []prometheus.Counter {
	return fw.NewRedisOpsCounters(numShards)
}

func incRedisOps(counters []prometheus.Counter, shard int) {
	fw.IncRedisOps(counters, shard)
}
