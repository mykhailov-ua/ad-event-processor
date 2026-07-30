package ingestion

import "espx/internal/config"

type RtbAuthorityController struct {
	cfg        *config.Config
	watcher    *SettingsWatcher
	unified    *UnifiedFilter
	catalog    *RtbCatalog
	budgetSync *RtbBudgetSync
}

func NewRtbAuthorityController(
	cfg *config.Config,
	watcher *SettingsWatcher,
	unified *UnifiedFilter,
	catalog *RtbCatalog,
	budgetSync *RtbBudgetSync,
) *RtbAuthorityController {
	c := &RtbAuthorityController{
		cfg:        cfg,
		watcher:    watcher,
		unified:    unified,
		catalog:    catalog,
		budgetSync: budgetSync,
	}
	if watcher != nil {
		watcher.AddChangeListener(func(_ *DynamicConfig) { c.Apply() })
	}
	c.Apply()
	return c
}

func (c *RtbAuthorityController) Apply() {
	setting := ""
	if c.watcher != nil {
		setting = c.watcher.Get().RtbBudgetAuthority
	}
	auth := BudgetAuthorityFromSettings(c.cfg, setting)
	if c.unified != nil {
		c.unified.SetSkipBudgetDebit(RtbSkipLuaBudgetDebit(c.cfg, setting))
	}
	if c.catalog != nil {
		c.catalog.SetAuthority(auth)
	}
	if c.budgetSync != nil {
		c.budgetSync.Authority = auth
	}
}
