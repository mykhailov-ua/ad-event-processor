package probecluster

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"ad-event-processor/internal/domain"
)

const DefaultScoreThreshold = 65

type Tuple struct {
	CanvasHash [16]byte
	AudioHash  [16]byte
	WebGLHash  [16]byte
	TLSJA3     string
	TLSJA4     string
	TCPSig     uint32
	TCPSigSet  uint8
}

type Policy struct {
	MinSessions    int
	MinCampaigns   int
	ScoreThreshold uint8
}

func DefaultPolicy() Policy {
	return Policy{
		MinSessions:    5,
		MinCampaigns:   3,
		ScoreThreshold: DefaultScoreThreshold,
	}
}

type State struct {
	SessionCount  int64
	CampaignCount int64
	VerifyCount   int64
	AvgProbeScore uint8
}

func TupleFromEvent(evt *domain.Event, snap domain.AntifraudSnapshot) Tuple {
	var out Tuple
	if evt == nil {
		return out
	}
	if snap.CanvasHash[0] != 0 {
		out.CanvasHash = snap.CanvasHash
	}
	if snap.AudioHash[0] != 0 {
		out.AudioHash = snap.AudioHash
	}
	if snap.WebGLHash[0] != 0 {
		out.WebGLHash = snap.WebGLHash
	}
	out.TLSJA3 = evt.TLSJA3
	out.TLSJA4 = evt.TLSJA4
	out.TCPSig = evt.TCPSig
	out.TCPSigSet = evt.TCPSigSet
	return out
}

func ClusterID(secret []byte, t Tuple) [16]byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(t.CanvasHash[:])
	_, _ = mac.Write(t.AudioHash[:])
	_, _ = mac.Write(t.WebGLHash[:])
	_, _ = mac.Write([]byte(t.TLSJA3))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(t.TLSJA4))
	_, _ = mac.Write([]byte{0})
	if t.TCPSigSet != 0 {
		var sig [4]byte
		sig[0] = byte(t.TCPSig >> 24)
		sig[1] = byte(t.TCPSig >> 16)
		sig[2] = byte(t.TCPSig >> 8)
		sig[3] = byte(t.TCPSig)
		_, _ = mac.Write(sig[:])
	}
	sum := mac.Sum(nil)
	var out [16]byte
	copy(out[:], sum[:16])
	return out
}

func ClusterIDHex(id [16]byte) string {
	return hex.EncodeToString(id[:])
}

func ShouldRouteSandbox(s State, p Policy) bool {
	if p.MinSessions <= 0 || p.MinCampaigns <= 0 {
		return false
	}
	if s.SessionCount < int64(p.MinSessions) {
		return false
	}
	if s.CampaignCount < int64(p.MinCampaigns) {
		return false
	}
	if p.ScoreThreshold > 0 && s.AvgProbeScore < p.ScoreThreshold {
		return false
	}
	return true
}

func Hash16Hex(h [16]byte) string {
	if h[0] == 0 {
		return ""
	}
	return hex.EncodeToString(h[:])
}
