package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ad-event-processor/internal/trialregistry"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "expire-stale":
		os.Exit(runExpireStale(os.Args[2:]))
	case "list-pending":
		os.Exit(runListPending(os.Args[2:]))
	case "reject-pending":
		os.Exit(runRejectPending(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "trial-registry: unknown command %q\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func printUsage(w *os.File) {
	_, _ = fmt.Fprintf(w, `usage: trial-registry <command>

commands:
 expire-stale mark active anchors expired when valid_until < now
 list-pending print open pending trial requests
 reject-pending reject an open pending request by id

`)
}

func openRegistry(pathOverride string) *trialregistry.Registry {
	cfg := trialregistry.ConfigFromEnv()
	if path := strings.TrimSpace(pathOverride); path != "" {
		cfg.RegistryPath = path
	}
	return trialregistry.NewFromConfig(cfg)
}

func runExpireStale(args []string) int {
	fs := flag.NewFlagSet("expire-stale", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registryPath := fs.String("trial-registry", "", "trial registry file path override")
	atRaw := fs.String("at", "", "RFC3339 cutoff time (default: now UTC)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	reg := openRegistry(*registryPath)

	at := time.Now().UTC()
	if raw := strings.TrimSpace(*atRaw); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "trial-registry: parse --at: %v\n", err)
			return 1
		}
		at = parsed.UTC()
	}

	n, err := reg.ExpireStale(at)
	if err != nil {
		fmt.Fprintf(os.Stderr, "trial-registry: expire-stale: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "trial-registry: expired %d anchor(s) before %s\n", n, at.Format(time.RFC3339))
	return 0
}

func runListPending(args []string) int {
	fs := flag.NewFlagSet("list-pending", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registryPath := fs.String("trial-registry", "", "trial registry file path override")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	reg := openRegistry(*registryPath)
	pending, err := reg.ListPending()
	if err != nil {
		fmt.Fprintf(os.Stderr, "trial-registry: list-pending: %v\n", err)
		return 1
	}
	if len(pending) == 0 {
		fmt.Fprintln(os.Stderr, "trial-registry: no open pending requests")
		return 0
	}
	for _, req := range pending {
		_, _ = fmt.Fprintf(os.Stdout, "%s\ttelegram=%s\tuser=%s\trequested=%s\n",
			req.ID,
			req.TelegramID,
			req.TelegramUsername,
			req.RequestedAt.Format(time.RFC3339),
		)
	}
	return 0
}

func runRejectPending(args []string) int {
	fs := flag.NewFlagSet("reject-pending", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registryPath := fs.String("trial-registry", "", "trial registry file path override")
	id := fs.String("id", "", "pending request id")
	reason := fs.String("reason", "", "rejection reason for audit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(os.Stderr, "trial-registry: reject-pending requires --id")
		return 2
	}

	reg := openRegistry(*registryPath)
	if err := reg.RejectPending(*id, *reason); err != nil {
		fmt.Fprintf(os.Stderr, "trial-registry: reject-pending: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "trial-registry: rejected pending id=%s\n", strings.TrimSpace(*id))
	return 0
}
