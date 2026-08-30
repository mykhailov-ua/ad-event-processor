package testutil

import (
	"path/filepath"
	"runtime"
)

func ModuleRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func AdsMigrationsDir() string {
	return filepath.Join(ModuleRoot(), "internal", "ingest", "migrations")
}

func AuthMigrationsDir() string {
	return filepath.Join(ModuleRoot(), "internal", "auth", "migrations")
}

func PaymentMigrationsDir() string {
	return filepath.Join(ModuleRoot(), "internal", "payment", "migrations")
}

func BillingMigrationsDir() string {
	return filepath.Join(ModuleRoot(), "internal", "ledger", "migrations")
}

func ServiceMigrationsDir(service string) string {
	return filepath.Join(ModuleRoot(), "internal", service, "migrations")
}

func NotifyMigrationsDir() string {
	return ServiceMigrationsDir("notify")
}
