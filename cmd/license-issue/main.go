// Vendor-only CLI: issue offline license JWTs for on-prem pilot customers.
// Private key never ships in customer tarballs — keep deploy/vendor/license_private.key local.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/google/uuid"
)

func main() {
	skuFile := flag.String("sku-file", "deploy/vendor/sku.yaml", "path to SKU catalog")
	skuCode := flag.String("sku", "pilot", "SKU code")
	customer := flag.String("customer", "", "customer display name (required)")
	deploymentID := flag.String("deployment-id", "", "deployment UUID (generated if empty)")
	fingerprint := flag.String("fingerprint", "", "host fingerprint for hard bind (from customer support bundle)")
	kid := flag.String("kid", licensing.DefaultLicenseKeyID, "JWT key id (kid); uses deploy/vendor/keys/<kid>/ when set")
	revoke := flag.Bool("revoke", false, "issue revocation JWT (valid_until in past, revoked=true)")
	validDays := flag.Int("days", 0, "override valid_days from SKU")
	privFile := flag.String("private-key", "", "Ed25519 private key file (hex seed)")
	outFile := flag.String("out", "", "write JWT to file instead of stdout")
	flag.Parse()

	if strings.TrimSpace(*customer) == "" {
		fmt.Fprintln(os.Stderr, "license-issue: --customer is required")
		os.Exit(2)
	}

	keyID := strings.TrimSpace(*kid)
	if keyID == "" {
		keyID = licensing.DefaultLicenseKeyID
	}

	privPath := licensing.ResolvePrivateKeyFileForKID(keyID, strings.TrimSpace(*privFile))
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: read private key %s: %v\n", privPath, err)
		os.Exit(1)
	}
	priv, err := licensing.ParsePrivateKey(privBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: parse private key: %v\n", err)
		os.Exit(1)
	}

	doc, err := licensing.LoadSKUFile(*skuFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: load SKU file: %v\n", err)
		os.Exit(1)
	}
	sku, err := doc.GetSKU(*skuCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: %v\n", err)
		os.Exit(1)
	}
	if *validDays > 0 {
		sku.ValidDays = *validDays
	}

	depID := strings.TrimSpace(*deploymentID)
	if depID == "" {
		depID = uuid.NewString()
	}
	licenseID := uuid.NewString()

	claims := sku.BuildClaims(licensing.IssueLicenseInput{
		SKUCode:      sku.Code,
		CustomerName: *customer,
		DeploymentID: depID,
		LicenseID:    licenseID,
		Fingerprint:  strings.TrimSpace(*fingerprint),
		ValidFrom:    time.Now().UTC(),
	})
	if *revoke {
		claims.Revoked = true
		claims.ValidUntil = time.Now().UTC().Add(-time.Hour)
		claims.ValidFrom = claims.ValidUntil.Add(-24 * time.Hour)
	}

	token, err := licensing.SignJWT(claims, priv, keyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "license-issue: sign: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(*outFile) != "" {
		if err := os.WriteFile(*outFile, []byte(token), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "license-issue: write out file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "license-issue: wrote JWT to %s (kid=%s deployment_id=%s valid_until=%s)\n",
			*outFile, keyID, depID, claims.ValidUntil.Format(time.RFC3339))
		return
	}

	fmt.Println(token)
	fmt.Fprintf(os.Stderr, "kid=%s deployment_id=%s valid_until=%s\n", keyID, depID, claims.ValidUntil.Format(time.RFC3339))
}
