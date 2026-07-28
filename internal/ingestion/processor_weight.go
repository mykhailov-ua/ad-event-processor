package ingestion

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"espx/internal/config"
	"espx/internal/metrics"
)

const (
	defaultProcessorWeightFloor = 0.05
	defaultProcessorWeightCeil  = 0.95
)

// ProcessorWeightConfig tunes per-instance stream consume cadence from published weights.
type ProcessorWeightConfig struct {
	NodeID        string
	InstanceLabel string
	Floor         float64
	Ceil          float64
	EpochInterval time.Duration
	DrainPgWait   time.Duration
	WeightsURL    string
}

// ProcessorWeightConfigFromApp builds controller settings from service config.
func ProcessorWeightConfigFromApp(cfg *config.Config) ProcessorWeightConfig {
	nodeID := cfg.NodeID
	if nodeID == "" {
		nodeID, _ = os.Hostname()
	}
	if nodeID == "" {
		nodeID = "processor"
	}
	weightsURL := ""
	if cfg.ManagementURL != "" {
		weightsURL = strings.TrimRight(cfg.ManagementURL, "/") + "/ops/processor-weights"
	}
	return ProcessorWeightConfig{
		NodeID:        nodeID,
		InstanceLabel: nodeID,
		Floor:         cfg.ProcessorWeightFloor,
		Ceil:          cfg.ProcessorWeightCeil,
		EpochInterval: time.Duration(cfg.UDPSyncIntervalMs) * time.Millisecond,
		DrainPgWait:   time.Duration(cfg.ProcessorWeightDrainPgWaitMs) * time.Millisecond,
		WeightsURL:    weightsURL,
	}
}

// ProcessorWeightController applies regional capacity weights to stream read cadence.
type ProcessorWeightController struct {
	cfg    ProcessorWeightConfig
	pgGate *ProcessorPgGate
	udp    *UDPControl

	localWeight atomic.Uint64
	httpClient  *http.Client
}

// processorWeightsResponse mirrors GET /ops/processor-weights JSON.
type processorWeightsResponse struct {
	Epoch    int64                      `json:"epoch"`
	EpochLag int64                      `json:"epoch_lag"`
	Nodes    []processorWeightHTTPEntry `json:"node_weights"`
}

type processorWeightHTTPEntry struct {
	NodeID string  `json:"node_id"`
	Weight float64 `json:"weight"`
}

// NewProcessorWeightController builds a weight snapshot for one processor instance.
func NewProcessorWeightController(cfg ProcessorWeightConfig, pgGate *ProcessorPgGate, udp *UDPControl) *ProcessorWeightController {
	if cfg.Floor <= 0 {
		cfg.Floor = defaultProcessorWeightFloor
	}
	if cfg.Ceil <= 0 || cfg.Ceil > 1 {
		cfg.Ceil = defaultProcessorWeightCeil
	}
	if cfg.EpochInterval <= 0 {
		cfg.EpochInterval = 10 * time.Second
	}
	if cfg.DrainPgWait <= 0 {
		cfg.DrainPgWait = 50 * time.Millisecond
	}
	if cfg.InstanceLabel == "" {
		cfg.InstanceLabel = cfg.NodeID
	}
	c := &ProcessorWeightController{
		cfg:        cfg,
		pgGate:     pgGate,
		udp:        udp,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
	c.localWeight.Store(math.Float64bits(1.0))
	return c
}

// Start refreshes published weight on a fixed epoch interval.
func (c *ProcessorWeightController) Start(ctx context.Context) {
	if c == nil {
		return
	}
	go c.refreshLoop(ctx)
}

func (c *ProcessorWeightController) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.EpochInterval)
	defer ticker.Stop()
	c.refresh()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refresh()
		}
	}
}

func (c *ProcessorWeightController) refresh() {
	weight := c.LocalWeight()
	if weight <= 0 {
		weight = 1.0
	}
	if c.udp != nil {
		if w := lookupProcessorWeight(c.udp.NodeWeights(), c.cfg.NodeID); w > 0 {
			weight = w
		}
	} else if c.cfg.WeightsURL != "" {
		if w := c.pollHTTPWeights(); w > 0 {
			weight = w
		}
	}
	weight = clampProcessorWeight(weight, c.cfg.Floor, c.cfg.Ceil)
	if c.pgDrainActive() {
		weight = c.cfg.Floor
	}
	c.localWeight.Store(math.Float64bits(weight))
	metrics.ProcessorWeight.WithLabelValues(c.cfg.InstanceLabel).Set(weight)
}

func (c *ProcessorWeightController) pollHTTPWeights() float64 {
	req, err := http.NewRequest(http.MethodGet, c.cfg.WeightsURL, nil)
	if err != nil {
		return 0
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var doc processorWeightsResponse
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return 0
	}
	return lookupProcessorWeightFromHTTP(doc.Nodes, c.cfg.NodeID)
}

func lookupProcessorWeightFromHTTP(nodes []processorWeightHTTPEntry, nodeID string) float64 {
	for _, n := range nodes {
		if n.NodeID == nodeID && n.Weight > 0 {
			return n.Weight
		}
	}
	return 0
}

func lookupProcessorWeight(weights []UDPNodeWeight, nodeID string) float64 {
	for _, w := range weights {
		if w.NodeID == nodeID && w.Weight > 0 {
			return w.Weight
		}
	}
	return 0
}

func (c *ProcessorWeightController) pgDrainActive() bool {
	if c == nil || c.pgGate == nil {
		return false
	}
	return c.pgGate.WaitEMA() >= c.cfg.DrainPgWait
}

// LocalWeight returns the active consume weight in [floor, ceil].
func (c *ProcessorWeightController) LocalWeight() float64 {
	if c == nil {
		return 1.0
	}
	return math.Float64frombits(c.localWeight.Load())
}

// InstanceLabel returns the Prometheus instance label for this controller.
func (c *ProcessorWeightController) InstanceLabel() string {
	if c == nil {
		return "local"
	}
	return c.cfg.InstanceLabel
}

// EffectiveReadCount scales XREADGROUP batch size by local weight (minimum 1).
func (c *ProcessorWeightController) EffectiveReadCount(batchSize int) int64 {
	if c == nil {
		return int64(batchSize)
	}
	w := c.LocalWeight()
	if w >= 0.99 {
		return int64(batchSize)
	}
	n := int(float64(batchSize) * w)
	if n < 1 {
		n = 1
	}
	return int64(n)
}

// ThrottleBeforeRead sleeps proportionally when weight is below 1 to reduce read cadence.
func (c *ProcessorWeightController) ThrottleBeforeRead(ctx context.Context) {
	if c == nil {
		return
	}
	w := c.LocalWeight()
	if w >= 0.99 {
		return
	}
	base := c.cfg.EpochInterval / 20
	if base < 10*time.Millisecond {
		base = 10 * time.Millisecond
	}
	delay := time.Duration(float64(base) * (1.0/w - 1.0))
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
	}
}

// SetWeightForTest overrides local weight (tests only).
func (c *ProcessorWeightController) SetWeightForTest(w float64) {
	if c == nil {
		return
	}
	w = clampProcessorWeight(w, c.cfg.Floor, c.cfg.Ceil)
	c.localWeight.Store(math.Float64bits(w))
	metrics.ProcessorWeight.WithLabelValues(c.cfg.InstanceLabel).Set(w)
}

func clampProcessorWeight(w, floor, ceil float64) float64 {
	if w < floor {
		return floor
	}
	if w > ceil {
		return ceil
	}
	return w
}
