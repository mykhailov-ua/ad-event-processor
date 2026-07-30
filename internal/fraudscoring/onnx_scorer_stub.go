//go:build !fraudscoring_onnx

package fraudscoring

import (
	"context"
	"errors"
)

type ONNXScorer struct{}

func NewONNXScorer(modelPath string) (*ONNXScorer, error) {
	return nil, errors.New("ONNX scorer is not enabled in this build. Rebuild with -tags fraudscoring_onnx")
}

func (o *ONNXScorer) Name() string {
	return "onnx_stub"
}

func (o *ONNXScorer) Dims() int {
	return 0
}

func (o *ONNXScorer) ScoreBatch(ctx context.Context, rows []FeatureRow) ([]float64, error) {
	return nil, errors.New("ONNX scorer is not enabled in this build")
}
