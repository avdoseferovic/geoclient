package main

import "testing"

func TestWithinItemInteractionRange(t *testing.T) {
	tests := []struct {
		name        string
		playerX     int
		playerY     int
		tileX       int
		tileY       int
		wantInRange bool
	}{
		{name: "same tile", playerX: 10, playerY: 10, tileX: 10, tileY: 10, wantInRange: true},
		{name: "one step", playerX: 10, playerY: 10, tileX: 11, tileY: 10, wantInRange: true},
		{name: "two steps", playerX: 10, playerY: 10, tileX: 12, tileY: 10, wantInRange: true},
		{name: "diagonal two plus one", playerX: 10, playerY: 10, tileX: 11, tileY: 11, wantInRange: true},
		{name: "three steps", playerX: 10, playerY: 10, tileX: 13, tileY: 10, wantInRange: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := withinItemInteractionRange(tt.playerX, tt.playerY, tt.tileX, tt.tileY); got != tt.wantInRange {
				t.Fatalf("withinItemInteractionRange() = %v, want %v", got, tt.wantInRange)
			}
		})
	}
}
