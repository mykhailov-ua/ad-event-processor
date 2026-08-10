package ingestion

const (
	MaxJSONDepth     = 16
	OrtbMaxJSONDepth = 32
	// MaxWSkip caps bytes consumed by a single JSON whitespace skip (hostile WS bomb).
	MaxWSkip = 256
	// OrtbScanMaxBytes caps top-level OpenRTB 2.6 key scan prefix (quote-dense bomb).
	OrtbScanMaxBytes = 262144
	// OrtbMaxQuoteChecks caps quote examinations per OpenRTB 2.6 top-level scan.
	OrtbMaxQuoteChecks = 65536
	// MaxJSONKeyPairs caps key:value pairs per JSON document (track + ORTB3 FSM).
	MaxJSONKeyPairs = 10000
)

var ErrMalformed = errMalformedJSON
