package ingest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	auditTestBinary     string
	auditTestBinaryOnce sync.Once
	auditTestBinaryErr  error
)

var bceHotSymbols = []string{
	"ad-event-processor/internal/ingest.foldKeyU32",
	"ad-event-processor/internal/ingest.foldKeyU64",
	"ad-event-processor/internal/ingest.matchTelegramQueryKey",
	"ad-event-processor/internal/ingest.dispatchTelegramRedirectMacro",
	"ad-event-processor/internal/ingest.expandTelegramRedirectMacros",
	"ad-event-processor/internal/ingest.parseTelegramQuery",
	"ad-event-processor/internal/ingest.unsafeString",
}

func TestBCEAudit_hotSymbolsNoPanicIndexInMainBody(t *testing.T) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("objdump BCE gate runs on linux amd64/arm64")
	}
	bin := testBinaryPath(t)
	for _, sym := range bceHotSymbols {
		t.Run(sym, func(t *testing.T) {
			asm := objdumpSymbol(t, bin, sym)
			main := asmMainBody(asm)
			if strings.Contains(main, "runtime.panicIndex") {
				t.Fatalf("main body still references runtime.panicIndex:\n%s", excerptPanicLines(main))
			}
		})
	}
}

func TestBCEAudit_dispatchTelegramRedirectMacro_boundsChecks(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("CMP budget calibrated on amd64")
	}
	bin := testBinaryPath(t)
	asm := asmMainBody(objdumpSymbol(t, bin, "ad-event-processor/internal/ingest.dispatchTelegramRedirectMacro"))

	cmpLen := strings.Count(asm, "CMPQ BX,")
	if cmpLen > 12 {
		t.Fatalf("dispatchTelegramRedirectMacro len CMPQ BX count = %d, want <= 12; tighten window BCE", cmpLen)
	}
}

func testBinaryPath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("TEST_BINARY"); p != "" {
		return p
	}
	auditTestBinaryOnce.Do(func() {
		f, err := os.CreateTemp("", "ingestion-bce-*.test")
		if err != nil {
			auditTestBinaryErr = err
			return
		}
		out := f.Name()
		_ = f.Close()
		cmd := exec.Command("go", "test", "-c", "-o", out, ".")
		cmd.Dir = mustPackageDir()
		auditTestBinaryErr = cmd.Run()
		auditTestBinary = out
	})
	if auditTestBinaryErr != nil {
		t.Fatalf("build audit test binary: %v", auditTestBinaryErr)
	}
	return auditTestBinary
}

func mustPackageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func objdumpSymbol(t *testing.T, bin, sym string) string {
	t.Helper()
	out, err := exec.Command("go", "tool", "objdump", "-s", sym, bin).CombinedOutput()
	if err != nil {
		t.Fatalf("objdump %s: %v\n%s", sym, err, out)
	}
	return string(out)
}

func asmMainBody(asm string) string {
	lines := strings.Split(asm, "\n")
	var buf bytes.Buffer
	for _, line := range lines {
		if strings.Contains(line, "CALL runtime.panicIndex") {
			break
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func excerptPanicLines(main string) string {
	const maxLines = 12
	lines := strings.Split(main, "\n")
	var out []string
	for _, line := range lines {
		if strings.Contains(line, "panicIndex") {
			out = append(out, line)
			if len(out) >= maxLines {
				break
			}
		}
	}
	return strings.Join(out, "\n")
}

func TestUnsafeAudit_tgClickScratchLifetime(t *testing.T) {
	path := []byte("/tg/click?campaign_id=00000000-0000-0000-0000-000000000001&click_id=00000000-0000-0000-0000-000000000002&bridge_token=abc123")
	scratch := make([]byte, 0, 256)
	var parsed telegramQueryParsed
	scratch = parseTelegramQuery(path, scratch, &parsed)
	if !parsed.OK {
		t.Fatal("expected ok")
	}
	if parsed.BridgeToken != "abc123" {
		t.Fatalf("bridge_token = %q", parsed.BridgeToken)
	}
	bridgePtr := UnsafeBytes(parsed.BridgeToken)
	if len(bridgePtr) == 0 {
		t.Fatal("expected bridge token bytes")
	}
	orig := bridgePtr[0]
	bridgePtr[0] = 'Z'
	if parsed.BridgeToken[0] != 'Z' {
		t.Fatal("parsed.BridgeToken must alias scratch backing store")
	}
	bridgePtr[0] = orig

	loc, ok := buildTelegramRedirectLocation(scratch[:0], []byte("https://example.com/{bridge_token}"), parsed.ClickIDStr, parsed.BridgeToken, parsed.Subs, nil)
	if !ok {
		t.Fatal("buildTelegramRedirectLocation failed")
	}
	if !bytes.Contains(loc, []byte("abc123")) {
		t.Fatalf("location %q missing bridge token", loc)
	}
}

func TestBCEAudit_docPresent(t *testing.T) {
	for _, p := range []string{
		"../../docs/ARCHITECTURE.md",
		"docs/ARCHITECTURE.md",
	} {
		if _, err := os.Stat(p); err == nil {
			return
		}
	}
	t.Fatal("docs/ARCHITECTURE.md missing")
}
