package main

import (
	"testing"

	"github.com/avdo/eoweb/internal/movement"
)

func TestBlockedTileSpecMatchesReferenceWalkBlockers(t *testing.T) {
	for _, spec := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 16, 18, 19, 20, 21, 22, 23, 24, 25, 26} {
		if !movement.BlockedTileSpec(spec) {
			t.Fatalf("BlockedTileSpec(%d) = false, want true", spec)
		}
	}
	if movement.BlockedTileSpec(0) {
		t.Fatal("BlockedTileSpec(0) = true, want false")
	}
	for _, spec := range []int{12, 13, 14, 15, 17, 27} {
		if movement.BlockedTileSpec(spec) {
			t.Fatalf("BlockedTileSpec(%d) = true, want false", spec)
		}
	}
}

func TestPlayerPassThroughAllowedAfterHold(t *testing.T) {
	g := &Game{}
	if g.playerPassThroughAllowed(3) {
		t.Fatal("player pass-through should be blocked without hold")
	}
	g.moveHoldDir = 3
	g.moveHoldTicks = playerGhostHoldTicks - 1
	if g.playerPassThroughAllowed(3) {
		t.Fatal("player pass-through should still be blocked below threshold")
	}
	g.moveHoldTicks = playerGhostHoldTicks
	if !g.playerPassThroughAllowed(3) {
		t.Fatal("player pass-through should be allowed at threshold")
	}
	if g.playerPassThroughAllowed(2) {
		t.Fatal("player pass-through should require the same held direction")
	}
}
