package controlplane

import "testing"

func TestBestFloorBucketByPlacement(t *testing.T) {
	buckets := []PlacementFloorBucket{
		{PlacementID: "a", FloorBucketMicro: 10_000, SampleN: 5, WinRate: 0.5},
		{PlacementID: "a", FloorBucketMicro: 20_000, SampleN: 12, WinRate: 0.4},
		{PlacementID: "b", FloorBucketMicro: 30_000, SampleN: 15, WinRate: 0.6},
	}
	got := bestFloorBucketByPlacement(buckets)
	if got["a"] != 20_000 {
		t.Fatalf("a=%d want 20000", got["a"])
	}
	if got["b"] != 30_000 {
		t.Fatalf("b=%d want 30000", got["b"])
	}
}
