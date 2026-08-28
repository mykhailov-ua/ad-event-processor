package rtbadmin

import "testing"

func TestBestFloorBucketByPlacement(t *testing.T) {
	buckets := []placementFloorBucket{
		{placementID: "a", floorBucketMicro: 10_000, sampleN: 5, winRate: 0.5},
		{placementID: "a", floorBucketMicro: 20_000, sampleN: 12, winRate: 0.4},
		{placementID: "b", floorBucketMicro: 30_000, sampleN: 15, winRate: 0.6},
	}
	got := bestFloorBucketByPlacement(buckets)
	if got["a"] != 20_000 {
		t.Fatalf("a=%d want 20000", got["a"])
	}
	if got["b"] != 30_000 {
		t.Fatalf("b=%d want 30000", got["b"])
	}
}
