package fraud

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFraudEvidencePackBulk_zipMemberSignature_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("bulk-pack-secret")
	customerID := uuid.New()
	campaignID := uuid.New()
	dir := t.TempDir()
	path := filepath.Join(dir, "bulk.zip")
	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()
	pack := reports.FraudEvidencePackDTO{
		ClickID:    "bulk:" + campaignID.String(),
		CustomerID: customerID.String(),
		CampaignID: campaignID.String(),
		RangeFrom:  from.Format(time.RFC3339),
		RangeTo:    to.Format(time.RFC3339),
	}
	signed, err := BuildSignedFraudEvidencePack(secret, pack)
	require.NoError(t, err)
	body, err := json.Marshal(signed)
	require.NoError(t, err)
	zipFile, err := os.Create(path)
	require.NoError(t, err)
	archive := zip.NewWriter(zipFile)
	writer, err := archive.Create(campaignID.String() + ".json")
	require.NoError(t, err)
	_, err = writer.Write(body)
	require.NoError(t, err)
	require.NoError(t, archive.Close())
	require.NoError(t, zipFile.Close())

	reader, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	require.Len(t, reader.File, 1)
	member, err := reader.File[0].Open()
	require.NoError(t, err)
	raw, err := io.ReadAll(member)
	require.NoError(t, err)
	_ = member.Close()
	var decoded reports.FraudEvidencePackDTO
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.NoError(t, VerifyFraudEvidencePackSignature(secret, decoded))
}
