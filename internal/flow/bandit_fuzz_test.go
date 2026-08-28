package flow

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/google/uuid"
)

func FuzzBanditWeightSnapshot(f *testing.F) {
	f.Add(`[{"weight":100,"landers":[{"lander_id":"00000000-0000-4000-8000-000000000001","weight":50},{"lander_id":"00000000-0000-4000-8000-000000000002","weight":50}],"offers":[]}]`, int64(200), int64(20), int64(200), int64(5))
	f.Fuzz(func(t *testing.T, pathsJSON string, clicksA, convA, clicksB, convB int64) {
		if len(pathsJSON) > 4096 {
			return
		}
		if clicksA < 0 || convA < 0 || clicksB < 0 || convB < 0 {
			return
		}
		var paths []banditPathJSON
		if err := json.Unmarshal([]byte(pathsJSON), &paths); err != nil {
			return
		}
		for _, p := range paths {
			if p.Weight < 0 {
				return
			}
			for _, l := range p.Landers {
				if l.Weight < 0 {
					return
				}
			}
			for _, o := range p.Offers {
				if o.Weight < 0 {
					return
				}
			}
		}
		raw := []byte(pathsJSON)
		campID := uuidFromFuzz(t.Name())
		stats := map[uuid.UUID]map[uuid.UUID]EntityBanditStat{
			campID: {
				uuid.MustParse("00000000-0000-4000-8000-000000000001"): {Clicks: clicksA, Conversions: convA},
				uuid.MustParse("00000000-0000-4000-8000-000000000002"): {Clicks: clicksB, Conversions: convB},
			},
		}
		_, _, _ = ApplyFlowBanditThompson(raw, []uuid.UUID{campID}, stats, nil, rand.New(rand.NewSource(1)))
	})
}

func uuidFromFuzz(s string) uuid.UUID {
	var b [16]byte
	for i := 0; i < len(s) && i < 16; i++ {
		b[i] = byte(s[i])
	}
	return uuid.UUID(b)
}
