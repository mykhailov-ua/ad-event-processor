package ingestion

import "unicode/utf8"

func utf8ValidBytes(b []byte) bool {
	return utf8.Valid(b)
}
