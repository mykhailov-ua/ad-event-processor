package fraud

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFraudEvidencePackBulkQuery_holdoutNoClickIDFilter(t *testing.T) {
	t.Parallel()
	require.NotContains(t, fraudEvidencePackBulkQuery, "click_id = ?")
	require.Contains(t, fraudEvidencePackBulkQuery, "campaign_id IN (?)")
}

func TestQueryFraudEvidencePackFraudCH_bulkMode_holdoutUsesBulkQuery(t *testing.T) {
	t.Parallel()
	rows, err := queryFraudEvidencePackFraudCH(context.TODO(), nil, []uuid.UUID{uuid.New()}, "", time.Now(), time.Now())
	require.NoError(t, err)
	require.Nil(t, rows)
}
