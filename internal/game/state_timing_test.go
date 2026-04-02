package game

import "testing"

func TestNearbyCharacterWalkDurationMatchesReferenceTiming(t *testing.T) {
	ch := NearbyCharacter{Walking: true}

	for i := 0; i < WalkDuration-1; i++ {
		if done := ch.TickWalk(); done {
			t.Fatalf("walk completed early at tick %d", i+1)
		}
	}

	if ch.WalkTick != WalkDuration-1 {
		t.Fatalf("WalkTick = %d, want %d", ch.WalkTick, WalkDuration-1)
	}

	if done := ch.TickWalk(); !done {
		t.Fatal("walk did not complete on final tick")
	}

	if ch.Walking {
		t.Fatal("Walking = true after completion")
	}
}

func TestNearbyCharacterWalkFrameAdvancesEveryQuarter(t *testing.T) {
	ch := NearbyCharacter{Walking: true}

	tests := []struct {
		tick int
		want int
	}{
		{0, 0},
		{5, 0},
		{6, 1},
		{11, 1},
		{12, 2},
		{17, 2},
		{18, 3},
		{23, 3},
	}

	for _, tt := range tests {
		ch.WalkTick = tt.tick
		if got := ch.WalkFrame(); got != tt.want {
			t.Fatalf("WalkFrame(%d) = %d, want %d", tt.tick, got, tt.want)
		}
	}
}

func TestNearbyNPCIdleFrameMatchesReferenceTiming(t *testing.T) {
	npc := NearbyNPC{}

	tests := []struct {
		tick int
		want int
	}{
		{0, 0},
		{5, 0},
		{6, 1},
		{11, 1},
		{12, 0},
	}

	for _, tt := range tests {
		npc.IdleTick = tt.tick
		if got := npc.IdleFrame(); got != tt.want {
			t.Fatalf("IdleFrame(%d) = %d, want %d", tt.tick, got, tt.want)
		}
	}
}

func TestNearbyNPCWalkFrameAdvancesEveryQuarter(t *testing.T) {
	npc := NearbyNPC{Walking: true}

	tests := []struct {
		tick int
		want int
	}{
		{0, 0},
		{5, 0},
		{6, 1},
		{11, 1},
		{12, 2},
		{17, 2},
		{18, 3},
		{23, 3},
	}

	for _, tt := range tests {
		npc.WalkTick = tt.tick
		if got := npc.WalkFrame(); got != tt.want {
			t.Fatalf("WalkFrame(%d) = %d, want %d", tt.tick, got, tt.want)
		}
	}
}
