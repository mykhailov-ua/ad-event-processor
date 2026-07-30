package billing

import "errors"

var (
	ErrCustomerNotFound    = errors.New("customer not found")
	ErrInvalidCustomerID   = errors.New("invalid customer id")
	ErrInvalidInvoiceID    = errors.New("invalid invoice id")
	ErrInvalidBillingMonth = errors.New("billing_month must be the first day of a calendar month")
	ErrInvoiceNotFound     = errors.New("invoice not found")
	ErrLedgerDrift         = errors.New("ledger balance drift detected")
	ErrNoSpend             = errors.New("no spend in billing period")
)
