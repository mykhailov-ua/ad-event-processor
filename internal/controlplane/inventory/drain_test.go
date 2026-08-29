package inventory

import "testing"

func TestCompositionDrainInventory_holdout(t *testing.T) {
	if len(CompositionDrain) == 0 {
		t.Fatal("composition drain inventory must not be empty")
	}
	for _, row := range CompositionDrain {
		if row.File == "" || row.Role == "" {
			t.Fatalf("incomplete drain row: %+v", row)
		}
	}
}
