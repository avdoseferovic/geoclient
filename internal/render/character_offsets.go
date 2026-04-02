package render

type characterFrameOffset struct {
	X float64
	Y float64
}

const (
	characterFrameSize     = 100
	halfCharacterFrameSize = characterFrameSize / 2
	characterFootAnchorY   = TileHeight - HalfHalfTileH
	npcFootAnchorY         = 23
)

func CharacterDrawPosition(gender int, frame CharacterFrame, direction int, sx, sy float64, w, h int) (float64, float64) {
	offset := characterFrameOffsetFor(gender, frame, direction)
	drawX := sx - float64(w)/2 + offset.X
	drawY := sy + float64(halfCharacterFrameSize-h/2-characterFrameSize+characterFootAnchorY) + offset.Y
	return drawX, drawY
}

func characterFrameOffsetFor(gender int, frame CharacterFrame, direction int) characterFrameOffset {
	isRight := direction == 3
	isLeft := direction == 1
	isUp := direction == 2

	switch gender {
	case 0: // female
		switch frame {
		case FrameRaisedDown, FrameRaisedUp:
			return characterFrameOffset{Y: -2}
		case FrameMeleeDown1:
			if isRight {
				return characterFrameOffset{X: 1}
			}
			return characterFrameOffset{X: -1}
		case FrameMeleeDown2:
			if isRight {
				return characterFrameOffset{X: 5, Y: 1}
			}
			return characterFrameOffset{X: -5, Y: 1}
		case FrameMeleeUp1:
			if isUp {
				return characterFrameOffset{X: 1}
			}
			if isLeft {
				return characterFrameOffset{X: -1}
			}
		case FrameMeleeUp2:
			if isUp {
				return characterFrameOffset{X: 5, Y: -1}
			}
			if isLeft {
				return characterFrameOffset{X: -5, Y: -1}
			}
		case FrameChairDown, FrameFloorDown:
			if isRight {
				return characterFrameOffset{X: 2, Y: 13}
			}
			return characterFrameOffset{X: -2, Y: 13}
		case FrameChairUp:
			if isUp {
				return characterFrameOffset{X: 3, Y: 15}
			}
			if isLeft {
				return characterFrameOffset{X: -3, Y: 15}
			}
		case FrameFloorUp:
			if isUp {
				return characterFrameOffset{X: 2, Y: 16}
			}
			if isLeft {
				return characterFrameOffset{X: -2, Y: 16}
			}
		case FrameRangeDown:
			if isRight {
				return characterFrameOffset{X: 7, Y: 1}
			}
			return characterFrameOffset{X: -7, Y: 1}
		case FrameRangeUp:
			if isUp {
				return characterFrameOffset{X: 5}
			}
			if isLeft {
				return characterFrameOffset{X: -5}
			}
		}
	case 1: // male
		switch frame {
		case FrameStandDown, FrameStandUp:
			return characterFrameOffset{Y: 1}
		case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
			if isRight {
				return characterFrameOffset{X: 1, Y: 1}
			}
			return characterFrameOffset{X: -1, Y: 1}
		case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
			return characterFrameOffset{Y: 1}
		case FrameRaisedDown, FrameRaisedUp:
			return characterFrameOffset{Y: -1}
		case FrameMeleeDown1:
			if isRight {
				return characterFrameOffset{X: 2, Y: 1}
			}
			return characterFrameOffset{X: -2, Y: 1}
		case FrameMeleeDown2:
			if isRight {
				return characterFrameOffset{X: 4, Y: 2}
			}
			return characterFrameOffset{X: -4, Y: 2}
		case FrameMeleeUp1:
			if isUp {
				return characterFrameOffset{X: 2, Y: 1}
			}
			if isLeft {
				return characterFrameOffset{X: -2, Y: 1}
			}
		case FrameMeleeUp2:
			if isUp {
				return characterFrameOffset{X: 4}
			}
			if isLeft {
				return characterFrameOffset{X: -4}
			}
		case FrameChairDown, FrameFloorDown:
			if isRight {
				return characterFrameOffset{X: 3, Y: 12}
			}
			return characterFrameOffset{X: -3, Y: 12}
		case FrameChairUp:
			if isUp {
				return characterFrameOffset{X: 3, Y: 14}
			}
			if isLeft {
				return characterFrameOffset{X: -3, Y: 14}
			}
		case FrameFloorUp:
			if isUp {
				return characterFrameOffset{X: 2, Y: 15}
			}
			if isLeft {
				return characterFrameOffset{X: -2, Y: 15}
			}
		case FrameRangeDown:
			if isRight {
				return characterFrameOffset{X: 8, Y: 2}
			}
			return characterFrameOffset{X: -8, Y: 2}
		case FrameRangeUp:
			if isUp {
				return characterFrameOffset{X: 6, Y: 1}
			}
			if isLeft {
				return characterFrameOffset{X: -6, Y: 1}
			}
		}
	}

	return characterFrameOffset{}
}
