package render

// GFX file IDs for sprite types.
const (
	GfxSkinSprites   = 8
	GfxMaleHair      = 9
	GfxFemaleHair    = 10
	GfxMaleShoes     = 11
	GfxFemaleShoes   = 12
	GfxMaleArmor     = 13
	GfxFemaleArmor   = 14
	GfxMaleHat       = 15
	GfxFemaleHat     = 16
	GfxMaleWeapons   = 17
	GfxFemaleWeapons = 18
	GfxMaleBack      = 19
	GfxFemaleBack    = 20
	GfxNPC           = 21
	GfxShadows       = 22
	GfxItems         = 23
	GfxSpells        = 24
)

// Character frame dimensions.
const (
	CharWidth        = 18
	CharHeight       = 58
	CharWalkWidth    = 26
	CharWalkHeight   = 61
	CharMeleeWidth   = 24
	CharRaisedHeight = 62
	CharSitChairW    = 24
	CharSitChairH    = 52
	CharSitGroundW   = 24
	CharSitGroundH   = 43
	CharRangeWidth   = 25
)

// CharacterFrame identifies a specific animation frame.
type CharacterFrame int

const (
	FrameStandDown  CharacterFrame = 0
	FrameStandUp    CharacterFrame = 1
	FrameWalkDown1  CharacterFrame = 2
	FrameWalkDown2  CharacterFrame = 3
	FrameWalkDown3  CharacterFrame = 4
	FrameWalkDown4  CharacterFrame = 5
	FrameWalkUp1    CharacterFrame = 6
	FrameWalkUp2    CharacterFrame = 7
	FrameWalkUp3    CharacterFrame = 8
	FrameWalkUp4    CharacterFrame = 9
	FrameRaisedDown CharacterFrame = 10
	FrameRaisedUp   CharacterFrame = 11
	FrameMeleeDown1 CharacterFrame = 12
	FrameMeleeDown2 CharacterFrame = 13
	FrameMeleeUp1   CharacterFrame = 14
	FrameMeleeUp2   CharacterFrame = 15
	FrameChairDown  CharacterFrame = 16
	FrameChairUp    CharacterFrame = 17
	FrameFloorDown  CharacterFrame = 18
	FrameFloorUp    CharacterFrame = 19
	FrameRangeDown  CharacterFrame = 20
	FrameRangeUp    CharacterFrame = 21
)

// skinGfxID returns the GFX resource ID for the skin sprite sheet of the given frame.
// Skin sprites are organized: 1=standing, 2=walking, 3=melee, 4=raised, 5=chair, 6=floor, 7=range.
func skinGfxID(frame CharacterFrame) int {
	switch frame {
	case FrameStandDown, FrameStandUp:
		return 1
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
		FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return 2
	case FrameMeleeDown1, FrameMeleeDown2, FrameMeleeUp1, FrameMeleeUp2:
		return 3
	case FrameRaisedDown, FrameRaisedUp:
		return 4
	case FrameChairDown, FrameChairUp:
		return 5
	case FrameFloorDown, FrameFloorUp:
		return 6
	case FrameRangeDown, FrameRangeUp:
		return 7
	}
	return 1
}

// frameSize returns the pixel dimensions for a character frame.
func frameSize(frame CharacterFrame) (w, h int) {
	switch frame {
	case FrameStandDown, FrameStandUp:
		return CharWidth, CharHeight
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
		FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return CharWalkWidth, CharWalkHeight
	case FrameMeleeDown1, FrameMeleeDown2, FrameMeleeUp1, FrameMeleeUp2:
		return CharMeleeWidth, CharHeight
	case FrameRaisedDown, FrameRaisedUp:
		return CharWidth, CharRaisedHeight
	case FrameChairDown, FrameChairUp:
		return CharSitChairW, CharSitChairH
	case FrameFloorDown, FrameFloorUp:
		return CharSitGroundW, CharSitGroundH
	case FrameRangeDown, FrameRangeUp:
		return CharRangeWidth, CharHeight
	}
	return CharWidth, CharHeight
}

// frameCount returns how many sub-frames exist in the sprite sheet for this frame type.
func frameCount(frame CharacterFrame) int {
	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
		FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return 4
	case FrameMeleeDown1, FrameMeleeDown2, FrameMeleeUp1, FrameMeleeUp2:
		return 2
	default:
		return 1
	}
}

// frameNumber returns the sub-frame index within the sprite sheet.
func frameNumber(frame CharacterFrame) int {
	switch frame {
	case FrameWalkDown1, FrameWalkUp1:
		return 0
	case FrameWalkDown2, FrameWalkUp2:
		return 1
	case FrameWalkDown3, FrameWalkUp3:
		return 2
	case FrameWalkDown4, FrameWalkUp4:
		return 3
	case FrameMeleeDown2, FrameMeleeUp2:
		return 1
	default:
		return 0
	}
}

// isUpLeft returns true if this frame faces up-left.
func isUpLeft(frame CharacterFrame) bool {
	switch frame {
	case FrameStandUp, FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4,
		FrameRaisedUp, FrameMeleeUp1, FrameMeleeUp2, FrameChairUp, FrameFloorUp, FrameRangeUp:
		return true
	}
	return false
}

// GenderGfx returns the appropriate GFX file ID for a gender (0=female, 1=male).
func GenderGfx(gender int, female, male int) int {
	if gender == 0 {
		return female
	}
	return male
}
