package coldpath

import (
	"encoding/json"
	"fmt"
)

func MarshalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return b, nil
}

func UnmarshalJSON(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}
	return nil
}

func RedactStripeWebhookPayload(payload []byte) ([]byte, error) {
	var redacted map[string]any
	if err := json.Unmarshal(payload, &redacted); err != nil {
		return nil, fmt.Errorf("unmarshal stripe webhook payload: %w", err)
	}
	delete(redacted, "client_secret")
	delete(redacted, "customer_details")
	return MarshalJSON(redacted)
}
