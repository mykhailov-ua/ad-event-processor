package fraud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"ad-event-processor/internal/reports"
)

const fraudEvidencePackVersion = "1"

var errFraudEvidencePackBadSignature = errors.New("fraud evidence pack signature mismatch")

func aggregateFraudEvidenceSignals(rows []reports.FraudEvidenceFraudRowDTO) reports.FraudEvidenceSignalsDTO {
	var out reports.FraudEvidenceSignalsDTO
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

func BuildSignedFraudEvidencePack(secret []byte, pack reports.FraudEvidencePackDTO) (reports.FraudEvidencePackDTO, error) {
	pack.Version = fraudEvidencePackVersion
	if pack.GeneratedAt == "" {
		pack.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	unsigned := pack
	unsigned.Signature = ""
	unsigned.DigestSHA256 = ""
	body, err := json.Marshal(unsigned)
	if err != nil {
		return reports.FraudEvidencePackDTO{}, err
	}
	sum := sha256.Sum256(body)
	pack.DigestSHA256 = hex.EncodeToString(sum[:])
	pack.Signature = signFraudEvidencePack(secret, body)
	return pack, nil
}

func VerifyFraudEvidencePackSignature(secret []byte, pack reports.FraudEvidencePackDTO) error {
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
