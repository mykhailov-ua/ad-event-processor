package ads

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/pb"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/proto"
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

		var campaignID uuid.UUID
		var eventType string
		var payload []byte

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/x-protobuf" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				l.Warn("failed to read body", "error", err)
				status = http.StatusBadRequest
				http.Error(w, "invalid body", status)
				return
			}
			var pbReq pb.AdEvent
			if err := proto.Unmarshal(body, &pbReq); err != nil {
				l.Warn("invalid protobuf body", "error", err)
				status = http.StatusBadRequest
				http.Error(w, "invalid protobuf", status)
				return
			}
			
			cid, err := uuid.Parse(pbReq.CampaignId)
			if err != nil {
				l.Warn("invalid campaign id in proto", "error", err)
				status = http.StatusBadRequest
				http.Error(w, "invalid campaign_id", status)
				return
			}
			campaignID = cid
			eventType = pbReq.EventType
			// Marshal metadata to JSON for storage in JSONB column
			if pbReq.Metadata != nil {
				payload, _ = json.Marshal(pbReq.Metadata)
			}
		} else {
			// Fallback to JSON
			var req struct {
				CampaignID uuid.UUID       `json:"campaign_id"`
				Type       string          `json:"type"`
				Payload    json.RawMessage `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				l.Warn("invalid json body", "error", err)
				status = http.StatusBadRequest
				http.Error(w, "invalid json", status)
				return
			}
			campaignID = req.CampaignID
			eventType = req.Type
			payload = req.Payload
		}

		if !registry.Exists(campaignID) {
			l.Warn("campaign not found", "campaign_id", campaignID)
			status = http.StatusNotFound
			http.Error(w, "campaign not found", status)
			return
		}

		err := proc.Process(Event{
			CampaignID: campaignID,
			Type:       eventType,
			Payload:    payload,
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

		agg.Increment(campaignID, eventType)

		if r.Header.Get("Accept") == "application/x-protobuf" {
			resp := &pb.TrackResponse{
				RequestId: requestID,
				Status:    "accepted",
			}
			out, _ := proto.Marshal(resp)
			w.Header().Set("Content-Type", "application/x-protobuf")
			w.WriteHeader(status)
			_, _ = w.Write(out)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"request_id": requestID,
				"status":     "accepted",
			})
		}
	})

	return mux
}
