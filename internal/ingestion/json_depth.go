package ingestion

const (
	MaxJSONDepth     = 16
	OrtbMaxJSONDepth = 32

	MaxWSkip = 256

	OrtbScanMaxBytes = 262144

	OrtbMaxQuoteChecks = 65536

	MaxJSONKeyPairs = 10000
)

var ErrMalformed = errMalformedJSON
