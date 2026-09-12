package crowdwave

import "time"

const (
	DefaultWaveWindowSec           = 3600
	DefaultWaveMinUniqueClusters   = 10
	DefaultWaveMinSimhashNeighbors = 8
	DefaultWaveSimhashMaxDist      = 2
	DefaultWaveRingSize            = 256
)

type WavePolicy struct {
	WindowSec           int64
	MinUniqueClusters   int
	MinSimhashNeighbors int
	SimhashMaxDist      int
}

func DefaultWavePolicy() WavePolicy {
	return WavePolicy{
		WindowSec:           DefaultWaveWindowSec,
		MinUniqueClusters:   DefaultWaveMinUniqueClusters,
		MinSimhashNeighbors: DefaultWaveMinSimhashNeighbors,
		SimhashMaxDist:      DefaultWaveSimhashMaxDist,
	}
}

type WaveEntry struct {
	ClusterID [16]byte
	Simhash   uint64
	Observed  int64
}

type WaveState struct {
	UniqueClusters   int
	SimhashNeighbors int
	Active           bool
	Score            uint16
	EntryCount       int
}

func EvaluateWave(entries []WaveEntry, nowUnix int64, policy WavePolicy) WaveState {
	if policy.WindowSec <= 0 {
		policy.WindowSec = DefaultWaveWindowSec
	}
	if policy.MinUniqueClusters <= 0 {
		policy.MinUniqueClusters = DefaultWaveMinUniqueClusters
	}
	if policy.MinSimhashNeighbors <= 0 {
		policy.MinSimhashNeighbors = DefaultWaveMinSimhashNeighbors
	}
	if policy.SimhashMaxDist <= 0 {
		policy.SimhashMaxDist = DefaultWaveSimhashMaxDist
	}
	cutoff := nowUnix - policy.WindowSec
	active := make([]WaveEntry, 0, len(entries))
	for _, e := range entries {
		if e.Observed >= cutoff {
			active = append(active, e)
		}
	}
	uniqClusters := countUniqueClusters(active)
	neighbors := countSimhashNeighbors(active, policy.SimhashMaxDist)
	score := waveScore(uniqClusters, neighbors)
	state := WaveState{
		UniqueClusters:   uniqClusters,
		SimhashNeighbors: neighbors,
		EntryCount:       len(active),
		Score:            score,
	}
	state.Active = uniqClusters >= policy.MinUniqueClusters &&
		neighbors >= policy.MinSimhashNeighbors
	return state
}

func countUniqueClusters(entries []WaveEntry) int {
	if len(entries) == 0 {
		return 0
	}
	seen := make(map[[16]byte]struct{}, len(entries))
	for _, e := range entries {
		if e.ClusterID[0] == 0 {
			continue
		}
		seen[e.ClusterID] = struct{}{}
	}
	return len(seen)
}

func countSimhashNeighbors(entries []WaveEntry, maxDist int) int {
	if len(entries) < 2 {
		return 0
	}
	neighbors := 0
	for i := range len(entries) {
		if entries[i].Simhash == 0 {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Simhash == 0 {
				continue
			}
			if HammingDistance(entries[i].Simhash, entries[j].Simhash) <= maxDist {
				neighbors++
			}
		}
	}
	return neighbors
}

func waveScore(uniqClusters, neighbors int) uint16 {
	score := uniqClusters*10 + neighbors
	if score > 65535 {
		return 65535
	}
	return uint16(score)
}

func NowUnix() int64 {
	return time.Now().Unix()
}
