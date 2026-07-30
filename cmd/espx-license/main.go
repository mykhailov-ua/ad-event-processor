package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"
	"time"

	"espx/internal/licensing"

	"github.com/google/uuid"
)

func main() {
	skuPath := flag.String("sku-file", "deploy/vendor/sku.yaml", "path to vendor sku.yaml")
	skuCode := flag.String("sku", "", "SKU code to issue (required)")
	customerName := flag.String("customer-name", "", "customer name for JWT claim (required)")
	deploymentID := flag.String("deployment-id", "", "deployment UUID (required)")
	licenseID := flag.String("license-id", "", "license subject UUID (generated if empty)")
	fingerprint := flag.String("fingerprint", "", "optional node fingerprint")
	privateKeyFile := flag.String("private-key-file", "", "Ed25519 seed/private key file (or ESPX_LICENSE_PRIVATE_KEY)")
	keyID := flag.String("kid", licensing.DefaultLicenseKeyID, "JWT kid header")
	output := flag.String("output", "", "write JWT to file (default stdout)")
	flag.Parse()

	if *skuCode == "" || *customerName == "" || *deploymentID == "" {
		flag.Usage()
		os.Exit(2)
	}
	if _, err := uuid.Parse(*deploymentID); err != nil {
		fmt.Fprintf(os.Stderr, "invalid deployment-id: %v\n", err)
		os.Exit(1)
	}
	licID := *licenseID
	if licID == "" {
		licID = uuid.NewString()
	}

	priv, err := loadPrivateKey(*privateKeyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "private key: %v\n", err)
		os.Exit(1)
	}

	doc, err := licensing.LoadSKUFile(*skuPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sku file: %v\n", err)
		os.Exit(1)
	}
	sku, err := doc.GetSKU(*skuCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	claims := sku.BuildClaims(licensing.IssueLicenseInput{
		SKUCode:      *skuCode,
		CustomerName: *customerName,
		DeploymentID: *deploymentID,
		LicenseID:    licID,
		Fingerprint:  *fingerprint,
		ValidFrom:    time.Now().UTC(),
	})
	token, err := licensing.SignJWT(claims, priv, *keyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign: %v\n", err)
		os.Exit(1)
	}
	if *output == "" {
		fmt.Println(token)
		return
	}
	if err := os.WriteFile(*output, []byte(token), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	var raw []byte
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		raw = data
	} else if env := os.Getenv("ESPX_LICENSE_PRIVATE_KEY"); env != "" {
		raw = []byte(env)
	} else {
		return nil, fmt.Errorf("set --private-key-file or ESPX_LICENSE_PRIVATE_KEY")
	}
	return licensing.ParsePrivateKey(raw)
}
