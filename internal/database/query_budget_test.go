package database

import (
	"testing"
)

func TestAssertQueryBudgetSublinear_passesFlatAndSublinear(t *testing.T) {
	t.Parallel()
	AssertQueryBudgetSublinear(t, "flat", []QueryBudgetPoint{
		{N: 2, Queries: 10},
		{N: 10, Queries: 11},
	}, 2)
	AssertQueryBudgetSublinear(t, "sublinear", []QueryBudgetPoint{
		{N: 2, Queries: 8},
		{N: 5, Queries: 10},
		{N: 10, Queries: 12},
	}, 2)
}

func TestAssertQueryBudgetSublinear_holdoutFailsLinear(t *testing.T) {
	t.Parallel()
	points := []QueryBudgetPoint{
		{N: 2, Queries: 10},
		{N: 10, Queries: 50},
	}
	deltaN := int64(points[1].N - points[0].N)
	deltaQ := points[1].Queries - points[0].Queries
	slope := divCeil(deltaQ, deltaN)
	if slope <= 2 {
		t.Fatalf("holdout broken: linear N+1 slope=%d should exceed max_per_item=2", slope)
	}
}
