package fraud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedConversionMobileBiometrics(
	t *testing.T,
	conn interface {
		Exec(context.Context, string, ...any) error
	},
	ip string,
	touchCount, gyroSamples, gyroFlat, biometricMobile uint8,
	copies int,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h := testPIIHasher()
	ipHash := piihash.FixedString16(h.HashIP(ip))
	uaHash := piihash.FixedString16(h.HashUA("ua-mobile-bio"))
	campaignID := uuid.New()
	now := time.Now().UTC()

	for i := 0; i < copies; i++ {
		clickID := fmt.Sprintf("mbio-%s-%d", ip, i)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO ad_event_processor.conversions
			(click_id, campaign_id, ip_hash, ua_hash, pii_salt_version, payload,
			 mobile_touch_count, mobile_gyro_samples, mobile_gyro_flat,
			 mobile_biometric_set, mobile_biometric_mobile, created_at)
			VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, 1, ?, ?)`,
			clickID, campaignID, ipHash, uaHash, h.Version(),
			touchCount, gyroSamples, gyroFlat, biometricMobile, now,
		))
	}
}

func TestMobileBiometricsRule_holdoutFlatGyro(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	ctx := context.Background()
	ip := "203.0.113.88"
	seedConversionMobileBiometrics(t, conn, ip, 2, 4, 1, 1, 6)

	rule := &mobileBiometricsRule{
		clickhouseQuery: database.NewClickHouseQuery(conn, database.ClickHouseQueryConfig{}),
		cfg: AnalyzerConfig{
			Window:                         time.Hour,
			MobileBiometricsEnabled:        true,
			MobileBiometricsMinSamples:     5,
			MobileBiometricsMinFlatHits:    4,
			MobileBiometricsMinGyroSamples: 3,
		},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "ivt_mobile_biometrics_flat_gyro", candidates[0].Reason)
}

func TestMobileBiometricsRule_holdoutDisabledFailOpen(t *testing.T) {
	rule := &mobileBiometricsRule{
		cfg: AnalyzerConfig{MobileBiometricsEnabled: false},
	}
	out, err := rule.Find(context.Background())
	require.NoError(t, err)
	assert.Nil(t, out)
}
