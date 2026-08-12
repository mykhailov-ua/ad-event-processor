package ingestion

import (
	"hash/fnv"

	"github.com/bidshard/ad-event-processor/internal/domain"
)

func AssignCohortVariant(salt, subjectID string, variants []domain.CohortVariant) (variantID string, flags map[string]string) {
	if len(variants) == 0 {
		return "", nil
	}
	var total uint32
	for _, v := range variants {
		total += v.Weight
	}
	if total == 0 {
		v := variants[0]
		return v.ID, cloneFlags(v.Flags)
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(subjectID))
	bucket := h.Sum32() % total

	var cursor uint32
	for _, v := range variants {
		cursor += v.Weight
		if bucket < cursor {
			return v.ID, cloneFlags(v.Flags)
		}
	}
	last := variants[len(variants)-1]
	return last.ID, cloneFlags(last.Flags)
}

func cloneFlags(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
