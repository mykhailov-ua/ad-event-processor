package fraud

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testNativeStumpModel = `tree
version=v3
num_class=1
num_tree_per_iteration=1
label_index=0
max_feature_idx=15
objective=binary sigmoid:1
Tree=0
num_leaves=2
num_cat=0
split_feature=0
split_gain=1
threshold=10.0
decision_type=2
left_child=-1
right_child=-2
leaf_value=-1.0 1.0
`

func TestNativeTree_stump(t *testing.T) {
	tree := nativeTree{
		splitFeature: []int{0},
		threshold:    []float64{10},
		leftChild:    []int{-1},
		rightChild:   []int{-2},
		leafValue:    []float64{-1, 1},
	}
	low := tree.predict([]float64{5})
	high := tree.predict([]float64{20})
	require.Equal(t, -1.0, low)
	require.Equal(t, 1.0, high)
}

func TestLoadNativeModel_inline(t *testing.T) {
	path := writeTempModel(t, testNativeStumpModel)
	model, err := LoadNativeModel(path)
	require.NoError(t, err)
	require.Len(t, model.trees, 1)

	low := model.PredictProbability([]float64{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	high := model.PredictProbability([]float64{20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	require.Less(t, low, high)
}

func TestNewLGBMScorer_prefersNative(t *testing.T) {
	path := writeTempModel(t, testNativeStumpModel)
	scorer, err := NewLGBMScorer(path)
	require.NoError(t, err)
	require.Equal(t, "lightgbm_native", scorer.Name())
}

func writeTempModel(t *testing.T, body string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "lgbm-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString(body)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}
