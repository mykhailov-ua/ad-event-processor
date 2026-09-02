package campaign

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCampaignDisplayID_eightDigits(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	got := CampaignDisplayID(id)
	require.Len(t, got, 8)
	require.Regexp(t, `^\d{8}$`, got)
}

func TestCampaignDisplayID_stable(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	require.Equal(t, CampaignDisplayID(id), CampaignDisplayID(id))
}

func TestCampaignDisplayIDFromString_uuid(t *testing.T) {
	raw := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	parsed, err := uuid.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, CampaignDisplayID(parsed), CampaignDisplayIDFromString(raw))
}

func TestCampaignDisplayID_seedCatalogCampaign_holdout(t *testing.T) {
	id := seedCatalogCampaignUUID(1)
	got := CampaignDisplayID(id)
	require.Equal(t, "91821918", got)
}

func seedCatalogCampaignUUID(seq int) uuid.UUID {
	ns := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("ad-event-processor.local.seed"))
	return uuid.NewSHA1(ns, []byte(fmt.Sprintf("campaign:%d", seq)))
}
