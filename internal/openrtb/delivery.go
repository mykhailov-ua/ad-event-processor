package openrtb

// DefaultNURLTemplate is the win-notice URL with P0 macros (SSP substitutes on notification).
var DefaultNURLTemplate = []byte("https://track.local/openrtb/win?price=${AUCTION_PRICE}&id=${AUCTION_ID}&bid=${AUCTION_BID_ID}&imp=${AUCTION_IMP_ID}&seat=${AUCTION_SEAT_ID}")

const (
	ExchangeDeliveryADM  = "adm"
	ExchangeDeliveryNURL = "nurl"
)
