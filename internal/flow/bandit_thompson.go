package flow

import (
	"math"
	"math/rand"

	"github.com/google/uuid"
)

type ArmStat struct {
	Clicks      int64
	Conversions int64
}

func ThompsonWeights(arms map[uuid.UUID]ArmStat, rng *rand.Rand) map[uuid.UUID]int32 {
	if len(arms) == 0 {
		return nil
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	samples := make(map[uuid.UUID]float64, len(arms))
	var sum float64
	for id, stat := range arms {
		alpha := float64(stat.Conversions) + 1
		failures := stat.Clicks - stat.Conversions
		if failures < 0 {
			failures = 0
		}
		beta := float64(failures) + 1
		s := sampleBeta(rng, alpha, beta)
		samples[id] = s
		sum += s
	}
	out := make(map[uuid.UUID]int32, len(arms))
	if sum <= 0 {
		for id := range arms {
			out[id] = 1
		}
		return out
	}
	for id, s := range samples {
		w := int32(math.Max(1, math.Round(100*s/sum)))
		out[id] = w
	}
	return out
}

func sampleBeta(rng *rand.Rand, alpha, beta float64) float64 {
	x := sampleGamma(rng, alpha)
	y := sampleGamma(rng, beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

func sampleGamma(rng *rand.Rand, alpha float64) float64 {
	if alpha < 1 {
		return sampleGamma(rng, alpha+1) * math.Pow(rng.Float64(), 1/alpha)
	}
	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)
	for {
		x := rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1-0.0331*x*x*x*x {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v
		}
	}
}
