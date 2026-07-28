package campaignmodel

import "github.com/google/uuid"

// CohortVariant is one weighted arm in an A/B experiment (hot-path safe; no json tags).
type CohortVariant struct {
	ID     string
	Weight uint32
	Flags  map[string]string
}

// ExperimentCohort is a stable-hash experiment definition loaded into the registry snapshot.
type ExperimentCohort struct {
	ID       uuid.UUID
	Name     string
	Salt     string
	Variants []CohortVariant
}
