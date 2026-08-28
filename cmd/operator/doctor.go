package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ad-event-processor/internal/doctor"

	"github.com/spf13/cobra"
)

var (
	doctorOnly      string
	doctorProfile   string
	doctorChecklist bool
	doctorTimeout   time.Duration
	bundleOut       string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run host and dependency health probes",
	RunE:  runDoctor,
}

var doctorBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Write a redacted support bundle tarball",
	RunE:  runDoctorBundle,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorOnly, "only", "", "comma-separated probe names (kernel,sysctl,listen,redis,slotmap,license,clickhouse,disk,tls)")
	doctorCmd.Flags().StringVar(&doctorProfile, "profile", "", "deploy profile to validate (ingest_only, network_operator, analytics_ml)")
	doctorCmd.Flags().BoolVar(&doctorChecklist, "checklist", false, "print MVSS checklist from DATA_SECURITY.md")
	doctorCmd.Flags().DurationVar(&doctorTimeout, "timeout", 60*time.Second, "overall probe timeout")

	doctorBundleCmd.Flags().StringVar(&bundleOut, "out", "", "output tar.gz path (required)")
	doctorBundleCmd.Flags().StringVar(&doctorOnly, "only", "", "comma-separated probe names to include in bundle report")
	doctorBundleCmd.Flags().DurationVar(&doctorTimeout, "timeout", 60*time.Second, "overall probe timeout")

	doctorCmd.AddCommand(doctorBundleCmd)
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	only := splitCSV(doctorOnly)
	if strings.TrimSpace(doctorProfile) != "" {
		rows := doctor.DeployProfileChecklist(doctorProfile, nil)
		doctor.WriteChecklist(os.Stdout, rows)
		os.Exit(doctor.ChecklistExitCode(rows))
		return nil
	}
	if doctorChecklist {
		rows := doctor.MVSSChecklist(cfg)
		doctor.WriteChecklist(os.Stdout, rows)
		os.Exit(doctor.ChecklistExitCode(rows))
		return nil
	}
	if err := ensureConfig(only); err != nil {
		return err
	}
	deps := doctor.WithCLILicenseDeps(doctor.ProbeDeps{Config: cfg})

	report := doctor.Run(context.Background(), doctor.Options{
		Only:    only,
		Timeout: doctorTimeout,
		Probes:  doctor.DefaultProbes(deps),
	})
	doctor.WriteReport(os.Stdout, report)
	os.Exit(report.ExitCode())
	return nil
}

func runDoctorBundle(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(bundleOut) == "" {
		return fmt.Errorf("--out is required")
	}
	only := splitCSV(doctorOnly)
	if err := ensureConfig(only); err != nil {
		return err
	}
	deps := doctor.WithCLILicenseDeps(doctor.ProbeDeps{Config: cfg})
	if err := doctor.WriteBundle(context.Background(), doctor.BundleOptions{
		Out:     bundleOut,
		Deps:    deps,
		Only:    only,
		Timeout: doctorTimeout,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "bundle written to %s\n", bundleOut)
	return nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
