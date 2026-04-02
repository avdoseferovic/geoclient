package main

import "testing"

func TestBlockedTileSpecMatchesReferenceWalkBlockers(t *testing.T) {
	for _, spec := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26} {
		if !blockedTileSpec(spec) {
			t.Fatalf("blockedTileSpec(%d) = false, want true", spec)
		}
	}
	if blockedTileSpec(0) {
		t.Fatal("blockedTileSpec(0) = true, want false")
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
