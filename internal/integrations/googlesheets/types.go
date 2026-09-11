package googlesheets

import "time"

const (
	OAuthScope              = "https://www.googleapis.com/auth/spreadsheets"
	defaultSheetHTTPTimeout = 30 * time.Second
	maxCellsPerBatch        = 10000
)

type Config struct {
	ClientID     string
	ClientSecret string
	PublicURL    string
	StateSecret  []byte
	TokenURL     string
}

type TokenRow struct {
	UserID       string
	RefreshToken string
	AccessToken  string
	ExpiresAt    time.Time
	Scopes       string
}

type StatusDTO struct {
	Connected    bool   `json:"connected"`
	AccountEmail string `json:"account_email,omitempty"`
	Message      string `json:"message,omitempty"`
}

type UploadResult struct {
	SpreadsheetID  string
	SpreadsheetURL string
}
