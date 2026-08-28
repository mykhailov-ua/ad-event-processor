package reports

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

	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
)

func writeFraudEvidencePackBulkZip(ctx context.Context, deps ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
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
	campaignUUIDs, err := listCustomerCampaignIDs(ctx, deps.Pool, customerID)
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
	for _, campUUID := range campaignUUIDs {
		campaignID := campUUID.String()
		pack := FraudEvidencePackDTO{
			ClickID:    "bulk:" + campaignID,
			CustomerID: customerID.String(),
			CampaignID: campaignID,
			RangeFrom:  rangeFrom,
			RangeTo:    rangeTo,
		}
		if deps.ClickHouseQuery != nil {
			fraudRows, qerr := queryFraudEvidencePackFraudCH(ctx, deps.ClickHouseQuery, []uuid.UUID{campUUID}, "", from, to)
			if qerr != nil {
				return qerr
			}
			pack.FraudEvents = fraudRows
			pack.Signals = aggregateFraudEvidenceSignals(fraudRows)
		}
		signed, serr := BuildSignedFraudEvidencePack(deps.FraudEvidencePackHMACSecret, pack)
		if serr != nil {
			return serr
		}
		body, merr := json.Marshal(signed)
		if merr != nil {
			return merr
		}
		memberName := campaignID + ".json"
		writer, werr := archive.Create(memberName)
		if werr != nil {
			return werr
		}
		if _, err := io.Copy(writer, bytes.NewReader(body)); err != nil {
			return err
		}
	}
	return nil
}
