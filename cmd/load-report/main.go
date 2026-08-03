package main

import (
	"errors"
	"fmt"
	"os"

	"espx/internal/loadreport"
)

const defaultPromURL = "http://127.0.0.1:9190"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "prom":
		sessionDir, promURL, err := parseSessionFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			usage()
			os.Exit(1)
		}
		if err := runProm(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "bpf":
		sessionDir, _, err := parseSessionFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			usage()
			os.Exit(1)
		}
		if err := runBPF(sessionDir); err != nil {
			if errors.Is(err, loadreport.ErrNoBPFSummary) {
				fmt.Fprintf(os.Stderr, "load-report bpf: no bpf/maps/summary.json — skipping\n")
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "all":
		sessionDir, promURL, err := parseSessionFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			usage()
			os.Exit(1)
		}
		if err := runBPF(sessionDir); err != nil && !errors.Is(err, loadreport.ErrNoBPFSummary) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else if errors.Is(err, loadreport.ErrNoBPFSummary) {
			fmt.Fprintf(os.Stderr, "load-report bpf: no bpf/maps/summary.json — skipping\n")
		}
		if err := runProm(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := runSLA(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			if os.Getenv("LOAD_SLA_GATE") == "1" {
				os.Exit(1)
			}
		}
		if err := runTelegram(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			if os.Getenv("LOAD_TG_GATE") == "1" {
				os.Exit(1)
			}
		}
	case "telegram":
		sessionDir, promURL, err := parseSessionFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			usage()
			os.Exit(1)
		}
		if err := runTelegram(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "sla":
		sessionDir, promURL, err := parseSessionFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			usage()
			os.Exit(1)
		}
		if err := runSLA(sessionDir, promURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "load-report: unknown subcommand %q\n", cmd)
		usage()
		os.Exit(1)
	}
}

func parseSessionFlags(args []string) (sessionDir, promURL string, err error) {
	if len(args) < 1 {
		return "", "", errors.New("load-report: session directory required")
	}
	sessionDir = args[0]
	promURL = os.Getenv("PROMETHEUS_URL")
	if promURL == "" {
		promURL = defaultPromURL
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--prom":
			if i+1 >= len(args) {
				return "", "", errors.New("load-report: --prom requires URL")
			}
			promURL = args[i+1]
			i++
		default:
			return "", "", fmt.Errorf("load-report: unknown flag %q", args[i])
		}
	}
	return sessionDir, promURL, nil
}

func runBPF(sessionDir string) error {
	path, err := loadreport.WriteBPFReport(sessionDir)
	if err != nil {
		return err
	}
	fmt.Printf("load-report bpf: wrote %s\n", path)
	return nil
}

func runProm(sessionDir, promURL string) error {
	path, err := loadreport.WritePromReports(sessionDir, promURL)
	if err != nil {
		return err
	}
	fmt.Printf("load-report prom: wrote %s\n", path)
	fmt.Println("load-report prom: Grafana dashboard: http://127.0.0.1:3100/d/espx-main/espx-operations (or browse Dashboards tagged load-test)")
	return nil
}

func runTelegram(sessionDir, promURL string) error {
	path, err := loadreport.WriteTelegramGateReport(sessionDir, promURL)
	if err != nil {
		return err
	}
	fmt.Printf("load-report telegram: wrote %s\n", path)
	return nil
}

func runSLA(sessionDir, promURL string) error {
	path, err := loadreport.WriteSLAReport(sessionDir, promURL)
	if err != nil {
		return err
	}
	fmt.Printf("load-report sla: wrote %s\n", path)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  load-report prom <session-dir> [--prom URL]
  load-report bpf <session-dir>
  load-report sla <session-dir> [--prom URL]
  load-report telegram <session-dir> [--prom URL]
  load-report all <session-dir> [--prom URL]

Default Prometheus URL: %s (override with PROMETHEUS_URL or --prom)
Set LOAD_SLA_GATE=1 to fail load-report all on SLA breach.
Set LOAD_TG_GATE=1 to fail load-report all on Telegram T9 gate breach.
`, defaultPromURL)
}
