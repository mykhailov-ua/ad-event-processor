package controlplane

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/domain"
)

func (s *Service) SetRtbMode(ctx context.Context, mode string) error {
	norm, err := domain.NormalizeRtbModeSetting(mode)
	if err != nil {
		return err
	}
	return s.UpdateSettings(ctx, map[string]string{domain.SystemSettingRtbMode: norm})
}

func (s *Service) GetRtbMode(ctx context.Context, cfg *config.Config) string {
	settings, err := s.GetSettings(ctx)
	if err == nil {
		if v, ok := settings[domain.SystemSettingRtbMode]; ok && v != "" {
			return v
		}
	}
	if cfg != nil && cfg.RtbMode != "" {
		return cfg.RtbMode
	}
	return "off"
}

func ValidateRtbModeSetting(mode string) (string, error) {
	norm, err := domain.NormalizeRtbModeSetting(mode)
	if err != nil {
		return "", fmt.Errorf("invalid rtb mode: %w", err)
	}
	return norm, nil
}
