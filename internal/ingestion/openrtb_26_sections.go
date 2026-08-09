package ingestion

// openrtb26Sections holds first-hit byte offsets for top-level OpenRTB 2.6
// object keys. Values are -1 when absent. Populated by scanOpenRTB26Payload.
type openrtb26Sections struct {
	imp    int
	device int
	site   int
	app    int
	user   int
	source int
	dooh   int
}

func asciiUpperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}
