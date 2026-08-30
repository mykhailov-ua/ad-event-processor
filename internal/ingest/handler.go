package ingest

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/gnet"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/ingest/pool"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	pkgnet "github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var (
	adEventPool = sync.Pool{
		New: func() any {
			return &pb.AdEvent{
				Metadata: &pb.EventMetadata{},
			}
		},
	}
	trackResponsePool = sync.Pool{
		New: func() any { return &pb.TrackResponse{} },
	}
	trackRequestPool = sync.Pool{
		New: func() any {
			return &TrackRequest{}
		},
	}
	bufferPool = sync.Pool{
		New: func() any { return new(bytes.Buffer) },
	}
	responseBytesPool = sync.Pool{
		New: func() any {
			s := make([]byte, 4096)
			return &s
		},
	}
	extraBufPool = sync.Pool{
		New: func() any {
			s := make([]byte, 0, 1024)
			return &s
		},
	}
	statusStrings          [600]string
	maxPoolObjectSize      = 64 * 1024
	contentTypeProtoHeader = []string{"application/x-protobuf"}
	contentTypeJSONHeader  = []string{"application/json"}
)

type Reactor = gnet.Reactor

type (
	ConnContext      = gnet.ConnContext
	PinnedWorkerPool = gnet.PinnedWorkerPool
	Request          = gnet.Request
)

var NewPinnedWorkerPool = gnet.NewPinnedWorkerPool

func ServeGnetHarness(h *AdsPacketHandler, inbound []byte) (pkgnet.Action, *gnet.GnetHarnessConn) {
	return gnet.ServeGnetHarness(h.Server, inbound)
}

func BuildGnetHTTP(method, path string, headers map[string]string, body []byte) []byte {
	return gnet.BuildGnetHTTP(method, path, headers, body)
}

func BuildGnetPostTrackJSON(body []byte) []byte { return gnet.BuildGnetPostTrackJSON(body) }

func PostTrackGnetJSON(h *AdsPacketHandler, body []byte) (int, []byte) {
	return gnet.PostTrackGnetJSON(h.Server, body)
}

func PostTrackGnet(h *AdsPacketHandler, body []byte, contentType, accept string) (int, []byte) {
	return gnet.PostTrackGnet(h.Server, body, contentType, accept)
}

func PostTrackGnetJSONWait(h *AdsPacketHandler, body []byte, timeout time.Duration) (int, []byte) {
	return gnet.PostTrackGnetJSONWait(h.Server, body, timeout)
}

func GetHealthGnet(h *AdsPacketHandler) (int, string) { return gnet.GetHealthGnet(h.Server) }

func GetReadyGnet(h *AdsPacketHandler) (int, string) { return gnet.GetReadyGnet(h.Server) }

func ParseGnetHTTPStatus(resp []byte) int { return gnet.ParseGnetHTTPStatus(resp) }

func ParseGnetHTTPBody(resp []byte) []byte { return gnet.ParseGnetHTTPBody(resp) }

func init() {
	for i := range 600 {
		statusStrings[i] = strconv.Itoa(i)
	}
}

func putBuffer(buf *bytes.Buffer) {
	if buf == nil || buf.Cap() > maxPoolObjectSize {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

func putAdEvent(evt *pb.AdEvent) {
	if evt == nil {
		return
	}
	if evt.Metadata != nil && (len(evt.Metadata.ExtraKeys) > 100 || cap(evt.Metadata.ExtraBytes) > 4096) {
		evt.Reset()
		adEventPool.Put(evt)
		return
	}
	evt.CampaignId = evt.CampaignId[:0]
	evt.EventType = evt.EventType[:0]
	if evt.Metadata != nil {
		evt.Metadata.ClickId = evt.Metadata.ClickId[:0]
		evt.Metadata.UserId = evt.Metadata.UserId[:0]
		evt.Metadata.DeviceType = evt.Metadata.DeviceType[:0]
		evt.Metadata.Os = evt.Metadata.Os[:0]
		for i := range evt.Metadata.ExtraKeys {
			evt.Metadata.ExtraKeys[i] = evt.Metadata.ExtraKeys[i][:0]
		}
		evt.Metadata.ExtraKeys = evt.Metadata.ExtraKeys[:0]
		for i := range evt.Metadata.ExtraValues {
			evt.Metadata.ExtraValues[i] = evt.Metadata.ExtraValues[i][:0]
		}
		evt.Metadata.ExtraValues = evt.Metadata.ExtraValues[:0]
		evt.Metadata.ExtraBytes = evt.Metadata.ExtraBytes[:0]
	}
	adEventPool.Put(evt)
}

func putTrackResponse(resp *pb.TrackResponse) {
	if resp == nil {
		return
	}
	resp.Reset()
	trackResponsePool.Put(resp)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

func NewRouter(cfg *config.Config, registry domain.CampaignRegistry, filterEngine *FilterEngine, pool Pinger, redisShards []redis.UniversalClient, sharder Sharder, fraudStream string, creativeStore *BrandCreativeStore, streamProducers []*StreamProducer, brokerProducers *BrokerProducerSet) http.Handler {
	mux := http.NewServeMux()
	trackCORS := newTrackCORS(cfg.TrackCORSOrigins)

	trackDurationObserver := metrics.HTTPRequestDuration.WithLabelValues("POST", "/track")
	var trackStatusCounters [600]prometheus.Counter
	for i := range 600 {
		trackStatusCounters[i] = metrics.HTTPRequestsTotal.WithLabelValues("POST", "/track", statusStrings[i])
	}

	trackLatencyRing := NewLatencyRing(defaultLatencyRingCap)
	fraudWriter := NewFraudStreamWriter(redisShards, fraudStream, int64(cfg.StreamMaxLen))
	trackProc := newTrackProcessor(filterEngine, registry, creativeStore)

	mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackLatencyRing.FlushTo(trackDurationObserver)
		promhttp.Handler().ServeHTTP(w, r)
	}))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.Error("health check failed: postgres", "error", err)
			http.Error(w, "postgres unreachable", http.StatusServiceUnavailable)
			return
		}

		if !pingConnectedRedisShards(ctx, redisShards) {
			http.Error(w, "redis shard unreachable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.HandleFunc("OPTIONS /track", func(w http.ResponseWriter, r *http.Request) {
		serveHTTPTrackCORSPreflight(w, r, trackCORS)
	})

	registerHTTPTrackPixel(mux)
	registerHTTPTrackClientStatic(mux)

	mux.HandleFunc("POST /track", func(w http.ResponseWriter, r *http.Request) {
		telemetry.RecordTrack()
		startMono := monotonicNano()
		status := http.StatusAccepted
		applyHTTPTrackCORSHeaders(w, r.Header.Get("Origin"), trackCORS)

		defer func() {
			if status >= 0 && status < 600 {
				trackStatusCounters[status].Inc()
			} else {
				metrics.HTTPRequestsTotal.WithLabelValues("POST", "/track", strconv.Itoa(status)).Inc()
			}
			trackLatencyRing.RecordMono(startMono)
		}()

		if r.ContentLength > cfg.MaxRequestBodySize {
			metrics.HTTPParseErrors.WithLabelValues("payload_too_large").Inc()
			status = http.StatusBadRequest
			http.Error(w, "invalid body", status)
			return
		}
		if r.ContentLength < 0 {
			r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestBodySize)
		}

		id := NewFastUUID()
		wReqID := bufPool.Get().(*BufWrapper)
		wReqID.Buf = wReqID.Buf[:0]
		wReqID.Buf = appendUUID(wReqID.Buf, id)
		defer bufPool.Put(wReqID)

		var campaignID uuid.UUID
		var eventType string
		var userID string
		var payload []byte

		ip := extractClientIP(r, cfg.TrustedProxies)
		var clickID string
		var requestIDStr string
		var ortbSlot *openRTBScratchSlot
		var jsonSerFlags uint8
		var telemetrySet uint8
		var telemetryEvents []domain.BehaviorTelemetryEvent

		contentType := ""
		if ctSlice := r.Header["Content-Type"]; len(ctSlice) > 0 {
			contentType = ctSlice[0]
		}
		adEventProcessorNative := cfg.IsAdEventProcessorNativeIngress()
		if adEventProcessorNative && (contentType == "application/x-protobuf" || contentType == "") {
			buf := bufferPool.Get().(*bytes.Buffer)
			defer putBuffer(buf)

			if _, err := io.Copy(buf, r.Body); err != nil {
				metrics.HTTPParseErrors.WithLabelValues("read_body").Inc()
				status = http.StatusBadRequest
				http.Error(w, "invalid body", status)
				return
			}

			pbReq := adEventPool.Get().(*pb.AdEvent)
			defer putAdEvent(pbReq)

			if err := unmarshalAdEventVT(pbReq, buf.Bytes()); err != nil {
				metrics.HTTPParseErrors.WithLabelValues("invalid_proto").Inc()
				status = http.StatusBadRequest
				http.Error(w, "invalid protobuf", status)
				return
			}

			var cid uuid.UUID
			if !ParseUUID(pbReq.CampaignId, &cid) {
				metrics.HTTPParseErrors.WithLabelValues("invalid_campaign_id").Inc()
				status = http.StatusBadRequest
				http.Error(w, "invalid campaign_id", status)
				return
			}
			campaignID = cid
			eventType = unsafeString(pbReq.EventType)
			if pbReq.Metadata != nil {
				userID = unsafeString(pbReq.Metadata.UserId)
				if len(pbReq.Metadata.ClickId) > 0 {
					clickID = unsafeString(pbReq.Metadata.ClickId)
				}
				if len(pbReq.Metadata.ExtraBytes) > 0 {
					payload = pbReq.Metadata.ExtraBytes
				} else if len(pbReq.Metadata.ExtraKeys) > 0 {
					bufPtr := extraBufPool.Get().(*[]byte)
					*bufPtr = marshalExtra(*bufPtr, pbReq.Metadata.ExtraKeys, pbReq.Metadata.ExtraValues)
					payload = *bufPtr
					defer extraBufPool.Put(bufPtr)
				}
			}
		} else {
			buf := bufferPool.Get().(*bytes.Buffer)
			defer putBuffer(buf)

			if _, err := io.Copy(buf, r.Body); err != nil {
				metrics.HTTPParseErrors.WithLabelValues("read_body").Inc()
				status = http.StatusBadRequest
				http.Error(w, "invalid body", status)
				return
			}

			req := trackRequestPool.Get().(*TrackRequest)
			req.Reset()
			defer trackRequestPool.Put(req)

			var err error
			if !adEventProcessorNative {
				err = ParseOpenRTB3Ingress(req, buf.Bytes())
			} else {
				err = ParseTrackRequestJSONOpt(req, buf.Bytes())
				if err == nil {
					jsonSerFlags = scanTrackJSONSerialization(buf.Bytes())
				}
			}
			if err != nil {
				metrics.HTTPParseErrors.WithLabelValues("invalid_json").Inc()
				status = http.StatusBadRequest
				http.Error(w, "invalid json", status)
				return
			}
			campaignID = req.CampaignID
			userID = req.UserID
			eventType = req.Type
			payload = appendAttributionPayload(nil, req.Payload, req.subs, req.fbclid, req.gclid, req.ttclid, req.msclkid, req.tblci, req.obClickID, req.eventID, req.txID)
			if req.ClickID != "" {
				clickID = req.ClickID
			}
			ortbSlot = req.ortbSlot
			telemetrySet = req.TelemetrySet
			if len(req.TelemetryEvents) > 0 {
				telemetryEvents = append(telemetryEvents[:0], req.TelemetryEvents...)
			}
			req.ortbSlot = nil
		}

		if clickID == "" {
			requestIDStr = unsafeString(wReqID.Buf)
			clickID = requestIDStr
		}

		evt := domain.EventPool.Get().(*domain.Event)
		evt.Reset()
		evt.ClickID = clickID
		evt.CampaignID = campaignID
		evt.UserID = userID
		evt.Type = eventType
		evt.Payload = append(evt.Payload[:0], payload...)
		evt.IP = ip
		ua := ""
		if uaSlice := r.Header["User-Agent"]; len(uaSlice) > 0 {
			ua = uaSlice[0]
		}
		evt.UA = ua
		evt.JSONSerializationFlags = jsonSerFlags
		if telemetrySet != 0 {
			evt.TelemetrySet = 1
			if len(telemetryEvents) > 0 {
				evt.TelemetryEvents = append(evt.TelemetryEvents[:0], telemetryEvents...)
			}
		}
		if cfg != nil && cfg.MobileBiometricsEnabled {
			applyMobileBiometricSummary(evt)
		}

		if ortbSlot != nil {
			attachOpenRTB3Scratch(evt, ortbSlot)
		}

		var landing string
		if filterEngine != nil {
			lease, kind, acquired := tryAcquireStreamAdmissionForFilter(cfg, sharder, streamProducers, brokerProducers, campaignID, filterEngine)
			if !acquired {
				spec := filterRejectSpecs[kind]
				recordHTTPFilterReject(kind, evt)
				w.Header().Set("Retry-After", "1")
				http.Error(w, spec.body, spec.status)
				status = spec.status
				return
			}
			leaseHeld := true
			releaseLease := func() {
				if leaseHeld {
					lease.Release()
					leaseHeld = false
				}
			}

			outcome := processTrack(r.Context(), trackProc, evt, nil)
			switch outcome.Status {
			case trackStatusFraudAccepted:
				recordHTTPFilterReject(outcome.RejectKind, evt)
				shard := sharder.GetShard(evt.CampaignID)
				enqueueFraudReject(fraudWriter, shard, evt)
				domain.EventPool.Put(evt)
				accept := ""
				if accSlice := r.Header["Accept"]; len(accSlice) > 0 {
					accept = accSlice[0]
				}
				writeHTTPTrackAccepted(w, wReqID, requestIDStr, accept, "")
				releaseLease()
				return
			case trackStatusRejected:
				spec := filterRejectSpecs[outcome.RejectKind]
				recordHTTPFilterReject(outcome.RejectKind, evt)
				if outcome.RejectKind == filterRejectFraudBlocked {
					shard := sharder.GetShard(evt.CampaignID)
					enqueueFraudReject(fraudWriter, shard, evt)
				}
				domain.EventPool.Put(evt)
				if outcome.RejectKind == filterRejectConsent {
					w.WriteHeader(http.StatusNoContent)
					releaseLease()
					return
				}
				if outcome.RejectKind == filterRejectInfra {
					w.Header().Set("Retry-After", "1")
				}
				if outcome.RejectKind == filterRejectRateLimit || outcome.RejectKind == filterRejectPacing {
					w.Header().Set("Retry-After", "60")
				}
				http.Error(w, spec.body, spec.status)
				releaseLease()
				return
			case trackStatusInternalError:
				domain.EventPool.Put(evt)
				http.Error(w, "internal error", http.StatusInternalServerError)
				releaseLease()
				return
			case trackStatusAccepted:
				landing = outcome.LandingURL
				if !publishAcceptedTrackIngress(sharder, streamProducers, brokerProducers, filterEngine, evt, &lease) {
					status = httpTrackRejectProducerOverload(r.Context(), w, filterEngine, evt, registry)
					releaseLease()
					return
				}
				releaseLease()
			}
		} else {
			releaseOpenRTB3Scratch(evt)
			landing = ResolveLandingURL(r.Context(), registry, creativeStore, evt)
		}
		domain.EventPool.Put(evt)

		accept := ""
		if accSlice := r.Header["Accept"]; len(accSlice) > 0 {
			accept = accSlice[0]
		}
		writeHTTPTrackAccepted(w, wReqID, requestIDStr, accept, landing)
	})

	return mux
}

func isTrustedProxy(ipStr string, trustedProxies []string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return false
	}
	for _, p := range trustedProxies {
		if p == "" {
			continue
		}
		if p == ipStr {
			return true
		}
		if _, ipNet, err := net.ParseCIDR(p); err == nil {
			if ipNet.Contains(parsedIP) {
				return true
			}
		}
	}
	return false
}

func getIPOnly(addr string) string {
	if idx := strings.LastIndexByte(addr, ':'); idx != -1 {
		if idx > 0 && addr[idx-1] == ']' {
			if addr[0] == '[' {
				return addr[1 : idx-1]
			}
		}
		return addr[:idx]
	}
	return addr
}

func extractClientIP(r *http.Request, trustedProxies []string) string {
	remoteIP := getIPOnly(r.RemoteAddr)
	if !isTrustedProxy(remoteIP, trustedProxies) {
		return remoteIP
	}

	var xff string
	if xffSlice := r.Header["X-Forwarded-For"]; len(xffSlice) > 0 {
		xff = xffSlice[0]
	}
	if xff != "" {
		last := len(xff)
		for i := len(xff) - 1; i >= -1; i-- {
			if i == -1 || xff[i] == ',' {
				start := i + 1
				for start < last && xff[start] == ' ' {
					start++
				}
				end := last
				for end > start && xff[end-1] == ' ' {
					end--
				}

				if start < end {
					ipStr := xff[start:end]
					parsedIP := net.ParseIP(ipStr)
					if parsedIP != nil && !parsedIP.IsPrivate() && !parsedIP.IsLoopback() && !parsedIP.IsLinkLocalUnicast() {
						return ipStr
					}
				}
				last = i
			}
		}
	}

	if xriSlice := r.Header["X-Real-Ip"]; len(xriSlice) > 0 {
		xri := xriSlice[0]
		ipStr := strings.TrimSpace(xri)
		parsedIP := net.ParseIP(ipStr)
		if parsedIP != nil && !parsedIP.IsPrivate() && !parsedIP.IsLoopback() && !parsedIP.IsLinkLocalUnicast() {
			return ipStr
		}
	}

	return remoteIP
}

var (
	respInvalidProto         = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: 16\r\nConnection: keep-alive\r\n\r\ninvalid protobuf")
	respInvalidCampaign      = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: 19\r\nConnection: keep-alive\r\n\r\ninvalid campaign_id")
	respInvalidJSON          = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: 12\r\nConnection: keep-alive\r\n\r\ninvalid json")
	respEmergencyBreaker     = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nContent-Length: 32\r\nConnection: keep-alive\r\n\r\nservice temporarily unavailable")
	respWorkerPoolOverload   = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\nserver overloaded")
	respProducerOverload     = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 18\r\nConnection: keep-alive\r\n\r\nproducer overloaded")
	respInfraUnavailable     = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 19\r\nConnection: keep-alive\r\n\r\nservice unavailable")
	respRateLimit            = []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: text/plain\r\nRetry-After: 60\r\nContent-Length: 19\r\nConnection: keep-alive\r\n\r\nrate limit exceeded")
	respDuplicate            = []byte("HTTP/1.1 409 Conflict\r\nContent-Type: text/plain\r\nContent-Length: 15\r\nConnection: close\r\n\r\nduplicate event")
	respBudget               = []byte("HTTP/1.1 402 Payment Required\r\nContent-Type: text/plain\r\nContent-Length: 16\r\nConnection: keep-alive\r\n\r\nbudget exhausted")
	respPacing               = []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: text/plain\r\nRetry-After: 60\r\nContent-Length: 20\r\nConnection: keep-alive\r\n\r\npacing limit reached")
	respFreq                 = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 23\r\nConnection: keep-alive\r\n\r\nfrequency limit reached")
	respGeo                  = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 21\r\nConnection: keep-alive\r\n\r\ngeo-targeting blocked")
	respSchedule             = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 26\r\nConnection: keep-alive\r\n\r\noutside delivery schedule")
	respCampaignNotFound     = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\ncampaign not found")
	respBidFloorNotMet       = []byte("HTTP/1.1 402 Payment Required\r\nContent-Type: text/plain\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\nbid floor not met")
	respFilterTimeout        = []byte("HTTP/1.1 504 Gateway Timeout\r\nContent-Type: text/plain\r\nContent-Length: 15\r\nConnection: keep-alive\r\n\r\nfilter timeout")
	respConsentDenied        = []byte("HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	respClickSafePage        = []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n" + branding.HTTPSafePageHeader + ": 1\r\nConnection: keep-alive\r\n\r\n")
	respInternalError        = []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nContent-Length: 14\r\nConnection: keep-alive\r\n\r\ninternal error")
	respBadRequestClose      = []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	respNotFound             = []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	respMethodNotAllowed     = []byte("HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	respPayloadTooLarge      = []byte("HTTP/1.1 413 Payload Too Large\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	respLicenseExpired       = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 15\r\nConnection: keep-alive\r\n\r\nlicense expired")
	respDailyQuotaExceeded   = []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: text/plain\r\nRetry-After: 60\r\nContent-Length: 21\r\nConnection: keep-alive\r\n\r\ndaily quota exceeded")
	respPlacementBlocked     = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\nplacement blocked")
	respFraudBlocked         = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 13\r\nConnection: keep-alive\r\n\r\nfraud blocked")
	respSegmentExcluded      = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 16\r\nConnection: keep-alive\r\n\r\nsegment excluded")
	respSegmentNotIncluded   = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 21\r\nConnection: keep-alive\r\n\r\nsegment not included")
	respLinkSigForbidden     = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 19\r\nConnection: keep-alive\r\n\r\ninvalid link signature")
	respReviewTrafficBlocked = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\nContent-Length: 22\r\nConnection: keep-alive\r\n\r\nreview traffic blocked")
	respRegistryStale        = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 14\r\nConnection: keep-alive\r\n\r\nregistry_stale")
	respShardUnavailable     = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\nshard_unavailable")
	respHealthzOK            = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 2\r\nConnection: keep-alive\r\n\r\nOK")
	respReadyzOK             = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 2\r\nConnection: keep-alive\r\n\r\nOK")
	respReadyz503            = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nContent-Length: 9\r\nConnection: keep-alive\r\n\r\nnot ready")
	respClickBadRequest      = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: 20\r\nConnection: keep-alive\r\n\r\ninvalid click query")
	respClickBadLanding      = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: 21\r\nConnection: keep-alive\r\n\r\ninvalid landing url")
	respClickNoLanding       = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 15\r\nConnection: keep-alive\r\n\r\nno landing url")
)

type AdsPacketHandler struct {
	*gnet.Server
	filterEngine            *FilterEngine
	registry                domain.CampaignRegistry
	creativeStore           *BrandCreativeStore
	cfg                     *config.Config
	pool                    Pinger
	redisShards             []redis.UniversalClient
	sharder                 Sharder
	fraudStream             string
	trackDurationObserver   prometheus.Observer
	trackStatusCounters     [600]prometheus.Counter
	trackMetrics            PreboundTrackMetrics
	trackLatencyRing        *LatencyRing
	healthy                 atomic.Int32
	healthzHits             atomic.Uint64
	startedAtNano           atomic.Int64
	redisShardsHealthy      []atomic.Int32
	logger                  *logger.Logger
	auditLogSeq             atomic.Uint64
	auditLogSampleMask      uint64
	fraudWriter             *FraudStreamWriter
	trackProc               trackProcessor
	udpControl              *UDPControl
	brokerProducers         *BrokerProducerSet
	streamProducers         []*StreamProducer
	trackCORS               trackCORS
	cidrTable               *CIDRTable
	cidrMetrics             cidrBlockMetrics
	ipv6RotationTable       *IPv6RotationTable
	ipv6RotationMetrics     l1IPv6RotationMetrics
	ipv4RotationTable       *IPv4RotationTable
	ipv4RotationMetrics     l1IPv4RotationMetrics
	mobileCarrierASN        *MobileCarrierASNTable
	tlsFingerprintTable     *TLSFingerprintTable
	tlsFingerprintMetrics   tlsFingerprintMetrics
	proxyVPNTable           *ProxyVPNTable
	proxyVPNBlockMetrics    proxyVPNBlockMetrics
	moderatorIPTable        *ModeratorIPTable
	moderatorMetrics        moderatorIntelMetrics
	attestationKeys         []attestationHMACKey
	attestationInnerScratch [linkHMACBlockSize + attestationPayloadLen]byte
	linkSigningSecret       []byte
	linkHMACIpad            [linkHMACBlockSize]byte
	linkHMACOpad            [linkHMACBlockSize]byte
	linkSignInnerScratch    [linkSignInnerScratchLen]byte
	domainPoolTable         *pool.Table
	campaignFlowTable       *CampaignFlowTable
	clickProxyClient        *http.Client
}

func (h *AdsPacketHandler) ConfigureCIDR(table *CIDRTable) {
	if h == nil {
		return
	}
	h.cidrTable = table
	h.cidrMetrics = newCIDRBlockMetrics()
}

func (h *AdsPacketHandler) ConfigureIPv6Rotation(table *IPv6RotationTable) {
	if h == nil {
		return
	}
	h.ipv6RotationTable = table
	h.ipv6RotationMetrics = newL1IPv6RotationMetrics()
}

func (h *AdsPacketHandler) ConfigureIPv4Rotation(table *IPv4RotationTable) {
	if h == nil {
		return
	}
	h.ipv4RotationTable = table
	h.ipv4RotationMetrics = newL1IPv4RotationMetrics()
}

func (h *AdsPacketHandler) ConfigureMobileCarrierASN(table *MobileCarrierASNTable) {
	if h != nil {
		h.mobileCarrierASN = table
	}
}

func (h *AdsPacketHandler) ConfigureTLSFingerprint(table *TLSFingerprintTable) {
	if h == nil {
		return
	}
	h.tlsFingerprintTable = table
	h.tlsFingerprintMetrics = newTLSFingerprintMetrics()
}

func (h *AdsPacketHandler) ConfigureLinkSigning(secret []byte) {
	if h == nil {
		return
	}
	h.linkSigningSecret = append([]byte(nil), secret...)
	if len(h.linkSigningSecret) == 0 {
		return
	}
	linkInitHMACPads(h.linkSigningSecret, &h.linkHMACIpad, &h.linkHMACOpad)
}

func (h *AdsPacketHandler) ConfigureProxyVPN(table *ProxyVPNTable) {
	if h == nil {
		return
	}
	h.proxyVPNTable = table
	h.proxyVPNBlockMetrics = newProxyVPNBlockMetrics()
}

func (h *AdsPacketHandler) ConfigureModeratorIntel(table *ModeratorIPTable) {
	if h == nil {
		return
	}
	h.moderatorIPTable = table
	h.moderatorMetrics = newModeratorIntelMetrics()
}

func (h *AdsPacketHandler) ConfigureDomainPool(table *pool.Table) {
	if h == nil {
		return
	}
	h.domainPoolTable = table
}

func (h *AdsPacketHandler) ConfigureCampaignFlow(table *CampaignFlowTable) {
	if h == nil {
		return
	}
	h.campaignFlowTable = table
}

func (h *AdsPacketHandler) SetPool(p Pinger) {
	if h != nil {
		h.pool = p
	}
}

func (h *AdsPacketHandler) SetBrokerProducer(bp *BrokerProducer) {
	if h == nil {
		return
	}
	if bp == nil {
		h.SetBrokerProducers(nil)
		return
	}
	h.SetBrokerProducers(NewBrokerProducerSet([]*BrokerProducer{bp}))
}

func (h *AdsPacketHandler) SetBrokerProducers(set *BrokerProducerSet) {
	if h == nil {
		return
	}
	h.brokerProducers = set
	if set != nil && set.Len() > 0 && h.filterEngine != nil {
		h.filterEngine.SetDeferStreamToProducer(true)
	}
}

func (h *AdsPacketHandler) SetStreamProducers(producers []*StreamProducer) {
	if h == nil {
		return
	}
	h.streamProducers = producers
	if h.filterEngine != nil && len(producers) > 0 {
		h.filterEngine.SetDeferStreamToProducer(true)
	}
}

func (h *AdsPacketHandler) publishAcceptedTrack(evt *domain.Event, lease *streamAdmissionLease) bool {
	if h == nil || evt == nil {
		return true
	}
	return publishAcceptedTrackIngress(h.sharder, h.streamProducers, h.brokerProducers, h.filterEngine, evt, lease)
}

func (h *AdsPacketHandler) SetUDPControl(ctrl *UDPControl) {
	if h != nil {
		h.udpControl = ctrl
	}
}

func (h *AdsPacketHandler) SetLogger(l *logger.Logger) {
	h.logger = l
	if h.Server != nil {
		h.Server.SetLogger(l)
	}
}

func (h *AdsPacketHandler) SetWorkerPool(wp *PinnedWorkerPool) {
	if h.Server != nil {
		h.Server.SetWorkerPool(wp)
	}
}

func (h *AdsPacketHandler) write(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	h.Server.Write(c, data, ctx)
}

func (h *AdsPacketHandler) writeClose(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	h.Server.WriteClose(c, data, ctx)
}

func (h *AdsPacketHandler) writeFilterReject(c pkgnet.Conn, data []byte, ctx *ConnContext) {
	h.Server.WriteFilterReject(c, data, ctx, bytes.Equal(data, respDuplicate))
}

func (h *AdsPacketHandler) writeMaybeClose(c pkgnet.Conn, data []byte, ctx *ConnContext, closeAfter bool) {
	if closeAfter {
		h.writeClose(c, data, ctx)
		return
	}
	h.write(c, data, ctx)
}

func NewAdsPacketHandler(cfg *config.Config, registry domain.CampaignRegistry, filterEngine *FilterEngine, pool Pinger, redisShards []redis.UniversalClient, sharder Sharder, fraudStream string, creativeStore *BrandCreativeStore) *AdsPacketHandler {
	trackDurationObserver := metrics.HTTPRequestDuration.WithLabelValues("POST", "/track")
	var trackStatusCounters [600]prometheus.Counter
	for i := range 600 {
		trackStatusCounters[i] = metrics.HTTPRequestsTotal.WithLabelValues("POST", "/track", statusStrings[i])
	}

	h := &AdsPacketHandler{
		filterEngine:          filterEngine,
		registry:              registry,
		creativeStore:         creativeStore,
		cfg:                   cfg,
		pool:                  pool,
		redisShards:           redisShards,
		sharder:               sharder,
		fraudStream:           fraudStream,
		fraudWriter:           NewFraudStreamWriter(redisShards, fraudStream, int64(cfg.StreamMaxLen)),
		trackProc:             newTrackProcessor(filterEngine, registry, creativeStore),
		trackDurationObserver: trackDurationObserver,
		trackStatusCounters:   trackStatusCounters,
		trackMetrics:          NewPreboundTrackMetrics(),
		trackLatencyRing:      NewLatencyRing(defaultLatencyRingCap),
		auditLogSampleMask:    auditLogSampleMaskFromConfig(cfg.AuditLogSampleMask),
		trackCORS:             newTrackCORS(cfg.TrackCORSOrigins),
	}
	h.Server = gnet.NewServer(gnet.ServerConfig{
		Cfg:    cfg,
		Logger: nil,
		RecordTrackStatus: func(status int) {
			h.recordTrackStatus(status)
		},
		MonoElapsed: monoElapsedSeconds,
	})
	h.Server.SetReactor(h)
	h.startedAtNano.Store(time.Now().UnixNano())
	configureOrtbScanLimits(cfg)
	configureJSONParseSecurity(cfg)
	configureProtoMaxFields(cfg)
	if n := len(redisShards); n > 0 {
		h.redisShardsHealthy = make([]atomic.Int32, n)
		for i := range h.redisShardsHealthy {
			h.redisShardsHealthy[i].Store(1)
		}
	}

	configureOpenRTBExchangeLimiter(cfg)

	return h
}

func (h *AdsPacketHandler) recordTrackStatus(status int) {
	if h == nil {
		return
	}
	if status >= 0 && status < len(h.trackStatusCounters) {
		h.trackStatusCounters[status].Inc()
		return
	}
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "/track", strconv.Itoa(status)).Inc()
}

func (h *AdsPacketHandler) recordMetrics(startMono int64, status int) {
	h.recordTrackStatus(status)
	h.trackLatencyRing.RecordMono(startMono)
}

func (h *AdsPacketHandler) FlushLatency() {
	if h.trackLatencyRing != nil {
		h.trackLatencyRing.FlushTo(h.trackDurationObserver)
	}
}

func (h *AdsPacketHandler) HealthzHits() uint64 {
	if h == nil {
		return 0
	}
	return h.healthzHits.Load()
}

func (h *AdsPacketHandler) Ready() bool {
	return h != nil && h.healthy.Load() == 1
}

func (h *AdsPacketHandler) Uptime() time.Duration {
	if h == nil || h.startedAtNano.Load() == 0 {
		return 0
	}
	return time.Since(time.Unix(0, h.startedAtNano.Load()))
}

func (h *AdsPacketHandler) WarmupElapsed() bool {
	if h == nil {
		return false
	}
	sec := 300
	if h.cfg != nil && h.cfg.NodeWarmupSec > 0 {
		sec = h.cfg.NodeWarmupSec
	}
	return h.Uptime() >= time.Duration(sec)*time.Second
}

func (h *AdsPacketHandler) WarmReady() bool {
	return h.Ready() && h.WarmupElapsed()
}

func (h *AdsPacketHandler) SetStartedAt(t time.Time) {
	if h != nil {
		h.startedAtNano.Store(t.UnixNano())
	}
}

func (h *AdsPacketHandler) SetHealthProbeState(healthy bool, shardOK ...bool) {
	if healthy {
		h.healthy.Store(1)
	} else {
		h.healthy.Store(0)
	}
	for i, ok := range shardOK {
		if i >= len(h.redisShardsHealthy) {
			break
		}
		if ok {
			h.redisShardsHealthy[i].Store(1)
		} else {
			h.redisShardsHealthy[i].Store(0)
		}
	}
}

func (h *AdsPacketHandler) writeGnetTrackAccepted(ctx *ConnContext, accept string, origin string, c pkgnet.Conn, startMono int64, wReqID *BufWrapper, requestIDStr, landingURL string) {
	if requestIDStr == "" {
		requestIDStr = unsafeString(wReqID.Buf)
	}

	if accept == "application/x-protobuf" {
		resp := &ctx.Resp
		resp.Reset()
		resp.RequestId = requestIDStr
		resp.Status = "accepted"

		respSize := resp.SizeVT()
		headerBudget := gnetTrackAcceptedHeaderBudget(origin, h.trackCORS, respSize, true)
		bufSlice := ctx.BufSlice
		if cap(bufSlice) < headerBudget+respSize {
			bufSlice = make([]byte, headerBudget+respSize)
			ctx.BufSlice = bufSlice
		} else {
			bufSlice = bufSlice[:headerBudget+respSize]
		}

		offset := copy(bufSlice, "HTTP/1.1 202 Accepted\r\n")
		offset = len(appendTrackCORSHeaders(bufSlice[:offset], origin, h.trackCORS))
		offset += copy(bufSlice[offset:], "Content-Type: application/x-protobuf\r\nContent-Length: ")
		var contentLenScratch [20]byte
		clen := appendInt(contentLenScratch[:], int64(respSize))
		offset += copy(bufSlice[offset:], contentLenScratch[:clen])
		offset += copy(bufSlice[offset:], "\r\nConnection: keep-alive\r\n\r\n")

		n, err := resp.MarshalToVT(bufSlice[offset : offset+respSize])
		if err != nil {
			h.write(c, respInternalError, ctx)
			h.recordMetrics(startMono, http.StatusInternalServerError)
			return
		}
		outSlice := bufSlice[:offset+n]
		metrics.GnetBytesSent.Add(float64(len(outSlice)))
		metrics.GnetPacketsSent.Inc()
		h.write(c, outSlice, ctx)
	} else {
		reqID := wReqID.Buf
		if requestIDStr != "" {
			reqID = UnsafeBytes(requestIDStr)
		}

		const jsonPrefix = `{"request_id":"`
		const jsonMid = `","status":"accepted"`
		respSize := len(jsonPrefix) + len(reqID) + len(jsonMid) + 1
		if landingURL != "" {
			const jsonLand = `,"landing_url":"`
			respSize += len(jsonLand) + len(landingURL) + 1
		}

		bufSlice := ctx.BufSlice
		headerBudget := gnetTrackAcceptedHeaderBudget(origin, h.trackCORS, respSize, false)
		if cap(bufSlice) < headerBudget+respSize {
			bufSlice = make([]byte, headerBudget+respSize)
			ctx.BufSlice = bufSlice
		} else {
			bufSlice = bufSlice[:headerBudget+respSize]
		}

		offset := copy(bufSlice, "HTTP/1.1 202 Accepted\r\n")
		offset = len(appendTrackCORSHeaders(bufSlice[:offset], origin, h.trackCORS))
		offset += copy(bufSlice[offset:], "Content-Type: application/json\r\nContent-Length: ")
		var contentLenScratch [20]byte
		clen := appendInt(contentLenScratch[:], int64(respSize))
		offset += copy(bufSlice[offset:], contentLenScratch[:clen])
		offset += copy(bufSlice[offset:], "\r\nConnection: keep-alive\r\n\r\n")
		offset += copy(bufSlice[offset:], jsonPrefix)
		offset += copy(bufSlice[offset:], reqID)
		offset += copy(bufSlice[offset:], jsonMid)
		if landingURL != "" {
			offset += copy(bufSlice[offset:], `,"landing_url":"`)
			offset += copy(bufSlice[offset:], landingURL)
			bufSlice[offset] = '"'
			offset++
		}
		bufSlice[offset] = '}'
		offset++

		metrics.GnetBytesSent.Add(float64(offset))
		metrics.GnetPacketsSent.Inc()
		h.write(c, bufSlice[:offset], ctx)
	}

	h.recordMetrics(startMono, http.StatusAccepted)
}

func writeHTTPTrackAccepted(w http.ResponseWriter, wReqID *BufWrapper, requestIDStr string, accept string, landingURL string) {
	if requestIDStr == "" {
		requestIDStr = unsafeString(wReqID.Buf)
	}
	if accept == "application/x-protobuf" {
		resp := trackResponsePool.Get().(*pb.TrackResponse)
		defer putTrackResponse(resp)
		resp.RequestId = requestIDStr
		resp.Status = "accepted"

		respSize := resp.SizeVT()
		bufSlicePtr := responseBytesPool.Get().(*[]byte)
		bufSlice := *bufSlicePtr
		if cap(bufSlice) < respSize {
			bufSlice = make([]byte, respSize)
		} else {
			bufSlice = bufSlice[:respSize]
		}

		n, err := resp.MarshalToVT(bufSlice)
		if err != nil {
			*bufSlicePtr = bufSlice
			responseBytesPool.Put(bufSlicePtr)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := bufSlice[:n]
		w.Header()["Content-Type"] = contentTypeProtoHeader
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(out)
		*bufSlicePtr = bufSlice
		responseBytesPool.Put(bufSlicePtr)
		return
	}

	w.Header()["Content-Type"] = contentTypeJSONHeader
	w.WriteHeader(http.StatusAccepted)
	buf := bufferPool.Get().(*bytes.Buffer)
	defer putBuffer(buf)
	buf.WriteString(`{"request_id":"`)
	buf.Write(wReqID.Buf)
	buf.WriteString(`","status":"accepted"`)
	if landingURL != "" {
		buf.WriteString(`,"landing_url":"`)
		buf.WriteString(landingURL)
		buf.WriteByte('"')
	}
	buf.WriteByte('}')
	_, _ = w.Write(buf.Bytes())
}

func (h *AdsPacketHandler) FraudWriter() *FraudStreamWriter {
	if h == nil {
		return nil
	}
	return h.fraudWriter
}

func (h *AdsPacketHandler) Stop(ctx context.Context) error {
	if h.fraudWriter != nil {
		h.fraudWriter.Stop()
	}
	for _, p := range h.streamProducers {
		if p != nil {
			p.Close()
		}
	}
	if h.Server != nil {
		return h.Server.Stop(ctx)
	}
	return nil
}

func (h *AdsPacketHandler) StartHealthProbe(ctx context.Context) {
	h.healthy.Store(1)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				ok := true
				if h.pool != nil {
					if err := h.pool.Ping(probeCtx); err != nil {
						ok = false
						slog.Error("health probe: postgres unreachable", "error", err)
					}
				}
				for i, redisClient := range h.redisShards {
					if err := redisClient.Ping(probeCtx).Err(); err != nil {
						ok = false
						if i < len(h.redisShardsHealthy) {
							h.redisShardsHealthy[i].Store(0)
						}
						slog.Error("health probe: redis shard unreachable", "shard", i, "error", err)
					} else if i < len(h.redisShardsHealthy) {
						h.redisShardsHealthy[i].Store(1)
					}
				}
				cancel()
				if ok {
					h.healthy.Store(1)
				} else {
					h.healthy.Store(0)
				}
				shardStates := make([]int32, len(h.redisShardsHealthy))
				for i := range h.redisShardsHealthy {
					shardStates[i] = h.redisShardsHealthy[i].Load()
				}
				exportHealthProbeMetrics(ok, shardStates)
			}
		}
	}()
}

func (h *AdsPacketHandler) React(req Request, c pkgnet.Conn) pkgnet.Action {
	ctx, ok := c.Context().(*ConnContext)
	if !ok {
		ctx = h.Server.AllocConnContext(c)
		c.SetContext(ctx)
	}

	if len(req.Method) == 7 && req.Method[0] == 'O' && req.Method[1] == 'P' && req.Method[2] == 'T' &&
		req.Method[3] == 'I' && req.Method[4] == 'O' && req.Method[5] == 'N' && req.Method[6] == 'S' {
		if bytesEqual(req.Path, "/track") {
			return h.reactTrackOPTIONS(req, c, ctx)
		}
		h.write(c, respMethodNotAllowed, ctx)
		return pkgnet.None
	}

	if len(req.Method) == 3 && req.Method[0] == 'G' && req.Method[1] == 'E' && req.Method[2] == 'T' {
		if bytesEqual(req.Path, "/healthz") || bytesEqual(req.Path, "/health") {
			h.healthzHits.Add(1)
			h.write(c, respHealthzOK, ctx)
			return pkgnet.None
		}
		if bytesEqual(req.Path, "/readyz") || bytesEqual(req.Path, "/ready") {
			if h.WarmReady() {
				h.write(c, respReadyzOK, ctx)
			} else {
				h.write(c, respReadyz503, ctx)
			}
			return pkgnet.None
		}
		if bytesEqual(req.Path, "/metrics") {
			h.write(c, respNotFound, ctx)
			return pkgnet.None
		}
		if httpPathHasPrefix(req.Path, safePageStubPathPrefix) {
			return h.reactSafePageStub(req, c, ctx)
		}
		if resp, ok := trackClientStaticGnetResponse(req.Path); ok {
			h.write(c, resp, ctx)
			return pkgnet.None
		}
		if httpPathHasPrefix(req.Path, "/click") {
			return h.reactClickRedirect(req, c, ctx)
		}
		if httpPathHasPrefix(req.Path, telegramPathClick) {
			return h.reactTelegramClick(req, c, ctx)
		}
		if httpPathHasPrefix(req.Path, telegramPathImpression) {
			return h.reactTelegramImpression(req, c, ctx)
		}
		h.write(c, respMethodNotAllowed, ctx)
		return pkgnet.None
	}

	isPOST := len(req.Method) == 4 && req.Method[0] == 'P' && req.Method[1] == 'O' && req.Method[2] == 'S' && req.Method[3] == 'T'
	if !isPOST {
		h.write(c, respMethodNotAllowed, ctx)
		return pkgnet.None
	}

	if !bytesEqual(req.Path, "/track") {
		if bytesEqual(req.Path, safePageVerifyPath) {
			if !req.HasContentLength {
				h.writeClose(c, respBadRequestClose, ctx)
				return pkgnet.Close
			}
			return h.reactTrackVerify(req, c, ctx)
		}
		if bytesEqual(req.Path, "/openrtb/bid") {
			if !req.HasContentLength {
				h.writeClose(c, respBadRequestClose, ctx)
				return pkgnet.Close
			}
			return h.reactOpenRTBBid(req, c, ctx)
		}
		if bytesEqual(req.Path, "/tg/bid") {
			if !req.HasContentLength {
				h.writeClose(c, respBadRequestClose, ctx)
				return pkgnet.Close
			}
			return h.reactTelegramBid(req, c, ctx)
		}
		h.write(c, respNotFound, ctx)
		return pkgnet.None
	}

	if !req.HasContentLength {
		h.writeClose(c, respBadRequestClose, ctx)
		return pkgnet.Close
	}

	startMono := monotonicNano()
	telemetry.RecordTrack()

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	ua := unsafeString(req.UserAgent)

	id := NewFastUUID()

	wReqID := &ctx.WReqID
	wReqID.Buf = wReqID.Buf[:0]
	wReqID.Buf = appendUUID(wReqID.Buf, id)

	fields, badResp, status, ok := h.parseTrackIngest(ctx, req)
	if !ok {
		h.write(c, badResp, ctx)
		h.recordMetrics(startMono, status)
		return pkgnet.None
	}

	if matched, kind := h.tlsFingerprintShouldSafeView(req.TLSJA3, req.TLSJA4, fields.campaignID, ua); matched {
		h.writeGnetSafeViewTLS(c, ctx, startMono, kind)
		return pkgnet.None
	}

	if matched, connType := h.proxyVPNBlockShouldSafeView(ip, fields.campaignID); matched {
		h.writeGnetSafeViewProxyVPN(c, ctx, startMono, connType)
		return pkgnet.None
	}

	var requestIDStr string
	if fields.clickID == "" {
		requestIDStr = unsafeString(wReqID.Buf)
		fields.clickID = requestIDStr
	}

	evt := &ctx.Evt
	fillTrackEventWithMobileBiometrics(evt, fields, ip, ua, h.cfg != nil && h.cfg.MobileBiometricsEnabled)
	if ctx.WorkerID >= 0 {
		if w := ctx.WorkerID; w >= 0 && w <= 127 {
			evt.FilterWorkerIdx = int8(w)
		}
	}
	evt.TLSHash = unsafeString(req.TLSHash)
	evt.TLSJA3 = unsafeString(req.TLSJA3)
	evt.TLSJA4 = unsafeString(req.TLSJA4)
	evt.SecCHUA = unsafeString(req.SecCHUA)
	evt.AcceptLang = unsafeString(req.AcceptLang)
	fillIngressH2(evt, ctx != nil && ctx.ProtoH2)
	fillWireMetadataFromRequest(evt, &req)
	if req.TCPMSSSet != 0 {
		evt.TCPMSS = req.TCPMSS
		evt.TCPMSSSet = 1
	}
	if req.TCPTTLSet != 0 {
		evt.TCPTTL = req.TCPTTL
		evt.TCPTTLSet = 1
	}
	if req.TCPWindowSet != 0 {
		evt.TCPWindow = req.TCPWindow
		evt.TCPWindowSet = 1
	}
	if req.TCPSigSet != 0 {
		evt.TCPSig = req.TCPSig
		evt.TCPSigSet = 1
	}
	fillConnTimingFromRequest(evt, &req)

	if h.udpControl != nil {
		shard := h.sharder.GetShard(evt.CampaignID)
		workerID := ctx.WorkerID
		if !h.udpControl.TryIngress(shard, workerID) {
			h.writeFilterReject(c, respRateLimit, ctx)
			h.recordMetrics(startMono, http.StatusTooManyRequests)
			h.recordTrackReject(ctx, evt, filterRejectRateLimit)
			return pkgnet.None
		}
	}

	if h.filterEngine != nil {
		lease, kind, acquired := h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.recordTrackReject(ctx, evt, kind)
			return pkgnet.None
		}

		outcome := processTrack(context.Background(), h.trackProc, evt, fields.deviceType)
		act := h.deliverGnetTrack(ctx, string(req.Accept), string(req.Origin), c, evt, startMono, wReqID, requestIDStr, outcome, &lease)
		lease.Release()
		return act
	}

	releaseOpenRTB3Scratch(evt)
	lease, kind, acquired := h.tryAcquireStreamAdmission(evt.CampaignID)
	if !acquired {
		spec := filterRejectSpecs[kind]
		h.writeFilterReject(c, spec.gnetResp, ctx)
		h.recordMetrics(startMono, spec.status)
		h.trackMetrics.recordFilterReject(kind)
		return pkgnet.None
	}
	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	if !h.publishAcceptedTrack(evt, &lease) {
		spec := filterRejectSpecs[filterRejectProducerOverload]
		h.writeFilterReject(c, spec.gnetResp, ctx)
		h.recordMetrics(startMono, spec.status)
		h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
		lease.Release()
		return pkgnet.None
	}
	landing := ResolveLandingURL(context.Background(), h.registry, h.creativeStore, &ctx.Evt)
	h.writeGnetTrackAccepted(ctx, string(req.Accept), string(req.Origin), c, startMono, wReqID, requestIDStr, landing)
	lease.Release()
	return pkgnet.None
}

func extractClientIPGnet(ctx *ConnContext, req *Request, c pkgnet.Conn, trustedProxies []string) string {
	if ctx.RemoteIP == "" {
		ctx.RemoteIP = getIPOnly(c.RemoteAddr().String())
	}
	remoteIP := ctx.RemoteIP
	if !isTrustedProxy(remoteIP, trustedProxies) {
		return remoteIP
	}

	if len(req.ClientIP) > 0 {
		xff := unsafeString(req.ClientIP)
		last := len(xff)
		for i := len(xff) - 1; i >= -1; i-- {
			if i == -1 || xff[i] == ',' {
				start := i + 1
				for start < last && xff[start] == ' ' {
					start++
				}
				end := last
				for end > start && xff[end-1] == ' ' {
					end--
				}

				if start < end {
					ipStr := xff[start:end]
					parsedIP := net.ParseIP(ipStr)
					if parsedIP != nil && !parsedIP.IsPrivate() && !parsedIP.IsLoopback() && !parsedIP.IsLinkLocalUnicast() {
						return ipStr
					}
				}
				last = i
			}
		}
	}

	return remoteIP
}
