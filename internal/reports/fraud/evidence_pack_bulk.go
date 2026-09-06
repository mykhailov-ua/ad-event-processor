package fraud

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
)

func writeFraudEvidencePackBulkZip(ctx context.Context, deps reports.ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	if len(deps.FraudEvidencePackHMACSecret) == 0 {
		return fmt.Errorf("fraud evidence pack signing secret not configured")
	}
	if deps.Pool == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	from, to, err := reportjob.ParseReportRangeFromStrings(spec.From, spec.To)
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(strings.TrimSpace(spec.CustomerID))
	if err != nil {
		return fmt.Errorf("invalid customer_id")
	}
	campaignUUIDs, err := reports.ListCustomerCampaignIDs(ctx, deps.Pool, customerID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = zipFile.Close() }()
	archive := zip.NewWriter(zipFile)
	defer func() { _ = archive.Close() }()
	rangeFrom := from.UTC().Format(time.RFC3339)
	rangeTo := to.UTC().Format(time.RFC3339)

	fraudByCampaign := map[string][]reports.FraudEvidenceFraudRowDTO{}
	if deps.ClickHouseQuery != nil && len(campaignUUIDs) > 0 {
		allRows, qerr := queryFraudEvidencePackFraudCH(ctx, deps.ClickHouseQuery, campaignUUIDs, "", from, to)
		if qerr != nil {
			return qerr
		}
		for i := range allRows {
			fraudByCampaign[allRows[i].CampaignID] = append(fraudByCampaign[allRows[i].CampaignID], allRows[i])
		}
	}

	for _, campUUID := range campaignUUIDs {
		campaignID := campUUID.String()
		pack := reports.FraudEvidencePackDTO{
			ClickID:     "bulk:" + campaignID,
			CustomerID:  customerID.String(),
			CampaignID:  campaignID,
			RangeFrom:   rangeFrom,
			RangeTo:     rangeTo,
			FraudEvents: fraudByCampaign[campaignID],
		}
		if len(pack.FraudEvents) > 0 {
			pack.Signals = aggregateFraudEvidenceSignals(pack.FraudEvents)
		}
		signed, serr := BuildSignedFraudEvidencePack(deps.FraudEvidencePackHMACSecret, pack)
		if serr != nil {
			return serr
		}
		body, merr := json.Marshal(signed)
		if merr != nil {
			return merr
		}
		writer, werr := archive.Create(campaignID + ".json")
		if werr != nil {
			return werr
		}
		if _, err := io.Copy(writer, bytes.NewReader(body)); err != nil {
			return err
		}
	}
	return nil
}
