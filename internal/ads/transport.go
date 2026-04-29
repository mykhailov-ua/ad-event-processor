package ads

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(cfg *config.Config, registry *Registry, proc *Processor, agg *Aggregator) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /track", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		status := http.StatusAccepted

		defer func() {
			duration := time.Since(start).Seconds()
			HttpRequestsTotal.WithLabelValues("POST", "/track", strconv.Itoa(status)).Inc()
			HttpRequestDuration.WithLabelValues("POST", "/track").Observe(duration)
		}()

		requestID := uuid.New().String()
		l := slog.With("request_id", requestID)

		var req struct {
			CampaignID uuid.UUID       `json:"campaign_id"`
			Type       string          `json:"type"`
			Payload    json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			l.Warn("invalid request body", "error", err)
			status = http.StatusBadRequest
			http.Error(w, "invalid request", status)
			return
		}

		if !registry.Exists(req.CampaignID) {
			l.Warn("campaign not found", "campaign_id", req.CampaignID)
			status = http.StatusNotFound
			http.Error(w, "campaign not found", status)
			return
		}

		err := proc.Process(Event{
			CampaignID: req.CampaignID,
			Type:       req.Type,
			Payload:    req.Payload,
			IP:         r.RemoteAddr,
			UA:         r.UserAgent(),
		})

		if err != nil {
			if errors.Is(err, ErrBufferFull) {
				l.Error("processor buffer full")
				status = http.StatusTooManyRequests
				http.Error(w, "server overloaded", status)
				return
			}
			l.Error("failed to process event", "error", err)
			status = http.StatusInternalServerError
			http.Error(w, "internal error", status)
			return
		}

		agg.Increment(req.CampaignID, req.Type)
		w.WriteHeader(status)
	})

	return mux
}
