package main

import (
	"testing"

	"github.com/avdo/eoweb/internal/movement"
)

func TestBlockedTileSpecMatchesReferenceWalkBlockers(t *testing.T) {
	// Blocked: Wall(0), Chairs(1-7), Chest(9), BankVault(16), Edge(18), Boards(20-27), Jukebox(28)
	for _, spec := range []int{0, 1, 2, 3, 4, 5, 6, 7, 9, 16, 18, 20, 21, 22, 23, 24, 25, 26, 27, 28} {
		if !movement.BlockedTileSpec(spec) {
			t.Fatalf("BlockedTileSpec(%d) = false, want true", spec)
		}
	}
	// Walkable: no-spec(-1), reserved(8,10,11), warps(12-15), NpcBoundary(17), FakeWall(19)
	for _, spec := range []int{-1, 8, 10, 11, 12, 13, 14, 15, 17, 19, 29} {
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
