package branding

import (
	"os"
	"runtime/debug"
	"sync"
)

const (
	defaultProductName  = "BidShard"
	defaultVendorName   = "BidShard"
	defaultSiteURL      = "https://bidshard.com"
	defaultSupportEmail = "support@bidshard.com"
)

var (
	productName  string
	vendorName   string
	siteURL      string
	supportEmail string
	once         sync.Once
)

func initFromEnv() {
	productName = envOr("BRAND_PRODUCT_NAME", defaultProductName)
	vendorName = envOr("BRAND_VENDOR_NAME", defaultVendorName)
	siteURL = envOr("BRAND_SITE_URL", defaultSiteURL)
	supportEmail = envOr("BRAND_SUPPORT_EMAIL", defaultSupportEmail)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ProductName() string {
	once.Do(initFromEnv)
	return productName
}

func VendorName() string {
	once.Do(initFromEnv)
	return vendorName
}

func SiteURL() string {
	once.Do(initFromEnv)
	return siteURL
}

func SupportEmail() string {
	once.Do(initFromEnv)
	return supportEmail
}

func AlertTitle(subject string) string {
	return ProductName() + ": " + subject
}

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
