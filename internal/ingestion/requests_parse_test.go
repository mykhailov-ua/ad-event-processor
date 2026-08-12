package ingestion

import (
	"encoding/json"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseTrackRequestJSON(t *testing.T) {
	data := testTrackRequestJSON(t)

	var reqReflect trackRequestReflect
	err := json.Unmarshal(data, &reqReflect)
	require.NoError(t, err)

	var req TrackRequest
	err = ParseTrackRequestJSON(&req, data)
	require.NoError(t, err)

	require.Equal(t, reqReflect.CampaignID, req.CampaignID)
	require.Equal(t, reqReflect.UserID, req.UserID)
	require.Equal(t, reqReflect.Type, req.Type)
	require.Equal(t, reqReflect.ClickID, req.ClickID)
	require.JSONEq(t, string(reqReflect.Payload), string(req.Payload))
}

func TestParseTrackRequestJSONOptParity(t *testing.T) {
	data := testTrackRequestJSON(t)

	var req TrackRequest
	require.NoError(t, ParseTrackRequestJSON(&req, data))

	var reqOpt TrackRequest
	require.NoError(t, ParseTrackRequestJSONOpt(&reqOpt, data))

	require.Equal(t, req, reqOpt)
}

func TestParseTrackRequestJSONOpt_ZeroAlloc(t *testing.T) {
	data := testTrackRequestJSON(t)
	var req TrackRequest

	avg := testing.AllocsPerRun(100, func() {
		req.Reset()
		if err := ParseTrackRequestJSONOpt(&req, data); err != nil {
			t.Fatal(err)
		}
	})
	if avg > 0 {
		t.Fatalf("ParseTrackRequestJSONOpt allocated %f times per run, want 0", avg)
	}
}

func TestParseTrackRequestJSON_ZeroAlloc(t *testing.T) {
	data := testTrackRequestJSON(t)
	var req TrackRequest

	avg := testing.AllocsPerRun(100, func() {
		req.Reset()
		if err := ParseTrackRequestJSON(&req, data); err != nil {
			t.Fatal(err)
		}
	})
	if avg > 0 {
		t.Fatalf("ParseTrackRequestJSON allocated %f times per run, want 0", avg)
	}
}

func testProtoTrackBody(t testing.TB) []byte {
	t.Helper()
	id := uuid.New()
	evt := &pb.AdEvent{
		CampaignId: id[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId: []byte("test-click"),
			UserId:  []byte("user123"),
		},
	}
	body, err := evt.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func resetPooledAdEvent(evt *pb.AdEvent) {
	evt.CampaignId = evt.CampaignId[:0]
	evt.EventType = evt.EventType[:0]
	if evt.Metadata == nil {
		evt.Metadata = &pb.EventMetadata{}
		return
	}
	md := evt.Metadata
	md.ClickId = md.ClickId[:0]
	md.UserId = md.UserId[:0]
	md.DeviceType = md.DeviceType[:0]
	md.Os = md.Os[:0]
	for i := range md.ExtraKeys {
		md.ExtraKeys[i] = md.ExtraKeys[i][:0]
	}
	md.ExtraKeys = md.ExtraKeys[:0]
	for i := range md.ExtraValues {
		md.ExtraValues[i] = md.ExtraValues[i][:0]
	}
	md.ExtraValues = md.ExtraValues[:0]
	md.ExtraBytes = md.ExtraBytes[:0]
}

func TestAdEvent_UnmarshalVT_ZeroAlloc(t *testing.T) {
	body := testProtoTrackBody(t)
	var evt pb.AdEvent
	evt.Metadata = &pb.EventMetadata{}

	for i := 0; i < 100; i++ {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	}

	avg := testing.AllocsPerRun(100, func() {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	})
	if avg > 0 {
		t.Fatalf("AdEvent.UnmarshalVT allocated %f times per run, want 0", avg)
	}
}

func testProtoTrackBodyExtraBytes(t testing.TB) []byte {
	t.Helper()
	id := uuid.New()
	evt := &pb.AdEvent{
		CampaignId: id[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId:    []byte("test-click"),
			UserId:     []byte("user123"),
			ExtraBytes: []byte(`{"slot":"top","cpm":"1.25"}`),
		},
	}
	body, err := evt.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func testProtoTrackBodyExtraRepeatedLegacy(t testing.TB) []byte {
	t.Helper()
	id := uuid.New()
	evt := &pb.AdEvent{
		CampaignId: id[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId:     []byte("test-click"),
			UserId:      []byte("user123"),
			ExtraKeys:   [][]byte{[]byte("slot"), []byte("cpm")},
			ExtraValues: [][]byte{[]byte("top"), []byte("1.25")},
		},
	}
	body, err := evt.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAdEvent_UnmarshalVT_ExtraBytes_ZeroAlloc(t *testing.T) {
	body := testProtoTrackBodyExtraBytes(t)
	var evt pb.AdEvent
	evt.Metadata = &pb.EventMetadata{}

	for i := 0; i < 100; i++ {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	}

	avg := testing.AllocsPerRun(100, func() {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	})
	if avg > 0 {
		t.Fatalf("AdEvent.UnmarshalVT extra_bytes allocated %f times per run, want 0", avg)
	}
}

func TestAdEvent_UnmarshalVT_ExtraRepeated_LegacyZeroAlloc(t *testing.T) {
	body := testProtoTrackBodyExtraRepeatedLegacy(t)
	var evt pb.AdEvent
	evt.Metadata = &pb.EventMetadata{}

	for i := 0; i < 100; i++ {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	}

	avg := testing.AllocsPerRun(100, func() {
		resetPooledAdEvent(&evt)
		if err := evt.UnmarshalVT(body); err != nil {
			t.Fatal(err)
		}
	})
	if avg > 0 {
		t.Fatalf("AdEvent.UnmarshalVT extra repeated allocated %f times per run, want 0", avg)
	}
}

func BenchmarkTrackRequest_ParseJSON(b *testing.B) {
	data := testTrackRequestJSON(b)
	var req TrackRequest

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req.Reset()
		if err := ParseTrackRequestJSON(&req, data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrackRequest_ParseJSONOpt(b *testing.B) {
	data := testTrackRequestJSON(b)
	var req TrackRequest

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req.Reset()
		if err := ParseTrackRequestJSONOpt(&req, data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrackRequest_ParseJSON_Legacy(b *testing.B) {
	data := testTrackRequestJSON(b)
	var req TrackRequest

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req.Reset()
		if err := parseTrackJSONLegacy(&req, data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrackRequest_Unmarshal_Reflect(b *testing.B) {
	data := testTrackRequestJSON(b)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var req trackRequestReflect
		resetTrackRequestReflect(&req)
		if err := json.Unmarshal(data, &req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrackRequest_UnmarshalJSON(b *testing.B) {
	data := testTrackRequestJSON(b)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var req TrackRequest
		req.Reset()
		if err := req.UnmarshalJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}

func TestParseUUID(t *testing.T) {
	id := uuid.New()
	idStr := id.String()
	idBytes := []byte(idStr)

	var got uuid.UUID
	ok := ParseUUID(idBytes, &got)
	require.True(t, ok)
	require.Equal(t, id, got)

	idRaw := id[:]
	var gotRaw uuid.UUID
	ok = ParseUUID(idRaw, &gotRaw)
	require.True(t, ok)
	require.Equal(t, id, gotRaw)

	require.False(t, ParseUUID([]byte("invalid-uuid-length-not-36"), &got))
	require.False(t, ParseUUID([]byte("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a1g"), &got))
}

func BenchmarkUUID_ParseBytes_Reflect(b *testing.B) {
	id := uuid.New()
	idBytes := []byte(id.String())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		_, err = uuid.ParseBytes(idBytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_ParseUUID_Custom(b *testing.B) {
	id := uuid.New()
	idBytes := []byte(id.String())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var got uuid.UUID
		if !ParseUUID(idBytes, &got) {
			b.Fatal("failed to parse")
		}
	}
}

func parseTrackJSONLegacy(v *TrackRequest, data []byte) error {
	v.Reset()
	if len(data) == 0 {
		return errMalformedJSON
	}

	_ = data[len(data)-1]

	n := len(data)
	i := 0

	for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
		i++
	}

	if i >= n || data[i] != '{' {
		return errMalformedJSON
	}
	i++

	for i < n {
		for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			i++
		}
		if i >= n {
			return errMalformedJSON
		}

		if data[i] == '}' {
			return nil
		}

		if data[i] != '"' {
			return errMalformedJSON
		}
		i++

		keyStart := i
		for i < n && data[i] != '"' {
			if data[i] == '\\' {
				return errMalformedJSON
			}
			i++
		}
		if i >= n {
			return errMalformedJSON
		}
		keyEnd := i
		i++

		key := data[keyStart:keyEnd]

		for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			i++
		}
		if i >= n || data[i] != ':' {
			return errMalformedJSON
		}
		i++

		for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			i++
		}
		if i >= n {
			return errMalformedJSON
		}

		isCampaignID := false
		isUserID := false
		isType := false
		isClickID := false
		isPayload := false
		isPlacementID := false

		switch len(key) {
		case 4:
			if key[0] == 't' && key[1] == 'y' && key[2] == 'p' && key[3] == 'e' {
				isType = true
			}
		case 7:
			if key[0] == 'p' && key[1] == 'a' && key[2] == 'y' && key[3] == 'l' && key[4] == 'o' && key[5] == 'a' && key[6] == 'd' {
				isPayload = true
			} else if key[0] == 'u' && key[1] == 's' && key[2] == 'e' && key[3] == 'r' && key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
				isUserID = true
			}
		case 8:
			if key[0] == 'c' && key[1] == 'l' && key[2] == 'i' && key[3] == 'c' && key[4] == 'k' && key[5] == '_' && key[6] == 'i' && key[7] == 'd' {
				isClickID = true
			}
		case 11:
			if key[0] == 'c' && key[1] == 'a' && key[2] == 'm' && key[3] == 'p' && key[4] == 'a' && key[5] == 'i' && key[6] == 'g' && key[7] == 'n' && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
				isCampaignID = true
			}
		case 12:
			if key[0] == 'p' && key[1] == 'l' && key[2] == 'a' && key[3] == 'c' && key[4] == 'e' && key[5] == 'm' && key[6] == 'e' && key[7] == 'n' && key[8] == 't' && key[9] == '_' && key[10] == 'i' && key[11] == 'd' {
				isPlacementID = true
			}
		}

		if isCampaignID || isUserID || isType || isClickID || isPlacementID {
			if data[i] != '"' {
				return errMalformedJSON
			}
			i++
			valStart := i
			for i < n && data[i] != '"' {
				if data[i] == '\\' {
					i += 2
				} else {
					i++
				}
			}
			if i >= n {
				return errMalformedJSON
			}
			valEnd := i
			i++

			valBytes := data[valStart:valEnd]
			if isCampaignID {
				if !ParseUUID(valBytes, &v.CampaignID) {
					return errMalformedJSON
				}
			} else if isUserID {
				v.UserID = unsafeString(valBytes)
			} else if isType {
				v.Type = unsafeString(valBytes)
			} else if isClickID {
				v.ClickID = unsafeString(valBytes)
			} else if isPlacementID {
				v.PlacementID = unsafeString(valBytes)
			}
		} else if isPayload {
			valStart := i
			valEnd, err := skipJSONValue(data, i)
			if err != nil {
				return err
			}
			v.Payload = data[valStart:valEnd]
			i = valEnd
		} else {
			valEnd, err := skipJSONValue(data, i)
			if err != nil {
				return err
			}
			i = valEnd
		}

		for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			i++
		}
		if i >= n {
			return errMalformedJSON
		}

		if data[i] == ',' {
			i++
			continue
		} else if data[i] == '}' {
			return nil
		}
		return errMalformedJSON
	}

	return errMalformedJSON
}
