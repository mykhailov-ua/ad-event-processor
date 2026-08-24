package main

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
)

type statusBucket struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Count  int64  `json:"count"`
}

type statusHistogram struct {
	Generated string           `json:"generated"`
	ByStatus  map[string]int64 `json:"by_status"`
	Histogram []statusBucket   `json:"histogram"`
}

type histogram struct {
	mu       sync.Mutex
	byStatus map[string]int64
	buckets  map[string]int64
}

func newHistogram() *histogram {
	return &histogram{
		byStatus: make(map[string]int64),
		buckets:  make(map[string]int64),
	}
}

func (h *histogram) inc(status, errKind string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.byStatus[status]++
	h.buckets[status+"\x00"+errKind]++
}

func (h *histogram) write(path, generated string) error {
	h.mu.Lock()
	by := make(map[string]int64, len(h.byStatus))
	for k, v := range h.byStatus {
		by[k] = v
	}
	var buckets []statusBucket
	for k, v := range h.buckets {
		parts := splitKey(k)
		buckets = append(buckets, statusBucket{Status: parts[0], Error: parts[1], Count: v})
	}
	h.mu.Unlock()

	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Count > buckets[j].Count })
	out := statusHistogram{Generated: generated, ByStatus: by, Histogram: buckets}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	return enc.Encode(out)
}

func splitKey(k string) [2]string {
	for i := 0; i < len(k); i++ {
		if k[i] == 0 {
			return [2]string{k[:i], k[i+1:]}
		}
	}
	return [2]string{k, "none"}
}
