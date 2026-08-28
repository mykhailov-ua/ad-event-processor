package rtbadmin

import (
	"errors"

	"ad-event-processor/internal/rtb"
)

var (
	ErrRtbDealNotFound     = errors.New("rtb deal not found")
	ErrInvalidDealPacing   = rtb.ErrInvalidDealPacing
	ErrDuplicateDealID     = errors.New("deal_id already exists")
	ErrDealCustomerMissing = errors.New("customer not found")
	ErrInvalidDealSeats    = errors.New("seats must be at least 1")
)
