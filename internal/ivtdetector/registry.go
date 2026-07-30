package ivtdetector

import (
	"context"
)

type SuspiciousFinder interface {
	FindSuspiciousIPs(ctx context.Context) ([]SuspiciousIP, error)
}

type Rule interface {
	Name() string
	Find(ctx context.Context) ([]SuspiciousIP, error)
}

type RuleRegistry struct {
	rules []Rule
}

func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{}
}

func (r *RuleRegistry) Register(rule Rule) {
	if r == nil || rule == nil {
		return
	}
	r.rules = append(r.rules, rule)
}

func (r *RuleRegistry) FindSuspiciousIPs(ctx context.Context) ([]SuspiciousIP, error) {
	if r == nil || len(r.rules) == 0 {
		return nil, nil
	}
	var groups [][]SuspiciousIP
	for _, rule := range r.rules {
		found, err := rule.Find(ctx)
		if err != nil {
			return nil, err
		}
		if len(found) > 0 {
			groups = append(groups, found)
		}
		ivtCandidatesTotal.WithLabelValues(rule.Name()).Add(float64(len(found)))
	}
	return mergeSuspiciousIPs(groups...), nil
}
