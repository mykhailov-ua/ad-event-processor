package controlplane

import (
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/probecluster"

	"github.com/redis/go-redis/v9"
)

func (s *Service) ProbeClusterRedis() redis.UniversalClient {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	for _, client := range s.redisShards {
		if client != nil {
			return client
		}
	}
	return nil
}

func (s *Service) ProbeClusterExportEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.ProbeClusterEnabled
}

func (s *Service) ProbeClusterStore() *filter.ProbeClusterStore {
	if s == nil || s.cfg == nil || !s.cfg.ProbeClusterEnabled {
		return nil
	}
	ttl := time.Duration(s.cfg.ProbeClusterTTLDays) * 24 * time.Hour
	return filter.NewProbeClusterStore(s.ProbeClusterRedis(), ttl)
}

func (s *Service) ProbeClusterPolicy() probecluster.Policy {
	if s == nil || s.cfg == nil {
		return probecluster.DefaultPolicy()
	}
	return probecluster.Policy{
		MinSessions:    s.cfg.ProbeClusterMinSessions,
		MinCampaigns:   s.cfg.ProbeClusterMinCampaigns,
		ScoreThreshold: s.cfg.ProbeClusterScoreThreshold,
	}
}

func (s *Service) ProbeClusterService() *fraudadmin.ProbeCluster {
	return fraudadmin.NewProbeCluster(s, s.ProbeClusterStore())
}

func (s *Service) StartProbeClusterExportWorker() {
	if s == nil || s.cfg == nil || !s.cfg.ProbeClusterEnabled {
		return
	}
	export := s.ProbeClusterService()
	worker := fraudadmin.NewProbeClusterExportWorker(s, export, s.ProbeClusterPolicy(), s.cfg.ProbeClusterExportInterval)
	if worker == nil {
		return
	}
	s.startWorker(func() {
		worker.Start(s.ctx)
	})
}
