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

func TestHatOffsetMatchesReference(t *testing.T) {
	tests := []struct {
		name   string
		gender int
		frame  CharacterFrame
		wantX  float64
		wantY  float64
	}{
		{name: "female stand down", gender: 0, frame: FrameStandDown, wantX: 0, wantY: 23},
		{name: "female melee down 2", gender: 0, frame: FrameMeleeDown2, wantX: -3, wantY: 28},
		{name: "male stand down", gender: 1, frame: FrameStandDown, wantX: 0, wantY: 22},
		{name: "male walk down", gender: 1, frame: FrameWalkDown1, wantX: 1, wantY: 22},
		{name: "male melee up 2", gender: 1, frame: FrameMeleeUp2, wantX: -2, wantY: 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hatOffset(tt.gender, tt.frame)
			if got.X != tt.wantX || got.Y != tt.wantY {
				t.Fatalf("hatOffset(%d, %v) = {%v, %v}, want {%v, %v}", tt.gender, tt.frame, got.X, got.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestHatDrawPositionAnchorsToCharacterFrame(t *testing.T) {
	x, y := hatDrawPosition(91, 146, CharWidth, CharHeight, 30, 40, attachmentOffset{X: 0, Y: 22}, false)
	if x != 85 {
		t.Fatalf("x = %v, want 85", x)
	}
	if y != 147 {
		t.Fatalf("y = %v, want 147", y)
	}
}
