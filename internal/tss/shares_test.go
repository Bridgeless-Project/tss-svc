package tss

import "testing"

func TestMaxMaliciousParties(t *testing.T) {
	if actual := MaxMaliciousParties(4, 2); actual != 1 {
		t.Fatalf("unexpected max malicious parties: %d", actual)
	}
}
