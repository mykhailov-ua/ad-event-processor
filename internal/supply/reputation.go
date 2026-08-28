package supply

import (
	"ad-event-processor/pkg/domainhealth"
)

type ReputationConfig struct {
	Enabled            bool
	SafeBrowsingAPIKey string
	FacebookToken      string
	FacebookGraphBase  string
}

func ReputationChecker(cfg ReputationConfig, cached **domainhealth.ReputationChecker) *domainhealth.ReputationChecker {
	if cached != nil && *cached != nil {
		return *cached
	}
	if !cfg.Enabled {
		return nil
	}
	checker := domainhealth.NewReputationChecker(domainhealth.ReputationConfig{
		SafeBrowsingAPIKey: cfg.SafeBrowsingAPIKey,
		FacebookToken:      cfg.FacebookToken,
		FacebookGraphBase:  cfg.FacebookGraphBase,
	})
	if cached != nil {
		*cached = checker
	}
	return checker
}

func SetReputationChecker(cached **domainhealth.ReputationChecker, checker *domainhealth.ReputationChecker) {
	if cached == nil {
		return
	}
	*cached = checker
}
