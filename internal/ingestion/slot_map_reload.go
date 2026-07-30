package ingestion

import (
	"encoding/json"
	"fmt"
	"time"
)

const DefaultSlotMapReloadTopic = "shards:reload"

type SlotMapReloadMessage struct {
	Version      int32 `json:"version"`
	RoutingEpoch int64 `json:"routing_epoch,omitempty"`
	AtUnix       int64 `json:"at_unix"`
}

func EncodeSlotMapReloadMessage(version int32, routingEpoch int64) ([]byte, error) {
	msg := SlotMapReloadMessage{
		Version:      version,
		RoutingEpoch: routingEpoch,
		AtUnix:       time.Now().Unix(),
	}
	return json.Marshal(msg)
}

func EncodeSlotMapReloadMessageVersion(version int32) ([]byte, error) {
	return EncodeSlotMapReloadMessage(version, 0)
}

func DecodeSlotMapReloadMessage(payload []byte) (SlotMapReloadMessage, error) {
	var msg SlotMapReloadMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return msg, fmt.Errorf("slot map reload decode: %w", err)
	}
	if msg.Version <= 0 {
		return msg, fmt.Errorf("slot map reload decode: invalid version %d", msg.Version)
	}
	return msg, nil
}

type OpsSlotMapResponse struct {
	Version       int32    `json:"version"`
	ActiveVersion int32    `json:"active_version"`
	RoutingEpoch  int64    `json:"routing_epoch"`
	Slots         []uint16 `json:"slots"`
}
