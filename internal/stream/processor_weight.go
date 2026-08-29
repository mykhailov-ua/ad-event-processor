package stream

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
)

const (
	defaultProcessorWeightFloor = 0.05
	defaultProcessorWeightCeil  = 0.95
)

type ProcessorWeightConfig struct {
	NodeID        string
	InstanceLabel string
	Floor         float64
	Ceil          float64
	EpochInterval time.Duration
	DrainPgWait   time.Duration
	WeightsURL    string
}

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

type ProcessorWeightController struct {
	cfg          ProcessorWeightConfig
	postgresGate *ProcessorPostgresGate
	udp          ProcessorUDPWeightsSource

	localWeight atomic.Uint64
	httpClient  *http.Client
}

type processorWeightsResponse struct {
	Epoch    int64                      `json:"epoch"`
	EpochLag int64                      `json:"epoch_lag"`
	Nodes    []processorWeightHTTPEntry `json:"node_weights"`
}

type processorWeightHTTPEntry struct {
	NodeID string  `json:"node_id"`
	Weight float64 `json:"weight"`
}

func NewProcessorWeightController(cfg ProcessorWeightConfig, postgresGate *ProcessorPostgresGate, udp ProcessorUDPWeightsSource) *ProcessorWeightController {
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
		cfg:          cfg,
		postgresGate: postgresGate,
		udp:          udp,
		httpClient:   &http.Client{Timeout: 3 * time.Second},
	}
	c.localWeight.Store(math.Float64bits(1.0))
	return c
}

func (c *ProcessorWeightController) Start(ctx context.Context) {
	if c == nil {
		return
	}
	go c.refreshLoop(ctx)
}

func (c *ProcessorWeightController) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.EpochInterval)
	defer ticker.Stop()
	c.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refresh(ctx)
		}
	}
}

func (c *ProcessorWeightController) refresh(ctx context.Context) {
	weight := c.LocalWeight()
	if weight <= 0 {
		weight = 1.0
	}
	if c.udp != nil {
		if w := lookupProcessorWeight(c.udp.NodeWeights(), c.cfg.NodeID); w > 0 {
			weight = w
		}
	} else if c.cfg.WeightsURL != "" {
		if w := c.pollHTTPWeights(ctx); w > 0 {
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

func (c *ProcessorWeightController) pollHTTPWeights(ctx context.Context) float64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.WeightsURL, http.NoBody)
	if err != nil {
		return 0
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return 0
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
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
	if c == nil || c.postgresGate == nil {
		return false
	}
	return c.postgresGate.WaitEMA() >= c.cfg.DrainPgWait
}

func (c *ProcessorWeightController) LocalWeight() float64 {
	if c == nil {
		return 1.0
	}
	return math.Float64frombits(c.localWeight.Load())
}

func (c *ProcessorWeightController) InstanceLabel() string {
	if c == nil {
		return "local"
	}
	return c.cfg.InstanceLabel
}

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
