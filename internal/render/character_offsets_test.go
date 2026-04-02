package render

import "testing"

func TestCharacterDrawPositionMatchesReferenceStandingMale(t *testing.T) {
	drawX, drawY := CharacterDrawPosition(1, FrameStandDown, 0, 100, 200, CharWidth, CharHeight)
	if drawX != 91 {
		t.Fatalf("drawX = %v, want 91", drawX)
	}
	if drawY != 146 {
		t.Fatalf("drawY = %v, want 146", drawY)
	}
}

func TestCharacterDrawPositionMatchesReferenceWalkingFemaleRight(t *testing.T) {
	drawX, drawY := CharacterDrawPosition(0, FrameWalkDown1, 3, 100, 200, CharWalkWidth, CharWalkHeight)
	if drawX != 87 {
		t.Fatalf("drawX = %v, want 87", drawX)
	}
	if drawY != 143.5 {
		t.Fatalf("drawY = %v, want 143.5", drawY)
	}
}

func TestNPCDrawPositionUsesReferenceFootAnchorAndMetadata(t *testing.T) {
	drawX, drawY := NPCDrawPosition(27, false, 100, 200, 40, 64)
	if drawX != 83 {
		t.Fatalf("drawX = %v, want 83", drawX)
	}
	if drawY != 146 {
		t.Fatalf("drawY = %v, want 146", drawY)
	}
}

func TestNPCDrawPositionMirrorsHorizontalMetadata(t *testing.T) {
	leftX, _ := NPCDrawPosition(27, false, 100, 200, 40, 64)
	rightX, _ := NPCDrawPosition(27, true, 100, 200, 40, 64)
	if leftX != 83 {
		t.Fatalf("leftX = %v, want 83", leftX)
	}
	if rightX != 77 {
		t.Fatalf("rightX = %v, want 77", rightX)
	}
}
