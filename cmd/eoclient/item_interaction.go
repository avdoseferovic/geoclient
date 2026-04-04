package main

import (
	"github.com/avdo/eoweb/internal/movement"
)

const itemInteractionRange = 2

func tileStepDistance(ax, ay, bx, by int) int {
	return movement.AbsInt(ax-bx) + movement.AbsInt(ay-by)
}

func withinItemInteractionRange(playerX, playerY, tileX, tileY int) bool {
	return tileStepDistance(playerX, playerY, tileX, tileY) <= itemInteractionRange
}
