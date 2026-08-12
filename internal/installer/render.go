package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func renderTemplates(profile *InstallProfile, dryRun bool) error {
	if err := profile.Validate(); err != nil {
		return err
	}

	unit, err := renderSystemdUnit(profile)
	if err != nil {
		return err
	}

	secrets := renderSecrets(profile)
	composeEnv := renderComposeEnv(profile)

	manifests := []struct {
		path    string
		content []byte
		mode    os.FileMode
	}{
		{systemdUnitPath(TrackerSystemdUnitName), unit, 0o644},
		{secretsPath(), secrets, 0o600},
	}
	if profile.Type == ProfileComposeDev {
		manifests = append(manifests, struct {
			path    string
			content []byte
			mode    os.FileMode
		}{composeEnvPath(), composeEnv, 0o644})
	}

	for _, m := range manifests {
		if dryRun {
			fmt.Printf("[Dry-Run] Would render %s (sha256=%s)\n", m.path, checksum(m.content))
			continue
		}

		if unchanged, err := fileUnchanged(m.path, m.content); err != nil {
			return err
		} else if unchanged {
			fmt.Printf("Skipping %s (unchanged)\n", m.path)
			continue
		}

		if err := writeFile(m.path, m.content, m.mode); err != nil {
			return err
		}
		fmt.Printf("Rendered %s\n", m.path)
	}

	edgeManifests, err := edgeSystemdManifests(profile)
	if err != nil {
		return err
	}
	for _, m := range edgeManifests {
		if dryRun {
			fmt.Printf("[Dry-Run] Would render %s (sha256=%s)\n", m.path, checksum(m.content))
			continue
		}
		if unchanged, err := fileUnchanged(m.path, m.content); err != nil {
			return err
		} else if unchanged {
			fmt.Printf("Skipping %s (unchanged)\n", m.path)
			continue
		}
		if err := writeFile(m.path, m.content, m.mode); err != nil {
			return err
		}
		fmt.Printf("Rendered %s\n", m.path)
	}

	if err := writeRollbackUnits(profile, dryRun); err != nil {
		return err
	}
	if err := applyServiceBinaries(profile, dryRun); err != nil {
		return err
	}
	if err := syncEdgeSystemdUnits(profile, dryRun); err != nil {
		return err
	}

	if profile.Type == ProfileComposeDev {
		script := filepath.Join(repoRoot(), "scripts", "dev", "stack.sh")
		if dryRun {
			fmt.Printf("[Dry-Run] Would invoke %s\n", script)
		}
	}

	return nil
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fileUnchanged(path string, content []byte) (bool, error) {
	current, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return checksum(current) == checksum(content), nil
}

func writeFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}
