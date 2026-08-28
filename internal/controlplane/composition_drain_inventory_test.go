package controlplane

import "testing"

func TestCompositionDrainInventory_holdout(t *testing.T) {
	if len(compositionDrainInventory) == 0 {
		t.Fatal("composition drain inventory must not be empty")
	}
	for _, row := range compositionDrainInventory {
		if row.File == "" || row.Role == "" {
			t.Fatalf("incomplete drain row: %+v", row)
		}
	}
}
