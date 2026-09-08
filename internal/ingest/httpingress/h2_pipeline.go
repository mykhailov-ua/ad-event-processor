package httpingress

// CountCompleteH2Messages returns fully parsed HTTP/2 requests in data.
// st is the live connection state (Established, fingerprints); it is not mutated.
func CountCompleteH2Messages(data []byte, maxBody int64, st *H2ConnState) int {
	if len(data) == 0 {
		return 0
	}
	scratch := H2ConnState{}
	if st != nil {
		scratch = *st
	}
	if cap(scratch.HeaderBlock) == 0 {
		scratch.HeaderBlock = make([]byte, 0, 256)
	}

	count := 0
	offset := 0
	for offset < len(data) {
		n, req, _, _, err := ParseH2Ingress(data[offset:], &scratch, maxBody)
		if n <= 0 {
			break
		}
		offset += n
		if err == nil && len(req.Method) > 0 {
			count++
			continue
		}
		break
	}
	return count
}
