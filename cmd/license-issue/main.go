package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "license-issue: flag parse error")
		os.Exit(exitUsage)
	}

	res, code := runIssue(opts, os.Stderr)
	if code != 0 {
		os.Exit(code)
	}
	if err := writeIssueOutput(res, opts.OutFile, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: write out file: %v\n", err)
		os.Exit(1)
	}
}
