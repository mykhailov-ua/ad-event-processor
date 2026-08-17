package coldpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	t.Parallel()
	assert.True(t, Contains([]string{"a", "b"}, "b"))
	assert.False(t, Contains([]string{"a"}, "z"))
}

func TestFilterSlice(t *testing.T) {
	t.Parallel()
	out := FilterSlice([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })
	assert.Equal(t, []int{2, 4}, out)
}

func TestUniqueSlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"a", "b"}, UniqueSlice([]string{"", "a", "b", "a", ""}))
}

func TestAppendUnique(t *testing.T) {
	t.Parallel()
	slice := []string{"a"}
	slice = AppendUnique(slice, "b")
	assert.Equal(t, []string{"a", "b"}, slice)
	slice = AppendUnique(slice, "a")
	assert.Equal(t, []string{"a", "b"}, slice)
}

func TestPtr(t *testing.T) {
	t.Parallel()
	v := Ptr(42)
	assert.NotNil(t, v)
	assert.Equal(t, 42, *v)
}
