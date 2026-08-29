package ingest

import (
	"testing"

	"ad-event-processor/internal/rtb"

	"github.com/stretchr/testify/assert"
)

func TestNoBidToRejectKind(t *testing.T) {
	assert.Equal(t, filterRejectPacing, noBidToRejectKind(rtb.NoBidPacingClosed))
	assert.Equal(t, filterRejectBudget, noBidToRejectKind(rtb.NoBidDailyCapExceeded))
	assert.Equal(t, filterRejectBudget, noBidToRejectKind(rtb.NoBidSpendFailed))
	assert.Equal(t, filterRejectBidFloor, noBidToRejectKind(rtb.NoBidNoCandidates))
	assert.Equal(t, filterRejectInfra, noBidToRejectKind(rtb.NoBidCorruptCatalog))
	assert.Equal(t, filterRejectSchedule, noBidToRejectKind(rtb.NoBidDaypartClosed))
	assert.Equal(t, filterRejectFreq, noBidToRejectKind(rtb.NoBidFreqCapExceeded))
}
