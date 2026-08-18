package ingestion

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
