package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"espx/pkg/platformconfig"
)

func RunUp() error {
	root := repoRoot()
	loadDotEnv(root)
	if err := ensureEnvFile(root); err != nil {
		return err
	}

	buildScript := filepath.Join(root, "scripts", "dev", "stack.sh")
	buildCmd := exec.Command("bash", buildScript, "build")
	buildCmd.Dir = root
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("stack build: %w", err)
	}

	script := filepath.Join(root, "scripts", "dev", "stack.sh")
	cmd := exec.Command("bash", script, "single-vps")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stack up: %w", err)
	}

	if err := waitControlHealth(root); err != nil {
		return err
	}

	fmt.Printf("Control URL: %s\n", managementBaseURL())
	if cfg, err := LoadPlatformConfigFromJSONFile(platformConfigJSONPath()); err == nil {
		if tpl := platformconfig.ClickURLTemplate(cfg.TrackingDomain); tpl != "" {
			fmt.Printf("Click URL template: %s\n", tpl)
		}
	}
	return nil
}

func waitControlHealth(root string) error {
	url := managementBaseURL() + "/health"
	script := fmt.Sprintf(
		`for i in $(seq 1 60); do if curl -sf "%s" >/dev/null; then exit 0; fi; sleep 2; done; exit 1`,
		url,
	)
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("control health check failed on %s", url)
	}
	return nil
}
