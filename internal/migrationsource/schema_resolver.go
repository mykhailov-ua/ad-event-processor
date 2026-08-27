package migrationsource

import (
	"strings"
)

type SchemaResolver struct {
	keitaro map[string]SourceEntry
	binom   map[string]SourceEntry
}

func NewSchemaResolver(m *Maps) *SchemaResolver {
	if m == nil {
		return &SchemaResolver{}
	}
	k := make(map[string]SourceEntry, len(m.KeitaroSources))
	for _, row := range m.KeitaroSources {
		name := strings.TrimSpace(row.KeitaroName)
		if name == "" {
			continue
		}
		k[normalizeSourceName(name)] = row
	}
	b := make(map[string]SourceEntry, len(m.BinomSources))
	for _, row := range m.BinomSources {
		name := strings.TrimSpace(row.BinomName)
		if name == "" {
			continue
		}
		b[normalizeSourceName(name)] = row
	}
	return &SchemaResolver{keitaro: k, binom: b}
}

func (r *SchemaResolver) ResolveKeitaro(name string) (SourceEntry, bool) {
	if r == nil {
		return SourceEntry{}, false
	}
	row, ok := r.keitaro[normalizeSourceName(name)]
	return row, ok
}

func (r *SchemaResolver) ResolveBinom(name string) (SourceEntry, bool) {
	if r == nil {
		return SourceEntry{}, false
	}
	row, ok := r.binom[normalizeSourceName(name)]
	return row, ok
}

func normalizeSourceName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
