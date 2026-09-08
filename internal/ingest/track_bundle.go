package ingest

import (
	"context"
	"net/http"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	trackStatusAccepted      = track.StatusAccepted
	trackStatusFraudAccepted = track.StatusFraudAccepted
	trackStatusRejected      = track.StatusRejected
	trackStatusInternalError = track.StatusInternalError
)

type trackOutcome = track.Outcome

type trackCORS = track.CORS

func newTrackCORS(allowed []string) trackCORS { return track.NewCORS(allowed) }

func appendTrackCORSHeaders(dst []byte, origin string, cors trackCORS) []byte {
	return track.AppendCORSHeaders(dst, origin, cors)
}

func buildTrackCORSPreflight(origin string, cors trackCORS) []byte {
	return track.BuildCORSPreflight(origin, cors)
}

func gnetTrackAcceptedHeaderBudget(origin string, cors trackCORS, bodyLen int, protobuf bool) int {
	return track.GnetTrackAcceptedHeaderBudget(origin, cors, bodyLen, protobuf)
}

func applyHTTPTrackCORSHeaders(w http.ResponseWriter, origin string, cors trackCORS) {
	track.ApplyHTTPCORSHeaders(w, origin, cors)
}

func serveHTTPTrackCORSPreflight(w http.ResponseWriter, r *http.Request, cors trackCORS) {
	track.ServeHTTPCORSPreflight(w, r, cors)
}

const (
	trackPixelPath     = track.TrackPixelPath
	trackTelemetryPath = track.TrackTelemetryPath
)

func serveHTTPTrackPixel(w http.ResponseWriter) {
	track.ServeHTTPTrackPixel(w)
}

func serveHTTPTrackClientJS(w http.ResponseWriter, js []byte) {
	track.ServeHTTPTrackClientJS(w, js)
}

var (
	trackTelemetryJS  = track.TrackTelemetryJS
	trackBiometricsJS = track.TrackBiometricsJS
)

func registerHTTPTrackPixel(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET "+trackPixelPath, func(w http.ResponseWriter, _ *http.Request) {
		track.ServeHTTPTrackPixel(w)
	})
}

func registerHTTPTrackClientStatic(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET "+track.TrackTelemetryPath, func(w http.ResponseWriter, _ *http.Request) {
		track.ServeHTTPTrackClientJS(w, track.TrackTelemetryJS)
	})
	mux.HandleFunc("GET "+track.TrackBiometricsPath, func(w http.ResponseWriter, _ *http.Request) {
		track.ServeHTTPTrackClientJS(w, track.TrackBiometricsJS)
	})
}

func isTrackPixelPath(path []byte) bool {
	return track.IsTrackClientStaticPath(path) && bytesEqual(path, trackPixelPath)
}

func trackClientStaticGnetResponse(path []byte) ([]byte, bool) {
	return track.TrackClientStaticGnetResponse(path)
}

func (h *AdsPacketHandler) reactTrackOPTIONS(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	resp := buildTrackCORSPreflight(unsafeString(req.Origin), h.trackCORS)
	if resp == nil {
		h.write(c, respMethodNotAllowed, ctx)
		return gnet.None
	}
	h.write(c, resp, ctx)
	h.recordMetrics(monotonicNano(), http.StatusNoContent)
	return gnet.None
}

type trackIngestFields struct {
	campaignID             uuid.UUID
	eventType              string
	userID                 string
	payload                []byte
	clickID                string
	placementID            string
	deviceType             []byte
	subs                   SubIDSlots
	fbclid                 string
	gclid                  string
	ttclid                 string
	msclkid                string
	tblci                  string
	obClickID              string
	eventID                string
	txID                   string
	ortbSlot               *openRTBScratchSlot
	jsonSerializationFlags uint8
	telemetrySet           uint8
	telemetryEvents        []domain.BehaviorTelemetryEvent
}

func (h *AdsPacketHandler) parseTrackIngest(
	ctx *ConnContext,
	req *Request,
) (fields trackIngestFields, badResp []byte, httpStatus int, ok bool) {
	contentType := unsafeString(req.ContentType)
	adEventProcessorNative := h.cfg == nil || h.cfg.IsAdEventProcessorNativeIngress()
	if adEventProcessorNative && (contentType == "application/x-protobuf" || contentType == "") {
		h.trackMetrics.throughputProto.Inc()
		pbReq := &ctx.PBReq
		pbReq.CampaignId = pbReq.CampaignId[:0]
		pbReq.EventType = pbReq.EventType[:0]
		if pbReq.Metadata != nil {
			pbReq.Metadata.ClickId = pbReq.Metadata.ClickId[:0]
			pbReq.Metadata.UserId = pbReq.Metadata.UserId[:0]
			pbReq.Metadata.DeviceType = pbReq.Metadata.DeviceType[:0]
			pbReq.Metadata.Os = pbReq.Metadata.Os[:0]
			for i := range pbReq.Metadata.ExtraKeys {
				pbReq.Metadata.ExtraKeys[i] = pbReq.Metadata.ExtraKeys[i][:0]
			}
			pbReq.Metadata.ExtraKeys = pbReq.Metadata.ExtraKeys[:0]
			for i := range pbReq.Metadata.ExtraValues {
				pbReq.Metadata.ExtraValues[i] = pbReq.Metadata.ExtraValues[i][:0]
			}
			pbReq.Metadata.ExtraValues = pbReq.Metadata.ExtraValues[:0]
			pbReq.Metadata.ExtraBytes = pbReq.Metadata.ExtraBytes[:0]
		}

		if err := unmarshalAdEventVT(pbReq, req.Body); err != nil {
			return fields, respInvalidProto, http.StatusBadRequest, false
		}

		if len(pbReq.CampaignId) != 16 {
			return fields, respInvalidCampaign, http.StatusBadRequest, false
		}
		copy(fields.campaignID[:], pbReq.CampaignId)

		fields.eventType = unsafeString(pbReq.EventType)
		if pbReq.Metadata != nil {
			fields.userID = unsafeString(pbReq.Metadata.UserId)
			if len(pbReq.Metadata.ClickId) > 0 {
				fields.clickID = unsafeString(pbReq.Metadata.ClickId)
			}
			if len(pbReq.Metadata.ExtraBytes) > 0 {
				fields.payload = pbReq.Metadata.ExtraBytes
			} else if len(pbReq.Metadata.ExtraKeys) > 0 {
				ctx.ExtraBuf = marshalExtra(ctx.ExtraBuf, pbReq.Metadata.ExtraKeys, pbReq.Metadata.ExtraValues)
				fields.payload = ctx.ExtraBuf
			}
			fields.deviceType = pbReq.Metadata.DeviceType
		}
		return fields, nil, 0, true
	}

	h.trackMetrics.throughputJSON.Inc()
	trackReq := trackRequestPool.Get().(*TrackRequest)
	trackReq.Reset()
	defer trackRequestPool.Put(trackReq)

	if !adEventProcessorNative {
		if err := ParseOpenRTB3Ingress(trackReq, req.Body); err != nil {
			return fields, respInvalidJSON, http.StatusBadRequest, false
		}
	} else if err := ParseTrackRequestJSONOpt(trackReq, req.Body); err != nil {
		return fields, respInvalidJSON, http.StatusBadRequest, false
	}
	fields.jsonSerializationFlags = scanTrackJSONSerialization(req.Body)
	fields.campaignID = trackReq.CampaignID
	fields.userID = trackReq.UserID
	fields.eventType = trackReq.Type
	fields.payload = trackReq.Payload
	fields.placementID = trackReq.PlacementID
	fields.subs = trackReq.subs
	fields.fbclid = trackReq.fbclid
	fields.gclid = trackReq.gclid
	fields.ttclid = trackReq.ttclid
	fields.msclkid = trackReq.msclkid
	fields.tblci = trackReq.tblci
	fields.obClickID = trackReq.obClickID
	fields.eventID = trackReq.eventID
	fields.txID = trackReq.txID
	if trackReq.ClickID != "" {
		fields.clickID = trackReq.ClickID
	}
	fields.ortbSlot = trackReq.ortbSlot
	fields.telemetrySet = trackReq.TelemetrySet
	if len(trackReq.TelemetryEvents) > 0 {
		fields.telemetryEvents = append(fields.telemetryEvents[:0], trackReq.TelemetryEvents...)
	}
	trackReq.ortbSlot = nil
	return fields, nil, 0, true
}

func fillTrackEvent(evt *domain.Event, fields trackIngestFields, ip, ua string) {
	evt.Reset()
	evt.ClickID = fields.clickID
	evt.CampaignID = fields.campaignID
	evt.UserID = fields.userID
	evt.Type = fields.eventType
	evt.PlacementID = fields.placementID
	if len(fields.payload) > 0 && !trackAttributionExtrasPresent(fields) {
		evt.Payload = fields.payload
	} else {
		evt.Payload = appendAttributionPayload(evt.Payload[:0], fields.payload, fields.subs, fields.fbclid, fields.gclid, fields.ttclid, fields.msclkid, fields.tblci, fields.obClickID, fields.eventID, fields.txID)
		if evt.Payload == nil {
			evt.Payload = evt.Payload[:0]
		}
	}
	evt.IP = ip
	evt.UA = ua
	if fields.ortbSlot != nil {
		attachOpenRTB3Scratch(evt, fields.ortbSlot)
	}
	evt.JSONSerializationFlags = fields.jsonSerializationFlags
	if fields.telemetrySet != 0 {
		evt.TelemetrySet = 1
		if len(fields.telemetryEvents) > 0 {
			evt.TelemetryEvents = append(evt.TelemetryEvents[:0], fields.telemetryEvents...)
		}
	}
}

func fillTrackEventWithMobileBiometrics(evt *domain.Event, fields trackIngestFields, ip, ua string, mobileBiometrics bool) {
	fillTrackEvent(evt, fields, ip, ua)
	if mobileBiometrics {
		applyMobileBiometricSummary(evt)
	}
}

func shouldApplyMobileBiometrics(cfg *config.Config, registry domain.CampaignRegistry, campaignID uuid.UUID, eventType string) bool {
	if cfg == nil {
		return false
	}
	if cfg.MobileBiometricsEnabled {
		return true
	}
	if eventType != "click" || !cfg.MobileBiometricsClickEnabled {
		return false
	}
	if registry == nil {
		return false
	}
	camp, ok := registry.GetCampaign(campaignID)
	return ok && camp != nil && camp.MobileBiometricsClickEnabled
}

// deliverGnetTrack maps processTrack outcome to gnet response. On accept, publishAcceptedOrRollback consumes
// the admission lease (ProcessReserved); publish false -> RollbackDebit + filterRejectProducerOverload 503.
// Caller always Release() lease after return (clears reserve if publish skipped).
func (h *AdsPacketHandler) deliverGnetTrack(
	ctx *ConnContext,
	accept string,
	origin string,
	c gnet.Conn,
	evt *domain.Event,
	startMono int64,
	wReqID *BufWrapper,
	requestIDStr string,
	outcome trackOutcome,
	lease *streamAdmissionLease,
) gnet.Action {
	switch outcome.Status {
	case trackStatusFraudAccepted:
		h.recordTrackReject(ctx, evt, outcome.RejectKind)
		shard := h.sharder.GetShard(evt.CampaignID)
		enqueueFraudReject(h.fraudWriter, shard, evt)
		h.writeGnetTrackAccepted(ctx, accept, origin, c, startMono, wReqID, requestIDStr, "")
		return gnet.None
	case trackStatusRejected:
		// Filter reject before accept: no publish; caller Release() returns TryReserve slot.
		spec := filterRejectSpecs[outcome.RejectKind]
		h.recordTrackReject(ctx, evt, outcome.RejectKind)
		if outcome.RejectKind == filterRejectFraudBlocked {
			shard := h.sharder.GetShard(evt.CampaignID)
			enqueueFraudReject(h.fraudWriter, shard, evt)
		}
		h.writeFilterReject(c, spec.gnetResp, ctx)
		h.recordMetrics(startMono, spec.status)
		return gnet.None
	case trackStatusInternalError:
		h.writeFilterReject(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return gnet.None
	case trackStatusAccepted:
		h.trackMetrics.decisionAccepted.Inc()
		writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
		if !h.publishAcceptedOrRollback(context.Background(), evt, lease) {
			spec := filterRejectSpecs[filterRejectProducerOverload]
			h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			return gnet.None
		}
		h.writeGnetTrackAccepted(ctx, accept, origin, c, startMono, wReqID, requestIDStr, outcome.LandingURL)
		return gnet.None
	default:
		h.writeFilterReject(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return gnet.None
	}
}

const (
	jsonSerFlagSortedKeys    uint8 = 1 << 0
	jsonSerFlagPythonSpacing uint8 = 1 << 1
	jsonSerFlagLongTimestamp uint8 = 1 << 2

	jsonSerPythonSpacingMaxBody = 4096
)

func scanTrackJSONSerialization(data []byte) uint8 {
	n := len(data)
	if n == 0 {
		return 0
	}
	bud := newJSONScanBudget()
	i, ok := skipJSONWSBudget(data, 0, n, &bud)
	if !ok || i >= n || data[i] != '{' {
		return 0
	}
	i++

	var flags uint8
	var prevKey []byte
	keyCount := 0
	sortedKeys := false
	checkPython := n <= jsonSerPythonSpacingMaxBody

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}
		if data[i] == '}' {
			break
		}
		if data[i] != '"' {
			return flags
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return flags
			}
			i++
		}
		if i+1 >= n {
			return flags
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n || data[i] != ':' {
			return flags
		}
		if checkPython && i+1 < n && data[i+1] == ' ' {
			flags |= jsonSerFlagPythonSpacing
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}

		key := data[keyStart:keyEnd]
		if isTrackTimestampKey(key) && data[i] != '"' && scanLongIntegerDigits(data, i, n) >= 16 {
			flags |= jsonSerFlagLongTimestamp
		}

		valEnd, err := skipJSONValueBudget(data, i, &bud)
		if err != nil {
			return flags
		}
		i = valEnd

		if keyCount > 0 {
			if bytesCompareLex(prevKey, key) < 0 {
				if keyCount == 1 {
					sortedKeys = true
				}
			} else {
				sortedKeys = false
			}
		}
		prevKey = key
		keyCount++

		if !bud.ConsumeKeyPair() {
			return flags
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			i = n
		default:
			return flags
		}
	}

	if sortedKeys && keyCount >= 2 {
		flags |= jsonSerFlagSortedKeys
	}
	return flags
}

func bytesCompareLex(a, b []byte) int {
	ln := len(a)
	if len(b) < ln {
		ln = len(b)
	}
	for i := range ln {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func isTrackTimestampKey(key []byte) bool {
	switch len(key) {
	case 2:
		return key[0] == 't' && key[1] == 's'
	case 4:
		return key[0] == 't' && key[1] == 'i' && key[2] == 'm' && key[3] == 'e'
	case 9:
		return key[0] == 't' && key[1] == 'i' && key[2] == 'm' && key[3] == 'e' &&
			key[4] == 's' && key[5] == 't' && key[6] == 'a' && key[7] == 'm' && key[8] == 'p'
	case 10:
		return key[0] == 'e' && key[1] == 'v' && key[2] == 'e' && key[3] == 'n' &&
			key[4] == 't' && key[5] == '_' && key[6] == 't' && key[7] == 'i' && key[8] == 'm' && key[9] == 'e'
	default:
		return false
	}
}

func scanLongIntegerDigits(data []byte, i, n int) int {
	if i >= n {
		return 0
	}
	if data[i] == '-' {
		i++
	}
	digits := 0
	for i < n && data[i] >= '0' && data[i] <= '9' {
		digits++
		i++
	}
	return digits
}

const trackTelemetryMaxEvents = 64

func matchTelemetryKey(key []byte) bool {
	return len(key) == 9 &&
		httpingress.FoldKeyU32(key, 0) == 0x656c6574 &&
		httpingress.FoldKeyU32(key, 4) == 0x7274656d &&
		key[8] == 'y'
}

func parseTrackTelemetryValue(data []byte, start, n int, bud *jsonScanBudget, telemetryScratch []domain.BehaviorTelemetryEvent) (int, []domain.BehaviorTelemetryEvent, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '{' {
		return start, nil, false
	}
	i++

	var events []domain.BehaviorTelemetryEvent

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}
		if data[i] == '}' {
			i++
			return i, events, true
		}
		if data[i] != '"' {
			return start, nil, false
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return start, nil, false
			}
			i++
		}
		if i+1 >= n {
			return start, nil, false
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return start, nil, false
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}

		key := data[keyStart:keyEnd]
		if len(key) == 6 && key[0] == 'e' && key[1] == 'v' && key[2] == 'e' && key[3] == 'n' && key[4] == 't' && key[5] == 's' {
			parsed, end, ok := parseTrackTelemetryEventsArray(data, i, n, bud, telemetryScratch)
			if !ok {
				return start, nil, false
			}
			events = parsed
			i = end
		} else {
			valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
			if err != nil {
				return start, nil, false
			}
			i = valEnd
		}

		if !bud.ConsumeKeyPair() {
			return start, nil, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			i++
			return i, events, true
		default:
			return start, nil, false
		}
	}
	return start, nil, false
}

func parseTrackTelemetryEventsArray(data []byte, start, n int, bud *jsonScanBudget, scratch []domain.BehaviorTelemetryEvent) ([]domain.BehaviorTelemetryEvent, int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '[' {
		return nil, start, false
	}
	i++

	events := scratch[:0]
	if cap(events) < 8 {
		events = make([]domain.BehaviorTelemetryEvent, 0, 8)
	}
	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return nil, start, false
		}
		if data[i] == ']' {
			return events, i + 1, true
		}
		if len(events) >= trackTelemetryMaxEvents {
			return nil, start, false
		}
		evt, end, ok := parseTrackTelemetryEventObject(data, i, n, bud)
		if !ok {
			return nil, start, false
		}
		events = append(events, evt)
		i = end

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return nil, start, false
		}
		switch data[i] {
		case ',':
			i++
		case ']':
			return events, i + 1, true
		default:
			return nil, start, false
		}
	}
	return nil, start, false
}

func parseTrackTelemetryEventObject(data []byte, start, n int, bud *jsonScanBudget) (domain.BehaviorTelemetryEvent, int, bool) {
	var evt domain.BehaviorTelemetryEvent
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '{' {
		return evt, start, false
	}
	i++

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}
		if data[i] == '}' {
			return evt, i + 1, true
		}
		if data[i] != '"' {
			return evt, start, false
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return evt, start, false
			}
			i++
		}
		if i+1 >= n {
			return evt, start, false
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return evt, start, false
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}

		key := data[keyStart:keyEnd]
		switch len(key) {
		case 1:
			switch key[0] {
			case 't':
				if data[i] != '"' {
					return evt, start, false
				}
				valStart := i + 1
				end, ok := scanJSONStringEnd(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				evt.T = unsafeString(data[valStart : end-1])
				i = end
			case 'x', 'y', 'z':
				v, end, ok := parseJSONIntValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				switch key[0] {
				case 'x':
					evt.X = v
				case 'y':
					evt.Y = v
				default:
					evt.Z = v
				}
				i = end
			default:
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		case 2:
			if key[0] == 't' && key[1] == 's' {
				v, end, ok := parseJSONIntValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				evt.TS = int64(v)
				i = end
			} else {
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		case 5:
			if key[0] == 'f' && key[1] == 'o' && key[2] == 'r' && key[3] == 'c' && key[4] == 'e' {
				v, end, ok := parseJSONFloatValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				evt.Force = v
				i = end
			} else {
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		case 8:
			if key[0] == 'r' && key[1] == 'a' && key[2] == 'd' && key[3] == 'i' && key[4] == 'u' && key[5] == 's' && key[6] == '_' {
				v, end, ok := parseJSONFloatValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				switch key[7] {
				case 'x':
					evt.RadiusX = v
				case 'y':
					evt.RadiusY = v
				default:
					return evt, start, false
				}
				i = end
			} else {
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		default:
			valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
			if err != nil {
				return evt, start, false
			}
			i = valEnd
		}

		if !bud.ConsumeKeyPair() {
			return evt, start, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return evt, i + 1, true
		default:
			return evt, start, false
		}
	}
	return evt, start, false
}

func parseJSONIntValue(data []byte, start, n int, bud *jsonScanBudget) (int, int, bool) {
	i := start
	if i >= n {
		return 0, start, false
	}
	neg := false
	if data[i] == '-' {
		neg = true
		i++
	}
	if i >= n || data[i] < '0' || data[i] > '9' {
		return 0, start, false
	}
	val := 0
	for i < n && data[i] >= '0' && data[i] <= '9' {
		val = val*10 + int(data[i]-'0')
		if val > 1_000_000_000 {
			return 0, start, false
		}
		i++
	}
	if neg {
		val = -val
	}
	if i < n && !isDelimiter(data[i]) {
		return 0, start, false
	}
	return val, i, true
}

func parseJSONFloatValue(data []byte, start, n int, bud *jsonScanBudget) (float32, int, bool) {
	i := start
	if i >= n {
		return 0, start, false
	}
	neg := false
	if data[i] == '-' {
		neg = true
		i++
	}
	if i >= n {
		return 0, start, false
	}
	intPart := 0
	hasInt := false
	for i < n && data[i] >= '0' && data[i] <= '9' {
		hasInt = true
		intPart = intPart*10 + int(data[i]-'0')
		i++
	}
	fracPart := 0
	fracDiv := 1.0
	if i < n && data[i] == '.' {
		i++
		for i < n && data[i] >= '0' && data[i] <= '9' {
			fracPart = fracPart*10 + int(data[i]-'0')
			fracDiv *= 10
			i++
		}
	}
	if !hasInt && fracDiv == 1.0 {
		return 0, start, false
	}
	val := float32(intPart) + float32(fracPart)/float32(fracDiv)
	if neg {
		val = -val
	}
	if i < n && !isDelimiter(data[i]) {
		return 0, start, false
	}
	return val, i, true
}

const MaxSubIDs = track.MaxSubIDs

func subKeyIndex(key []byte) (int, bool) {
	return track.SubKeyIndex(key)
}

func shouldSampleHistogram(seq uint64, mask uint64) bool {
	return filter.ShouldSampleHistogram(seq, mask)
}

var (
	BudgetFrozenKeyPrefix        = filter.BudgetFrozenKeyPrefix
	WarmBudgetKeyNX              = filter.WarmBudgetKeyNX
	TryRecoverBudgetFromRegistry = filter.TryRecoverBudgetFromRegistry
)

func UnsafeBytes(s string) []byte {
	return filter.UnsafeBytes(s)
}

func cachedUnixSec() uint64 {
	return filter.CachedUnixSec()
}

func NewFastUUID() uuid.UUID {
	return filter.NewFastUUID()
}
