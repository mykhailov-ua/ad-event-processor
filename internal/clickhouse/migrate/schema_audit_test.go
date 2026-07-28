package migrate_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"espx/internal/clickhouse/migrate"

	"github.com/stretchr/testify/require"
)

func TestSchemaAudit_initSQL_noRawPIIColumns(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	initPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "deploy", "clickhouse", "init.sql")
	initBody, err := os.ReadFile(initPath)
	require.NoError(t, err)

	violations := migrate.AuditSchemaDDL(map[string]string{
		"deploy/clickhouse/init.sql": string(initBody),
	})
	for _, v := range violations {
		t.Errorf("forbidden PII column %s.%s in %s", v.Table, v.Column, v.File)
	}
}

func TestMigration00010_dropsRawPIIColumns(t *testing.T) {
	body, err := fs.ReadFile(migrate.ClickHouseMigrationFS(), "migrations/00010_pii_hash_columns.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(body))
	for _, col := range migrate.ForbiddenCHColumns {
		require.Contains(t, sql, "drop column if exists "+col, "migration 00010 must drop %s", col)
	}
}
