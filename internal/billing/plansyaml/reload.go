package plansyaml

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	billingdb "espx/internal/billing/db"
	"espx/internal/ingestion"
	lic "espx/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

type PlansFile struct {
	Plans       []PlanDefinition       `yaml:"plans"`
	Assignments []AssignmentDefinition `yaml:"assignments"`
}

type PlanDefinition struct {
	Code         string         `yaml:"code"`
	DisplayName  string         `yaml:"display_name"`
	BaseFeeMicro int64          `yaml:"base_fee_micro"`
	Limits       lic.Limits     `yaml:"limits"`
	Features     lic.FeatureSet `yaml:"features"`
}

type AssignmentDefinition struct {
	CustomerID  string `yaml:"customer_id"`
	PlanCode    string `yaml:"plan_code"`
	Status      string `yaml:"status"`
	PeriodStart string `yaml:"period_start"`
	PeriodEnd   string `yaml:"period_end"`
	Overrides   struct {
		Limits   *lic.Limits     `yaml:"limits,omitempty"`
		Features *lic.FeatureSet `yaml:"features,omitempty"`
	} `yaml:"overrides"`
}

type ReloadReport struct {
	Path          string `json:"path"`
	DryRun        bool   `json:"dry_run"`
	PlansUpserted int    `json:"plans_upserted"`
	Assignments   int    `json:"assignments_upserted"`
}

func DefaultPlansPath() string {
	if p := os.Getenv("OPERATOR_PLANS_YAML"); p != "" {
		return p
	}
	for _, candidate := range []string{"deploy/operator/plans.yaml", "../deploy/operator/plans.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "deploy/operator/plans.yaml"
}

func Load(path string) (*PlansFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc PlansFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(doc.Plans))
	for i, plan := range doc.Plans {
		code := strings.TrimSpace(plan.Code)
		if code == "" {
			return nil, fmt.Errorf("plans[%d]: code is required", i)
		}
		if _, ok := seen[code]; ok {
			return nil, fmt.Errorf("duplicate plan code %q", code)
		}
		seen[code] = struct{}{}
		doc.Plans[i].Code = code
		if strings.TrimSpace(plan.DisplayName) == "" {
			doc.Plans[i].DisplayName = code
		}
	}
	for i, a := range doc.Assignments {
		if _, err := uuid.Parse(strings.TrimSpace(a.CustomerID)); err != nil {
			return nil, fmt.Errorf("assignments[%d]: invalid customer_id", i)
		}
		if strings.TrimSpace(a.PlanCode) == "" {
			return nil, fmt.Errorf("assignments[%d]: plan_code is required", i)
		}
		doc.Assignments[i].PlanCode = strings.TrimSpace(a.PlanCode)
		if strings.TrimSpace(a.Status) == "" {
			doc.Assignments[i].Status = "active"
		}
	}
	return &doc, nil
}

type Fanout func(ctx context.Context) error

func Reload(ctx context.Context, pool *pgxpool.Pool, path string, dryRun bool, fanout Fanout) (ReloadReport, error) {
	report := ReloadReport{Path: path, DryRun: dryRun}
	doc, err := Load(path)
	if err != nil {
		return report, err
	}
	if dryRun {
		report.PlansUpserted = len(doc.Plans)
		report.Assignments = len(doc.Assignments)
		return report, nil
	}
	if pool == nil {
		return report, fmt.Errorf("postgres pool is nil")
	}
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := billingdb.New(tx)
		planCodes := make(map[string]struct{}, len(doc.Plans))
		for _, plan := range doc.Plans {
			limitsJSON, err := json.Marshal(plan.Limits)
			if err != nil {
				return err
			}
			featuresJSON, err := json.Marshal(plan.Features.Normalized())
			if err != nil {
				return err
			}
			if _, err := q.UpsertSubscriptionPlan(ctx, billingdb.UpsertSubscriptionPlanParams{
				Code:         plan.Code,
				DisplayName:  plan.DisplayName,
				LimitsJson:   limitsJSON,
				FeaturesJson: featuresJSON,
				BaseFeeMicro: plan.BaseFeeMicro,
			}); err != nil {
				return err
			}
			planCodes[plan.Code] = struct{}{}
			report.PlansUpserted++
		}
		for _, assignment := range doc.Assignments {
			if _, ok := planCodes[assignment.PlanCode]; !ok {
				if _, err := q.GetSubscriptionPlan(ctx, assignment.PlanCode); err != nil {
					return fmt.Errorf("assignment plan %q not found", assignment.PlanCode)
				}
			}
			custID, err := uuid.Parse(strings.TrimSpace(assignment.CustomerID))
			if err != nil {
				return err
			}
			periodStart := time.Now().UTC()
			if assignment.PeriodStart != "" {
				periodStart, err = time.Parse("2006-01-02", assignment.PeriodStart)
				if err != nil {
					return fmt.Errorf("invalid period_start for %s: %w", assignment.CustomerID, err)
				}
			}
			var periodEnd pgtype.Date
			if assignment.PeriodEnd != "" {
				t, parseErr := time.Parse("2006-01-02", assignment.PeriodEnd)
				if parseErr != nil {
					return fmt.Errorf("invalid period_end for %s: %w", assignment.CustomerID, parseErr)
				}
				periodEnd = pgtype.Date{Time: t, Valid: true}
			}
			var overridesRaw []byte
			if assignment.Overrides.Limits != nil || assignment.Overrides.Features != nil {
				overridesRaw, err = json.Marshal(assignment.Overrides)
				if err != nil {
					return err
				}
			}
			if _, err := q.UpsertCustomerSubscription(ctx, billingdb.UpsertCustomerSubscriptionParams{
				CustomerID:    ingestion.ToUUID(custID),
				PlanCode:      assignment.PlanCode,
				Status:        assignment.Status,
				PeriodStart:   pgtype.Date{Time: periodStart, Valid: true},
				PeriodEnd:     periodEnd,
				OverridesJson: overridesRaw,
			}); err != nil {
				return err
			}
			report.Assignments++
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	if fanout != nil {
		if err := fanout(ctx); err != nil {
			return report, err
		}
	}
	return report, nil
}
