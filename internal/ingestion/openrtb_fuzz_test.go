package ingestion

import (
	"bytes"
	"testing"

	"ad-event-processor/internal/openrtb"
)

var (
	fuzzOpenRTB26Seed = []byte(`{
 "id":"req-1",
 "tmax":250,
 "imp":[{"id":"1","bidfloor":1.25,"pmp":{"deals":[{"id":"deal-a","wseat":["seat-1"]}]},"banner":{"w":300,"h":250}}],
 "device":{"ip":"1.1.1.1","ua":"Mozilla","devicetype":1,"geo":{"country":"US"}},
 "site":{"cat":["IAB1"],"domain":"example.com"},
 "bcat":["IAB2-3"],"badv":["evil.com"],"bapp":["com.blocked"],"bseat":["blocked-seat"],
 "regs":{"gdpr":1,"us_privacy":"1YNN"}
}`)

	fuzzOpenRTB3Seed = []byte(`{"openrtb":{"ver":"3.0","item":[{"id":"1","flr":1.5,"deal_id":"d1","category_mask":3}]}}`)
)

func fuzzNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: panic: %v", name, r)
		}
	}()
	fn()
}

func FuzzParseOpenRTB26Split(f *testing.F) {
	f.Add(fuzzOpenRTB26Seed)
	f.Add([]byte(`{"imp":[{"id":"x","banner":{}}],"site":{},"device":{"ip":"x","ua":"y"}}`))
	f.Add([]byte{})
	f.Add([]byte(`{"imp":`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var hot OpenRTB26Hot
		var cold OpenRTB26Cold
		fuzzNoPanic(t, "ParseOpenRTB26Split", func() {
			ParseOpenRTB26Split(data, &hot, &cold)
		})
		if hot.OK {
			fuzzNoPanic(t, "mapParsedToTargeting", func() {
				_ = mapParsedToTargeting(&hot, &cold, nil, "203.0.113.1")
			})
			fuzzNoPanic(t, "exchangeReady", func() {
				_ = exchangeReady(&hot, &cold, openrtb.ExchangeConfig{MultiImpMax: 10})
			})
			fuzzNoPanic(t, "blockedCatMaskFromCold", func() {
				_ = blockedCatMaskFromCold(&cold)
			})
			fuzzNoPanic(t, "checkBlocklistsParsed", func() {
				_ = checkBlocklistsParsed(hot, &cold, true)
			})
			fuzzNoPanic(t, "seatBlockedByBSeat", func() {
				_ = seatBlockedByBSeat(&cold, []byte("seat-1"))
			})
			for i := 0; i < int(cold.ImpSlots); i++ {
				slot := &cold.Imps[i]
				fuzzNoPanic(t, "mapImpSlotToTargeting", func() {
					_ = mapImpSlotToTargeting(&hot, &cold, slot, nil, "")
				})
				fuzzNoPanic(t, "seatAllowedInWSeat", func() {
					_ = seatAllowedInWSeat(slot, []byte("seat-1"))
				})
				fuzzNoPanic(t, "impSlotExchangeReady", func() {
					_ = impSlotExchangeReady(slot)
				})
			}
		}
	})
}

func FuzzParseOpenRTB26Helpers(f *testing.F) {
	f.Add([]byte(`"abc"`))
	f.Add([]byte(`1.234567`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`USA`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var buf [256]byte
		fuzzNoPanic(t, "parseQuotedField", func() {
			_ = parseQuotedField(data, 0, buf[:])
			_ = parseQuotedField(data, 0, nil)
		})
		fuzzNoPanic(t, "parseJSONIntField", func() {
			_ = parseJSONIntField(data, 0)
		})
		fuzzNoPanic(t, "parseDecimalMicro", func() {
			_ = parseDecimalMicro(data)
		})
		fuzzNoPanic(t, "parseDecimalMicroField", func() {
			_ = parseDecimalMicroField(data, 0)
		})
		fuzzNoPanic(t, "parseCategoryMaskFromArray", func() {
			_ = parseCategoryMaskFromArray(data, 0)
		})
		fuzzNoPanic(t, "categoryBitFromIABCode", func() {
			_ = categoryBitFromIABCode(data)
		})
		fuzzNoPanic(t, "normalizeCountryBytes", func() {
			var dst [3]byte
			_ = normalizeCountryBytes(data, dst[:])
		})
		fuzzNoPanic(t, "normalizeRegionBytes", func() {
			var dst [3]byte
			_ = normalizeRegionBytes(data, dst[:])
		})
		fuzzNoPanic(t, "parseImpObjectCountAt", func() {
			idx := bytes.Index(data, openrtbKeyImp)
			_ = parseImpObjectCountAt(data, idx)
		})
		fuzzNoPanic(t, "parseSchainNodesAt", func() {
			idx := bytes.Index(data, openrtbKeySchain)
			_ = parseSchainNodesAt(data, idx)
		})
		fuzzNoPanic(t, "parseSeatJSONArrayAt", func() {
			var seats [openrtb26SeatMax][openrtb26SeatIDMax]byte
			var lens [openrtb26SeatMax]uint8
			_ = parseSeatJSONArrayAt(data, 0, seats[:], lens[:])
		})
	})
}

func FuzzParseOpenRTB3FSM(f *testing.F) {
	f.Add(fuzzOpenRTB3Seed)
	f.Add([]byte(`{"openrtb":{"ver":"3.0","item":[{"id":"a","flr":1.5},{"id":"b","flr":9.9}]}}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var parsed OpenRTB3Parsed
		fuzzNoPanic(t, "parseOpenRTB3FSMInto", func() {
			_ = parseOpenRTB3FSMInto(&parsed, data)
		})
		fuzzNoPanic(t, "ParseOpenRTB3Payload", func() {
			_, _, _, _ = ParseOpenRTB3Payload(data)
		})
		fuzzNoPanic(t, "ParseDealIDBytes", func() {
			var buf [ortbDealIDMax]byte
			_ = ParseDealIDBytes(data, buf[:])
		})
		var req TrackRequest
		fuzzNoPanic(t, "ParseOpenRTB3Ingress", func() {
			_ = ParseOpenRTB3Ingress(&req, data)
		})
	})
}

func FuzzOpenRTB26ImpSlotWalk(f *testing.F) {
	f.Add(fuzzOpenRTB26Seed)
	f.Add([]byte(`{"imp":[{"id":"1"},{"id":"2","banner":{}},{"id":"3","video":{}}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		idx := bytes.Index(data, openrtbKeyImp)
		fuzzNoPanic(t, "foreachImpObject", func() {
			foreachImpObject(data, idx, func(obj []byte) bool {
				var slot OpenRTB26ImpSlot
				_ = parseImpSlot(obj, &slot)
				return true
			})
		})
	})
}
