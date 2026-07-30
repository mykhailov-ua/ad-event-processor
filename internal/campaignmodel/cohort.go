package campaignmodel

import "github.com/google/uuid"

type CohortVariant struct {
	ID     string
	Weight uint32
	Flags  map[string]string
}

type ExperimentCohort struct {
	ID       uuid.UUID
	Name     string
	Salt     string
	Variants []CohortVariant
}
