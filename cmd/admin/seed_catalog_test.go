package main

import "testing"

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

func TestSeedCatalog_customerBalanceMicroNotRoundThousands(t *testing.T) {
	for i := 1; i <= 100; i++ {
		balance := seedCustomerBalanceMicro(i)
		if balance%1_000_000 == 0 {
			t.Fatalf("customer balance ends on .00 at seq %d: %d micros", i, balance)
		}
	}
}
