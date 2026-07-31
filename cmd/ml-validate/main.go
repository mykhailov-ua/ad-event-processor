package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"espx/internal/fraudscoring"
)

type featureFixture struct {
	ID           string     `json:"id"`
	FeatureNames []string   `json:"feature_names"`
	Row          fixtureRow `json:"row"`
	Vector       []float64  `json:"vector"`
	Score        *float64   `json:"score,omitempty"`
}

type fixtureRow struct {
	Events           uint64 `json:"events"`
	Clicks           uint64 `json:"clicks"`
	SpendMicro       int64  `json:"spend_micro"`
	BudgetLimitMicro int64  `json:"budget_limit_micro"`
	UniqueUsers      uint64 `json:"unique_users"`
	UniqueUAs        uint64 `json:"unique_uas"`
}

func validateModel(modelPath string) (*fraudscoring.LGBMScorer, error) {
	scorer, err := fraudscoring.NewLGBMScorer(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}
	if scorer.Dims() != fraudscoring.Dims() {
		return nil, fmt.Errorf("model NFeatures=%d want %d", scorer.Dims(), fraudscoring.Dims())
	}
	return scorer, nil
}

func validateFixtures(scorer *fraudscoring.LGBMScorer, fixturesDir string) error {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return fmt.Errorf("read fixtures dir: %w", err)
	}

	var errs []error
	var scored int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "features_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(fixturesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}

		var fixture featureFixture
		if err := json.Unmarshal(data, &fixture); err != nil {
			errs = append(errs, fmt.Errorf("%s: decode: %w", entry.Name(), err))
			continue
		}

		row := fraudscoring.FeatureRow{
			Events:           fixture.Row.Events,
			Clicks:           fixture.Row.Clicks,
			SpendMicro:       fixture.Row.SpendMicro,
			BudgetLimitMicro: fixture.Row.BudgetLimitMicro,
			UniqueUsers:      fixture.Row.UniqueUsers,
			UniqueUAs:        fixture.Row.UniqueUAs,
		}
		vec := row.ToVector()
		for i := range fixture.Vector {
			if i >= len(vec) || math.Abs(vec[i]-fixture.Vector[i]) > 1e-9 {
				errs = append(errs, fmt.Errorf("%s: vector[%d] got %v want %v", entry.Name(), i, vec, fixture.Vector))
				break
			}
		}

		if fixture.Score == nil {
			continue
		}
		scores, err := scorer.ScoreBatch(context.Background(), []fraudscoring.FeatureRow{row})
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: score: %w", entry.Name(), err))
			continue
		}
		if len(scores) != 1 || math.Abs(scores[0]-*fixture.Score) > 1e-4 {
			got := 0.0
			if len(scores) > 0 {
				got = scores[0]
			}
			errs = append(errs, fmt.Errorf("%s: score got %.5f want %.5f", entry.Name(), got, *fixture.Score))
			continue
		}
		scored++
	}

	if scored == 0 {
		errs = append(errs, errors.New("no fixtures with score field were validated"))
	}
	return errors.Join(errs...)
}

func main() {
	modelPath := "internal/fraudscoring/testdata/model.txt"
	fixturesDir := "testdata/ml"
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "-model":
				i++
				if i >= len(os.Args) {
					fmt.Fprintln(os.Stderr, "ml-validate: missing value for -model")
					os.Exit(2)
				}
				modelPath = os.Args[i]
			case "-fixtures":
				i++
				if i >= len(os.Args) {
					fmt.Fprintln(os.Stderr, "ml-validate: missing value for -fixtures")
					os.Exit(2)
				}
				fixturesDir = os.Args[i]
			}
		}
	}

	scorer, err := validateModel(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ml-validate: %v\n", err)
		os.Exit(1)
	}

	if err := validateFixtures(scorer, fixturesDir); err != nil {
		fmt.Fprintf(os.Stderr, "ml-validate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ml-validate: OK model=%s dims=%d fixtures=%s\n", modelPath, fraudscoring.Dims(), fixturesDir)
}
