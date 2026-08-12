package installer

import (
	"github.com/bidshard/ad-event-processor/pkg/naming"
	"os"
	"path/filepath"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/runtimepaths"
)

func repoRoot() string {
	if root := os.Getenv("AD_EVENT_PROCESSOR_REPO_ROOT"); root != "" {
		return root
	}
	if root := os.Getenv(naming.LegacyVendorEnvKey("REPO_ROOT")); root != "" {
		return root
	}
	if root := os.Getenv("ROOT"); root != "" {
		return root
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}

func installRoot() string {
	return config.InstallRootFromEnv()
}

func secretsPath() string {
	if root := installRoot(); root != "" {
		return filepath.Join(root, "etc/ad-event-processor/secrets.env")
	}
	return runtimepaths.SecretsEnvPath()
}

func licensePath() string {
	if root := installRoot(); root != "" {
		return filepath.Join(root, "etc/ad-event-processor/license.jwt")
	}
	return runtimepaths.LicensePath()
}

func systemdUnitPath(name string) string {
	if root := installRoot(); root != "" {
		return filepath.Join(root, "etc/systemd/system", name)
	}
	return filepath.Join("/etc/systemd/system", name)
}

func packagesYAMLPath() string {
	return filepath.Join(repoRoot(), "deploy", "installer", "packages.yaml")
}

func checkDepsScript() string {
	return filepath.Join(repoRoot(), "scripts", "ci", "deps.sh")
}
