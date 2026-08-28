package doctor

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/naming"
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
		checkEventsRetention(cfg),
		checkRedisTLS(),
	}
}

func checkManual(id, detail string) ChecklistRow {
	return ChecklistRow{ID: id, Status: StatusSkip, Detail: "manual: " + detail}
}

func checkDBTLS(cfg *config.Config) ChecklistRow {
	if os.Getenv(naming.LegacyVendorEnvKey("PROFILE")) != "production" {
		return ChecklistRow{ID: "db_tls", Status: StatusSkip, Detail: naming.LegacyVendorEnvKey("PROFILE") + "!=production"}
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
	raw, ok := os.LookupEnv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"))
	if !ok {
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusPass, Detail: "unset (default off)"}
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "no", "off":
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusPass, Detail: naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN") + "=0"}
	case "1", "true", "yes", "on":
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusWarn, Detail: naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN") + " enabled; review policy"}
	default:
		return ChecklistRow{ID: "telemetry_opt_in", Status: StatusWarn, Detail: "unrecognized " + naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN") + " value"}
	}
}

func checkEventsRetention(cfg *config.Config) ChecklistRow {
	days := 0
	if cfg != nil && cfg.EventsRetentionDays > 0 {
		days = cfg.EventsRetentionDays
	} else if raw, ok := os.LookupEnv("EVENTS_RETENTION_DAYS"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
			days = n
		}
	}
	if days > 0 {
		return ChecklistRow{ID: "pg_events_retention", Status: StatusPass, Detail: fmt.Sprintf("EVENTS_RETENTION_DAYS=%d", days)}
	}
	return ChecklistRow{ID: "pg_events_retention", Status: StatusWarn, Detail: "set EVENTS_RETENTION_DAYS (default 90 when loaded via config.Load)"}
}

func checkRedisTLS() ChecklistRow {
	if os.Getenv(naming.LegacyVendorEnvKey("PROFILE")) != "production" {
		return ChecklistRow{ID: "redis_tls", Status: StatusSkip, Detail: naming.LegacyVendorEnvKey("PROFILE") + "!=production"}
	}
	ca := strings.TrimSpace(os.Getenv("REDIS_TLS_CA"))
	cert := strings.TrimSpace(os.Getenv("REDIS_TLS_CERT"))
	key := strings.TrimSpace(os.Getenv("REDIS_TLS_KEY"))
	if ca == "" || cert == "" || key == "" {
		return ChecklistRow{ID: "redis_tls", Status: StatusFail, Detail: "set REDIS_TLS_CA, REDIS_TLS_CERT, REDIS_TLS_KEY"}
	}
	return ChecklistRow{ID: "redis_tls", Status: StatusPass, Detail: "redis client TLS cert paths configured"}
}

func isExampleSecret(value string, examples ...string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, ex := range examples {
		if strings.EqualFold(lower, ex) {
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
