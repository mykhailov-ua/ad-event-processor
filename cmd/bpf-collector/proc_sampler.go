package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type procSampleRow struct {
	TsNs        int64  `json:"ts_ns"`
	PID         uint32 `json:"pid"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	OpenFDs     uint64 `json:"open_fds"`
	SocketFDs   uint64 `json:"socket_fds"`
	Threads     uint64 `json:"threads"`
	VmRSSKB     uint64 `json:"vm_rss_kb"`
	VmHWMKB     uint64 `json:"vm_hwm_kb"`
	RssAnonKB   uint64 `json:"rss_anon_kb"`
	MinFlt      uint64 `json:"minflt"`
	MajFlt      uint64 `json:"majflt"`
}

type procSamplePeak struct {
	PID          uint32  `json:"pid"`
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	PeakOpenFDs  uint64  `json:"peak_open_fds"`
	PeakSocketFD uint64  `json:"peak_socket_fds"`
	PeakThreads  uint64  `json:"peak_threads"`
	StartOpenFDs uint64  `json:"start_open_fds"`
	EndOpenFDs   uint64  `json:"end_open_fds"`
	FdDelta      int64   `json:"fd_delta"`
	ThreadDelta  int64   `json:"thread_delta"`
	SampleCount  uint64  `json:"sample_count"`
	FdOpenRate   float64 `json:"fd_open_per_sec,omitempty"`
	FdCloseRate  float64 `json:"fd_close_per_sec,omitempty"`
	SocketOpen   uint64  `json:"socket_open"`
	SocketAccept uint64  `json:"socket_accept"`
	ThreadFork   uint64  `json:"thread_fork"`
	ThreadExit   uint64  `json:"thread_exit"`
}

func (r *probeRun) procSampleLoop(ctx context.Context) {
	defer r.sampleWG.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	outPath := filepath.Join(r.session.Dir, "proc-samples.ndjson")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			targets := make([]targetEntry, 0, len(r.tracked))
			for _, t := range r.tracked {
				targets = append(targets, t)
			}
			r.mu.Unlock()

			ts := time.Now().UnixNano()
			for _, t := range targets {
				mem, err := readProcMem(t)
				if err != nil {
					continue
				}
				fds, socks, _, _ := readProcFDAndThreads(t.PID)
				_ = enc.Encode(procSampleRow{
					TsNs:      ts,
					PID:       t.PID,
					Name:      t.Name,
					Role:      roleName(t.Role),
					OpenFDs:   fds,
					SocketFDs: socks,
					Threads:   mem.Threads,
					VmRSSKB:   mem.VmRSSKB,
					VmHWMKB:   mem.VmHWMKB,
					RssAnonKB: mem.RssAnonKB,
					MinFlt:    mem.MinFlt,
					MajFlt:    mem.MajFlt,
				})
			}
		}
	}
}

func aggregateProcSamples(sessionDir string, durationSec float64, bpfStats []dumpedPIDStats) ([]procSamplePeak, error) {
	path := filepath.Join(sessionDir, "proc-samples.ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return peaksFromMemSnapshots(sessionDir, bpfStats, durationSec)
		}
		return nil, err
	}

	type acc struct {
		name         string
		role         string
		peakFD       uint64
		peakSock     uint64
		peakThreads  uint64
		firstFD      uint64
		lastFD       uint64
		firstThreads uint64
		lastThreads  uint64
		count        uint64
		seenFirst    bool
	}
	byPID := map[uint32]*acc{}

	for line := range splitLines(data) {
		var row procSampleRow
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		a := byPID[row.PID]
		if a == nil {
			a = &acc{name: row.Name, role: row.Role}
			byPID[row.PID] = a
		}
		if !a.seenFirst {
			a.firstFD = row.OpenFDs
			a.firstThreads = row.Threads
			a.seenFirst = true
		}
		a.lastFD = row.OpenFDs
		a.lastThreads = row.Threads
		a.count++
		if row.OpenFDs > a.peakFD {
			a.peakFD = row.OpenFDs
		}
		if row.SocketFDs > a.peakSock {
			a.peakSock = row.SocketFDs
		}
		if row.Threads > a.peakThreads {
			a.peakThreads = row.Threads
		}
	}

	bpfByPID := map[uint32]dumpedPIDStats{}
	for _, s := range bpfStats {
		bpfByPID[s.PID] = s
	}

	var out []procSamplePeak
	for pid, a := range byPID {
		peak := procSamplePeak{
			PID:          pid,
			Name:         a.name,
			Role:         a.role,
			PeakOpenFDs:  a.peakFD,
			PeakSocketFD: a.peakSock,
			PeakThreads:  a.peakThreads,
			StartOpenFDs: a.firstFD,
			EndOpenFDs:   a.lastFD,
			FdDelta:      int64(a.lastFD) - int64(a.firstFD),
			ThreadDelta:  int64(a.lastThreads) - int64(a.firstThreads),
			SampleCount:  a.count,
		}
		if st, ok := bpfByPID[pid]; ok && durationSec > 0 {
			peak.FdOpenRate = float64(st.FdOpen) / durationSec
			peak.FdCloseRate = float64(st.FdClose) / durationSec
			peak.SocketOpen = st.SocketOpen
			peak.SocketAccept = st.SocketAccept
			peak.ThreadFork = st.ThreadFork
			peak.ThreadExit = st.ThreadExit
		}
		out = append(out, peak)
	}
	return out, nil
}

func peaksFromMemSnapshots(sessionDir string, bpfStats []dumpedPIDStats, durationSec float64) ([]procSamplePeak, error) {
	memStart := readMemSnap(filepath.Join(sessionDir, "mem-start.json"))
	memEnd := readMemSnap(filepath.Join(sessionDir, "mem-end.json"))

	var out []procSamplePeak
	for _, st := range bpfStats {
		peak := procSamplePeak{
			PID:          st.PID,
			Name:         st.Name,
			Role:         st.Role,
			SocketOpen:   st.SocketOpen,
			SocketAccept: st.SocketAccept,
			ThreadFork:   st.ThreadFork,
			ThreadExit:   st.ThreadExit,
		}
		if s, ok := memStart[st.PID]; ok {
			peak.StartOpenFDs = s.OpenFDs
			peak.PeakOpenFDs = s.OpenFDs
			peak.PeakSocketFD = s.SocketFDs
			peak.PeakThreads = s.Threads
		}
		if e, ok := memEnd[st.PID]; ok {
			peak.EndOpenFDs = e.OpenFDs
			if e.OpenFDs > peak.PeakOpenFDs {
				peak.PeakOpenFDs = e.OpenFDs
			}
			if e.SocketFDs > peak.PeakSocketFD {
				peak.PeakSocketFD = e.SocketFDs
			}
			if e.Threads > peak.PeakThreads {
				peak.PeakThreads = e.Threads
			}
		}
		peak.FdDelta = int64(peak.EndOpenFDs) - int64(peak.StartOpenFDs)
		if durationSec > 0 {
			peak.FdOpenRate = st.FdOpenPerSec
			peak.FdCloseRate = st.FdClosePerSec
		}
		out = append(out, peak)
	}
	return out, nil
}

func readMemSnap(path string) map[uint32]procMemSnapshot {
	out := map[uint32]procMemSnapshot{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var snap memSnapshot
	if json.Unmarshal(data, &snap) != nil {
		return out
	}
	for _, p := range snap.Processes {
		out[p.PID] = p
	}
	return out
}

func splitLines(data []byte) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		start := 0
		for i := 0; i < len(data); i++ {
			if data[i] != '\n' {
				continue
			}
			if i > start {
				ch <- string(data[start:i])
			}
			start = i + 1
		}
		if start < len(data) {
			ch <- string(data[start:])
		}
	}()
	return ch
}

// readProcFDAndThreads counts open FDs, socket FDs, and OS threads for a process.
func readProcFDAndThreads(pid uint32) (openFDs, socketFDs, threads uint64, err error) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return 0, 0, 0, err
	}
	openFDs = uint64(len(entries))
	for _, e := range entries {
		link, err := os.Readlink(filepath.Join(fdDir, e.Name()))
		if err != nil {
			continue
		}
		if len(link) >= 6 && link[:6] == "socket" {
			socketFDs++
		}
	}

	row, err := readProcMem(targetEntry{PID: pid})
	if err == nil {
		threads = row.Threads
	}
	return openFDs, socketFDs, threads, nil
}
