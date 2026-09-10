package licenseissue

import (
	"fmt"
	"strings"
	"time"
)

type TelegramDelivery struct {
	Text      string             `json:"text"`
	ParseMode string             `json:"parse_mode,omitempty"`
	ButtonRows [][]TelegramButton `json:"button_rows,omitempty"`
}

func IssueTelegramDelivery(res IssueResult, sku string) TelegramDelivery {
	skuLabel := strings.TrimSpace(sku)
	if skuLabel == "" {
		skuLabel = "pilot"
	}
	valid := res.ValidUntil.UTC().Format(time.RFC3339)
	text := fmt.Sprintf(
		"%s license issued.\nDeployment: %s\nValid until: %s\n\nOn your VPS:\n  license-apply %s",
		skuLabel,
		res.DeploymentID,
		valid,
		res.Token,
	)
	return TelegramDelivery{Text: text}
}
