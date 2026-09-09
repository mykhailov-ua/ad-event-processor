package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLanderListWhere_hostingExternal(t *testing.T) {
	where, args := buildLanderListWhere(ListLandersFilter{Hosting: "external"})
	assert.Equal(t, " WHERE l.hosted_asset_id IS NULL AND COALESCE(l.url, '') <> ''", where)
	assert.Empty(t, args)
}

func TestBuildLanderListWhere_searchUsesSinglePattern(t *testing.T) {
	where, args := buildLanderListWhere(ListLandersFilter{Search: "demo"})
	assert.Contains(t, where, "ILIKE $1")
	assert.Contains(t, where, "ILIKE $2")
	assert.Contains(t, where, "ILIKE $3")
	assert.Equal(t, []any{"%demo%", "%demo%", "%demo%"}, args)
}

func TestBuildLanderListWhere_holdoutCombinesHostingAndSearch(t *testing.T) {
	where, args := buildLanderListWhere(ListLandersFilter{Hosting: "hosted", Search: "lp"})
	assert.Contains(t, where, "hosted_asset_id IS NOT NULL")
	assert.Contains(t, where, "ILIKE $1")
	assert.Len(t, args, 3)
}
