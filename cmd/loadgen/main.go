// loadgen entrypoint. Package documentation: doc.go.
package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func init() {
	// Preload load-test env files before flag defaults resolve tracker URLs.
	applyLoadTestRuntimeEnv()
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	mode := flag.String("mode", "full", "smoke|business|full")
	rate := flag.Int("rate", 0, "target RPS (0 = mode default)")
	duration := flag.Duration("duration", 0, "test duration (0 = mode default)")
	outDir := flag.String("out", "", "session output directory")
	trackers := flag.String("trackers", envDefault("LOAD_TEST_CONSTRAINED_TRACKER_BASES_CSV", "http://127.0.0.1:8181,http://127.0.0.1:8182"), "comma-separated tracker bases")
	edgeURL := flag.String("edge", envDefault("LOAD_TEST_EDGE_URL", "http://127.0.0.1:8180"), "nginx edge URL (empty to disable)")
	oversize := flag.Int("oversize-bytes", 65536, "oversize payload for invalid traffic (bytes, default 64 KiB)")
	pctBroken := flag.Int("pct-broken", 0, "business mode: broken traffic %")
	pctGray := flag.Int("pct-gray", 0, "business mode: gray/fraud %")
	pctClickProxy := flag.Int("pct-click-proxy", 0, "GET /click % carved out of the valid bucket (proxy delivery when campaign is configured)")
	pctProxyVPN := flag.Int("pct-proxy-vpn", 0, "POST /track % with mock proxy/VPN X-Forwarded-For (carved from valid bucket)")
	pctFlowRoute := flag.Int("pct-flow-route", 0, "GET /click % with flow_id query (carved from valid bucket; flow-route drill)")
	pctJA3Block := flag.Int("pct-ja3-block", 0, "GET /click % with X-TLS-JA3 header (carved from valid bucket; JA3 block drill)")
	profile := flag.String("profile", "constant", "constant|spike")
	baseRate := flag.Int("base-rate", 200, "spike profile base RPS")
	spikeMult := flag.Int("spike-mult", 10, "spike profile peak multiplier")
	rampUp := flag.Duration("ramp-up", 10*time.Second, "spike ramp up")
	hold := flag.Duration("hold", 30*time.Second, "spike hold at peak")
	rampDown := flag.Duration("ramp-down", 10*time.Second, "spike ramp down")
	workers := flag.Int("workers", 0, "concurrent workers (0 = GOMAXPROCS*4)")
	campaignID := flag.String("campaign-id", "", "pin all /track traffic to this campaign UUID (load-test drill)")
	flag.Parse()

	if *outDir == "" {
		slog.Error("loadgen: -out is required")
		os.Exit(1)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		slog.Error("loadgen: mkdir output dir", "error", err, "dir", *outDir)
		os.Exit(1)
	}

	rps, dur := modeDefaults(*mode, *rate, *duration)
	bases := splitComma(*trackers)
	if err := healthCheck(bases); err != nil {
		slog.Error("loadgen: health check", "error", err)
		os.Exit(1)
	}

	mix := defaultMix(*mode, *pctBroken, *pctGray)
	if pct := *pctClickProxy; pct > 0 {
		if pct > mix.pctValid {
			pct = mix.pctValid
		}
		mix.pctValid -= pct
		mix.pctClickProxy = pct
	}
	if pct := *pctProxyVPN; pct > 0 {
		if pct > mix.pctValid {
			pct = mix.pctValid
		}
		mix.pctValid -= pct
		mix.pctProxyVPN = pct
	}
	if pct := *pctFlowRoute; pct > 0 {
		if pct > mix.pctValid {
			pct = mix.pctValid
		}
		mix.pctValid -= pct
		mix.pctFlowRoute = pct
	}
	if pct := *pctJA3Block; pct > 0 {
		if pct > mix.pctValid {
			pct = mix.pctValid
		}
		mix.pctValid -= pct
		mix.pctJA3Block = pct
	}
	hist := newHistogram()
	run := newRunner(bases, *edgeURL, *oversize, mix, hist)
	run.campaignID = strings.TrimSpace(*campaignID)

	w := *workers
	if w <= 0 {
		w = runtime.GOMAXPROCS(0) * 4
		if w < 8 {
			w = 8
		}
	}

	log.Printf("loadgen: mode=%s profile=%s rate=%d duration=%s workers=%d trackers=%v",
		*mode, *profile, rps, dur, w, bases)

	startedAt := time.Now()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	if *profile == "spike" {
		sched := spikeSchedule{
			base: *baseRate, mult: *spikeMult,
			rampUp: *rampUp, hold: *hold, rampDown: *rampDown,
			start: startedAt,
		}
		go runSpike(run, w, &wg, stop, sched)
		time.Sleep(sched.total())
	} else {
		go runConstant(run, &wg, stop, rps, w)
		time.Sleep(dur)
	}

	close(stop)
	wg.Wait()

	histPath := *outDir + "/status-histogram.json"
	if err := hist.write(histPath, startedAt.UTC().Format(time.RFC3339)); err != nil {
		slog.Error("loadgen: write histogram", "error", err, "path", histPath)
		os.Exit(1)
	}
	elapsed := time.Since(startedAt).Round(time.Millisecond)
	log.Printf("loadgen: done in %s - %s", elapsed, histPath)
}

func modeDefaults(mode string, rate int, dur time.Duration) (int, time.Duration) {
	if dur == 0 {
		switch mode {
		case "smoke":
			dur = 2 * time.Minute
		default:
			dur = 5 * time.Minute
		}
	}
	if rate > 0 {
		return rate, dur
	}
	switch mode {
	case "smoke":
		return 500, dur
	case "business":
		return 2000, dur
	default:
		return 2000, dur
	}
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runConstant(run *runner, wg *sync.WaitGroup, stop <-chan struct{}, rps, workers int) {
	if rps <= 0 {
		return
	}
	if workers <= 0 {
		workers = 1
	}
	perWorker := rps / workers
	if perWorker < 1 {
		perWorker = 1
	}
	interval := time.Second / time.Duration(perWorker)
	if interval < time.Microsecond {
		interval = time.Microsecond
	}
	for i := range workers {
		wg.Add(1)
		workerIdx := i
		go func() {
			defer wg.Done()
			next := time.Now()
			for {
				select {
				case <-stop:
					return
				default:
				}
				now := time.Now()
				if now.Before(next) {
					timer := time.NewTimer(next.Sub(now))
					select {
					case <-stop:
						timer.Stop()
						return
					case <-timer.C:
					}
				}
				run.doOnceWorker(workerIdx)
				next = next.Add(interval)
				if next.Before(time.Now().Add(-interval)) {
					next = time.Now()
				}
			}
		}()
	}
}

type spikeSchedule struct {
	base, mult int
	rampUp     time.Duration
	hold       time.Duration
	rampDown   time.Duration
	start      time.Time
}

func (s spikeSchedule) total() time.Duration {
	return s.rampUp + s.hold + s.rampDown
}

func runSpike(run *runner, workers int, wg *sync.WaitGroup, stop <-chan struct{}, s spikeSchedule) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for workerIdx := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var tokens float64
			last := time.Now()
			for {
				select {
				case <-stop:
					return
				case now := <-ticker.C:
					elapsed := now.Sub(last).Seconds()
					last = now
					rps := spikeRPS(s, now)
					tokens += float64(rps) * elapsed / float64(workers)
					for tokens >= 1 {
						run.doOnceWorker(idx)
						tokens--
					}
				}
			}
		}(workerIdx)
	}
}

func spikeRPS(s spikeSchedule, now time.Time) int {
	elapsed := now.Sub(s.start)
	switch {
	case elapsed < s.rampUp:
		frac := float64(elapsed) / float64(s.rampUp)
		return s.base + int(float64(s.base*(s.mult-1))*frac)
	case elapsed < s.rampUp+s.hold:
		return s.base * s.mult
	case elapsed < s.rampUp+s.hold+s.rampDown:
		left := s.rampUp + s.hold + s.rampDown - elapsed
		frac := float64(left) / float64(s.rampDown)
		return s.base + int(float64(s.base*(s.mult-1))*frac)
	default:
		return s.base
	}
}

func applyLoadTestRuntimeEnv() {
	root := loadgenRepoRoot()
	paths := []string{
		filepath.Join(root, "deploy/compose/.env.load-test.runtime"),
		filepath.Join(root, ".env.load-test"),
	}
	for _, path := range paths {
		if loadgenApplyEnvFile(path) {
			return
		}
	}
}

func loadgenRepoRoot() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_REPO_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
		dir = parent
	}
}

func loadgenApplyEnvFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	applied := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			continue
		}
		applied = true
	}
	return applied
}
