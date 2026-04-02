package render

import "testing"

func TestIsBackGraphic(t *testing.T) {
	backGraphics := []int{10, 11, 14, 15, 16, 18, 19}
	for _, id := range backGraphics {
		if !isBackGraphic(id) {
			t.Fatalf("expected shield %d to render as back graphic", id)
		}
	}

	normalGraphics := []int{1, 2, 9, 12, 13, 17, 20}
	for _, id := range normalGraphics {
		if isBackGraphic(id) {
			t.Fatalf("expected shield %d to render as normal shield", id)
		}
	}
}

func TestBackFrameNumber(t *testing.T) {
	tests := []struct {
		frame CharacterFrame
		want  int
	}{
		{FrameStandDown, 0},
		{FrameStandUp, 1},
		{FrameWalkDown3, 0},
		{FrameWalkUp2, 1},
		{FrameMeleeDown1, 2},
		{FrameMeleeUp2, 3},
		{FrameRaisedDown, 0},
		{FrameRaisedUp, 1},
		{FrameChairDown, 0},
		{FrameChairUp, 1},
		{FrameFloorDown, 0},
		{FrameFloorUp, 1},
		{FrameRangeDown, 2},
		{FrameRangeUp, 3},
	}

	for _, tt := range tests {
		if got := backFrameNumber(tt.frame); got != tt.want {
			t.Fatalf("backFrameNumber(%v) = %d, want %d", tt.frame, got, tt.want)
		}
	}
}

func TestHatMask(t *testing.T) {
	faceMasks := []int{7, 12, 15, 32, 48, 50}
	for _, id := range faceMasks {
		if got := hatMask(id); got != hatMaskFaceMask {
			t.Fatalf("hatMask(%d) = %v, want face mask", id, got)
		}
	}

	hideHair := []int{16, 21, 30, 37, 44, 47}
	for _, id := range hideHair {
		if got := hatMask(id); got != hatMaskHideHair {
			t.Fatalf("hatMask(%d) = %v, want hide hair", id, got)
		}
	}

	standard := []int{0, 1, 6, 22, 24, 27, 29, 39, 45, 49}
	for _, id := range standard {
		if got := hatMask(id); got != hatMaskStandard {
			t.Fatalf("hatMask(%d) = %v, want standard", id, got)
		}
	}
}

func TestHairGraphicIDs(t *testing.T) {
	baseID := (3-1)*40 + 2*4

	behindDown := baseID + 1
	frontDown := baseID + 2
	behindUp := baseID + 3
	frontUp := baseID + 4

	if behindDown != 89 || frontDown != 90 || behindUp != 91 || frontUp != 92 {
		t.Fatalf("unexpected hair graphic ids: got %d %d %d %d", behindDown, frontDown, behindUp, frontUp)
	}
}
