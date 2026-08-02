package fraud

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type nativeTree struct {
	splitFeature []int
	threshold    []float64
	leftChild    []int
	rightChild   []int
	leafValue    []float64
}

func (t *nativeTree) predict(features []float64) float64 {
	if len(t.splitFeature) == 0 {
		if len(t.leafValue) > 0 {
			return t.leafValue[0]
		}
		return 0
	}
	node := 0
	for {
		fidx := t.splitFeature[node]
		val := 0.0
		if fidx >= 0 && fidx < len(features) {
			val = features[fidx]
		}
		var next int
		if val <= t.threshold[node] {
			next = t.leftChild[node]
		} else {
			next = t.rightChild[node]
		}
		if next < 0 {
			return t.leafValue[^next]
		}
		node = next
	}
}

type NativeModel struct {
	trees      []nativeTree
	numClasses int
	dims       int
}

func LoadNativeModel(path string) (*NativeModel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	model := &NativeModel{numClasses: 1, dims: featureVectorDims}
	var tree *nativeTree
	parsingTree := false

	flushTree := func() error {
		if tree == nil {
			return nil
		}
		if len(tree.splitFeature) == 0 && len(tree.leafValue) == 0 {
			return fmt.Errorf("native lgbm: empty tree")
		}
		model.trees = append(model.trees, *tree)
		tree = nil
		parsingTree = false
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Tree=") {
			if err := flushTree(); err != nil {
				return nil, err
			}
			tree = &nativeTree{}
			parsingTree = true
			continue
		}
		if !parsingTree {
			if strings.HasPrefix(line, "max_feature_idx=") {
				v, err := strconv.Atoi(strings.TrimPrefix(line, "max_feature_idx="))
				if err == nil && v >= 0 {
					model.dims = v + 1
				}
			}
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "split_feature":
			tree.splitFeature = parseIntFields(val)
		case "threshold":
			tree.threshold = parseFloatFields(val)
		case "left_child":
			tree.leftChild = parseIntFields(val)
		case "right_child":
			tree.rightChild = parseIntFields(val)
		case "leaf_value":
			tree.leafValue = parseFloatFields(val)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := flushTree(); err != nil {
		return nil, err
	}
	if len(model.trees) == 0 {
		return nil, fmt.Errorf("native lgbm: no trees in %s", path)
	}
	return model, nil
}

func parseIntFields(s string) []int {
	fields := strings.Fields(s)
	out := make([]int, len(fields))
	for i, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil {
			out[i] = 0
			continue
		}
		out[i] = v
	}
	return out
}

func parseFloatFields(s string) []float64 {
	fields := strings.Fields(s)
	out := make([]float64, len(fields))
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			out[i] = 0
			continue
		}
		out[i] = v
	}
	return out
}

func (m *NativeModel) PredictProbability(features []float64) float64 {
	if m == nil {
		return 0
	}
	var sum float64
	for i := range m.trees {
		sum += m.trees[i].predict(features)
	}
	if m.numClasses <= 1 {
		return sigmoid(2 * sum)
	}
	return sigmoid(sum)
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		z := math.Exp(-x)
		return 1 / (1 + z)
	}
	z := math.Exp(x)
	return z / (1 + z)
}
