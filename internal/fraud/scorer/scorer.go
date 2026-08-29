package scorer

import (
	"context"
	"log/slog"
	"sync"

	"ad-event-processor/internal/fraud/features"

	"github.com/zhongdai/go-lgbm"
)

type Scorer interface {
	Name() string
	ScoreBatch(ctx context.Context, rows []features.FeatureRow) ([]float64, error)
	Dims() int
}

type LGBMScorer struct {
	model *lgbm.Model
	dims  int
	pool  sync.Pool
}

func NewLGBMScorer(modelPath string) (Scorer, error) {
	if native, err := NewNativeScorer(modelPath); err == nil {
		return native, nil
	}
	return newCGOLGBMScorer(modelPath)
}

func newCGOLGBMScorer(modelPath string) (*LGBMScorer, error) {
	model, err := lgbm.ModelFromFile(modelPath, true)
	if err != nil {
		model, err = lgbm.ModelFromFile(modelPath, false)
		if err != nil {
			slog.Error("failed to load LightGBM model", "model_path", modelPath, "error", err)
			return nil, err
		}
	}

	dims := model.NFeatures()
	return &LGBMScorer{
		model: model,
		dims:  dims,
		pool: sync.Pool{
			New: func() any {
				buf := make([]float64, 0, 10000)
				return &buf
			},
		},
	}, nil
}

func (lg *LGBMScorer) Name() string {
	return "lightgbm"
}

func (lg *LGBMScorer) Dims() int {
	return lg.dims
}

func (lg *LGBMScorer) ScoreBatch(ctx context.Context, rows []features.FeatureRow) ([]float64, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	nRows := len(rows)
	nCols := lg.dims

	pBuf := lg.pool.Get().(*[]float64)
	defer func() {
		*pBuf = (*pBuf)[:0]
		lg.pool.Put(pBuf)
	}()

	neededCap := nRows * nCols
	if cap(*pBuf) < neededCap {
		*pBuf = make([]float64, neededCap)
	} else {
		*pBuf = (*pBuf)[:neededCap]
	}

	flat := *pBuf

	var vec [features.FeatureVectorDims]float64
	for i, row := range rows {
		row.ToVectorInto(vec[:])
		offset := i * nCols
		for j := range nCols {
			if j < features.FeatureVectorDims {
				flat[offset+j] = vec[j]
			} else {
				flat[offset+j] = 0.0
			}
		}
	}

	out := make([]float64, nRows)
	err := lg.model.PredictDense(flat, nRows, nCols, 0, 0, out)
	if err != nil {
		slog.Error("lgbm PredictDense failed", "error", err)
		return nil, err
	}

	return out, nil
}
