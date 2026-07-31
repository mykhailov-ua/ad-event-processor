package payment_test

type cryptoWebhookPayload struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	TxHash        string `json:"tx_hash"`
	AmountMicro   int64  `json:"amount_micro"`
	Currency      string `json:"currency"`
	Confirmations int    `json:"confirmations"`
	ProviderRef   string `json:"provider_ref"`
}
