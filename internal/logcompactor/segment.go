package logcompactor

const readySuffix = ".log.zst.ready"

type compactStats struct {
	OriginalCount int64
	KeptCount     int64
}
