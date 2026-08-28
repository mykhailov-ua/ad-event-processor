package supply

import (
	"errors"
	"fmt"
)

const MaxChainHops = 10

var (
	ErrSellerNotFound      = errors.New("seller not found")
	ErrAdsTxtEntryNotFound = errors.New("ads.txt entry not found")
	ErrInvalidSellerType   = errors.New("seller_type must be PUBLISHER, INTERMEDIARY, or BOTH")
	ErrInvalidRelationship = errors.New("relationship must be DIRECT or RESELLER")
	ErrChainTooLong        = fmt.Errorf("supply chain exceeds %d hops", MaxChainHops)
	ErrSellersJSONInvalid  = errors.New("sellers.json schema validation failed")
)
