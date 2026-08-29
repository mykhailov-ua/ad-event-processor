package main

import (
	"os"

	"ad-event-processor/internal/control"
	"ad-event-processor/internal/licensing"
)

func main() {
	if control.ProbeHealth(os.Args) {
		return
	}
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}
	control.RunCLI()
}
