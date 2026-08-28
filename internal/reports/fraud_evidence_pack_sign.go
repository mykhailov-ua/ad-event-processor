package reports

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

const fraudEvidencePackVersion = "1"

var errFraudEvidencePackBadSignature = errors.New("fraud evidence pack signature mismatch")

type FraudEvidenceTimelineRowDTO struct {
	EventType   string `json:"event_type"`
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	Country     string `json:"country,omitempty"`
	Sub1        string `json:"sub1,omitempty"`
}

type FraudEvidenceFraudRowDTO struct {
	EventType         string `json:"event_type"`
	CampaignID        string `json:"campaign_id"`
	PlacementID       string `json:"placement_id,omitempty"`
	FraudReason       string `json:"fraud_reason"`
	FraudScore        uint32 `json:"fraud_score"`
	LayerDesyncCount  uint8  `json:"layer_desync_count"`
	SilentRejectEvent bool   `json:"silent_reject_event"`
	CreatedAt         string `json:"created_at"`
}

type FraudEvidenceSignalsDTO struct {
	FraudReasons        []string `json:"fraud_reasons"`
	MaxFraudScore       uint32   `json:"max_fraud_score"`
	MaxLayerDesyncCount uint8    `json:"max_layer_desync_count"`
	SilentRejectEvents  int      `json:"silent_reject_events"`
}

type FraudEvidencePackDTO struct {
	Version      string                        `json:"version"`
	ClickID      string                        `json:"click_id"`
	CustomerID   string                        `json:"customer_id"`
	CampaignID   string                        `json:"campaign_id,omitempty"`
	GeneratedAt  string                        `json:"generated_at"`
	RangeFrom    string                        `json:"range_from"`
	RangeTo      string                        `json:"range_to"`
	Timeline     []FraudEvidenceTimelineRowDTO `json:"timeline"`
	FraudEvents  []FraudEvidenceFraudRowDTO    `json:"fraud_events"`
	Signals      FraudEvidenceSignalsDTO       `json:"signals"`
	DigestSHA256 string                        `json:"digest_sha256"`
	Signature    string                        `json:"signature"`
}

func aggregateFraudEvidenceSignals(rows []FraudEvidenceFraudRowDTO) FraudEvidenceSignalsDTO {
	var out FraudEvidenceSignalsDTO
	seen := make(map[string]struct{}, 8)
	for i := range rows {
		row := rows[i]
		if row.FraudScore > out.MaxFraudScore {
			out.MaxFraudScore = row.FraudScore
		}
		if row.LayerDesyncCount > out.MaxLayerDesyncCount {
			out.MaxLayerDesyncCount = row.LayerDesyncCount
		}
		if row.SilentRejectEvent {
			out.SilentRejectEvents++
		}
		for _, part := range splitFraudReasonParts(row.FraudReason) {
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			out.FraudReasons = append(out.FraudReasons, part)
		}
	}
	return out
}

func splitFraudReasonParts(reason string) []string {
	if reason == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i <= len(reason); i++ {
		if i == len(reason) || reason[i] == ',' {
			if i > start {
				part := reason[start:i]
				if part != "" {
					parts = append(parts, part)
				}
			}
			start = i + 1
		}
	}
	return parts
}

func signFraudEvidencePack(secret []byte, unsignedJSON []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(unsignedJSON)
	return hex.EncodeToString(mac.Sum(nil))
}

func BuildSignedFraudEvidencePack(secret []byte, pack FraudEvidencePackDTO) (FraudEvidencePackDTO, error) {
	pack.Version = fraudEvidencePackVersion
	if pack.GeneratedAt == "" {
		pack.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	unsigned := pack
	unsigned.Signature = ""
	unsigned.DigestSHA256 = ""
	body, err := json.Marshal(unsigned)
	if err != nil {
		return FraudEvidencePackDTO{}, err
	}
	sum := sha256.Sum256(body)
	pack.DigestSHA256 = hex.EncodeToString(sum[:])
	pack.Signature = signFraudEvidencePack(secret, body)
	return pack, nil
}

func VerifyFraudEvidencePackSignature(secret []byte, pack FraudEvidencePackDTO) error {
	unsigned := pack
	unsigned.Signature = ""
	unsigned.DigestSHA256 = ""
	body, err := json.Marshal(unsigned)
	if err != nil {
		return err
	}
	want := signFraudEvidencePack(secret, body)
	if !hmac.Equal([]byte(want), []byte(pack.Signature)) {
		return errFraudEvidencePackBadSignature
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != pack.DigestSHA256 {
		return errFraudEvidencePackBadSignature
	}
	return nil
}
