package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/domainhealth"
)

// SetReputationChecker overrides the domain reputation client (tests).
func (s *Service) SetReputationChecker(c *domainhealth.ReputationChecker) {
	if s == nil {
		return
	}
	s.reputation = c
}

func (s *Service) reputationChecker() *domainhealth.ReputationChecker {
	if s == nil {
		return nil
	}
	if s.reputation != nil {
		return s.reputation
	}
	if s.cfg == nil || !s.cfg.Management.DomainReputationEnabled {
		return nil
	}
	s.reputation = domainhealth.NewReputationChecker(domainhealth.ReputationConfig{
		SafeBrowsingAPIKey: string(s.cfg.Management.SafeBrowsingAPIKey),
		FacebookToken:      string(s.cfg.Management.FacebookGraphAccessToken),
		FacebookGraphBase:  s.cfg.Management.FacebookGraphAPIBase,
	})
	return s.reputation
}
