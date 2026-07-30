package rtb

type ClearingMode uint8

const (
	ClearingSecondPrice ClearingMode = iota
	ClearingFirstPrice
)

func (registry *Registry) clearingPrice(mode ClearingMode, floor int64, winnerBid int64, secondBid int64) int64 {
	if mode == ClearingFirstPrice {
		return winnerBid
	}
	price := floor
	if secondBid != -1 && secondBid > price {
		price = secondBid
	}
	return price
}

func applyReserve(price int64, reserve int64, winnerBid int64) int64 {
	if reserve > price {
		price = reserve
	}
	if price > winnerBid {
		price = winnerBid
	}
	return price
}
