package domain

import "unsafe"

func FNV64Bytes(b []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := range b {
		h ^= uint64(b[i])
		h *= prime64
	}
	return h
}

func FNV64String(s string) uint64 {
	if s == "" {
		return 0
	}
	return FNV64Bytes(unsafe.Slice(unsafe.StringData(s), len(s)))
}
