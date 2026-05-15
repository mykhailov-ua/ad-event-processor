package management

import (
	"context"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/db"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads"
	"github.com/mykhailov-ua/ad-event-processor/internal/database"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formatNumericLocal(n pgtype.Numeric) string {
	val, _ := n.Value()
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

func TestManagementQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	queries := db.New(pool)

	customerID := uuid.New()
	cust, err := queries.CreateCustomer(ctx, db.CreateCustomerParams{
		ID:       pgtype.UUID{Bytes: customerID, Valid: true},
		Name:     "Test db.Customer",
		Balance:  ads.ToNumeric(decimal.NewFromFloat(1000.00)),
		Currency: "USD",
	})
	require.NoError(t, err)
	assert.Equal(t, "1000.00", formatNumericLocal(cust.Balance))

	cust, err = queries.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
		ID:      pgtype.UUID{Bytes: customerID, Valid: true},
		Balance: ads.ToNumeric(decimal.NewFromFloat(500.00)),
	})
	require.NoError(t, err)
	assert.Equal(t, "1500.00", formatNumericLocal(cust.Balance))

	campaignID := uuid.New()
	camp, err := queries.CreateCampaign(ctx, db.CreateCampaignParams{
		ID:          pgtype.UUID{Bytes: campaignID, Valid: true},
		Name:        "Management Test Campaign",
		BudgetLimit: ads.ToNumeric(decimal.NewFromFloat(100.00)),
		Status:      db.CampaignStatusTypeACTIVE,
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		PacingMode:  db.PacingModeTypeASAP,
		DailyBudget: ads.ToNumeric(decimal.Zero),
		Timezone:    "UTC",
	})
	require.NoError(t, err)
	assert.Equal(t, campaignID, uuid.UUID(camp.ID.Bytes))
}
