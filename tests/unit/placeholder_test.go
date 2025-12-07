package unit

import "testing"

func TestPlaceholder(t *testing.T) {
	// trivial test to ensure test pipeline is configured in Step 1
	if 2+2 != 4 {
		t.Fatalf("math broken")
	}
}
