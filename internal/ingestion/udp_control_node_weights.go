package ingestion

import (
	"encoding/binary"
	"math"
)

const (
	udpProtocolVersion3   uint8 = 3
	UDPMaxNodeWeights           = 64
	UDPMaxNodeIDLen             = 63
	UDPNodeWeightStaleLag int64 = 2

	UDPFlagNodeWeights uint16 = 1 << 1

	udpProvOwnWindow           uint8 = 0
	udpProvNeighborMedian      uint8 = 1
	udpProvHistoricalDaily     uint8 = 2
	udpProvConservativeDefault uint8 = 3
)

// UDPNodeWeight is one peer entry in a control epoch (cold path only).
type UDPNodeWeight struct {
	NodeID     string
	Role       string
	Weight     float64
	Score      float64
	Provenance uint8
}

// UDPNodeWeightJSON is the §8 audit JSON shape for control_plane_epochs.
type UDPNodeWeightJSON struct {
	NodeID     string  `json:"node_id"`
	Role       string  `json:"role,omitempty"`
	Weight     float64 `json:"weight"`
	Score      float64 `json:"score"`
	Provenance string  `json:"provenance"`
}

// ProvenanceToUDPCode maps §0 provenance strings to on-wire codes.
func ProvenanceToUDPCode(provenance string) uint8 {
	switch provenance {
	case "own_window", "own":
		return udpProvOwnWindow
	case "neighbor_median", "neighbor":
		return udpProvNeighborMedian
	case "historical_daily", "historical":
		return udpProvHistoricalDaily
	default:
		return udpProvConservativeDefault
	}
}

// ProvenanceFromUDPCode maps on-wire codes back to §0 provenance strings.
func ProvenanceFromUDPCode(code uint8) string {
	switch code {
	case udpProvOwnWindow:
		return "own_window"
	case udpProvNeighborMedian:
		return "neighbor_median"
	case udpProvHistoricalDaily:
		return "historical_daily"
	default:
		return "conservative_default"
	}
}

// NodeWeightsToJSON converts live weights for Postgres payload_json.
func NodeWeightsToJSON(weights []UDPNodeWeight) []UDPNodeWeightJSON {
	if len(weights) == 0 {
		return nil
	}
	out := make([]UDPNodeWeightJSON, len(weights))
	for i, w := range weights {
		out[i] = UDPNodeWeightJSON{
			NodeID:     w.NodeID,
			Role:       w.Role,
			Weight:     w.Weight,
			Score:      w.Score,
			Provenance: ProvenanceFromUDPCode(w.Provenance),
		}
	}
	return out
}

// EqualizeNodeWeights assigns weight 1/N to each peer (stale epoch policy).
func EqualizeNodeWeights(weights []UDPNodeWeight) []UDPNodeWeight {
	if len(weights) == 0 {
		return nil
	}
	eq := 1.0 / float64(len(weights))
	out := make([]UDPNodeWeight, len(weights))
	for i, w := range weights {
		out[i] = w
		out[i].Weight = eq
	}
	return out
}

// NodeWeightsDrainFrozen reports whether routing must stop draining peers (§8 stale epoch).
func NodeWeightsDrainFrozen(channelStale bool, publisherEpoch, appliedEpoch int64) bool {
	if channelStale {
		return true
	}
	if publisherEpoch <= 0 || appliedEpoch <= 0 {
		return false
	}
	return publisherEpoch-appliedEpoch > UDPNodeWeightStaleLag
}

// EffectiveNodeWeights returns published weights or equalized weights when stale.
func EffectiveNodeWeights(weights []UDPNodeWeight, channelStale bool, publisherEpoch, appliedEpoch int64) []UDPNodeWeight {
	if len(weights) == 0 {
		return nil
	}
	if NodeWeightsDrainFrozen(channelStale, publisherEpoch, appliedEpoch) {
		return EqualizeNodeWeights(weights)
	}
	out := make([]UDPNodeWeight, len(weights))
	copy(out, weights)
	return out
}

func udpNodeWeightsEncodedLen(weights []UDPNodeWeight) int {
	if len(weights) == 0 {
		return 0
	}
	if len(weights) > UDPMaxNodeWeights {
		weights = weights[:UDPMaxNodeWeights]
	}
	n := 2
	for _, w := range weights {
		idLen := len(w.NodeID)
		if idLen > UDPMaxNodeIDLen {
			idLen = UDPMaxNodeIDLen
		}
		n += 1 + idLen + 8 + 8 + 1
	}
	return n
}

func udpEncodeNodeWeights(dst []byte, weights []UDPNodeWeight) int {
	if len(weights) == 0 {
		return 0
	}
	if len(weights) > UDPMaxNodeWeights {
		weights = weights[:UDPMaxNodeWeights]
	}
	need := 2
	for _, w := range weights {
		idLen := len(w.NodeID)
		if idLen > UDPMaxNodeIDLen {
			idLen = UDPMaxNodeIDLen
		}
		need += 1 + idLen + 8 + 8 + 1
	}
	if len(dst) < need {
		return 0
	}
	binary.LittleEndian.PutUint16(dst[0:2], uint16(len(weights)))
	off := 2
	for _, w := range weights {
		id := w.NodeID
		if len(id) > UDPMaxNodeIDLen {
			id = id[:UDPMaxNodeIDLen]
		}
		dst[off] = uint8(len(id))
		off++
		copy(dst[off:off+len(id)], id)
		off += len(id)
		binary.LittleEndian.PutUint64(dst[off:off+8], math.Float64bits(w.Weight))
		off += 8
		binary.LittleEndian.PutUint64(dst[off:off+8], math.Float64bits(w.Score))
		off += 8
		dst[off] = w.Provenance
		off++
	}
	return off
}

func udpDecodeNodeWeights(payload []byte, out *[]UDPNodeWeight) bool {
	if out == nil || len(payload) < 2 {
		return false
	}
	count := int(binary.LittleEndian.Uint16(payload[0:2]))
	if count == 0 {
		*out = nil
		return true
	}
	if count > UDPMaxNodeWeights {
		return false
	}
	off := 2
	weights := make([]UDPNodeWeight, 0, count)
	for i := 0; i < count; i++ {
		if off >= len(payload) {
			return false
		}
		idLen := int(payload[off])
		off++
		if idLen > UDPMaxNodeIDLen || off+idLen > len(payload) {
			return false
		}
		id := string(payload[off : off+idLen])
		off += idLen
		if off+17 > len(payload) {
			return false
		}
		weight := math.Float64frombits(binary.LittleEndian.Uint64(payload[off : off+8]))
		off += 8
		score := math.Float64frombits(binary.LittleEndian.Uint64(payload[off : off+8]))
		off += 8
		prov := payload[off]
		off++
		weights = append(weights, UDPNodeWeight{
			NodeID:     id,
			Weight:     weight,
			Score:      score,
			Provenance: prov,
		})
	}
	*out = weights
	return true
}

// EdgeControlEqualizeWeights reports whether edge must equalize peer weights (M5.3 conservative default).
func EdgeControlEqualizeWeights(stale, failOpen bool) bool {
	return stale && !failOpen
}

// EdgeControlDrainFrozen reports whether edge must freeze drain-oriented routing (M5.3 conservative default).
func EdgeControlDrainFrozen(stale, failOpen bool) bool {
	if failOpen {
		return false
	}
	return stale
}

// ControlFailOpenEnabled parses CONTROL_FAIL_OPEN (edge only; default false).
func ControlFailOpenEnabled(raw string) bool {
	return raw == "1" || raw == "true" || raw == "TRUE"
}

func hashNodeWeights(h interface {
	Write(p []byte) (n int, err error)
}, weights []UDPNodeWeight) {
	var buf [8]byte
	binary.LittleEndian.PutUint16(buf[:2], uint16(len(weights)))
	_, _ = h.Write(buf[:2])
	for _, w := range weights {
		id := w.NodeID
		if len(id) > UDPMaxNodeIDLen {
			id = id[:UDPMaxNodeIDLen]
		}
		buf[0] = uint8(len(id))
		_, _ = h.Write(buf[:1])
		_, _ = h.Write([]byte(id))
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(w.Weight))
		_, _ = h.Write(buf[:8])
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(w.Score))
		_, _ = h.Write(buf[:8])
		buf[0] = w.Provenance
		_, _ = h.Write(buf[:1])
	}
}
