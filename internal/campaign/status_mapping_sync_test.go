package campaign

import (
	"testing"

	"ad-event-processor/internal/integrationschema"

	"github.com/stretchr/testify/require"
)

func TestConversionMappingsFromStatusSchema_mapsInboundToGoal(t *testing.T) {
	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"status_map": {"lead": "pending", "sale": "approved"}
	}`))
	require.NoError(t, err)
	s := parsed.(*integrationschema.StatusMappingSchema)

	mappings, err := ConversionMappingsFromStatusSchema(s)
	require.NoError(t, err)
	require.Len(t, mappings, 2)

	byInbound := make(map[string]ConversionMappingDTO, len(mappings))
	for _, row := range mappings {
		byInbound[row.InboundStatus] = row
	}
	require.Equal(t, "pending", byInbound["lead"].GoalName)
	require.Equal(t, "approved", byInbound["sale"].GoalName)
	require.Equal(t, int64(0), byInbound["sale"].PayoutMicro)
}

func TestMapAffiliateStatus_usedByConversionMappingsFromStatusSchema_holdout(t *testing.T) {
	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"status_map": {"sale": "approved"}
	}`))
	require.NoError(t, err)
	s := parsed.(*integrationschema.StatusMappingSchema)

	mapped, ok := integrationschema.MapAffiliateStatus(s, "sale")
	require.True(t, ok)
	require.Equal(t, "approved", mapped)

	mappings, err := ConversionMappingsFromStatusSchema(s)
	require.NoError(t, err)
	require.Len(t, mappings, 1)
	require.Equal(t, "sale", mappings[0].InboundStatus)
	require.Equal(t, mapped, mappings[0].GoalName)
}
