package supply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type ExportHost interface {
	SupplyExportPath() string
	BuildSellersJSON(ctx context.Context) ([]byte, error)
	BuildAdsTxt(ctx context.Context) (string, error)
}

func ExportFiles(ctx context.Context, host ExportHost) error {
	if host == nil {
		return fmt.Errorf("supply export: host unavailable")
	}
	exportDir := host.SupplyExportPath()
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return fmt.Errorf("create supply export dir: %w", err)
	}

	sellersBody, err := host.BuildSellersJSON(ctx)
	if err != nil {
		return err
	}
	sellersPath := filepath.Join(exportDir, "sellers.json")
	if err := os.WriteFile(sellersPath, sellersBody, 0o644); err != nil {
		return fmt.Errorf("write sellers.json: %w", err)
	}

	adsTxt, err := host.BuildAdsTxt(ctx)
	if err != nil {
		return err
	}
	adsPath := filepath.Join(exportDir, "ads.txt")
	if err := os.WriteFile(adsPath, []byte(adsTxt), 0o644); err != nil {
		return fmt.Errorf("write ads.txt: %w", err)
	}

	InvalidateSellersJSONCache()
	return nil
}
