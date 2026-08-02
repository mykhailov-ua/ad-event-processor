package fraud

import (
	"context"
)

type NativeScorer struct {
	model *NativeModel
}

func NewNativeScorer(modelPath string) (*NativeScorer, error) {
	model, err := LoadNativeModel(modelPath)
	if err != nil {
		return nil, err
	}
	return &NativeScorer{model: model}, nil
}

func (s *NativeScorer) Name() string {
	return "lightgbm_native"
}

func (s *NativeScorer) Dims() int {
	if s == nil || s.model == nil {
		return featureVectorDims
	}
	if s.model.dims > 0 {
		return s.model.dims
	}
	return featureVectorDims
}

func (s *NativeScorer) ScoreBatch(ctx context.Context, rows []FeatureRow) ([]float64, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]float64, len(rows))
	var vec [featureVectorDims]float64
	for i := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows[i].ToVectorInto(vec[:])
		out[i] = s.model.PredictProbability(vec[:])
	}
	return out, nil
}

func ScoreBatchSoA(ctx context.Context, model *NativeModel, rows []FeatureRow, out []float64) error {
	if model == nil || len(rows) == 0 {
		return nil
	}
	if len(out) < len(rows) {
		return nil
	}
	n := len(rows)
	events := make([]float64, n)
	clicks := make([]float64, n)
	ctr := make([]float64, n)
	spendNorm := make([]float64, n)
	spendRatio := make([]float64, n)
	uniqueUsers := make([]float64, n)
	uniqueUAs := make([]float64, n)
	evPerUA := make([]float64, n)
	clkPerUA := make([]float64, n)
	usrPerUA := make([]float64, n)
	clkPerUsr := make([]float64, n)
	spendPerClk := make([]float64, n)
	uaPerEv := make([]float64, n)
	evPerUsr := make([]float64, n)
	evPerClk := make([]float64, n)
	usrPerClk := make([]float64, n)

	for i, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		ev := float64(row.Events)
		cl := float64(row.Clicks)
		uu := float64(row.UniqueUsers)
		ua := float64(row.UniqueUAs)
		sn := float64(row.SpendMicro) / 1e6
		sr := safeRatio(float64(row.SpendMicro), float64(row.BudgetLimitMicro))
		events[i] = ev
		clicks[i] = cl
		ctr[i] = safeRatio(cl, ev)
		spendNorm[i] = sn
		spendRatio[i] = sr
		uniqueUsers[i] = uu
		uniqueUAs[i] = ua
		evPerUA[i] = safeRatio(ev, ua)
		clkPerUA[i] = safeRatio(cl, ua)
		usrPerUA[i] = safeRatio(uu, ua)
		clkPerUsr[i] = safeRatio(cl, uu)
		spendPerClk[i] = safeRatio(sn, cl)
		uaPerEv[i] = safeRatio(ua, ev)
		evPerUsr[i] = safeRatio(ev, uu)
		evPerClk[i] = safeRatio(ev, cl+1)
		usrPerClk[i] = safeRatio(uu, cl+1)
	}

	var features [featureVectorDims]float64
	for i := range rows {
		features[0] = events[i]
		features[1] = clicks[i]
		features[2] = ctr[i]
		features[3] = spendNorm[i]
		features[4] = spendRatio[i]
		features[5] = uniqueUsers[i]
		features[6] = uniqueUAs[i]
		features[7] = evPerUA[i]
		features[8] = clkPerUA[i]
		features[9] = usrPerUA[i]
		features[10] = clkPerUsr[i]
		features[11] = spendPerClk[i]
		features[12] = uaPerEv[i]
		features[13] = evPerUsr[i]
		features[14] = evPerClk[i]
		features[15] = usrPerClk[i]
		out[i] = model.PredictProbability(features[:])
	}
	return nil
}
