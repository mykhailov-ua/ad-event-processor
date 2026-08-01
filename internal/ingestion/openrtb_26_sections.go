package ingestion

import "bytes"

type openrtb26Sections struct {
	imp    int
	device int
	site   int
	app    int
	user   int
	source int
	dooh   int
}

func locateOpenRTB26Sections(payload []byte) openrtb26Sections {
	return openrtb26Sections{
		imp:    bytes.Index(payload, openrtbKeyImp),
		device: bytes.Index(payload, openrtbKeyDevice),
		site:   bytes.Index(payload, openrtbKeySite),
		app:    bytes.Index(payload, openrtbKeyApp),
		user:   bytes.Index(payload, openrtbKeyUser),
		source: bytes.Index(payload, openrtbKeySource),
		dooh:   bytes.Index(payload, openrtbKeyDOOH),
	}
}

func asciiUpperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}
