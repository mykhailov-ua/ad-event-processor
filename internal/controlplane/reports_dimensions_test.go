package controlplane

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlacementReportQuery_usesColumnPredicate_holdout(t *testing.T) {
	require.Contains(t, placementReportQuery, "placement_id")
	require.NotContains(t, placementReportQuery, "JSONExtractString(payload")
}

func TestKeywordReportQuery_prefersDimensionColumns_holdout(t *testing.T) {
	require.Contains(t, keywordReportQuery, "nullIf(keyword, '')")
	require.True(t, strings.Contains(keywordReportQuery, clickhouseDimKeywordExpr),
		"keyword report must coalesce keyword column before JSONExtract fallback")
}
