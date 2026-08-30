package rtb

// ClearingMode selects first-price vs second-price after rank; stored on Registry
// via atomic.Uint32 (ClearingMode()) so hot path reads without mutex.
type ClearingMode uint8

const (
	ClearingSecondPrice ClearingMode = iota
	ClearingFirstPrice
)

// clearingPrice: second-price pays max(floor, secondBid); first-price pays winnerBid.
// secondBid stays -1 when only one eligible candidate cleared rank filters.
func (r *Registry) clearingPrice(mode ClearingMode, floor int64, winnerBid int64, secondBid int64) int64 {
	if mode == ClearingFirstPrice {
		return winnerBid
	}
	price := floor
	if secondBid != -1 && secondBid > price {
		price = secondBid
	}
	return price
}

// applyReserve lifts price to campaign reserve and caps at winnerBid (never overbid).
func applyReserve(price int64, reserve int64, winnerBid int64) int64 {
	if reserve > price {
		price = reserve
	}
	if price > winnerBid {
		price = winnerBid
	}
	return price
}
