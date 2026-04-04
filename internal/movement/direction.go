package movement

// NextTileInDirection returns the tile coordinates one step in the given direction.
func NextTileInDirection(tileX, tileY, dir int) (int, int) {
	switch dir {
	case 0:
		return tileX, tileY + 1
	case 1:
		return tileX - 1, tileY
	case 2:
		return tileX, tileY - 1
	case 3:
		return tileX + 1, tileY
	default:
		return tileX, tileY
	}
}

// DirectionTowardTile returns the cardinal direction from (fromX,fromY) toward (toX,toY).
func DirectionTowardTile(fromX, fromY, toX, toY int) (int, bool) {
	dx := toX - fromX
	dy := toY - fromY
	if dx == 0 && dy == 0 {
		return -1, false
	}
	if AbsInt(dx) >= AbsInt(dy) {
		if dx < 0 {
			return 1, true
		}
		return 3, true
	}
	if dy < 0 {
		return 2, true
	}
	return 0, true
}

// IsAdjacentTile returns true if two tiles are exactly one step apart (Manhattan distance 1).
func IsAdjacentTile(ax, ay, bx, by int) bool {
	return AbsInt(ax-bx)+AbsInt(ay-by) == 1
}

// AbsInt returns the absolute value of an integer.
func AbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// BlockedTileSpec returns true if the tile spec value represents a non-walkable tile.
func BlockedTileSpec(spec int) bool {
	switch spec {
	case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 16, 18, 19, 20, 21, 22, 23, 24, 25, 26:
		return true
	default:
		return false
	}
}
