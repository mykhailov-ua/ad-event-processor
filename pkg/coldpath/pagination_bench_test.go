package coldpath

import "testing"

func listFirstPagination(count func() (int64, error), list func() ([]int, error)) ([]int, int64, error) {
	rows, err := list()
	if err != nil {
		return nil, 0, err
	}
	total, err := count()
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func BenchmarkPagination_listFirst_zeroTotal(b *testing.B) {
	count := func() (int64, error) { return 0, nil }
	list := func() ([]int, error) { return nil, nil }
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := listFirstPagination(count, list)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPagination_countFirst_zeroTotal(b *testing.B) {
	count := func() (int64, error) { return 0, nil }
	list := func() ([]int, error) { return nil, nil }
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := PaginatedQuery(count, list)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPagination_listFirst_nonempty(b *testing.B) {
	rows := []int{1, 2, 3, 4, 5}
	count := func() (int64, error) { return int64(len(rows)), nil }
	list := func() ([]int, error) { return rows, nil }
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := listFirstPagination(count, list)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPagination_countFirst_nonempty(b *testing.B) {
	rows := []int{1, 2, 3, 4, 5}
	count := func() (int64, error) { return int64(len(rows)), nil }
	list := func() ([]int, error) { return rows, nil }
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := PaginatedQuery(count, list)
		if err != nil {
			b.Fatal(err)
		}
	}
}
