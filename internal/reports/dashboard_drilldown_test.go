package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDashboardDrilldownDimension_holdout(t *testing.T) {
	t.Parallel()

	got, err := ParseDashboardDrilldownDimension("country")
	require.NoError(t, err)
	require.Equal(t, DrilldownDimensionCountry, got)

	_, err = ParseDashboardDrilldownDimension("not-a-dimension")
	require.Error(t, err)
}

func TestDrilldownDimensionExpr_sub4Creative(t *testing.T) {
	t.Parallel()

	sub4Expr, _ := drilldownDimensionExpr(DrilldownDimensionSub4)
	creativeExpr, _ := drilldownDimensionExpr(DrilldownDimensionCreative)
	require.Equal(t, sub4Expr, creativeExpr)
}
