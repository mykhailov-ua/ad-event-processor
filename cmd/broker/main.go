// broker CLI entrypoint. Package documentation: doc.go.
package main

import (
	"fmt"
	"os"
	"strings"

	"ad-event-processor/pkg/lifecycle"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(os.Args) >= 2 && (os.Args[1] == "serve" || strings.HasPrefix(os.Args[1], "-")) {
		if os.Args[1] == "serve" {
			runServe(os.Args[2:])
			return
		}
		runServe(os.Args[1:])
		return
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "replay":
		runReplay(os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
