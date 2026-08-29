package main

import (
	"fmt"
	"os"
)

func printUsage() {
	_, _ = fmt.Fprintf(os.Stdout, "Usage: broker <command> [options]\n")
	_, _ = fmt.Fprintf(os.Stdout, "\nCommands:\n")
	_, _ = fmt.Fprintf(os.Stdout, " serve Run mmap WAL broker (gnet ingress)\n")
	_, _ = fmt.Fprintf(os.Stdout, " replay Replay historical WAL segments to target storage (e.g. clickhouse)\n")
	_, _ = fmt.Fprintf(os.Stdout, "\nOptions for replay:\n")
	_, _ = fmt.Fprintf(os.Stdout, " --data-dir Path to broker WAL data directory (e.g. /var/lib/ad-event-processor/broker)\n")
	_, _ = fmt.Fprintf(os.Stdout, " --topic Topic name to replay (default: ad-events)\n")
	_, _ = fmt.Fprintf(os.Stdout, " --from RFC3339 start timestamp filter (e.g. 2026-08-08T12:00:00Z)\n")
	_, _ = fmt.Fprintf(os.Stdout, " --to RFC3339 end timestamp filter (e.g. 2026-08-08T18:00:00Z)\n")
	_, _ = fmt.Fprintf(os.Stdout, " --target Target system: clickhouse, stdout, or null (default: clickhouse)\n")
	_, _ = fmt.Fprintf(os.Stdout, " --ch-dsn ClickHouse DSN connection string\n")
	_, _ = fmt.Fprintf(os.Stdout, " --batch-size Replay batch size (default: 50000)\n")
}
