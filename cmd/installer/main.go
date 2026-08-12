package main

import (
	"fmt"
	"os"

	"github.com/bidshard/ad-event-processor/internal/installer"
)

func main() {
	if err := installer.NewCLI().Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
