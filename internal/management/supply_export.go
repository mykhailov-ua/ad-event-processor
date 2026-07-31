package management

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Service) ExportSupplyFiles(ctx context.Context) error {
	exportDir := s.SupplyExportPath()
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("create supply export dir: %w", err)
	}

	sellersBody, err := s.BuildSellersJSON(ctx)
	if err != nil {
		return err
	}
	sellersPath := filepath.Join(exportDir, "sellers.json")
	if err := os.WriteFile(sellersPath, sellersBody, 0644); err != nil {
		return fmt.Errorf("write sellers.json: %w", err)
	}

	adsTxt, err := s.BuildAdsTxt(ctx)
	if err != nil {
		return err
	}
	adsPath := filepath.Join(exportDir, "ads.txt")
	if err := os.WriteFile(adsPath, []byte(adsTxt), 0644); err != nil {
		return fmt.Errorf("write ads.txt: %w", err)
	}

	invalidateSellersJSONCache()
	return nil
}

func (worker *OutboxWorker) handleUpdateSupplyFiles(ctx context.Context, payload []byte) error {
	_ = payload
	return worker.svc.ExportSupplyFiles(ctx)
}
