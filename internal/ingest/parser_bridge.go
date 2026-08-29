package ingest

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/parser"

	"github.com/google/uuid"
)

var (
	ErrMalformed           = parser.ErrMalformed
	MaxJSONDepth           = parser.MaxJSONDepth
	OrtbMaxJSONDepth       = parser.OrtbMaxJSONDepth
	MaxWSkip               = parser.MaxWSkip
	OrtbScanMaxBytes       = parser.OrtbScanMaxBytes
	OrtbMaxQuoteChecks     = parser.OrtbMaxQuoteChecks
	MaxJSONKeyPairs        = parser.MaxJSONKeyPairs
	MaxJSONTotalWSkip      = parser.MaxJSONTotalWSkip
	MaxJSONStringScanBytes = parser.MaxJSONStringScanBytes
	MaxJSONStringEscapes   = parser.MaxJSONStringEscapes
)

type jsonScanBudget struct {
	parser.ScanBudget
}

func (b *jsonScanBudget) consumeWS(n int) bool { return b.ScanBudget.ConsumeWS(n) }
func (b *jsonScanBudget) consumeStrByte() bool { return b.ScanBudget.ConsumeStrByte() }
func (b *jsonScanBudget) consumeEscape() bool  { return b.ScanBudget.ConsumeEscape() }
func (b *jsonScanBudget) consumeKeyPair() bool { return b.ScanBudget.ConsumeKeyPair() }

func configureJSONParseSecurity(cfg *config.Config) {
	parser.ConfigureSecurity(cfg)
}

func newJSONScanBudget() jsonScanBudget {
	return jsonScanBudget{ScanBudget: parser.NewScanBudget()}
}

func skipJSONWSBudget(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	return parser.SkipWSBudget(data, i, n, &b.ScanBudget)
}

func scanJSONStringEnd(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	return parser.ScanStringEnd(data, i, n, &b.ScanBudget)
}

func scanJSONLiteralStringEnd(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	return parser.ScanLiteralStringEnd(data, i, n, &b.ScanBudget)
}

func skipJSONValueBudget(data []byte, start int, bud *jsonScanBudget) (int, error) {
	return parser.SkipValueBudget(data, start, &bud.ScanBudget)
}

func skipJSONValueBudgetDepth(data []byte, start int, bud *jsonScanBudget, maxDepth int) (int, error) {
	return parser.SkipValueBudgetDepth(data, start, &bud.ScanBudget, maxDepth)
}

func jsonTrackKeyOK(key []byte) bool { return parser.TrackKeyOK(key) }

func ParseUUID(b []byte, dst *uuid.UUID) bool { return parser.ParseUUID(b, dst) }

func loadU32(b []byte) uint32 { return parser.LoadU32(b) }

func loadU64(b []byte) uint64 { return parser.LoadU64(b) }

func appendJSONString(dst []byte, s []byte) []byte { return parser.AppendJSONString(dst, s) }

func marshalExtra(dst []byte, keys, values [][]byte) []byte {
	return parser.MarshalExtra(dst, keys, values)
}

func isDelimiter(b byte) bool { return parser.IsDelimiter(b) }
