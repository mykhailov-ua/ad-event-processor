package migrationsource

import "fmt"

type Adapter interface {
	Kind() SourceKind
	Parse(payload []byte) (NormalizedBundle, error)
}

func Parse(kind SourceKind, payload []byte) (NormalizedBundle, error) {
	switch kind {
	case SourceKindKeitaroJSON:
		return ParseKeitaroJSON(payload)
	case SourceKindBinomJSON:
		return ParseBinomJSON(payload)
	case SourceKindNativeV1:
		return NormalizedBundle{}, fmt.Errorf("native_v1 uses campaign export import path")
	default:
		return NormalizedBundle{}, fmt.Errorf("unsupported source_kind %q", kind)
	}
}
