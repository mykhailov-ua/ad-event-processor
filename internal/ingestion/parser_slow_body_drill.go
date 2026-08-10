package ingestion

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"espx/internal/config"

	"github.com/panjf2000/gnet/v2"
)

type slowBodyDrillConfig struct {
	Connections int
	Duration    time.Duration
	P99Budget   time.Duration
	ControlRPS  int
}

type slowBodyDrillResult struct {
	ControlReqs int64
	P99Nanos    int64
	Elapsed     time.Duration
}

func slowBodyDrillConfigFromEnv() slowBodyDrillConfig {
	cfg := slowBodyDrillConfig{
		Connections: 64,
		Duration:    8 * time.Second,
		P99Budget:   80 * time.Millisecond,
		ControlRPS:  800,
	}
	if v := os.Getenv("SLOW_BODY_DRILL_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Connections = n
		}
	}
	if v := os.Getenv("SLOW_BODY_DRILL_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.Duration = d
		}
	}
	if v := os.Getenv("SLOW_BODY_DRILL_P99_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.P99Budget = time.Duration(n) * time.Millisecond
		}
	}
	if v := os.Getenv("SLOW_BODY_DRILL_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ControlRPS = n
		}
	}
	return cfg
}

func runParserSlowBodyDrill(cfg slowBodyDrillConfig) slowBodyDrillResult {
	if cfg.Connections <= 0 {
		cfg.Connections = 64
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 8 * time.Second
	}
	if cfg.P99Budget <= 0 {
		cfg.P99Budget = 80 * time.Millisecond
	}
	if cfg.ControlRPS <= 0 {
		cfg.ControlRPS = 800
	}

	h := NewAdsPacketHandler(
		&config.Config{
			MaxRequestBodySize: 1 << 20,
			HTTP1IncompleteMax: 3,
			HTTP1BodyIdleMs:    60_000,
		},
		&mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil,
	)
	valid := newChaosLoadGenerators().validProtoWire()
	headers := chaosSlowBodyHeaders()
	body := chaosSlowBodyPrefixBytes()

	deadline := time.Now().Add(cfg.Duration)
	stop := make(chan struct{})
	go func() {
		<-time.After(cfg.Duration)
		close(stop)
	}()

	var slowWG sync.WaitGroup
	slowWG.Add(cfg.Connections)
	for i := 0; i < cfg.Connections; i++ {
		go func() {
			defer slowWG.Done()
			conn := NewGnetHarnessConn(nil)
			conn.Append(headers)
			_ = h.OnTraffic(conn)
			pos := 0
			tick := time.NewTicker(time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-stop:
					return
				case <-tick.C:
					if pos < len(body) {
						conn.Append(body[pos : pos+1])
						pos++
					}
					if h.OnTraffic(conn) == gnet.Close {
						conn = NewGnetHarnessConn(nil)
						conn.Append(headers)
						pos = 0
						_ = h.OnTraffic(conn)
					}
				}
			}
		}()
	}

	interval := time.Second / time.Duration(cfg.ControlRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var (
		controlReqs atomic.Int64
		latMu       sync.Mutex
		latencies   []int64
	)

	for {
		select {
		case <-stop:
			slowWG.Wait()
			return slowBodyDrillResult{
				ControlReqs: controlReqs.Load(),
				P99Nanos:    latencyP99Nanos(latencies),
				Elapsed:     cfg.Duration,
			}
		case <-ticker.C:
			if time.Now().After(deadline) {
				continue
			}
			conn := NewGnetHarnessConn(valid)
			start := monotonicNano()
			_ = h.OnTraffic(conn)
			controlReqs.Add(1)
			elapsed := monotonicNano() - start
			latMu.Lock()
			latencies = append(latencies, elapsed)
			latMu.Unlock()
		}
	}
}
