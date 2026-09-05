package main

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSeedCatalog_customerNamesUniquePer100(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 1; i <= 100; i++ {
		name := seedCustomerName(i)
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate customer display name at seq %d: %q", i, name)
		}
		seen[name] = struct{}{}
	}
}

func TestSeedCatalog_campaignNamesUniquePer100(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 1; i <= 100; i++ {
		name := seedCampaignName(i)
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate campaign display name at seq %d: %q", i, name)
		}
		seen[name] = struct{}{}
	}
}

func TestSeedCatalog_campaignNamesUniquePer1000(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 1; i <= 1000; i++ {
		name := seedCampaignName(i)
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate campaign display name at seq %d: %q", i, name)
		}
		seen[name] = struct{}{}
	}
}

func TestSeedCatalog_deterministicUUIDsAreRealistic(t *testing.T) {
	placeholderPrefix := "00000000-0000-0000-0000-"
	legacyDevMockPrefix := "00000000-"
	seen := make(map[string]struct{})

	check := func(kind string, seq int, id uuid.UUID) {
		t.Helper()
		value := id.String()
		if strings.HasPrefix(value, placeholderPrefix) {
			t.Fatalf("%s seq=%d looks like placeholder uuid: %s", kind, seq, value)
		}
		if strings.HasPrefix(value, legacyDevMockPrefix) && strings.Contains(value, "-4000-8000-") {
			t.Fatalf("%s seq=%d looks like legacy dev_mock sequential uuid: %s", kind, seq, value)
		}
		if strings.HasSuffix(value, strings.Repeat("0", 11)+"1") {
			t.Fatalf("%s seq=%d has trivial sequential tail: %s", kind, seq, value)
		}
		if _, ok := seen[value]; ok {
			t.Fatalf("duplicate seed uuid %s for %s seq=%d", value, kind, seq)
		}
		seen[value] = struct{}{}
	}

	for i := 1; i <= 100; i++ {
		check("customer", i, seedCustomerUUID(i))
		check("brand", i, seedBrandUUID(i))
		check("creative", i, seedCreativeUUID(i))
		check("campaign", i, seedCampaignUUID(i))
	}
	check("deployment", 1, seedDeploymentUUID())
	check("license", 1, seedLicenseRecordUUID())
}

func TestSeedCatalog_customerBalanceMicroNotRoundThousands(t *testing.T) {
	for i := 1; i <= 100; i++ {
		balance := seedCustomerBalanceMicro(i)
		if balance%1_000_000 == 0 {
			t.Fatalf("customer balance ends on .00 at seq %d: %d micros", i, balance)
		}
	}
}

func TestSeedCatalog_displayNamesOmitGeoDeskAndWaveTokens(t *testing.T) {
	banned := []string{"Alpha desk", "Bravo desk", " · wave ", " · group ", " / GB /", " · GB ·", " · CA ·", " · US ·", " · "}
	for i := 1; i <= 1000; i++ {
		campaign := seedCampaignName(i)
		customer := seedCustomerName(i)
		creative := seedCreativeDisplayName(i)
		for _, token := range banned {
			if strings.Contains(campaign, token) {
				t.Fatalf("campaign seq %d contains banned token %q: %q", i, token, campaign)
			}
			if strings.Contains(customer, token) {
				t.Fatalf("customer seq %d contains banned token %q: %q", i, token, customer)
			}
			if strings.Contains(creative, token) {
				t.Fatalf("creative seq %d contains banned token %q: %q", i, token, creative)
			}
		}
	}
}
