// Package ortbreact maps OpenRTB 2.6 wire flags into rtb.RtbTargetingInput for ingest handlers.
//
// Role:
//   - WireTargeting scratch struct and MapParsedToTargeting for /openrtb/bid React path.
//   - MapImpSlotToTargeting for multi-imp auctions; MapWireToTargeting for legacy wire helpers.
//   - ImpSlotFromHot and SeatAllowedInWSeat bridge internal/openrtb parse output to internal/rtb inputs
//     without importing ingest handlers from openrtb.
//
// Topology:
//   - Runs on PinnedWorkerPool Tier B after DFA parse and OpenRTB26 split into hot/cold structs.
//   - RunAuction in-process on the same worker; no full FilterEngine chain on exchange path.
//   - Uses ingest/cold EnsureIngestGeo for geo hash when device geo fields are absent.
//
// Thread model (hot-path.mdc Tracker thread model):
//
//	Tier B (PinnedWorkerPool): MapParsedToTargeting and RunAuction run synchronously on the offload
//	  worker after parse; Tier A epoll must not call targeting or auction helpers.
//
// Invariants:
//   - Hot OpenRTB26Hot stays on worker stack; cold strings and blocklists live in OpenRTB26Cold arena.
//   - PublisherFloorMicro = max(imp BidFloorMicro, DealBidFloorMicro); imp-slot mapping overrides request-level floor.
//   - Default imp ID "1" when ImpIDLen zero (MapParsedToTargeting, MapWireToTargeting, empty imp.ID).
//   - MediaMask video when imp-slot or hot video flag set; else display. MaxDuration from slot or hot.
//   - GeoHash from wire country bytes when openrtb26FlagGeoCountry set; else sync EnsureIngestGeo(geo, clientIP).
//   - SeatAllowedInWSeat returns true when WSeatCount zero or seat empty (no whitelist configured).
//   - CurrencyUSD false when request Cur or imp BidFloorCur is EUR (MapWireToTargeting / hot EUR flag).
//
// Contracts:
//   - WireTargeting.Input carries rtb.RtbTargetingInput fixed buffers (DealIDBuf, Schain nodes).
//   - ImpIDBuf capped at openrtb26ImpIDMax; DealIDLen and SeatCount propagated from hot or imp slot.
//   - DeadlineMono from openrtb.DeadlineMonoFromTmax(hot.TmaxMs) bounds in-process RunAuction.
//   - categoryMaskFromWire: site/app Cat last digit sets bit; empty cats yields mask 1.
//   - deviceTypeFromWire maps OpenRTB device type ints to rtb device type uint8 (mobile/tablet/CTV).
//
// Tradeoffs:
//   - Rejected full FilterEngine.Check on /openrtb/bid: in-process RunAuction only (architecture.mdc).
//   - Rejected encoding/json bid response on hot path (openrtb.WriteBidHTTPResponse with stack buffers).
//   - Rejected heap strings for imp/deal IDs: ImpIDBuf and DealIDBuf on WireTargeting stack struct.
//   - Geo MaxMind lookup on Tier B worker when wire country absent (sync, bounded); no background geo reload here.
//   - Fail-open seat gating: empty wseat list allows any seat; fail-open default imp ID avoids no-bid on missing id.
//   - Multi-imp: MapImpSlotToTargeting resets deal from slot and recomputes seat count via impSlotSeatCount.
//
// Forbidden:
//   - Full FilterEngine.Check on /openrtb/bid exchange path.
//   - encoding/json bid response on production hot path (use openrtb.WriteBidHTTPResponse).
//
// Verify (subpackage has no *_test.go; exchange path tested from parent internal/ingest):
//
//	go test ./internal/ingest/ -short -run TestRunOpenRTBExchange -count=1
//	go test ./internal/ingest/ -short -run TestOpenRTBBid_gnetHandler -count=1
//	go test ./internal/openrtb/... -short -count=1
package ortbreact
