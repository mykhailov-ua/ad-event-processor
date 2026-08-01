package openrtb

// MacroWire is the zero-alloc macro substitution view. All fields are caller-owned []byte.
type MacroWire struct {
	AuctionPrice []byte
	AuctionID    []byte
	BidID        []byte
	ImpID        []byte
	SeatID       []byte
}

// MacroContext is the cold-path string view (admin/tests). Prefer MacroWire on exchange glue.
type MacroContext struct {
	AuctionPrice string
	AuctionID    string
	BidID        string
	ImpID        string
	SeatID       string
}

var (
	macroAuctionBidID  = []byte("${AUCTION_BID_ID}")
	macroAuctionImpID  = []byte("${AUCTION_IMP_ID}")
	macroAuctionSeatID = []byte("${AUCTION_SEAT_ID}")
	macroAuctionPrice  = []byte("${AUCTION_PRICE}")
	macroAuctionID     = []byte("${AUCTION_ID}")
)

// AppendApplyMacros expands P0 OpenRTB macros into dst without heap allocation when cap(dst) suffices.
func AppendApplyMacros(dst, template []byte, ctx MacroWire) []byte {
	if len(template) == 0 {
		return dst
	}
	for i := 0; i < len(template); {
		switch {
		case len(ctx.BidID) > 0 && macroAt(template, i, macroAuctionBidID):
			dst = append(dst, ctx.BidID...)
			i += len(macroAuctionBidID)
		case len(ctx.ImpID) > 0 && macroAt(template, i, macroAuctionImpID):
			dst = append(dst, ctx.ImpID...)
			i += len(macroAuctionImpID)
		case len(ctx.SeatID) > 0 && macroAt(template, i, macroAuctionSeatID):
			dst = append(dst, ctx.SeatID...)
			i += len(macroAuctionSeatID)
		case len(ctx.AuctionPrice) > 0 && macroAt(template, i, macroAuctionPrice):
			dst = append(dst, ctx.AuctionPrice...)
			i += len(macroAuctionPrice)
		case len(ctx.AuctionID) > 0 && macroAt(template, i, macroAuctionID):
			dst = append(dst, ctx.AuctionID...)
			i += len(macroAuctionID)
		default:
			dst = append(dst, template[i])
			i++
		}
	}
	return dst
}

// ApplyMacros is cold-path JSON/admin helper. Hot exchange uses AppendApplyMacros.
func ApplyMacros(template string, ctx MacroContext) string {
	if template == "" {
		return ""
	}
	var buf [512]byte
	wire := MacroWire{
		AuctionID: []byte(ctx.AuctionID),
		BidID:     []byte(ctx.BidID),
		ImpID:     []byte(ctx.ImpID),
		SeatID:    []byte(ctx.SeatID),
	}
	if ctx.AuctionPrice != "" {
		wire.AuctionPrice = []byte(ctx.AuctionPrice)
	}
	out := AppendApplyMacros(buf[:0], []byte(template), wire)
	return string(out)
}

// AppendAuctionPrice writes micro-unit price as decimal (same layout as bid JSON price).
func AppendAuctionPrice(dst []byte, micro int64) []byte {
	return appendMicroPrice(dst, micro)
}

// FormatAuctionPrice is cold-path helper; exchange encode uses appendMicroPrice inline.
func FormatAuctionPrice(micro int64) string {
	var buf [32]byte
	return string(appendMicroPrice(buf[:0], micro))
}

// AppendCreativeID writes decimal creative id into dst.
func AppendCreativeID(dst []byte, id uint64) []byte {
	return appendUint(dst, id)
}

// FormatCreativeID is cold-path helper.
func FormatCreativeID(id uint64) string {
	var buf [24]byte
	return string(appendUint(buf[:0], id))
}

func macroAt(src []byte, off int, macro []byte) bool {
	if off+len(macro) > len(src) {
		return false
	}
	for j := 0; j < len(macro); j++ {
		if src[off+j] != macro[j] {
			return false
		}
	}
	return true
}
