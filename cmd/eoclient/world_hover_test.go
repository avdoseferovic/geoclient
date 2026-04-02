package main

import (
	"testing"

	"github.com/avdo/eoweb/internal/game"
)

func TestGroundItemHoverData(t *testing.T) {
	snapshot := game.UISnapshot{
		NearbyItems: []game.NearbyItem{
			{UID: 17, ID: 5, Amount: 3},
		},
	}

	itemID, amount := groundItemHoverData(snapshot, 17)
	if itemID != 5 || amount != 3 {
		t.Fatalf("groundItemHoverData = (%d, %d), want (5, 3)", itemID, amount)
	}
	itemID, amount = groundItemHoverData(snapshot, 99)
	if itemID != 0 || amount != 0 {
		t.Fatalf("groundItemHoverData(miss) = (%d, %d), want (0, 0)", itemID, amount)
	}
}

func TestGroundItemHoverAllowed(t *testing.T) {
	if !groundItemHoverAllowed(worldHoverIntent{TileX: 10, TileY: 12, CursorType: 2, Valid: true}, 10, 12) {
		t.Fatal("expected item hover to be allowed on the hovered item tile")
	}
	if groundItemHoverAllowed(worldHoverIntent{TileX: 10, TileY: 12, CursorType: 0, Valid: true}, 10, 12) {
		t.Fatal("expected item hover to be blocked when the cursor is not the item cursor")
	}
	if groundItemHoverAllowed(worldHoverIntent{TileX: 9, TileY: 12, CursorType: 2, Valid: true}, 10, 12) {
		t.Fatal("expected item hover to be blocked off the item's tile")
	}
	if groundItemHoverAllowed(worldHoverIntent{TileX: 10, TileY: 12, CursorType: -1, Valid: false}, 10, 12) {
		t.Fatal("expected item hover to be blocked when the crosshair is hidden")
	}
}
