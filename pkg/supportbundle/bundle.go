package supportbundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strings"
	"time"
)

const (
	DefaultMaxBytes    = 50 << 20
	DefaultTimeout     = 30 * time.Second
	defaultMaxLogLines = 10_000
)

type Meta struct {
	DeploymentID     string `json:"deployment_id,omitempty"`
	LicenseState     string `json:"license_state,omitempty"`
	DaysToExpiry     int    `json:"days_to_expiry,omitempty"`
	HostFingerprint  string `json:"host_fingerprint,omitempty"`
	FingerprintMatch *bool  `json:"fingerprint_match,omitempty"`
	BinaryVersion    string `json:"binary_version,omitempty"`
}

type Options struct {
	Meta        Meta
	LogDir      string
	MaxBytes    int64
	MaxLogLines int
	ExtraJSON   map[string]any
}

func Write(ctx context.Context, w io.Writer, opts Options) error {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.LogDir == "" {
		opts.LogDir = os.Getenv("LOGGER_DIR")
	}
	if opts.LogDir == "" {
		opts.LogDir = "/var/log/ad-event-processor"
	}
	if opts.MaxLogLines <= 0 {
		opts.MaxLogLines = defaultMaxLogLines
	}

	lw := &limitedWriter{w: w, max: opts.MaxBytes}
	gz := gzip.NewWriter(lw)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	if err := writeVersionJSON(ctx, tw, opts.Meta); err != nil {
		return err
	}
	for name, payload := range opts.ExtraJSON {
		if err := writeTarJSON(tw, name, payload); err != nil {
			return err
		}
	}
	if err := writeSanitizedEnv(tw); err != nil {
		return err
	}
	if err := writeProfiles(ctx, tw); err != nil {
		return err
	}
	if err := writeRedactedLogs(ctx, tw, opts.LogDir, opts.MaxLogLines); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if lw.exceeded() {
		return fmt.Errorf("bundle exceeds %d byte limit", opts.MaxBytes)
	}
	return nil
}

func writeVersionJSON(ctx context.Context, tw *tar.Writer, meta Meta) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := map[string]string{
		"go_version":     runtime.Version(),
		"go_os":          runtime.GOOS,
		"go_arch":        runtime.GOARCH,
		"built_at":       time.Now().UTC().Format(time.RFC3339),
		"deployment_id":  meta.DeploymentID,
		"license_state":  meta.LicenseState,
		"binary_version": meta.BinaryVersion,
	}
	if v["binary_version"] == "" {
		v["binary_version"] = readBuildVersion()
	}
	return writeTarJSON(tw, "version.json", v)
}

func readBuildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func writeSanitizedEnv(tw *tar.Writer) error {
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
	return writeTarBytes(tw, "config/sanitized.env", []byte(strings.Join(keys, "\n")+"\n"))
}

func writeProfiles(ctx context.Context, tw *tar.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var goroutineBuf strings.Builder
	if err := pprof.Lookup("goroutine").WriteTo(io.Writer(&goroutineBuf), 2); err != nil {
		return fmt.Errorf("goroutine profile: %w", err)
	}
	if err := writeTarBytes(tw, "goroutine.pprof", []byte(goroutineBuf.String())); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var heapBuf strings.Builder
	if err := pprof.Lookup("heap").WriteTo(io.Writer(&heapBuf), 0); err != nil {
		return fmt.Errorf("heap profile: %w", err)
	}
	return writeTarBytes(tw, "heap.pprof", []byte(heapBuf.String()))
}

func writeRedactedLogs(ctx context.Context, tw *tar.Writer, dir string, maxLines int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lines, err := tailLogLines(dir, maxLines)
	if err != nil {
		return fmt.Errorf("read logs: %w", err)
	}
	return writeTarBytes(tw, "logs/redacted.log", RedactLog(lines))
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

type limitedWriter struct {
	w     io.Writer
	max   int64
	wrote int64
	limit bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.limit {
		return 0, fmt.Errorf("bundle size limit exceeded")
	}
	remain := lw.max - lw.wrote
	if remain <= 0 {
		lw.limit = true
		return 0, fmt.Errorf("bundle size limit exceeded")
	}
	if int64(len(p)) > remain {
		lw.limit = true
		_, _ = lw.w.Write(p[:remain])
		lw.wrote += remain
		return int(remain), fmt.Errorf("bundle size limit exceeded")
	}
	n, err := lw.w.Write(p)
	lw.wrote += int64(n)
	return n, err
}

func (lw *limitedWriter) exceeded() bool {
	return lw.limit
}
