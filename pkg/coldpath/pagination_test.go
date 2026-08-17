package coldpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginatedList_countBeforeList(t *testing.T) {
	t.Parallel()
	var callOrder []string
	_, total, err := PaginatedList(
		func() (int64, error) {
			callOrder = append(callOrder, "count")
			return 2, nil
		},
		func() ([]int, error) {
			callOrder = append(callOrder, "list")
			return []int{1, 2}, nil
		},
		func(v int) int { return v * 10 },
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []string{"count", "list"}, callOrder)
}

func TestPaginatedList_skipsListWhenCountZero(t *testing.T) {
	t.Parallel()
	var listCalls int
	items, total, err := PaginatedList(
		func() (int64, error) { return 0, nil },
		func() ([]int, error) {
			listCalls++
			return []int{99}, nil
		},
		func(v int) int { return v },
	)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, items)
	assert.Zero(t, listCalls, "list query must not run when count is zero")
}

func TestPaginatedQuery_skipsListWhenCountZero(t *testing.T) {
	t.Parallel()
	var listCalls int
	rows, total, err := PaginatedQuery(
		func() (int64, error) { return 0, nil },
		func() ([]string, error) {
			listCalls++
			return []string{"row"}, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, rows)
	assert.Zero(t, listCalls)
}
