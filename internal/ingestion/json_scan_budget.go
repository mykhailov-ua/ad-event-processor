package ingestion

const (
	// MaxJSONTotalWSkip caps cumulative whitespace skipped while parsing one JSON document.
	MaxJSONTotalWSkip = 4096
	// MaxJSONStringScanBytes caps bytes examined inside a single JSON string value.
	MaxJSONStringScanBytes = 65536
	// MaxJSONStringEscapes caps backslash escapes examined in one JSON string value.
	MaxJSONStringEscapes = 16384
)

type jsonScanBudget struct {
	wsLeft  int
	strLeft int
	escLeft int
}

func newJSONScanBudget() jsonScanBudget {
	return jsonScanBudget{
		wsLeft:  MaxJSONTotalWSkip,
		strLeft: MaxJSONStringScanBytes,
		escLeft: MaxJSONStringEscapes,
	}
}

func (b *jsonScanBudget) consumeWS(n int) bool {
	if b == nil {
		return true
	}
	b.wsLeft -= n
	return b.wsLeft >= 0
}

func (b *jsonScanBudget) consumeStrByte() bool {
	if b == nil {
		return true
	}
	b.strLeft--
	return b.strLeft >= 0
}

func (b *jsonScanBudget) consumeEscape() bool {
	if b == nil {
		return true
	}
	b.escLeft--
	return b.escLeft >= 0
}

func skipJSONWSBudget(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	skipped := 0
	for i < n && jsonWhitespace[data[i]] != 0 {
		if skipped >= MaxWSkip {
			return i, false
		}
		skipped++
		i++
	}
	if !b.consumeWS(skipped) {
		return i, false
	}
	return i, true
}

// jsonTrackKeyOK requires literal unescaped ASCII keys (no Unicode homoglyphs / escapes).
func jsonTrackKeyOK(key []byte) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		if c > 0x7f {
			return false
		}
	}
	return true
}
