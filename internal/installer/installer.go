package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CLI struct {
	Args []string
}

func NewCLI() *CLI {
	return &CLI{Args: os.Args}
}

func (c *CLI) Run() error {
	if len(c.Args) < 2 {
		c.PrintUsage()
		return nil
	}

	cmd := c.Args[1]
	switch cmd {
	case "preflight":
		strict := false
		asJSON := false
		for _, arg := range c.Args[2:] {
			if arg == "--strict" {
				strict = true
			}
			if arg == "--json" {
				asJSON = true
			}
		}
		_, err := RunPreflight(strict, asJSON)
		return err

	case "provision":
		yes := false
		for _, arg := range c.Args[2:] {
			if arg == "--yes" {
				yes = true
			}
		}
		return RunProvision(yes)

	case "configure":
		interactive := false
		for _, arg := range c.Args[2:] {
			if arg == "--interactive" {
				interactive = true
			}
		}
		return RunConfigure(interactive)

	case "up":
		return RunUp()

	case "bootstrap":
		return RunBootstrap()

	case "apply":
		dryRun := false
		for _, arg := range c.Args[2:] {
			if arg == "--dry-run" {
				dryRun = true
			}
		}
		return c.RunApply(dryRun)

	case "rollback":
		if len(c.Args) < 3 {
			return fmt.Errorf("usage: ad-event-processor-install rollback <tracker|processor>")
		}
		return RunRollbackCLI(c.Args[2])

	case "doctor":
		asJSON := false
		for _, arg := range c.Args[2:] {
			if arg == "--json" {
				asJSON = true
			}
		}
		return RunDoctor(asJSON)

	case "license":
		if len(c.Args) < 3 {
			fmt.Println("Usage: ad-event-processor-install license <install|activate|status>")
			return nil
		}
		return RunLicense(c.Args[2])

	default:
		c.PrintUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func (c *CLI) PrintUsage() {
	fmt.Println("Usage: ad-event-processor-install <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  preflight [--strict] [--json]")
	fmt.Println("  provision [--yes]")
	fmt.Println("  configure [--interactive]")
	fmt.Println("  up")
	fmt.Println("  bootstrap")
	fmt.Println("  apply     [--dry-run]")
	fmt.Println("  rollback  <tracker|processor>")
	fmt.Println("  doctor    [--json]")
	fmt.Println("  license   <install|activate|status>")
}

func (c *CLI) RunApply(dryRun bool) error {
	root := repoRoot()
	loadDotEnv(root)

	baseURL := managementBaseURL()
	adminKey := strings.TrimSpace(os.Getenv("ADMIN_API_KEY"))

	cfg, profile, err := loadConfigForApply(baseURL, adminKey)
	if err != nil {
		return err
	}

	if err := profile.Validate(); err != nil {
		return err
	}

	if err := writeInstallComposeEnv(cfg, dryRun); err != nil {
		return err
	}

	if err := renderTemplates(&profile, dryRun); err != nil {
		return err
	}

	if profile.Type == ProfileSingleVPS || profile.Type == ProfileComposeDev {
		if err := runDockerComposeUp(root, profile.Type, dryRun); err != nil {
			return err
		}
	}

	return nil
}

func runDockerComposeUp(root string, profile Profile, dryRun bool) error {
	envPath := filepath.Join(root, ".env")
	composeEnv := composeEnvPath()
	args := []string{
		"compose",
		"--env-file", envPath,
		"--env-file", composeEnv,
	}
	if profile == ProfileSingleVPS {
		args = append(args, "--profile", "single_vps")
	}
	args = append(args, "up", "-d")

	if dryRun {
		fmt.Printf("[Dry-Run] Would run docker %s\n", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	return nil
}
