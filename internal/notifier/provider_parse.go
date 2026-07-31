package notifier

import (
	"strings"

	"espx/internal/notifier/db"
)

func ParseProviderName(name string) (db.NotifierProvider, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrUnsupportedProvider
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "PROVIDER_") {
		upper = strings.TrimPrefix(upper, "PROVIDER_")
	}
	switch upper {
	case "TELEGRAM":
		return db.NotifierProviderTELEGRAM, nil
	case "SLACK":
		return db.NotifierProviderSLACK, nil
	case "SMTP":
		return db.NotifierProviderSMTP, nil
	case "SMS":
		return db.NotifierProviderSMS, nil
	default:
		return "", ErrUnsupportedProvider
	}
}

func ProviderDisplayName(provider db.NotifierProvider) string {
	switch provider {
	case db.NotifierProviderTELEGRAM:
		return "TELEGRAM"
	case db.NotifierProviderSLACK:
		return "SLACK"
	case db.NotifierProviderSMTP:
		return "SMTP"
	case db.NotifierProviderSMS:
		return "SMS"
	default:
		return "UNSPECIFIED"
	}
}
