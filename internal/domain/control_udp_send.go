package domain

import (
	"encoding/json"
)

func EncodeQuotaEpochDatagram(dst []byte, msgType uint8, hdr *UDPHeader, limits *UDPControlLimits) int {
	return EncodeQuotaEpochDatagramWithWeights(dst, msgType, hdr, limits, nil)
}

func EncodeQuotaEpochDatagramWithWeights(dst []byte, msgType uint8, hdr *UDPHeader, limits *UDPControlLimits, weights []UDPNodeWeight) int {
	if hdr == nil || limits == nil {
		return 0
	}
	hdr.Magic = udpMagic
	switch {
	case len(weights) > 0:
		hdr.Version = udpProtocolVersion3
		hdr.Flags |= UDPFlagNodeWeights
	case limits.MaxRPD > 0:
		hdr.Version = udpProtocolVersion2
	default:
		hdr.Version = udpProtocolVersion
	}
	hdr.MsgType = msgType
	hdr.NumShards = limits.NumShards
	payloadLen := udpShardPayloadLen(limits.NumShards)
	if limits.MaxRPD > 0 {
		payloadLen += 8
	}
	if len(weights) > 0 {
		payloadLen += udpNodeWeightsEncodedLen(weights)
	}
	hdr.PayloadLen = uint16(payloadLen)
	if len(dst) < UDPHeaderSize+payloadLen {
		return 0
	}
	udpEncodeHeader(dst, hdr)
	off := UDPHeaderSize + udpEncodeShardLimits(dst[UDPHeaderSize:], limits)
	if len(weights) > 0 {
		off += udpEncodeNodeWeights(dst[off:], weights)
	}
	return off
}

type EpochPayloadJSON struct {
	ShardLimits    []uint64            `json:"shard_limits_rps"`
	SlotMapVersion int32               `json:"slot_map_version"`
	KState         []float64           `json:"k_state,omitempty"`
	NodeWeights    []UDPNodeWeightJSON `json:"node_weights,omitempty"`
}

func MarshalEpochPayload(slotVersion int32, limits *UDPControlLimits, weights []UDPNodeWeight) ([]byte, error) {
	if limits == nil {
		return []byte("{}"), nil
	}
	p := EpochPayloadJSON{
		SlotMapVersion: slotVersion,
		ShardLimits:    make([]uint64, limits.NumShards),
		NodeWeights:    NodeWeightsToJSON(weights),
	}
	for i := uint8(0); i < limits.NumShards; i++ {
		p.ShardLimits[i] = limits.Limits[i]
	}
	return json.Marshal(p)
}
