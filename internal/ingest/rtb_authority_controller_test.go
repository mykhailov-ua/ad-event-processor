package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	filterunified "ad-event-processor/internal/filter/unified"
	"ad-event-processor/internal/rtb"

	"github.com/stretchr/testify/assert"
)

func TestRtbAuthorityController_luaKeepsBudgetInRedis(t *testing.T) {
	cfg := &config.Config{RtbMode: "live", RtbBudgetAuthority: "rtb"}
	sw := NewSettingsWatcher(nil, cfg)
	unified := &UnifiedFilter{}
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	sync := RtbBudgetSync{Authority: BudgetAuthorityRTB}

	ctrl := NewRtbAuthorityController(cfg, sw, unified, catalog, &sync)
	assert.True(t, unified.SkipBudgetDebitAny() == filterunified.OneAny)
	assert.Equal(t, BudgetAuthorityRTB, catalog.Authority())

	sw.StoreDynamicConfigForTest(&DynamicConfig{RtbBudgetAuthority: "lua"})
	ctrl.Apply()
	assert.True(t, unified.SkipBudgetDebitAny() == filterunified.ZeroAny)
	assert.Equal(t, BudgetAuthorityRedis, catalog.Authority())
}
