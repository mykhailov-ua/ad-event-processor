package domain

import (
	"fmt"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"
)

type MarginEconomicsSplit struct {
	RevenueMicro         int64
	RtbCostMicro         int64
	OperatorMarginMicro  int64
	PublisherPayoutMicro int64
}

func ComputeMarginEconomicsSplit(revenueMicro, rtbCostMicro int64) (MarginEconomicsSplit, error) {
	if revenueMicro <= 0 {
		return MarginEconomicsSplit{}, fmt.Errorf("revenue must be positive")
	}
	if rtbCostMicro < 0 {
		return MarginEconomicsSplit{}, fmt.Errorf("rtb cost must be non-negative")
	}
	if rtbCostMicro > revenueMicro {
		rtbCostMicro = revenueMicro
	}
	return MarginEconomicsSplit{
		RevenueMicro:         revenueMicro,
		RtbCostMicro:         rtbCostMicro,
		OperatorMarginMicro:  revenueMicro - rtbCostMicro,
		PublisherPayoutMicro: rtbCostMicro,
	}, nil
}

func marginLegHash(prefix, txID string) string {
	return prefix + ":" + txID
}

type marginEconomicsLeg struct {
	amountMicro int64
	ledgerType  db.LedgerType
	hashPrefix  string
}

func marginEconomicsLegs(split MarginEconomicsSplit) []marginEconomicsLeg {
	return []marginEconomicsLeg{
		{amountMicro: split.RtbCostMicro, ledgerType: db.LedgerTypeRtbCost, hashPrefix: "margin:rtb"},
		{amountMicro: split.PublisherPayoutMicro, ledgerType: db.LedgerTypePublisherPayout, hashPrefix: "margin:pub"},
		{amountMicro: split.OperatorMarginMicro, ledgerType: db.LedgerTypeOperatorMargin, hashPrefix: "margin:op"},
	}
}
