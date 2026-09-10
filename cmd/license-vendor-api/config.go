package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"ad-event-processor/internal/licenseissue"
)

const envAPIToken = "LICENSE_VENDOR_API_TOKEN"
const envListen = "LICENSE_VENDOR_API_LISTEN"

type serverConfig struct {
	Listen string
	Token  string
	Issue  licenseissue.Config
}

func loadConfig(args []string) (serverConfig, error) {
	fs := flag.NewFlagSet("license-vendor-api", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.String("listen", "", "listen address (default 127.0.0.1:8199 or LICENSE_VENDOR_API_LISTEN)")
	skuFile := fs.String("sku-file", "", "SKU catalog path override")
	privKey := fs.String("private-key", "", "Ed25519 private key file override")
	trialReg := fs.String("trial-registry", "", "trial registry JSON override")

	if err := fs.Parse(args); err != nil {
		return serverConfig{}, err
	}

	cfg := serverConfig{
		Listen: strings.TrimSpace(os.Getenv(envListen)),
		Token:  strings.TrimSpace(os.Getenv(envAPIToken)),
		Issue:  licenseissue.ConfigFromEnv(),
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8199"
	}
	if addr := strings.TrimSpace(*listen); addr != "" {
		cfg.Listen = addr
	}
	if path := strings.TrimSpace(*skuFile); path != "" {
		cfg.Issue.SKUFile = path
	}
	if path := strings.TrimSpace(*privKey); path != "" {
		cfg.Issue.PrivateKeyFile = path
	}
	if path := strings.TrimSpace(*trialReg); path != "" {
		cfg.Issue.TrialRegistry = path
	}
	if cfg.Token == "" {
		return serverConfig{}, fmt.Errorf("%s is required", envAPIToken)
	}
	return cfg, nil
}
