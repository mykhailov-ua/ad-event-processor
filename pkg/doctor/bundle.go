package doctor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/config"
)

type BundleOptions struct {
	Out     string
	Deps    ProbeDeps
	Only    []string
	Timeout time.Duration
}

func WriteBundle(ctx context.Context, opts BundleOptions) error {
	if strings.TrimSpace(opts.Out) == "" {
		return fmt.Errorf("bundle output path required")
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil && filepath.Dir(opts.Out) != "." {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	report := Run(ctx, Options{
		Only:    opts.Only,
		Timeout: opts.Timeout,
		Probes:  DefaultProbes(WithCLILicenseDeps(opts.Deps)),
	})
	checklist := MVSSChecklist(opts.Deps.Config)
	deps := WithCLILicenseDeps(opts.Deps)

	f, err := os.Create(opts.Out)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz := gzip.NewWriter(f)
	defer func() { _ = gz.Close() }()
	tw := tar.NewWriter(gz)
	defer func() { _ = tw.Close() }()

	if err := writeTarJSON(tw, "doctor/report.json", report); err != nil {
		return err
	}
	if err := writeTarJSON(tw, "doctor/checklist.json", checklist); err != nil {
		return err
	}
	if err := writeTarJSON(tw, "version.json", bundleVersion()); err != nil {
		return err
	}
	if err := writeTarBytes(tw, "config/sanitized.env", sanitizedEnv()); err != nil {
		return err
	}
	if config.LicenseProbeEnabled() {
		if err := writeTarJSON(tw, "doctor/license.json", bundleLicenseInfo(deps)); err != nil {
			return err
		}
	}
	readme := []byte("Redacted operator bundle from ad-event-processor doctor. Full pprof/log redaction ships in GAP-SUP-01.\n")
	return writeTarBytes(tw, "README.txt", readme)
}

func bundleVersion() map[string]string {
	return map[string]string{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"built_at":   time.Now().UTC().Format(time.RFC3339),
	}
}

func sanitizedEnv() []byte {
	keys := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			keys = append(keys, kv+"=***")
			continue
		}
		keys = append(keys, key+"=***")
	}
	sort.Strings(keys)
	return []byte(strings.Join(keys, "\n") + "\n")
}

func writeTarJSON(tw *tar.Writer, name string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	return writeTarBytes(tw, name, raw)
}

func writeTarBytes(tw *tar.Writer, name string, payload []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    int64(len(payload)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header %s: %w", name, err)
	}
	if _, err := tw.Write(payload); err != nil {
		return fmt.Errorf("tar body %s: %w", name, err)
	}
	return nil
}
