// Vendor-only: seal edge BPF objects with MCK derived from a customer license JWT.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/licensing"
)

func main() {
	in := flag.String("in", "", "input asset bytes (BPF ELF, unified-filter.lua, …)")
	out := flag.String("out", "internal/edge/bpf/edge_sealed.bin", "output sealed blob path")
	label := flag.String("label", licensing.AssetLabelEdge, "asset label (edge-bpf, unified-filter)")
	licenseFile := flag.String("license", "", "license JWT file used to derive MCK")
	token := flag.String("token", "", "license JWT inline (alternative to --license)")
	hwid := flag.String("hwid", "", "host HWID v2 used for MCK (default: HostHWID())")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "license-asset-seal: --in is required")
		os.Exit(2)
	}
	jwt := strings.TrimSpace(*token)
	if jwt == "" && *licenseFile != "" {
		data, err := os.ReadFile(*licenseFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "license-asset-seal: read license: %v\n", err)
			os.Exit(1)
		}
		jwt = strings.TrimSpace(string(data))
	}
	if jwt == "" {
		fmt.Fprintln(os.Stderr, "license-asset-seal: --license or --token is required")
		os.Exit(2)
	}
	hostHWID := strings.TrimSpace(*hwid)
	if hostHWID == "" {
		hostHWID = licensing.HostHWID()
	}
	mck, err := licensing.DeriveMCK(jwt, hostHWID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-asset-seal: derive mck: %v\n", err)
		os.Exit(1)
	}
	plain, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-asset-seal: read input: %v\n", err)
		os.Exit(1)
	}
	sealed, err := licensing.SealAsset(strings.TrimSpace(*label), plain, mck)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-asset-seal: seal: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, sealed, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "license-asset-seal: write out: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "license-asset-seal: wrote %s (%d bytes plaintext -> %d bytes sealed)\n", *out, len(plain), len(sealed))
}
