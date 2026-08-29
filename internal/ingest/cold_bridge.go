package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/filter/netintel"
	"ad-event-processor/internal/ingest/cold"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	filter.AcceptEncodingBrowserMismatchFn = cold.AcceptEncodingBrowserMismatch
	filter.SecFetchAnomalyFn = cold.SecFetchAnomaly
	filter.ClientHintsPlatformMismatchFn = cold.ClientHintsPlatformMismatch
	filter.TLSALPNBrowserMismatchFn = cold.TLSALPNBrowserMismatch
}

const wireSecFetchAllBits = cold.WireSecFetchAllBits

func classifySecFetchSite(b []byte) uint8  { return cold.ClassifySecFetchSite(b) }
func classifySecFetchMode(b []byte) uint8  { return cold.ClassifySecFetchMode(b) }
func classifySecFetchDest(b []byte) uint8  { return cold.ClassifySecFetchDest(b) }
func classifySecCHUAMobile(b []byte) uint8 { return cold.ClassifySecCHUAMobile(b) }

func classifyAcceptEncoding(b []byte) uint8 { return cold.ClassifyAcceptEncoding(b) }

func acceptLangGeoMismatch(acceptLang, geoCountry string) bool {
	return cold.AcceptLangGeoMismatch(acceptLang, geoCountry)
}

func http1AssignConnTimingHeaders(req *Request, key, val []byte) {
	if req == nil {
		return
	}
	switch {
	case httpingress.KeyMatchFold(key, "x-rtt-syn-ms"):
		if ms, ok := cold.ParseConnTimingMSHeader(val); ok {
			req.RTTSynMS = ms
			req.ConnTimingSet |= connTimingRTTBit
		}
	case httpingress.KeyMatchFold(key, "x-ttfb-app-ms"):
		if ms, ok := cold.ParseConnTimingMSHeader(val); ok {
			req.TTFBAppMS = ms
			req.ConnTimingSet |= connTimingTTFBBit
		}
	}
}

func fillConnTimingFromRequest(evt *domain.Event, req *Request) {
	if evt == nil || req == nil || req.ConnTimingSet == 0 {
		return
	}
	evt.ConnTimingSet = req.ConnTimingSet
	if req.ConnTimingSet&connTimingRTTBit != 0 {
		evt.RTTSynMS = req.RTTSynMS
	}
	if req.ConnTimingSet&connTimingTTFBBit != 0 {
		evt.TTFBAppMS = req.TTFBAppMS
	}
	if req.ConnTimingSet == connTimingRTTBit|connTimingTTFBBit {
		evt.RTTSplitDeltaMS = cold.RttSplitDeltaMS(evt.RTTSynMS, evt.TTFBAppMS)
	}
}

func ComputeCompositeHashFromTrackReq(req *TrackRequest) uint32 {
	return cold.ComputeCompositeHashUUID(req.CampaignID, UnsafeBytes(req.UserID))
}

func trackAttributionExtrasPresent(fields trackIngestFields) bool {
	if fields.fbclid != "" || fields.gclid != "" || fields.ttclid != "" || fields.msclkid != "" ||
		fields.tblci != "" || fields.obClickID != "" || fields.eventID != "" || fields.txID != "" {
		return true
	}
	return fields.subs.HasAny()
}

func FormatUUIDCanonical(dst *[36]byte, id uuid.UUID) { cold.FormatUUIDCanonical(dst, id) }

func ComputeCompositeHashUUID(campaignID uuid.UUID, userID []byte) uint32 {
	return cold.ComputeCompositeHashUUID(campaignID, userID)
}

func ComputeCompositeHashFromTrackCampaignUser(campaignID uuid.UUID, userID string) uint32 {
	return cold.ComputeCompositeHashFromTrackCampaignUser(campaignID, userID)
}

func ComputeCompositeHashFromProto(req *pb.AdEvent) uint32 {
	return cold.ComputeCompositeHashFromProto(req)
}

func resetAdEventInPlace(evt *pb.AdEvent) { cold.ResetAdEventInPlace(evt) }

func ensureIngestGeo(geo netintel.GeoProvider, evt *domain.Event) { cold.EnsureIngestGeo(geo, evt) }

func parseCategoryMask(payload []byte) uint64 { return cold.ParseCategoryMask(payload) }

func exportHealthProbeMetrics(healthy bool, shardHealthy []int32) {
	cold.ExportHealthProbeMetrics(healthy, shardHealthy)
}

func appendAttributionPayload(dst, payload []byte, subs SubIDSlots, fbclid, gclid, ttclid, msclkid, tblci, obClickID, eventID, txID string) []byte {
	return cold.AppendAttributionPayload(dst, payload, subs, fbclid, gclid, ttclid, msclkid, tblci, obClickID, eventID, txID)
}

func appendAttributionPassthrough(dst []byte, fbclid, gclid, ttclid string) []byte {
	return cold.AppendAttributionPassthrough(dst, fbclid, gclid, ttclid)
}

func appendFlowAttribution(dst []byte, landerID, offerID uuid.UUID) []byte {
	return cold.AppendFlowAttribution(dst, landerID, offerID)
}

type CampaignTripletPick = cold.CampaignTripletPick

type ConversionDatacenterIPChecker = cold.ConversionDatacenterIPChecker

var NewConversionDatacenterIPChecker = cold.NewConversionDatacenterIPChecker

type preboundFraudMetrics struct {
	tierPass    prometheus.Counter
	tierSuspect prometheus.Counter
	tierIVT     prometheus.Counter
	tierBlock   prometheus.Counter

	reason [fraudReasonCount]prometheus.Counter

	l1Reject prometheus.Counter
}

var boundFraudMetrics = newPreboundFraudMetrics()

func newPreboundFraudMetrics() preboundFraudMetrics {
	pm := preboundFraudMetrics{
		tierPass:    metrics.FraudTierTotal.WithLabelValues("pass"),
		tierSuspect: metrics.FraudTierTotal.WithLabelValues("suspect"),
		tierIVT:     metrics.FraudTierTotal.WithLabelValues("ivt"),
		tierBlock:   metrics.FraudTierTotal.WithLabelValues("block"),
		l1Reject:    metrics.L1RejectTotal,
	}
	for id := FraudReasonID(1); id < fraudReasonCount; id++ {
		code := FraudReasonCode(id)
		if code != "" {
			pm.reason[id] = metrics.FraudReasonTotal.WithLabelValues(code)
		}
	}
	return pm
}

type (
	ReconciliationWorker   = stream.ReconciliationWorker
	Snapshot               = stream.Snapshot
	SnapshotReplicator     = stream.SnapshotReplicator
	ClickHouseConn         = stream.ClickHouseConn
	PostgresConn           = stream.PostgresConn
	TCPControlClient       = stream.TCPControlClient
	TCPControlClientConfig = stream.TCPControlClientConfig
	UDPControl             = stream.UDPControl
	UDPControlConfig       = stream.UDPControlConfig
	UDPChannelState        = stream.UDPChannelState
)

var (
	NewReconciliationWorker  = stream.NewReconciliationWorker
	ApplyRuntimeAutotune     = stream.ApplyRuntimeAutotune
	DefaultMaxWorkers        = stream.DefaultMaxWorkers
	NewSnapshotReplicator    = stream.NewSnapshotReplicator
	NewTCPControlClient      = stream.NewTCPControlClient
	NewUDPControl            = stream.NewUDPControl
	NewUDPControlFromConfig  = stream.NewUDPControlFromConfig
	EncodeQuotaEpochDatagram = stream.EncodeQuotaEpochDatagram
)

const (
	UDPChannelOK    = stream.UDPChannelOK
	UDPChannelStale = stream.UDPChannelStale
)
