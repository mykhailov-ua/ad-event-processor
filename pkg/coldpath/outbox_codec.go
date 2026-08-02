package coldpath

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

// OutboxProtoMagic prefixes protobuf-encoded outbox payloads stored in BYTEA/JSONB.
const OutboxProtoMagic byte = 0x1f

var (
	outboxCodecMu  sync.RWMutex
	outboxEncoders = map[reflect.Type]func(any) ([]byte, error){}
	outboxDecoders = map[reflect.Type]func([]byte) (any, error){}
)

func RegisterOutboxCodec[T any](encode func(T) ([]byte, error), decode func([]byte) (T, error)) {
	var zero T
	t := reflect.TypeOf(zero)
	outboxCodecMu.Lock()
	defer outboxCodecMu.Unlock()
	outboxEncoders[t] = func(v any) ([]byte, error) {
		return encode(v.(T))
	}
	outboxDecoders[t] = func(b []byte) (any, error) {
		return decode(b)
	}
}

func MarshalOutbox(v any) ([]byte, error) {
	t := reflect.TypeOf(v)
	outboxCodecMu.RLock()
	enc, ok := outboxEncoders[t]
	outboxCodecMu.RUnlock()
	if ok {
		body, err := enc(v)
		if err != nil {
			return nil, err
		}
		return wrapOutboxProto(body), nil
	}
	return MarshalJSON(v)
}

func wrapOutboxProto(body []byte) []byte {
	out := make([]byte, 1+len(body))
	out[0] = OutboxProtoMagic
	copy(out[1:], body)
	return out
}

func IsOutboxProto(payload []byte) bool {
	return len(payload) > 0 && payload[0] == OutboxProtoMagic
}

func OutboxProtoBody(payload []byte) []byte {
	if !IsOutboxProto(payload) {
		return payload
	}
	return payload[1:]
}

func UnmarshalStrict[T any](payload []byte) (T, error) {
	var zero T
	if IsOutboxProto(payload) {
		body := OutboxProtoBody(payload)
		outboxCodecMu.RLock()
		dec, ok := outboxDecoders[reflect.TypeOf(zero)]
		outboxCodecMu.RUnlock()
		if ok {
			v, err := dec(body)
			if err != nil {
				slog.Error("invalid outbox proto payload", "error", err)
				return zero, err
			}
			return v.(T), nil
		}
	}
	var p T
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid outbox payload", "error", err)
		return p, err
	}
	return p, nil
}

func UnmarshalLenient[T any](payload []byte) T {
	p, err := UnmarshalStrict[T](payload)
	if err != nil {
		slog.Warn("invalid outbox payload", "error", err)
	}
	return p
}

func MarshalProtoMessage(msg proto.Message) ([]byte, error) {
	b, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox proto: %w", err)
	}
	return b, nil
}

func UnmarshalProtoMessage(payload []byte, msg proto.Message) error {
	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("unmarshal outbox proto: %w", err)
	}
	return nil
}
