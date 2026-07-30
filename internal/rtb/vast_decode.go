package rtb

import (
	"unsafe"

	"espx/internal/rtb/pb"
)

func DecodeVASTWire(wire []byte, doc *pb.VastDocument) error {
	if doc == nil || len(wire) == 0 {
		return ErrVASTMalformed
	}
	return doc.UnmarshalVT(wire)
}

func VASTMediaMIME(doc *pb.VastDocument) string {
	if doc == nil {
		return ""
	}
	for _, ad := range doc.Ads {
		if ad == nil || ad.Inline == nil {
			continue
		}
		for _, cr := range ad.Inline.Creatives {
			if cr == nil || cr.Linear == nil {
				continue
			}
			for _, mf := range cr.Linear.MediaFiles {
				if mf != nil && len(mf.MimeType) > 0 {
					return vastBytesString(mf.MimeType)
				}
			}
		}
	}
	return ""
}

func vastBytesString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

func PrepareCreativeVASTWire(c *CreativeData) ([]byte, uint32, error) {
	if c == nil {
		return nil, 0, ErrVASTMalformed
	}
	if len(c.VASTWire) > 0 {
		doc, err := UnmarshalVASTDocument(c.VASTWire)
		if err != nil {
			return nil, 0, err
		}
		dur := c.DurationSec
		if dur == 0 {
			dur = VASTDurationSec(doc)
		}
		return c.VASTWire, dur, nil
	}
	if len(c.VASTXML) == 0 {
		return nil, c.DurationSec, nil
	}
	doc, err := ParseVASTXML(c.VASTXML)
	if err != nil {
		return nil, 0, err
	}
	wire, err := MarshalVASTDocument(doc)
	if err != nil {
		return nil, 0, err
	}
	dur := c.DurationSec
	if dur == 0 {
		dur = VASTDurationSec(doc)
	}
	return wire, dur, nil
}
