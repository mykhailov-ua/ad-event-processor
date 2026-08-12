package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"
)

const (
	ProfileIngestOnly      = "ingest_only"
	ProfileNetworkOperator = "network_operator"
	ProfileAnalyticsML     = "analytics_ml"
)

var knownDeployProfiles = map[string]struct{}{
	ProfileIngestOnly:      {},
	ProfileNetworkOperator: {},
	ProfileAnalyticsML:     {},
}

type profileComposeExpect struct {
	mustInclude     []string
	mustNotInclude  []string
	composeProfiles []string
}

var profileComposeSpecs = map[string]profileComposeExpect{
	ProfileIngestOnly: {
		mustInclude:     []string{"control", "processor", "tracker-0", "db"},
		mustNotInclude:  []string{"clickhouse", "payment", "billing", "ivt-detector", "fraud-scorer"},
		composeProfiles: []string{"ingest_only"},
	},
	ProfileNetworkOperator: {
		mustInclude:     []string{"control", "clickhouse", "processor", "db", "db-payment"},
		mustNotInclude:  []string{"payment", "billing"},
		composeProfiles: []string{"network_operator"},
	},
	ProfileAnalyticsML: {
		mustInclude:     []string{"clickhouse", "ivt-detector", "fraud-scorer"},
		composeProfiles: []string{"analytics_ml", "fraud-scorer"},
	},
}

func DeployProfileChecklist(profile string, cfg *config.Config) []ChecklistRow {
	profile = strings.TrimSpace(strings.ToLower(profile))
	if _, ok := knownDeployProfiles[profile]; !ok {
		return []ChecklistRow{{
			ID:     "profile_name",
			Status: StatusFail,
			Detail: fmt.Sprintf("unknown profile %q (want ingest_only, network_operator, analytics_ml)", profile),
		}}
	}

	var rows []ChecklistRow
	rows = append(rows, profileEnvRows(profile, cfg)...)
	rows = append(rows, profileComposeRows(profile)...)
	rows = append(rows, profileContainerRows(profile)...)
	return rows
}

func profileEnvRows(profile string, cfg *config.Config) []ChecklistRow {
	switch profile {
	case ProfileIngestOnly:
		return []ChecklistRow{
			checkEnvBool("ch_enabled", "CH_ENABLED", false, "ingest_only requires CH_ENABLED=0"),
			checkEnvBool("control_payment", "CONTROL_ENABLE_PAYMENT", false, "payment disabled in ingest_only"),
			checkEnvBool("control_billing", "CONTROL_ENABLE_BILLING", false, "billing disabled in ingest_only"),
			checkEnvBool("control_notifier", "CONTROL_ENABLE_NOTIFIER", false, "notifier disabled in ingest_only"),
			checkBaseDeployEnv(cfg),
		}
	case ProfileNetworkOperator:
		rows := []ChecklistRow{
			checkEnvBoolDefaultOn("control_payment", "CONTROL_ENABLE_PAYMENT", "wallet rail enabled"),
			checkEnvBoolDefaultOn("control_billing", "CONTROL_ENABLE_BILLING", "billing enabled"),
			checkBaseDeployEnv(cfg),
		}
		if !clickHouseConfigured(cfg) {
			rows = append(rows, ChecklistRow{
				ID:     "ch_enabled",
				Status: StatusWarn,
				Detail: "CH_ENABLED=0; analytics APIs stale without ClickHouse",
			})
		}
		return rows
	case ProfileAnalyticsML:
		rows := []ChecklistRow{checkBaseDeployEnv(cfg)}
		if !clickHouseConfigured(cfg) {
			rows = append(rows, ChecklistRow{ID: "ch_enabled", Status: StatusFail, Detail: "analytics_ml requires CH_ENABLED=1 and CH_DSN"})
		} else {
			rows = append(rows, ChecklistRow{ID: "ch_enabled", Status: StatusPass, Detail: "ClickHouse enabled"})
		}
		if strings.TrimSpace(os.Getenv("ADMIN_API_KEY")) == "" {
			rows = append(rows, ChecklistRow{ID: "admin_api_key", Status: StatusFail, Detail: "ADMIN_API_KEY required for ivt-detector/fraud-scorer"})
		} else {
			rows = append(rows, ChecklistRow{ID: "admin_api_key", Status: StatusPass, Detail: "ADMIN_API_KEY set"})
		}
		return rows
	default:
		return nil
	}
}

func checkBaseDeployEnv(cfg *config.Config) ChecklistRow {
	dsn := ""
	if cfg != nil {
		dsn = string(cfg.DBDSN)
	}
	if strings.TrimSpace(dsn) == "" {
		dsn = strings.TrimSpace(os.Getenv("DB_DSN"))
	}
	if dsn == "" {
		return ChecklistRow{ID: "db_dsn", Status: StatusFail, Detail: "DB_DSN not set"}
	}
	if strings.TrimSpace(os.Getenv("REDIS_PASSWORD")) == "" {
		return ChecklistRow{ID: "redis_password", Status: StatusFail, Detail: "REDIS_PASSWORD not set"}
	}
	return ChecklistRow{ID: "base_env", Status: StatusPass, Detail: "DB_DSN and REDIS_PASSWORD configured"}
}

func clickHouseConfigured(cfg *config.Config) bool {
	if cfg != nil && cfg.ClickHouseEnabled() {
		return true
	}
	enabled, unset := envBool("CH_ENABLED")
	if unset {
		enabled = true
	}
	return enabled && strings.TrimSpace(os.Getenv("CH_DSN")) != ""
}

func checkEnvBool(id, key string, want bool, failDetail string) ChecklistRow {
	got, unset := envBool(key)
	if unset {
		if want {
			return ChecklistRow{ID: id, Status: StatusFail, Detail: failDetail + " (" + key + " unset)"}
		}
		return ChecklistRow{ID: id, Status: StatusPass, Detail: key + " unset (default off)"}
	}
	if got != want {
		return ChecklistRow{ID: id, Status: StatusFail, Detail: failDetail}
	}
	return ChecklistRow{ID: id, Status: StatusPass, Detail: key + " ok"}
}

func checkEnvBoolDefaultOn(id, key, passDetail string) ChecklistRow {
	got, unset := envBool(key)
	if unset || got {
		return ChecklistRow{ID: id, Status: StatusPass, Detail: passDetail}
	}
	return ChecklistRow{ID: id, Status: StatusWarn, Detail: key + "=0; network_operator expects wallet rail on"}
}

func envBool(key string) (bool, bool) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return false, true
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "no", "off":
		return false, false
	default:
		return true, false
	}
}

func profileComposeRows(profile string) []ChecklistRow {
	spec, ok := profileComposeSpecs[profile]
	if !ok {
		return nil
	}
	services, err := composeServices(spec.composeProfiles...)
	if err != nil {
		return []ChecklistRow{{ID: "compose_config", Status: StatusSkip, Detail: err.Error()}}
	}
	set := make(map[string]struct{}, len(services))
	for _, s := range services {
		set[s] = struct{}{}
	}
	var rows []ChecklistRow
	for _, name := range spec.mustInclude {
		if _, ok := set[name]; !ok {
			rows = append(rows, ChecklistRow{
				ID:     "compose_include_" + name,
				Status: StatusFail,
				Detail: fmt.Sprintf("compose profile missing service %s", name),
			})
		} else {
			rows = append(rows, ChecklistRow{
				ID:     "compose_include_" + name,
				Status: StatusPass,
				Detail: name + " in compose profile",
			})
		}
	}
	for _, name := range spec.mustNotInclude {
		if _, ok := set[name]; ok {
			rows = append(rows, ChecklistRow{
				ID:     "compose_exclude_" + name,
				Status: StatusFail,
				Detail: fmt.Sprintf("compose profile must not include %s", name),
			})
		} else {
			rows = append(rows, ChecklistRow{
				ID:     "compose_exclude_" + name,
				Status: StatusPass,
				Detail: name + " absent from compose profile",
			})
		}
	}
	return rows
}

func profileContainerRows(profile string) []ChecklistRow {
	running, err := runningComposeServices()
	if err != nil {
		return []ChecklistRow{{ID: "containers", Status: StatusSkip, Detail: err.Error()}}
	}
	if len(running) == 0 {
		return []ChecklistRow{{ID: "containers", Status: StatusSkip, Detail: "no running compose containers detected"}}
	}
	set := make(map[string]struct{}, len(running))
	for _, s := range running {
		set[s] = struct{}{}
	}
	spec := profileComposeSpecs[profile]
	var rows []ChecklistRow
	for _, name := range spec.mustInclude {
		if _, ok := set[name]; !ok {
			rows = append(rows, ChecklistRow{
				ID:     "running_" + name,
				Status: StatusWarn,
				Detail: fmt.Sprintf("service %s not running (stack may be partial)", name),
			})
		} else {
			rows = append(rows, ChecklistRow{
				ID:     "running_" + name,
				Status: StatusPass,
				Detail: name + " running",
			})
		}
	}
	for _, name := range spec.mustNotInclude {
		if _, ok := set[name]; ok {
			rows = append(rows, ChecklistRow{
				ID:     "running_exclude_" + name,
				Status: StatusFail,
				Detail: fmt.Sprintf("service %s must not run under profile %s", name, profile),
			})
		}
	}
	return rows
}

func composeServices(profiles ...string) ([]string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not available for compose config check")
	}
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	args := []string{"compose"}
	for _, p := range profiles {
		args = append(args, "--profile", p)
	}
	args = append(args, "config", "--services")
	cmd := exec.Command("docker", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose config: %w", err)
	}
	var services []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "time=") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func runningComposeServices() ([]string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not available for container check")
	}
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("docker", "compose", "ps", "--status", "running", "--services")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose ps: %w", err)
	}
	var services []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}
	return services, nil
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("docker-compose.yaml not found from %s", wd)
}
