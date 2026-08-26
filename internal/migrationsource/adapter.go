package migrationsource

import "fmt"

// Adapter parses a foreign export payload into a NormalizedBundle.
type Adapter interface {
	Kind() SourceKind
	Parse(payload []byte) (NormalizedBundle, error)
}

// Parse dispatches payload parsing by source kind.
func Parse(kind SourceKind, payload []byte) (NormalizedBundle, error) {
	switch kind {
	case SourceKindKeitaroJSON:
		return ParseKeitaroJSON(payload)
	case SourceKindBinomJSON:
		return NormalizedBundle{}, fmt.Errorf("binom_json adapter not implemented")
	case SourceKindNativeV1:
		return NormalizedBundle{}, fmt.Errorf("native_v1 uses campaign export import path")
	default:
		return NormalizedBundle{}, fmt.Errorf("unsupported source_kind %q", kind)
	}
}
