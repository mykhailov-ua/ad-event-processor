package dedupkey

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const spendSyncPayloadKind = "spend_sync"

// SpendSyncTxn is one cross-region spend debit in a proxy uplink batch.
type SpendSyncTxn struct {
	CampaignID  uuid.UUID
	AmountMicro int64
	TxnID       string
}

type spendSyncPayload struct {
	Kind string           `json:"kind"`
	Txns []spendSyncTxnJS `json:"txns"`
}

type spendSyncTxnJS struct {
	CampaignID  string `json:"c"`
	AmountMicro int64  `json:"a"`
	TxnID       string `json:"t"`
}

// EncodeSpendSyncPayload serializes a spend sync batch for region-proxy uplink.
func EncodeSpendSyncPayload(txns []SpendSyncTxn) ([]byte, error) {
	if len(txns) == 0 {
		return nil, errors.New("dedupkey: empty spend sync batch")
	}
	body := spendSyncPayload{Kind: spendSyncPayloadKind, Txns: make([]spendSyncTxnJS, 0, len(txns))}
	for _, txn := range txns {
		if txn.CampaignID == uuid.Nil {
			return nil, errors.New("dedupkey: spend sync txn missing campaign_id")
		}
		if txn.AmountMicro <= 0 {
			return nil, fmt.Errorf("dedupkey: spend sync txn %q has non-positive amount", txn.TxnID)
		}
		if txn.TxnID == "" {
			return nil, errors.New("dedupkey: spend sync txn missing txn_id")
		}
		body.Txns = append(body.Txns, spendSyncTxnJS{
			CampaignID:  txn.CampaignID.String(),
			AmountMicro: txn.AmountMicro,
			TxnID:       txn.TxnID,
		})
	}
	return json.Marshal(body)
}

// DecodeSpendSyncPayload parses a region-proxy spend sync batch.
func DecodeSpendSyncPayload(payload []byte) ([]SpendSyncTxn, error) {
	if len(payload) == 0 {
		return nil, errors.New("dedupkey: empty spend sync payload")
	}
	var body spendSyncPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("dedupkey: spend sync payload: %w", err)
	}
	if body.Kind != spendSyncPayloadKind {
		return nil, fmt.Errorf("dedupkey: unexpected spend sync kind %q", body.Kind)
	}
	if len(body.Txns) == 0 {
		return nil, errors.New("dedupkey: spend sync batch has no txns")
	}
	out := make([]SpendSyncTxn, 0, len(body.Txns))
	for i, row := range body.Txns {
		campID, err := uuid.Parse(row.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("dedupkey: spend sync txn[%d] campaign_id: %w", i, err)
		}
		if row.AmountMicro <= 0 {
			return nil, fmt.Errorf("dedupkey: spend sync txn[%d] has non-positive amount", i)
		}
		if row.TxnID == "" {
			return nil, fmt.Errorf("dedupkey: spend sync txn[%d] missing txn_id", i)
		}
		out = append(out, SpendSyncTxn{
			CampaignID:  campID,
			AmountMicro: row.AmountMicro,
			TxnID:       row.TxnID,
		})
	}
	return out, nil
}

// IsSpendSyncPayload reports whether payload bytes look like a spend sync batch.
func IsSpendSyncPayload(payload []byte) bool {
	txns, err := DecodeSpendSyncPayload(payload)
	return err == nil && len(txns) > 0
}
