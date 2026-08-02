package openrtb

var DefaultNURLTemplate = []byte("https://track.local/openrtb/win?price=${AUCTION_PRICE}&id=${AUCTION_ID}&bid=${AUCTION_BID_ID}&imp=${AUCTION_IMP_ID}&seat=${AUCTION_SEAT_ID}")

const (
	ExchangeDeliveryADM  = "adm"
	ExchangeDeliveryNURL = "nurl"
)
