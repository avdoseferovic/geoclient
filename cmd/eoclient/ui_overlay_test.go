package main

import (
	"testing"

	"github.com/avdo/eoweb/internal/ui/overlay"
)

func TestTnlProgress(t *testing.T) {
	remaining, rng := overlay.TnlProgress(1, overlay.ExpForLevel(1))
	if remaining != overlay.ExpForLevel(2)-overlay.ExpForLevel(1) {
		t.Fatalf("remaining = %d, want %d", remaining, overlay.ExpForLevel(2)-overlay.ExpForLevel(1))
	}
	if rng != overlay.ExpForLevel(2)-overlay.ExpForLevel(1) {
		t.Fatalf("range = %d, want %d", rng, overlay.ExpForLevel(2)-overlay.ExpForLevel(1))
	}

	midExp := overlay.ExpForLevel(2) + 50
	remaining, rng = overlay.TnlProgress(2, midExp)
	if remaining != overlay.ExpForLevel(3)-midExp {
		t.Fatalf("remaining(mid) = %d, want %d", remaining, overlay.ExpForLevel(3)-midExp)
	}
	if rng != overlay.ExpForLevel(3)-overlay.ExpForLevel(2) {
		t.Fatalf("range(mid) = %d, want %d", rng, overlay.ExpForLevel(3)-overlay.ExpForLevel(2))
	}
}
