package ingestion

import (
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/protobuf/proto"
)

type chaosLoadConfig struct {
	Duration  time.Duration
	RPS       int
	ChaosPct  int
	P99Budget time.Duration
	Workers   int
}

type chaosLoadResult struct {
	TotalReqs   int64
	ControlReqs int64
	ChaosReqs   int64
	P99Nanos    int64
	PoolRejects float64
	Elapsed     time.Duration
	AchievedRPS float64
}

func chaosLoadConfigFromEnv() chaosLoadConfig {
	cfg := chaosLoadConfig{
		Duration:  8 * time.Second,
		RPS:       3000,
		ChaosPct:  10,
		P99Budget: 80 * time.Millisecond,
		Workers:   4,
	}
	if v := os.Getenv("CHAOS_LOAD_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.Duration = d
		}
	}
	if v := os.Getenv("CHAOS_LOAD_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RPS = n
		}
	}
	if v := os.Getenv("CHAOS_LOAD_CHAOS_PCT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 100 {
			cfg.ChaosPct = n
		}
	}
	if v := os.Getenv("CHAOS_LOAD_P99_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.P99Budget = time.Duration(n) * time.Millisecond
		}
	}
	if v := os.Getenv("CHAOS_LOAD_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Workers = n
		}
	}
	return cfg
}

func runParserChaosLoad(cfg chaosLoadConfig) chaosLoadResult {
	if cfg.RPS <= 0 {
		cfg.RPS = 3000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 8 * time.Second
	}

	h := NewAdsPacketHandler(
		&config.Config{MaxRequestBodySize: 1 << 20, HTTP1IncompleteMax: 3, HTTP1BodyIdleMs: 60_000},
		&mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil,
	)

	gens := newChaosLoadGenerators()
	valid := gens.validProtoWire()
	poolBefore := testutil.ToFloat64(metrics.WorkerPoolRejectTotal)

	deadline := time.Now().Add(cfg.Duration)
	interval := time.Second / time.Duration(cfg.RPS)

	var (
		totalReqs   atomic.Int64
		controlReqs atomic.Int64
		chaosReqs   atomic.Int64
		latMu       sync.Mutex
		latencies   []int64
	)

	var wg sync.WaitGroup
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	stop := make(chan struct{})
	jobs := make(chan bool, cfg.RPS*2)

	go func() {
		<-time.After(cfg.Duration)
		close(stop)
	}()

	rng := newChaosLoadRNG(42)
	go func() {
		defer close(jobs)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if time.Now().After(deadline) {
					return
				}
				control := rng.next100() >= cfg.ChaosPct
				select {
				case jobs <- control:
				default:
				}
			}
		}
	}()

	worker := func(seed int64) {
		defer wg.Done()
		wrng := newChaosLoadRNG(seed)
		for control := range jobs {
			var wire []byte
			if control {
				wire = valid
			} else {
				wire, _ = gens.pick(wrng, valid, 100)
			}
			conn := NewGnetHarnessConn(wire)
			start := monotonicNano()
			_ = h.OnTraffic(conn)
			totalReqs.Add(1)
			if control {
				controlReqs.Add(1)
				elapsed := monotonicNano() - start
				latMu.Lock()
				latencies = append(latencies, elapsed)
				latMu.Unlock()
			} else {
				chaosReqs.Add(1)
			}
		}
	}

	wg.Add(cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		go worker(int64(i) + 1)
	}
	wg.Wait()

	elapsed := cfg.Duration
	if elapsed <= 0 {
		elapsed = time.Second
	}
	total := totalReqs.Load()
	return chaosLoadResult{
		TotalReqs:   total,
		ControlReqs: controlReqs.Load(),
		ChaosReqs:   chaosReqs.Load(),
		P99Nanos:    latencyP99Nanos(latencies),
		PoolRejects: testutil.ToFloat64(metrics.WorkerPoolRejectTotal) - poolBefore,
		Elapsed:     elapsed,
		AchievedRPS: float64(total) / elapsed.Seconds(),
	}
}

func latencyP99Nanos(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := (len(cp)*99+99)/100 - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

type chaosLoadGenerators struct {
	validProto  []byte
	wsBomb      []byte
	ortbGarbage []byte
	chunkedORTB []byte
	slowBody    []byte
	protoFlood  []byte
	malformed   []byte
}

func newChaosLoadGenerators() *chaosLoadGenerators {
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	evt := &pb.AdEvent{
		CampaignId: cid[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId:    []byte("load-click"),
			ExtraBytes: []byte(`{"slot":"top"}`),
		},
	}
	body, _ := proto.Marshal(evt)
	valid := BuildGnetHTTP("POST", "/track", map[string]string{
		"Content-Type":   "application/x-protobuf",
		"Content-Length": strconv.Itoa(len(body)),
	}, body)

	garbage := make([]byte, 64*1024)
	for i := range garbage {
		garbage[i] = 'A'
	}
	ortbBody := append([]byte(`{"id":"x",`), garbage...)
	ortbBody = append(ortbBody, `}`...)

	return &chaosLoadGenerators{
		validProto: valid,
		wsBomb:     BuildGnetPostTrackJSON(chaosWSBomb(MaxWSkip, chaosValidTrackJSON)),
		ortbGarbage: BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
			"Content-Type":   "application/json",
			"Content-Length": strconv.Itoa(len(ortbBody)),
		}, ortbBody),
		chunkedORTB: fragmentedChunkedOpenRTBRequest(),
		slowBody:    append(chaosSlowBodyHeaders(), chaosSlowBodyPrefixBytes()[:16]...),
		protoFlood: BuildGnetHTTP("POST", "/track", map[string]string{
			"Content-Type":   "application/x-protobuf",
			"Content-Length": strconv.Itoa(len(chaosProtoWireFieldFlood(10_000))),
		}, chaosProtoWireFieldFlood(10_000)),
		malformed: randomWireGarbage(128),
	}
}

func (g *chaosLoadGenerators) validProtoWire() []byte {
	return g.validProto
}

func (g *chaosLoadGenerators) pick(rng *chaosLoadRNG, valid []byte, chaosPct int) (wire []byte, control bool) {
	if chaosPct <= 0 || rng.next100() >= chaosPct {
		return valid, true
	}
	switch rng.nextN(6) {
	case 0:
		return g.wsBomb, false
	case 1:
		return g.ortbGarbage, false
	case 2:
		return g.chunkedORTB, false
	case 3:
		return g.slowBody, false
	case 4:
		return g.protoFlood, false
	default:
		return g.malformed, false
	}
}

type chaosLoadRNG struct {
	state uint64
}

func newChaosLoadRNG(seed int64) *chaosLoadRNG {
	if seed == 0 {
		seed = 1
	}
	return &chaosLoadRNG{state: uint64(seed)}
}

func (r *chaosLoadRNG) next100() int {
	r.state = r.state*6364136223846793005 + 1
	return int((r.state >> 33) % 100)
}

func (r *chaosLoadRNG) nextN(n int) int {
	if n <= 0 {
		return 0
	}
	r.state = r.state*6364136223846793005 + 1
	return int((r.state >> 33) % uint64(n))
}
