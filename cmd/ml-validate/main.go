package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/fraud"
)

type featureFixture struct {
	ID           string     `json:"id"`
	FeatureNames []string   `json:"feature_names"`
	Row          fixtureRow `json:"row"`
	Vector       []float64  `json:"vector"`
}

type fixtureRow struct {
	Events           uint64 `json:"events"`
	Clicks           uint64 `json:"clicks"`
	SpendMicro       int64  `json:"spend_micro"`
	BudgetLimitMicro int64  `json:"budget_limit_micro"`
	UniqueUsers      uint64 `json:"unique_users"`
	UniqueUAs        uint64 `json:"unique_uas"`
}

func checkModel(modelPath string) error {
	scorer, err := fraud.NewLGBMScorer(modelPath)
	if err != nil {
		return fmt.Errorf("load model: %w", err)
	}
	if scorer.Dims() != fraud.Dims() {
		return fmt.Errorf("model NFeatures=%d want %d", scorer.Dims(), fraud.Dims())
	}
	return nil
}

func validateFixtures(fixturesDir string) error {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return fmt.Errorf("read fixtures dir: %w", err)
	}

	var errs []error
	var validated int
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

		row := fraud.FeatureRow{
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
		validated++
	}

	if validated == 0 {
		errs = append(errs, errors.New("no features_*.json fixtures were validated"))
	}
	return errors.Join(errs...)
}

func main() {
	modelPath := "var/fraudscore/artifacts/model.txt"
	fixturesDir := "var/fraudscore/fixtures"
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

	if err := checkModel(modelPath); err != nil {
		fmt.Fprintf(os.Stderr, "ml-validate: %v\n", err)
		os.Exit(1)
	}

	if err := validateFixtures(fixturesDir); err != nil {
		fmt.Fprintf(os.Stderr, "ml-validate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ml-validate: OK model=%s dims=%d fixtures=%s\n", modelPath, fraud.Dims(), fixturesDir)
}
