package opsadmin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type HTTPHandlers struct {
	OpsReader               ManagementOpsReader
	PaymentIntents          domain.PaymentAPI
	ConsentRecorder         ConsentRecorder
	ConsentVerifier         ConsentVerifier
	AuditLister             AuditLister
	RolesReloader           RolesReloader
	Blacklist               BlacklistAdmin
	Shard0Catchup           Shard0CatchupRunner
	FraudThreat             FraudThreatEnqueuer
	DoctorSnapshot          DoctorSnapshotBuilder
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError       func(http.ResponseWriter, error)
	AuthorizeCustomerAccess func(*http.Request, string) error
	SupportBundle           SupportBundleWriter
	RUMStore                RUMStore
	FraudPresets            FraudPresetsService
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.OpsReader == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/ops/incidents", limit(perm("shards:read", h.getIncidents)))
	mux.HandleFunc("GET /api/v1/ops/health/snapshot", limit(perm("shards:read", h.getStackHealthSnapshot)))
	mux.HandleFunc("GET /api/v1/ops/outbox", limit(perm("shards:read", h.listOutbox)))
	mux.HandleFunc("GET /api/v1/ops/dlq", limit(perm("shards:read", h.listDLQ)))
	mux.HandleFunc("GET /api/v1/ops/dlq/inbox", limit(perm("shards:read", h.listDLQInbox)))
	mux.HandleFunc("POST /api/v1/ops/dlq/{id}/retry", limit(perm("shards:write", h.retryDLQ)))
	mux.HandleFunc("POST /api/v1/ops/dlq/inbox/{id}/retry", limit(perm("shards:write", h.retryDLQInbox)))
	mux.HandleFunc("GET /api/v1/ops/shards", limit(perm("shards:read", h.getShards)))
	mux.HandleFunc("POST /api/v1/ops/shards/0/catchup", limit(perm("shards:write", h.postShard0Catchup)))
	mux.HandleFunc("GET /api/v1/audit/export", limit(perm("audit:read", h.exportAudit)))
	mux.HandleFunc("GET /api/v1/customers/{id}/payments", limit(perm("customers:read", h.listCustomerPayments)))
	h.registerReconRoutes(mux)
	h.registerConsentRoutes(mux)
	h.registerAuditRoutes(mux)
	h.registerRolesRoutes(mux)
	h.registerBlacklistRoutes(mux)
	h.RegisterFraudThreatRoutes(mux)
	h.registerDashboardRoutes(mux)
	h.registerHomeRoutes(mux)
	h.registerSupportBundleRoutes(mux)
	h.registerRUMRoutes(mux)
	h.registerMLModelRoutes(mux)
	h.RegisterFraudPresetOpsRoutes(mux, limit, perm)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}

func parsePaginationLimit(r *http.Request) int32 {
	limit, _ := coldpath.ParseAPIPagination(r)
	return limit
}

func stringsHasPrefixJSON(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return len(ct) >= 16 && ct[:16] == "application/json"
}

func ParseDLQShardFromRoute(dlqID string) int {
	return parseDLQShardFromRoute(dlqID)
}

func parseDLQShardFromRoute(dlqID string) int {
	const prefix = "shard-"
	if len(dlqID) < len(prefix)+2 {
		return 0
	}
	if dlqID[:6] != prefix {
		return 0
	}
	rest := dlqID[6:]
	for i, ch := range rest {
		if ch == '-' {
			n, err := strconv.Atoi(rest[:i])
			if err == nil {
				return n
			}
			break
		}
	}
	return 0
}

func ParseDLQEntryIDFromRoute(dlqID string) string {
	return parseDLQEntryIDFromRoute(dlqID)
}

func parseDLQEntryIDFromRoute(dlqID string) string {
	const prefix = "shard-"
	if !strings.HasPrefix(dlqID, prefix) {
		return ""
	}
	rest := dlqID[len(prefix):]
	dash := strings.Index(rest, "-")
	if dash < 0 || dash+1 >= len(rest) {
		return ""
	}
	return rest[dash+1:]
}

const (
	defaultFanOutMaxConcurrency = 8
	defaultFanOutPerSourceTO    = 2 * time.Second
)

type FanOutSourceError struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

type FanOutResult[T any] struct {
	Items      []T                 `json:"items"`
	Partial    bool                `json:"partial"`
	Errors     []FanOutSourceError `json:"errors,omitempty"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type FanOutSource[T any] struct {
	ID   string
	Poll func(ctx context.Context) ([]T, error)
}

type FanOutCollector struct {
	MaxConcurrency int
	PerSourceTO    time.Duration
	Route          string
}

func NewFanOutCollector(cfg *config.Config, route string) *FanOutCollector {
	maxConcurrency := defaultFanOutMaxConcurrency
	if cfg != nil && cfg.Management.AdminFanoutMaxConcurrency > 0 {
		maxConcurrency = cfg.Management.AdminFanoutMaxConcurrency
	}
	return &FanOutCollector{
		MaxConcurrency: maxConcurrency,
		PerSourceTO:    defaultFanOutPerSourceTO,
		Route:          route,
	}
}

type fanOutResultSlot[T any] struct {
	sourceID string
	items    []T
	err      error
}

func CollectFanOut[T any](ctx context.Context, c *FanOutCollector, sources []FanOutSource[T]) FanOutResult[T] {
	start := time.Now()
	defer func() {
		if c != nil && c.Route != "" {
			metrics.AdminFanoutLatencySeconds.WithLabelValues(c.Route).Observe(time.Since(start).Seconds())
		}
	}()

	if len(sources) == 0 {
		return FanOutResult[T]{Items: []T{}}
	}
	if c == nil {
		c = NewFanOutCollector(nil, "")
	}

	sem := make(chan struct{}, c.MaxConcurrency)
	slots := make([]fanOutResultSlot[T], len(sources))
	var wg sync.WaitGroup

	for i, src := range sources {
		wg.Add(1)
		go func(idx int, source FanOutSource[T]) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			srcCtx, cancel := context.WithTimeout(ctx, c.PerSourceTO)
			defer cancel()

			items, err := source.Poll(srcCtx)
			slots[idx] = fanOutResultSlot[T]{sourceID: source.ID, items: items, err: err}
		}(i, src)
	}
	wg.Wait()

	var (
		out    FanOutResult[T]
		ok     int
		failed int
	)
	for _, slot := range slots {
		if slot.err != nil {
			failed++
			code := "SOURCE_UNAVAILABLE"
			if errors.Is(slot.err, context.DeadlineExceeded) || errors.Is(slot.err, context.Canceled) {
				code = "TIMEOUT"
			}
			out.Errors = append(out.Errors, FanOutSourceError{Source: slot.sourceID, Code: code})
			continue
		}
		ok++
		if len(slot.items) > 0 {
			out.Items = append(out.Items, slot.items...)
		}
	}

	if failed > 0 && ok > 0 {
		out.Partial = true
	}
	if c.Route != "" {
		metrics.AdminFanoutSourcesTotal.WithLabelValues(c.Route).Add(float64(len(sources)))
		if out.Partial {
			metrics.AdminFanoutPartialTotal.WithLabelValues(c.Route).Inc()
		}
	}
	return out
}

type fanOutCursorState struct {
	Sources map[string]string `json:"sources"`
}

func EncodeFanOutCursor(state map[string]string) (string, error) {
	if len(state) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(fanOutCursorState{Sources: state})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodeFanOutCursor(raw string) (map[string]string, error) {
	if raw == "" {
		return map[string]string{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	var state fanOutCursorState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	if state.Sources == nil {
		state.Sources = map[string]string{}
	}
	return state.Sources, nil
}
