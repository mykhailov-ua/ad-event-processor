package management

import (
	"encoding/json"
	"fmt"
)

func parseReconciliationAdjustPayload(payload []byte) (ReconciliationAdjustPayload, error) {
	var p ReconciliationAdjustPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return p, err
	}
	if p.CampaignID == "" || p.CustomerID == "" {
		return p, fmt.Errorf("invalid reconciliation adjust payload")
	}
	if p.LedgerAmt == 0 && p.RedisDelta == 0 {
		return p, fmt.Errorf("empty reconciliation adjust")
	}
	return p, nil
}
