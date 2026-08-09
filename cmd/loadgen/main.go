package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

func main() {
	mode := flag.String("mode", "full", "smoke|business|full")
	rate := flag.Int("rate", 0, "target RPS (0 = mode default)")
	duration := flag.Duration("duration", 0, "test duration (0 = mode default)")
	outDir := flag.String("out", "", "session output directory")
	trackers := flag.String("trackers", "http://127.0.0.1:8181,http://127.0.0.1:8182", "comma-separated tracker bases")
	edgeURL := flag.String("edge", "http://127.0.0.1:8180", "nginx edge URL (empty to disable)")
	oversize := flag.Int("oversize-bytes", 65536, "oversize payload for invalid traffic")
	pctBroken := flag.Int("pct-broken", 0, "business mode: broken traffic %")
	pctGray := flag.Int("pct-gray", 0, "business mode: gray/fraud %")
	profile := flag.String("profile", "constant", "constant|spike")
	baseRate := flag.Int("base-rate", 200, "spike profile base RPS")
	spikeMult := flag.Int("spike-mult", 10, "spike profile peak multiplier")
	rampUp := flag.Duration("ramp-up", 10*time.Second, "spike ramp up")
	hold := flag.Duration("hold", 30*time.Second, "spike hold at peak")
	rampDown := flag.Duration("ramp-down", 10*time.Second, "spike ramp down")
	workers := flag.Int("workers", 0, "concurrent workers (0 = GOMAXPROCS*4)")
	flag.Parse()

	if *outDir == "" {
		log.Fatal("loadgen: -out is required")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	rps, dur := modeDefaults(*mode, *rate, *duration)
	bases := splitComma(*trackers)
	if err := healthCheck(bases); err != nil {
		log.Fatalf("loadgen: health check: %v", err)
	}

	mix := defaultMix(*mode, *pctBroken, *pctGray)
	hist := newHistogram()
	run := newRunner(bases, *edgeURL, *oversize, mix, hist)

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
		log.Fatalf("loadgen: write histogram: %v", err)
	}
	elapsed := time.Since(startedAt).Round(time.Millisecond)
	log.Printf("loadgen: done in %s — %s", elapsed, histPath)
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
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					run.doOnce()
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

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
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
						run.doOnce()
						tokens--
					}
				}
			}
		}()
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
