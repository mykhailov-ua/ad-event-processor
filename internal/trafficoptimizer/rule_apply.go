package trafficoptimizer

func RuleSupported(rule Rule) bool {
	switch rule.Scope {
	case ScopeLander, ScopeOffer:
		switch rule.Objective {
		case ObjectiveCR:
			return rule.Algorithm == AlgorithmThompson
		case ObjectiveEPC, ObjectiveRevenue, ObjectiveROI:
			return rule.Algorithm == AlgorithmProportional
		default:
			return false
		}
	case ScopeCreative:
		return rule.HasBrand && rule.Objective == ObjectiveROI && rule.Algorithm == AlgorithmProportional
	default:
		return false
	}
}
