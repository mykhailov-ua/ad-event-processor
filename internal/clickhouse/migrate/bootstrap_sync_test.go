package migrate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/clickhouse/migrate"

	"github.com/stretchr/testify/require"
)

func TestBootstrapInitSQLMatchesMigration(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)

	initPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "deploy", "clickhouse", "init.sql")
	initBody, err := os.ReadFile(initPath)
	require.NoError(t, err)

	migrationBody, err := migrate.BootstrapMigrationSQL()
	require.NoError(t, err)

	got := migrate.NormalizeBootstrapDDL(migrate.InitSQLTableSection(string(initBody)))
	want := migrate.NormalizeBootstrapDDL(migrationBody)
	require.Equal(t, want, got,
		"deploy/clickhouse/init.sql table section must match %s; edit migration first, then init.sql",
		migrate.BootstrapMigrationFile(),
	)
}
