package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(runBot(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "vendor-trial-bot: unknown command %q\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func printUsage(w *os.File) {
	fmt.Fprintf(w, `usage: vendor-trial-bot <command>

commands:
  run   long-poll Telegram and enqueue /trial requests

environment:
  %s       bot token from @BotFather (never commit)
  %s   trial registry file path

`, envBotToken, envTrialRegistry)
}
