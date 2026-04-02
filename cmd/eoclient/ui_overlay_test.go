package main

import "testing"

func TestTnlProgress(t *testing.T) {
	remaining, rng := tnlProgress(1, expForLevel(1))
	if remaining != expForLevel(2)-expForLevel(1) {
		t.Fatalf("remaining = %d, want %d", remaining, expForLevel(2)-expForLevel(1))
	}
	if rng != expForLevel(2)-expForLevel(1) {
		t.Fatalf("range = %d, want %d", rng, expForLevel(2)-expForLevel(1))
	}

	midExp := expForLevel(2) + 50
	remaining, rng = tnlProgress(2, midExp)
	if remaining != expForLevel(3)-midExp {
		t.Fatalf("remaining(mid) = %d, want %d", remaining, expForLevel(3)-midExp)
	}
	if rng != expForLevel(3)-expForLevel(2) {
		t.Fatalf("range(mid) = %d, want %d", rng, expForLevel(3)-expForLevel(2))
	}
}
