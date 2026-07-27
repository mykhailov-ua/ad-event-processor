// Dev-only BPF collector for load-test sessions.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"

	"espx/cmd/bpf-collector/bpfprobe"
)

const (
	roleTracker   = 1
	roleNginx     = 2
	roleRedis     = 3
	roleK6        = 4
	roleProcessor = 5
)

func main() {
	sessionDir := flag.String("session-dir", "", "session output directory (contains targets.json)")
	bpfObject := flag.String("bpf-object", "", "path to loadtest_probe.o (default: deploy/dev/bpf/loadtest_probe.o)")
	sampleRate := flag.Uint("sample-rate", 1, "syscall sample rate (1=every event)")
	slowUs := flag.Uint("slow-us", 10000, "slow syscall threshold microseconds")
	discoverK6 := flag.Bool("discover-k6", true, "watch for k6 load generator PIDs")
	discoverSec := flag.Duration("discover-interval", 2*time.Second, "dynamic target scan interval")
	flag.Parse()

	if *sessionDir == "" {
		fmt.Fprintln(os.Stderr, "session-dir is required")
		os.Exit(2)
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		slog.Error("rlimit", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sess, err := openSession(*sessionDir)
	if err != nil {
		slog.Error("open session", "error", err)
		os.Exit(1)
	}

	objPath := *bpfObject
	if objPath == "" {
		objPath = filepath.Join(repoRoot(), "deploy", "dev", "bpf", "loadtest_probe.o")
	}

	run := &probeRun{
		session:      sess,
		sampleRate:   uint32(*sampleRate),
		slowNs:       uint64(*slowUs) * 1000,
		discoverK6:   *discoverK6,
		discoverTick: *discoverSec,
		bpfObject:    objPath,
	}
	if err := run.start(ctx); err != nil {
		slog.Error("probe start", "error", err)
		os.Exit(1)
	}
	defer run.stop()

	slog.Info("bpf-collector running", "session", *sessionDir, "object", objPath)
	<-ctx.Done()
	slog.Info("bpf-collector stopping")
}

func repoRoot() string {
	if v := os.Getenv("ESPX_REPO_ROOT"); v != "" {
		return v
	}
	// scripts set ROOT; bpf-collector may be invoked from repo root.
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

type probeRun struct {
	session      *session
	sampleRate   uint32
	slowNs       uint64
	discoverK6   bool
	discoverTick time.Duration
	bpfObject    string

	coll     *bpfprobe.Collection
	links    []link.Link
	ringWG   sync.WaitGroup
	sampleWG sync.WaitGroup
	cancel   context.CancelFunc
	tracked  map[uint32]targetEntry
	mu       sync.Mutex
}

func (r *probeRun) start(ctx context.Context) error {
	coll, err := bpfprobe.Load(r.bpfObject)
	if err != nil {
		return err
	}
	r.coll = coll

	cfg := bpfprobe.Config{
		SampleRate:    r.sampleRate,
		SlowSyscallNs: uint32(minU64(r.slowNs, 0xffffffff)),
		Enabled:       1,
	}
	if err := coll.SetConfig(cfg); err != nil {
		return err
	}

	r.tracked = make(map[uint32]targetEntry)
	for _, t := range sessTargets(r.session) {
		if err := r.trackPID(t.PID, t.Role, t.Name); err != nil {
			slog.Warn("track pid", "pid", t.PID, "name", t.Name, "error", err)
		}
	}

	attach := []struct {
		group string
		name  string
		prog  *ebpf.Program
	}{
		{"raw_syscalls", "sys_enter", coll.Progs.SysEnter},
		{"raw_syscalls", "sys_exit", coll.Progs.SysExit},
		{"sched", "sched_wakeup", coll.Progs.SchedWakeup},
		{"sched", "sched_switch", coll.Progs.SchedSwitch},
		{"exceptions", "page_fault_user", coll.Progs.PageFaultUser},
		{"sched", "sched_process_fork", coll.Progs.SchedProcessFork},
		{"sched", "sched_process_exit", coll.Progs.SchedProcessExit},
	}

	for _, a := range attach {
		if a.prog == nil {
			continue
		}
		l, err := link.Tracepoint(a.group, a.name, a.prog, nil)
		if err != nil {
			slog.Warn("tracepoint attach skipped", "group", a.group, "name", a.name, "error", err)
			continue
		}
		r.links = append(r.links, l)
	}

	if coll.Progs.TcpRetransmit != nil {
		if l, err := link.Kprobe("tcp_retransmit_skb", coll.Progs.TcpRetransmit, nil); err != nil {
			slog.Warn("kprobe attach skipped", "symbol", "tcp_retransmit_skb", "error", err)
		} else {
			r.links = append(r.links, l)
		}
	}

	ringCtx, ringCancel := context.WithCancel(ctx)
	r.cancel = ringCancel
	r.ringWG.Add(1)
	go r.drainRingbuf(ringCtx, coll.Maps.SlowEvents)

	r.sampleWG.Add(1)
	go r.procSampleLoop(ringCtx)

	r.sampleWG.Add(1)
	go r.cgroupSampleLoop(ringCtx)

	if r.discoverK6 {
		go r.discoverLoop(ringCtx)
	}

	return r.writeMemSnapshot("mem-start.json")
}

func (r *probeRun) stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.ringWG.Wait()
	r.sampleWG.Wait()
	for _, l := range r.links {
		_ = l.Close()
	}
	if r.coll != nil {
		if err := r.dumpMaps(); err != nil {
			slog.Error("dump maps", "error", err)
		}
		if err := r.writeMemSnapshot("mem-end.json"); err != nil {
			slog.Error("mem-end snapshot", "error", err)
		}
		if err := r.session.markEnded(); err != nil {
			slog.Error("mark ended", "error", err)
		}
		r.coll.Close()
	}
}

func sessTargets(s *session) []targetEntry {
	return s.Meta.Targets
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (r *probeRun) drainRingbuf(ctx context.Context, m *ebpf.Map) {
	defer r.ringWG.Done()
	if m == nil {
		return
	}
	rd, err := ringbuf.NewReader(m)
	if err != nil {
		slog.Warn("ringbuf reader", "error", err)
		return
	}
	defer rd.Close()
	drainRingbufRecords(ctx, rd, r.session.Dir)
}
