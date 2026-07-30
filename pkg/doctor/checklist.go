package doctor

import (
	"os"
	"strings"

	"espx/internal/config"
)

type ChecklistRow struct {
	ID     string
	Status Status
	Detail string
}

func MVSSChecklist(cfg *config.Config) []ChecklistRow {
	return []ChecklistRow{
		checkManual("luks", "LUKS or cloud volume encryption on PG/Redis/CH/spool volumes"),
		checkManual("firewall", "Firewall: only ingress public; data stores on private subnet"),
		checkDBTLS(cfg),
		checkPIISalt(),
		checkRedisPassword(),
		checkTelemetryOptIn(),
		checkManual("rbac", "RBAC: operator admins separate from advertiser API keys"),
		checkManual("backups", "Encrypted backups with offline key"),
		checkEventsRetention(),
	}
}

func checkManual(id, detail string) ChecklistRow {
	return ChecklistRow{ID: id, Status: StatusSkip, Detail: "manual: " + detail}
}

func checkDBTLS(cfg *config.Config) ChecklistRow {
	if os.Getenv("ESPX_PROFILE") != "production" {
		return ChecklistRow{ID: "db_tls", Status: StatusSkip, Detail: "ESPX_PROFILE!=production"}
	}
	if cfg == nil || string(cfg.DBDSN) == "" {
		return ChecklistRow{ID: "db_tls", Status: StatusFail, Detail: "DB_DSN not set"}
	}
	mode, err := dsnSSLMode(string(cfg.DBDSN))
	if err != nil {
		return ChecklistRow{ID: "db_tls", Status: StatusFail, Detail: err.Error()}
	}
	if mode != "verify-full" {
		return ChecklistRow{ID: "db_tls", Status: StatusFail, Detail: "DB_DSN sslmode=" + mode + " want verify-full"}
	}
	return ChecklistRow{ID: "db_tls", Status: StatusPass, Detail: "postgres sslmode=verify-full"}
}

func checkPIISalt() ChecklistRow {
	salt := strings.TrimSpace(os.Getenv("PII_SALT_HEX"))
	if salt == "" {
		return ChecklistRow{ID: "pii_salt", Status: StatusFail, Detail: "PII_SALT_HEX not set"}
	}
	if isExampleSecret(salt, "your_pii_salt", "changeme", "deadbeef") {
		return ChecklistRow{ID: "pii_salt", Status: StatusFail, Detail: "PII_SALT_HEX looks like a placeholder"}
	}
	if len(salt) < 64 {
		return ChecklistRow{ID: "pii_salt", Status: StatusWarn, Detail: "PII_SALT_HEX shorter than 32 bytes (64 hex chars)"}
	}
	return ChecklistRow{ID: "pii_salt", Status: StatusPass, Detail: "unique salt configured"}
}

func checkRedisPassword() ChecklistRow {
	pass := os.Getenv("REDIS_PASSWORD")
	if pass == "" {
		return ChecklistRow{ID: "redis_password", Status: StatusFail, Detail: "REDIS_PASSWORD not set"}
	}
	if isExampleSecret(pass, "your_redis_password_here", "redis", "changeme") {
		return ChecklistRow{ID: "redis_password", Status: StatusFail, Detail: "REDIS_PASSWORD looks like a placeholder"}
	}
	return ChecklistRow{ID: "redis_password", Status: StatusPass, Detail: "password configured"}
}

func checkTelemetryOptIn() ChecklistRow {
	raw, ok := os.LookupEnv("ESPX_TELEMETRY_OPT_IN")
	if !ok {
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusPass, Detail: "unset (default off)"}
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "no", "off":
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusPass, Detail: "ESPX_TELEMETRY_OPT_IN=0"}
	case "1", "true", "yes", "on":
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusWarn, Detail: "ESPX_TELEMETRY_OPT_IN enabled; review policy"}
	default:
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusWarn, Detail: "unrecognized ESPX_TELEMETRY_OPT_IN value"}
	}
}

func checkEventsRetention() ChecklistRow {
	if _, ok := os.LookupEnv("EVENTS_RETENTION_DAYS"); ok {
		return ChecklistRow{ID: "pg_events_retention", Status: StatusPass, Detail: "EVENTS_RETENTION_DAYS set"}
	}
	if days := os.Getenv("MANAGEMENT_RETENTION_DAYS"); days != "" {
		return ChecklistRow{ID: "pg_events_retention", Status: StatusWarn, Detail: "MANAGEMENT_RETENTION_DAYS set; document PG events policy"}
	}
	return ChecklistRow{ID: "pg_events_retention", Status: StatusWarn, Detail: "set EVENTS_RETENTION_DAYS (see DATA_SECURITY.md)"}
}

func isExampleSecret(value string, examples ...string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, ex := range examples {
		if lower == strings.ToLower(ex) {
			return true
		}
	}
	return false
}

func ChecklistExitCode(rows []ChecklistRow) int {
	rep := Report{}
	for _, row := range rows {
		rep.Results = append(rep.Results, Result{Name: row.ID, Status: row.Status, Detail: row.Detail})
	}
	return rep.ExitCode()
}
