package controlplane

import (
	"encoding/json"
	"fmt"

	"espx/pkg/dedupkey"

	"github.com/google/uuid"
)

type LeaseDedupScope struct {
	RegionID    uuid.UUID `json:"region_id"`
	SourceID    uuid.UUID `json:"source_id"`
	SourceEpoch uint32    `json:"source_epoch"`
	SeqStart    int64     `json:"seq_start"`
	SeqEnd      int64     `json:"seq_end"`
	Attempt     int32     `json:"attempt"`
}

func EncodeLeaseDedupScope(scope dedupkey.Scope, attempt int32) ([]byte, error) {
	payload := LeaseDedupScope{
		RegionID:    scope.RegionID,
		SourceID:    scope.SourceID,
		SourceEpoch: scope.SourceEpoch,
		SeqStart:    scope.SeqStart,
		SeqEnd:      scope.SeqEnd,
		Attempt:     attempt,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode lease dedup scope: %w", err)
	}
	return raw, nil
}

func DecodeLeaseDedupScope(raw []byte) (dedupkey.Scope, int32, error) {
	var payload LeaseDedupScope
	if err := json.Unmarshal(raw, &payload); err != nil {
		return dedupkey.Scope{}, 0, fmt.Errorf("decode lease dedup scope: %w", err)
	}
	return dedupkey.Scope{
		RegionID:    payload.RegionID,
		SourceID:    payload.SourceID,
		SourceEpoch: payload.SourceEpoch,
		SeqStart:    payload.SeqStart,
		SeqEnd:      payload.SeqEnd,
	}, payload.Attempt, nil
}
