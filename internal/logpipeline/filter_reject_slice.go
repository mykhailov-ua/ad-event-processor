package logpipeline

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"time"

	"ad-event-processor/internal/ingestion/pb"
)

const filterRejectSampleEventType = "filter_reject"

type FilterRejectSliceRow struct {
	RollupHour  time.Time
	RejectKind  string
	PlacementID string
	Country     string
	RejectCount uint64
}

type filterRejectSliceKey struct {
	hour      time.Time
	kind      string
	placement string
	country   string
}

type filterRejectSamplePayload struct {
	Kind      string `json:"k"`
	Placement string `json:"p"`
	Country   string `json:"c"`
}

func aggregateFilterRejectSlices(r io.Reader) ([]FilterRejectSliceRow, error) {
	aggs := make(map[filterRejectSliceKey]uint64)
	var hdr [4]byte
	recordBuf := make([]byte, 0, 4096)
	evt := &pb.AdStreamEvent{}

	for {
		_, err := io.ReadFull(r, hdr[:])
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		length := binary.BigEndian.Uint32(hdr[:])
		if length == 0 {
			continue
		}
		if length > maxRecordBytes {
			return nil, ErrRecordTooLarge
		}
		if int(length) > cap(recordBuf) {
			recordBuf = make([]byte, length)
		}
		record := recordBuf[:length]
		if _, err := io.ReadFull(r, record); err != nil {
			return nil, err
		}

		kind, placement, country, ok := parseFilterRejectSampleRecord(evt, record)
		if !ok {
			continue
		}
		ts := evt.CreatedAtUnix
		if ts <= 0 {
			ts = time.Now().Unix()
		}
		hour := time.Unix(ts, 0).UTC().Truncate(time.Hour)
		key := filterRejectSliceKey{hour: hour, kind: kind, placement: placement, country: country}
		aggs[key]++
	}

	if len(aggs) == 0 {
		return nil, ErrEmptySegment
	}

	keys := make([]filterRejectSliceKey, 0, len(aggs))
	for key := range aggs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if !keys[i].hour.Equal(keys[j].hour) {
			return keys[i].hour.Before(keys[j].hour)
		}
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		if keys[i].country != keys[j].country {
			return keys[i].country < keys[j].country
		}
		return keys[i].placement < keys[j].placement
	})

	rows := make([]FilterRejectSliceRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, FilterRejectSliceRow{
			RollupHour:  key.hour,
			RejectKind:  key.kind,
			PlacementID: key.placement,
			Country:     key.country,
			RejectCount: aggs[key],
		})
	}
	return rows, nil
}

func parseFilterRejectSampleRecord(evt *pb.AdStreamEvent, record []byte) (kind, placement, country string, ok bool) {
	*evt = pb.AdStreamEvent{}
	if err := evt.UnmarshalVT(record); err != nil {
		return "", "", "", false
	}
	if string(evt.EventType) != filterRejectSampleEventType {
		return "", "", "", false
	}
	var payload filterRejectSamplePayload
	if len(evt.Payload) > 0 {
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			return "", "", "", false
		}
	}
	kind = payload.Kind
	placement = payload.Placement
	country = payload.Country
	if kind == "" {
		return "", "", "", false
	}
	return kind, placement, country, true
}

func aggregateWarmAndRejectSlices(plain []byte, sourceSegment, warmSHA string) ([]RollupRow, []FilterRejectSliceRow, error) {
	rows, warmErr := aggregateWarmSegment(bytes.NewReader(plain), sourceSegment, warmSHA)
	sliceRows, sliceErr := aggregateFilterRejectSlices(bytes.NewReader(plain))
	if warmErr != nil && !errors.Is(warmErr, ErrEmptySegment) {
		return nil, nil, warmErr
	}
	if sliceErr != nil && !errors.Is(sliceErr, ErrEmptySegment) {
		return nil, nil, sliceErr
	}
	if warmErr != nil && errors.Is(warmErr, ErrEmptySegment) {
		rows = nil
	}
	if sliceErr != nil && errors.Is(sliceErr, ErrEmptySegment) {
		sliceRows = nil
	}
	if len(rows) == 0 && len(sliceRows) == 0 {
		return nil, nil, ErrEmptySegment
	}
	return rows, sliceRows, nil
}
