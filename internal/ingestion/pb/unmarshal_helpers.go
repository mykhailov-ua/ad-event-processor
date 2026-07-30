package pb

func appendReuseBytes(dst [][]byte, src []byte) [][]byte {
	idx := len(dst)
	if idx < cap(dst) {
		dst = dst[:idx+1]
		dst[idx] = append(dst[idx][:0], src...)
		return dst
	}
	return append(dst, append([]byte(nil), src...))
}
