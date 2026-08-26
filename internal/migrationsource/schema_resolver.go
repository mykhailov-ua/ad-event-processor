package migrationsource

import (
	"strings"
)

// SchemaResolver maps foreign traffic source labels to bundled integration slugs.
type SchemaResolver struct {
	keitaro map[string]SourceEntry
	binom   map[string]SourceEntry
}

// NewSchemaResolver builds resolvers from loaded maps.
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

// ResolveKeitaro returns bundled slug and UI template id for a Keitaro source label.
func (r *SchemaResolver) ResolveKeitaro(name string) (SourceEntry, bool) {
	if r == nil {
		return SourceEntry{}, false
	}
	row, ok := r.keitaro[normalizeSourceName(name)]
	return row, ok
}

func normalizeSourceName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
