package controlplane

import (
	"ad-event-processor/internal/integrations/googlesheets"
)

func googleSheetsEncKey(s *Service) []byte {
	if s == nil || s.cfg == nil {
		return nil
	}
	encKey := []byte(s.cfg.ConsentHMACSecret)
	if len(encKey) < 32 && len(s.cfg.TokenSymmetricKey) >= 32 {
		encKey = []byte(s.cfg.TokenSymmetricKey)
	}
	return encKey
}

func googleSheetsConfig(s *Service, stateSecret []byte) googlesheets.Config {
	cfg := googlesheets.Config{StateSecret: stateSecret}
	if s == nil || s.cfg == nil {
		return cfg
	}
	cfg.ClientID = s.cfg.GoogleSheetsClientID
	cfg.ClientSecret = string(s.cfg.GoogleSheetsClientSecret)
	cfg.PublicURL = s.cfg.AdminPublicURL
	if cfg.PublicURL == "" {
		cfg.PublicURL = s.cfg.ManagementURL
	}
	return cfg
}
