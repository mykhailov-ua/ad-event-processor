package controlplane

import (
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/crowdwave"

	"github.com/redis/go-redis/v9"
)

func (s *Service) CrowdWaveRedis() redis.UniversalClient {
	return s.ProbeClusterRedis()
}

func (s *Service) CrowdWaveEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.CrowdWaveEnabled
}

func (s *Service) CrowdWavePolicy() crowdwave.WavePolicy {
	if s == nil || s.cfg == nil {
		return crowdwave.DefaultWavePolicy()
	}
	return crowdwave.WavePolicy{
		WindowSec:           int64(s.cfg.CrowdWaveWindowSec),
		MinUniqueClusters:   s.cfg.CrowdWaveMinUniqueClusters,
		MinSimhashNeighbors: s.cfg.CrowdWaveMinSimhashNeighbors,
		SimhashMaxDist:      s.cfg.CrowdWaveSimhashMaxDist,
	}
}

func (s *Service) CrowdWaveWindow() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.CrowdWaveWindowSec <= 0 {
		return time.Duration(crowdwave.DefaultWaveWindowSec) * time.Second
	}
	return time.Duration(s.cfg.CrowdWaveWindowSec) * time.Second
}

func (s *Service) CrowdWaveStore() *filter.CrowdWaveStore {
	if s == nil || s.cfg == nil || !s.cfg.CrowdWaveEnabled {
		return nil
	}
	return filter.NewCrowdWaveStore(s.CrowdWaveRedis(), s.CrowdWaveWindow(), s.CrowdWavePolicy())
}

func (s *Service) CrowdWaveService() *fraudadmin.CrowdWave {
	return fraudadmin.NewCrowdWave(s, s.CrowdWaveStore())
}

func (s *Service) StartCrowdWaveExportWorker() {
	if s == nil || s.cfg == nil || !s.cfg.CrowdWaveEnabled {
		return
	}
	export := s.CrowdWaveService()
	worker := fraudadmin.NewCrowdWaveExportWorker(s, export, s.cfg.CrowdWaveExportInterval)
	if worker == nil {
		return
	}
	go worker.Start(s.ctx)
}
