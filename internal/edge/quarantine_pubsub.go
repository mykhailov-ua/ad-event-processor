package edge

import (
	"encoding/json"
	"fmt"
)

const FraudQuarantineChannel = "fraud:quarantine"

type FraudQuarantinePayload struct {
	IPs []string `json:"ips"`
}


func MarshalFraudQuarantinePayload(ips []string) (string, error) {
	if len(ips) == 0 {
		return "", fmt.Errorf("fraud quarantine payload: empty ip list")
	}
	out := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("fraud quarantine payload: empty ip list")
	}
	raw, err := json.Marshal(FraudQuarantinePayload{IPs: out})
	if err != nil {
		return "", fmt.Errorf("marshal fraud quarantine payload: %w", err)
	}
	return string(raw), nil
}


func ParseFraudQuarantinePayload(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	if raw[0] == '{' {
		var payload FraudQuarantinePayload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("unmarshal fraud quarantine payload: %w", err)
		}
		return payload.IPs, nil
	}
	return []string{raw}, nil
}
