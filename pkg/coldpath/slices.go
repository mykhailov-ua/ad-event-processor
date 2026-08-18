package coldpath

func Ptr[T any](v T) *T {
	return &v
}

func MapSlice[T, R any](in []T, fn func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

func Contains[T comparable](slice []T, v T) bool {
	for _, item := range slice {
		if item == v {
			return true
		}
	}
	return false
}

func FilterSlice[T any](in []T, keep func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func UniqueSlice[T comparable](in []T) []T {
	var zero T
	seen := make(map[T]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		if v == zero {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func AppendUnique[T comparable](slice []T, v T) []T {
	if Contains(slice, v) {
		return slice
	}
	return append(slice, v)
}

func KeyBy[T any, K comparable](slice []T, key func(T) (K, bool)) map[K]T {
	m := make(map[K]T, len(slice))
	for _, v := range slice {
		if k, ok := key(v); ok {
			m[k] = v
		}
	}
	return m
}

func KeyByValue[T any, K comparable, V any](slice []T, key func(T) K, val func(T) V) map[K]V {
	m := make(map[K]V, len(slice))
	for _, v := range slice {
		m[key(v)] = val(v)
	}
	return m
}

func PaginatedList[T, R any](
	count func() (int64, error),
	list func() ([]T, error),
	mapFn func(T) R,
) ([]R, int64, error) {
	total, err := count()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []R{}, 0, nil
	}
	rows, err := list()
	if err != nil {
		return nil, 0, err
	}
	return MapSlice(rows, mapFn), total, nil
}

func PaginatedQuery[T any](
	count func() (int64, error),
	list func() ([]T, error),
) ([]T, int64, error) {
	total, err := count()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []T{}, 0, nil
	}
	rows, err := list()
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
