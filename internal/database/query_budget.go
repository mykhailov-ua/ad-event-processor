package database

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type QueryBudgetPoint struct {
	N       int
	Queries int64
}

func MeasureQueryBudget(counter *QueryCounter, fn func()) int64 {
	if counter == nil {
		panic("query budget counter is nil")
	}
	counter.Reset()
	fn()
	return counter.Snapshot()
}

// AssertQueryBudgetSublinear fails when PG query count grows ~linearly with input size N.
// maxQueriesPerItem is the ceiling on ceil(deltaQueries / deltaN) between consecutive samples.
func AssertQueryBudgetSublinear(t *testing.T, label string, points []QueryBudgetPoint, maxQueriesPerItem int64) {
	t.Helper()
	require.GreaterOrEqual(t, len(points), 2, "query_budget %s: need at least two samples", label)
	for i := 1; i < len(points); i++ {
		prev := points[i-1]
		cur := points[i]
		require.Greater(t, cur.N, prev.N, "query_budget %s: samples must be sorted by increasing N", label)
		deltaN := int64(cur.N - prev.N)
		deltaQ := cur.Queries - prev.Queries
		if deltaQ < 0 {
			deltaQ = 0
		}
		slope := divCeil(deltaQ, deltaN)
		t.Logf(
			"query_budget %s: N %d->%d queries %d->%d deltaN=%d deltaQ=%d slope=%d max_per_item=%d",
			label, prev.N, cur.N, prev.Queries, cur.Queries, deltaN, deltaQ, slope, maxQueriesPerItem,
		)
		if slope > maxQueriesPerItem {
			require.FailNow(t, fmt.Sprintf(
				"query_budget %s: N+1 slope %d exceeds max %d per item (N %d->%d, queries %d->%d)",
				label, slope, maxQueriesPerItem, prev.N, cur.N, prev.Queries, cur.Queries,
			))
		}
	}
}

func divCeil(num, den int64) int64 {
	if den <= 0 {
		return num
	}
	if num <= 0 {
		return 0
	}
	return (num + den - 1) / den
}
