package main

import (
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/fraudscoring"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type replayOptions struct {
	modelPath   string
	fixturesDir string
	useCH       bool
	limit       int
	minutes     int
	outputPath  string
}

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

type replayRow struct {
	Source      string
	WindowStart string
	IPHash      string
	CampaignID  string
	FeatureRow  fraudscoring.FeatureRow
}

func loadScorer(modelPath string) (*fraudscoring.LGBMScorer, error) {
	scorer, err := fraudscoring.NewLGBMScorer(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}
	if scorer.Dims() != fraudscoring.Dims() {
		return nil, fmt.Errorf("model NFeatures=%d want %d", scorer.Dims(), fraudscoring.Dims())
	}
	return scorer, nil
}

func loadFixtureRows(fixturesDir string) ([]replayRow, error) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return nil, fmt.Errorf("read fixtures dir: %w", err)
	}

	var rows []replayRow
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "features_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(fixturesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}

		var fixture featureFixture
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("%s: decode: %w", entry.Name(), err)
		}

		id := fixture.ID
		if id == "" {
			id = strings.TrimSuffix(entry.Name(), ".json")
		}
		rows = append(rows, replayRow{
			Source: id,
			FeatureRow: fraudscoring.FeatureRow{
				Events:           fixture.Row.Events,
				Clicks:           fixture.Row.Clicks,
				SpendMicro:       fixture.Row.SpendMicro,
				BudgetLimitMicro: fixture.Row.BudgetLimitMicro,
				UniqueUsers:      fixture.Row.UniqueUsers,
				UniqueUAs:        fixture.Row.UniqueUAs,
			},
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no features_*.json fixtures in %s", fixturesDir)
	}
	return rows, nil
}

func loadCHRows(ctx context.Context, conn driver.Conn, limit, minutes int) ([]replayRow, error) {
	query := `
SELECT
    window_start,
    ip_hash,
    campaign_id,
    events,
    clicks,
    spend_micro,
    budget_limit_micro,
    unique_users,
    unique_uas
FROM ml_features_1m
WHERE window_start >= now() - INTERVAL ? MINUTE
ORDER BY window_start DESC
LIMIT ?`

	rows, err := conn.Query(ctx, query, minutes, limit)
	if err != nil {
		return nil, fmt.Errorf("clickhouse query: %w", err)
	}
	defer rows.Close()

	var out []replayRow
	for rows.Next() {
		var fr fraudscoring.FeatureRow
		var campaignID string
		var ipHash []byte
		if err := rows.Scan(
			&fr.WindowStart,
			&ipHash,
			&campaignID,
			&fr.Events,
			&fr.Clicks,
			&fr.SpendMicro,
			&fr.BudgetLimitMicro,
			&fr.UniqueUsers,
			&fr.UniqueUAs,
		); err != nil {
			return nil, fmt.Errorf("clickhouse scan: %w", err)
		}
		fr.CampaignID = campaignID
		fr.IPAddress = hex.EncodeToString(ipHash)
		out = append(out, replayRow{
			Source:      "clickhouse",
			WindowStart: fr.WindowStart.UTC().Format(time.RFC3339),
			IPHash:      fr.IPAddress,
			CampaignID:  campaignID,
			FeatureRow:  fr,
		})
	}
	return out, rows.Err()
}

func writeReplayCSV(w io.Writer, rows []replayRow, scores []float64) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"source", "window_start", "ip_hash", "campaign_id",
		"events", "clicks", "spend_micro", "budget_limit_micro", "unique_users", "unique_uas",
		"ml_score", "fraud_score", "tier", "action",
	}); err != nil {
		return err
	}

	pass := uint8(fraudscoring.FraudTierPassMax)
	suspect := uint8(fraudscoring.FraudTierSuspectMax)
	ivt := uint8(fraudscoring.FraudTierIVTMax)
	block := uint8(100)

	for i, row := range rows {
		mlScore := scores[i]
		decision := fraudscoring.DecideWithCampaign(row.FeatureRow, mlScore, pass, suspect, ivt, block)
		action := shadowAction(decision.Tier, true)
		if err := cw.Write([]string{
			row.Source,
			row.WindowStart,
			row.IPHash,
			row.CampaignID,
			strconv.FormatUint(row.FeatureRow.Events, 10),
			strconv.FormatUint(row.FeatureRow.Clicks, 10),
			strconv.FormatInt(row.FeatureRow.SpendMicro, 10),
			strconv.FormatInt(row.FeatureRow.BudgetLimitMicro, 10),
			strconv.FormatUint(row.FeatureRow.UniqueUsers, 10),
			strconv.FormatUint(row.FeatureRow.UniqueUAs, 10),
			strconv.FormatFloat(mlScore, 'f', 6, 64),
			strconv.Itoa(decision.Score),
			string(decision.Tier),
			action,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func shadowAction(tier fraudscoring.FraudTier, ghostEnabled bool) string {
	switch tier {
	case fraudscoring.FraudTierSuspect:
		return "boost"
	case fraudscoring.FraudTierIVT:
		if ghostEnabled {
			return "ghost"
		}
	case fraudscoring.FraudTierBlock:
		return "blacklist"
	}
	return ""
}

func runReplay(ctx context.Context, opts replayOptions) error {
	scorer, err := loadScorer(opts.modelPath)
	if err != nil {
		return err
	}

	var rows []replayRow
	switch {
	case opts.useCH:
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		dsn := string(cfg.CHReadonlyDSN)
		if dsn == "" {
			dsn = string(cfg.CHDSN)
		}
		if dsn == "" {
			return fmt.Errorf("CH_READONLY_DSN or CH_DSN required for -clickhouse")
		}
		conn, err := database.ConnectCHReadonly(ctx, dsn)
		if err != nil {
			return fmt.Errorf("connect clickhouse: %w", err)
		}
		defer func() { _ = conn.Close() }()
		rows, err = loadCHRows(ctx, conn, opts.limit, opts.minutes)
		if err != nil {
			return err
		}
	default:
		rows, err = loadFixtureRows(opts.fixturesDir)
		if err != nil {
			return err
		}
	}

	if len(rows) == 0 {
		return fmt.Errorf("no rows to score")
	}

	featureRows := make([]fraudscoring.FeatureRow, len(rows))
	for i := range rows {
		featureRows[i] = rows[i].FeatureRow
	}
	scores, err := scorer.ScoreBatch(ctx, featureRows)
	if err != nil {
		return fmt.Errorf("score batch: %w", err)
	}
	if len(scores) != len(rows) {
		return fmt.Errorf("score count mismatch: got %d want %d", len(scores), len(rows))
	}

	var out io.Writer = os.Stdout
	var file *os.File
	if opts.outputPath != "" {
		file, err = os.Create(opts.outputPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer file.Close()
		out = file
	}

	if err := writeReplayCSV(out, rows, scores); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "ml-replay: scored %d rows model=%s\n", len(rows), opts.modelPath)
	return nil
}

func parseOptions(args []string) (replayOptions, error) {
	opts := replayOptions{
		modelPath:   "var/fraudscore/artifacts/model.txt",
		fixturesDir: "testdata/ml",
		limit:       1000,
		minutes:     5,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-model":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("missing value for -model")
			}
			opts.modelPath = args[i]
		case "-fixtures":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("missing value for -fixtures")
			}
			opts.fixturesDir = args[i]
		case "-clickhouse":
			opts.useCH = true
		case "-limit":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("missing value for -limit")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("invalid -limit: %s", args[i])
			}
			opts.limit = n
		case "-minutes":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("missing value for -minutes")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("invalid -minutes: %s", args[i])
			}
			opts.minutes = n
		case "-output":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("missing value for -output")
			}
			opts.outputPath = args[i]
		default:
			return opts, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return opts, nil
}

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ml-replay: %v\n", err)
		os.Exit(2)
	}
	if err := runReplay(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "ml-replay: %v\n", err)
		os.Exit(1)
	}
}
