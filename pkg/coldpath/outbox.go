package coldpath

import (
	"encoding/json"
	"log/slog"
)

func UnmarshalStrict[T any](payload []byte) (T, error) {
	var p T
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid outbox payload", "error", err)
		return p, err
	}
	return p, nil
}

func UnmarshalLenient[T any](payload []byte) T {
	var p T
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Warn("invalid outbox payload", "error", err)
	}
	return p
}
