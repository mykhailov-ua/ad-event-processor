package wal

// ScanDedupReady visits up to max records with WalFlagDedupReady starting at fromSeq.
// fn returns false to stop early (backpressure). Returns the number of records visited.
func (w *WAL) ScanDedupReady(fromSeq uint64, max int, fn func(seq uint64, factorU [32]byte) bool) int {
	if max <= 0 || fn == nil {
		return 0
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	visited := 0
	pos := int64(0)
	writePos := w.writePos.Load()
	for visited < max && pos < writePos {
		if len(w.mmap) <= int(pos)+HeaderSize {
			break
		}
		hdr := readHeaderWithFactor(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			break
		}
		if !hdr.Has(WalFlagAppended) {
			break
		}
		payloadLen := int(hdr.PayloadLen)
		recordLen := int64(HeaderSize + payloadLen)
		if pos+recordLen > writePos {
			break
		}
		if hdr.Seq < fromSeq {
			pos += recordLen
			continue
		}
		if !hdr.Has(WalFlagDedupReady) {
			break
		}
		if !fn(hdr.Seq, hdr.FactorU) {
			break
		}
		visited++
		pos += recordLen
	}
	return visited
}
