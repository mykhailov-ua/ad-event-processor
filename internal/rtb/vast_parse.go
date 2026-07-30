package rtb

import (
	"bytes"

	"espx/internal/rtb/pb"
)

const (
	vastTagVAST       = "VAST"
	vastTagAd         = "Ad"
	vastTagInLine     = "InLine"
	vastTagCreative   = "Creative"
	vastTagLinear     = "Linear"
	vastTagDuration   = "Duration"
	vastTagMediaFiles = "MediaFiles"
	vastTagMediaFile  = "MediaFile"
)

func ParseVASTXML(xml []byte) (*pb.VastDocument, error) {
	if len(xml) == 0 {
		return nil, ErrVASTMalformed
	}
	xml = bytes.TrimSpace(xml)
	if bytes.HasPrefix(xml, []byte("<?")) {
		if end := bytes.Index(xml, []byte("?>")); end >= 0 {
			xml = bytes.TrimSpace(xml[end+2:])
		}
	}
	doc := &pb.VastDocument{}
	p := vastParser{xml: xml}
	if !p.scanDocument(doc) {
		return nil, ErrVASTMalformed
	}
	if len(doc.Ads) == 0 {
		return nil, ErrVASTNoAds
	}
	return doc, nil
}

func MarshalVASTDocument(doc *pb.VastDocument) ([]byte, error) {
	if doc == nil {
		return nil, ErrVASTMalformed
	}
	return doc.MarshalVT()
}

func UnmarshalVASTDocument(wire []byte) (*pb.VastDocument, error) {
	if len(wire) == 0 {
		return nil, ErrVASTMalformed
	}
	doc := &pb.VastDocument{}
	if err := doc.UnmarshalVT(wire); err != nil {
		return nil, err
	}
	return doc, nil
}

func VASTDurationSec(doc *pb.VastDocument) uint32 {
	if doc == nil {
		return 0
	}
	for _, ad := range doc.Ads {
		if ad == nil {
			continue
		}
		if ad.Inline != nil {
			if d := vastCreativeDuration(ad.Inline.Creatives); d > 0 {
				return d
			}
		}
		if ad.Wrapper != nil {
			if d := vastCreativeDuration(ad.Wrapper.Creatives); d > 0 {
				return d
			}
		}
	}
	return 0
}

func vastCreativeDuration(creatives []*pb.VastCreative) uint32 {
	for _, cr := range creatives {
		if cr == nil || cr.Linear == nil {
			continue
		}
		if cr.Linear.DurationSec > 0 {
			return cr.Linear.DurationSec
		}
	}
	return 0
}

type vastParser struct {
	xml []byte
	i   int
}

func (p *vastParser) scanDocument(doc *pb.VastDocument) bool {
	if !p.findOpenTag(vastTagVAST) {
		return false
	}
	ver := p.attrValue("version")
	if len(ver) > 0 {
		doc.Version = append(doc.Version[:0], ver...)
	}
	for p.i < len(p.xml) {
		if p.atCloseTag(vastTagVAST) {
			p.i += len("</") + len(vastTagVAST) + 1
			return true
		}
		if !p.findOpenTag(vastTagAd) {
			p.advancePastTag()
			continue
		}
		ad := &pb.VastAd{}
		id := p.attrValue("id")
		if len(id) > 0 {
			ad.Id = append(ad.Id[:0], id...)
		}
		if p.scanAdBody(ad) {
			doc.Ads = append(doc.Ads, ad)
		}
	}
	return len(doc.Ads) > 0
}

func (p *vastParser) scanAdBody(ad *pb.VastAd) bool {
	for p.i < len(p.xml) {
		if p.atCloseTag(vastTagAd) {
			p.i += len("</") + len(vastTagAd) + 1
			return ad.Inline != nil || ad.Wrapper != nil
		}
		if p.findOpenTag(vastTagInLine) {
			inline := &pb.VastInline{}
			if p.scanInlineBody(inline) {
				ad.Inline = inline
			}
			continue
		}
		p.advancePastTag()
	}
	return false
}

func (p *vastParser) scanInlineBody(inline *pb.VastInline) bool {
	for p.i < len(p.xml) {
		if p.atCloseTag(vastTagInLine) {
			p.i += len("</") + len(vastTagInLine) + 1
			return len(inline.Creatives) > 0
		}
		if p.findOpenTag(vastTagCreative) {
			cr := &pb.VastCreative{}
			id := p.attrValue("id")
			if len(id) > 0 {
				cr.Id = append(cr.Id[:0], id...)
			}
			if p.scanCreativeBody(cr) {
				inline.Creatives = append(inline.Creatives, cr)
			}
			continue
		}
		p.advancePastTag()
	}
	return false
}

func (p *vastParser) scanCreativeBody(cr *pb.VastCreative) bool {
	for p.i < len(p.xml) {
		if p.atCloseTag(vastTagCreative) {
			p.i += len("</") + len(vastTagCreative) + 1
			return cr.Linear != nil
		}
		if p.findOpenTag(vastTagLinear) {
			linear := &pb.VastLinear{}
			if p.scanLinearBody(linear) {
				cr.Linear = linear
			}
			continue
		}
		p.advancePastTag()
	}
	return false
}

func (p *vastParser) scanLinearBody(linear *pb.VastLinear) bool {
	for p.i < len(p.xml) {
		if p.atCloseTag(vastTagLinear) {
			p.i += len("</") + len(vastTagLinear) + 1
			return linear.DurationSec > 0 || len(linear.MediaFiles) > 0
		}
		if p.findOpenTag(vastTagDuration) {
			body := p.readTextElement(vastTagDuration)
			linear.DurationSec = parseVASTDuration(body)
			continue
		}
		if p.findOpenTag(vastTagMediaFile) {
			mf := p.readMediaFile()
			if mf != nil {
				linear.MediaFiles = append(linear.MediaFiles, mf)
			}
			continue
		}
		p.advancePastTag()
	}
	return false
}

func (p *vastParser) readMediaFile() *pb.VastMediaFile {
	mime := p.attrValue("type")
	width := parseVASTUint(p.attrValue("width"))
	height := parseVASTUint(p.attrValue("height"))
	bitrate := parseVASTUint(p.attrValue("bitrate"))
	body := p.readTextElement(vastTagMediaFile)
	if len(body) == 0 && len(mime) == 0 {
		return nil
	}
	mf := &pb.VastMediaFile{
		Width:   width,
		Height:  height,
		Bitrate: bitrate,
	}
	if len(mime) > 0 {
		mf.MimeType = append(mf.MimeType[:0], mime...)
	}
	if len(body) > 0 {
		mf.Uri = append(mf.Uri[:0], body...)
	}
	return mf
}

func (p *vastParser) findOpenTag(name string) bool {
	needle := []byte("<" + name)
	n := len(p.xml)
	for p.i < n {
		j := bytes.Index(p.xml[p.i:], needle)
		if j < 0 {
			return false
		}
		p.i += j
		if p.i > 0 && p.xml[p.i-1] == '/' {
			p.i++
			continue
		}
		after := p.i + len(needle)
		if after < n && vastNameChar(p.xml[after]) {
			p.i++
			continue
		}
		p.i = after
		return true
	}
	return false
}

func vastNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func (p *vastParser) atCloseTag(name string) bool {
	needle := []byte("</" + name + ">")
	if p.i+len(needle) > len(p.xml) {
		return false
	}
	return bytes.Equal(p.xml[p.i:p.i+len(needle)], needle)
}

func (p *vastParser) attrValue(name string) []byte {
	needle := []byte(name + "=\"")
	start := bytes.Index(p.xml[p.i:], needle)
	if start < 0 {
		needle = []byte(name + "='")
		start = bytes.Index(p.xml[p.i:], needle)
		if start < 0 {
			return nil
		}
	}
	start += p.i + len(needle)
	end := start
	quote := p.xml[start-1]
	for end < len(p.xml) && p.xml[end] != quote {
		end++
	}
	return p.xml[start:end]
}

func (p *vastParser) readTextElement(tag string) []byte {
	closeSelf := bytes.IndexByte(p.xml[p.i:], '>')
	if closeSelf < 0 {
		return nil
	}
	contentStart := p.i + closeSelf + 1
	closeTag := []byte("</" + tag + ">")
	closeAt := bytes.Index(p.xml[contentStart:], closeTag)
	if closeAt < 0 {
		return nil
	}
	body := p.xml[contentStart : contentStart+closeAt]
	p.i = contentStart + closeAt + len(closeTag)
	return bytes.TrimSpace(body)
}

func (p *vastParser) advancePastTag() {
	if p.i < len(p.xml) && p.xml[p.i] == '<' {
		p.i++
	}
	for p.i < len(p.xml) && p.xml[p.i] != '<' {
		p.i++
	}
}

func (p *vastParser) skipToNextTag() {
	p.advancePastTag()
}

func parseVASTDuration(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	var h, m, s uint32
	part := 0
	val := uint32(0)
	for i := 0; i <= len(b); i++ {
		if i < len(b) && b[i] >= '0' && b[i] <= '9' {
			val = val*10 + uint32(b[i]-'0')
			continue
		}
		switch part {
		case 0:
			h = val
		case 1:
			m = val
		case 2:
			s = val
		}
		part++
		val = 0
		if i < len(b) && b[i] != ':' {
			break
		}
	}
	return h*3600 + m*60 + s
}

func parseVASTUint(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	var v uint32
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			break
		}
		v = v*10 + uint32(c-'0')
	}
	return v
}
