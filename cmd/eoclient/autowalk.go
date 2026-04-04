package main

import "github.com/avdo/eoweb/internal/movement"

type autoWalkPlan = movement.AutoWalkPlan

func (g *Game) clearAutoWalk() {
	g.autoWalk = autoWalkPlan{}
}

func (g *Game) advanceAutoWalk() bool {
	if !g.autoWalk.Active || g.isWalking || g.walkCooldown > 0 {
		return false
	}
	if len(g.autoWalk.Directions) == 0 {
		return g.completeAutoWalkAction()
	}

	dir := g.autoWalk.Directions[0]
	nextX, nextY := movement.NextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
	if !g.canStepTo(nextX, nextY) {
		g.clearAutoWalk()
		return false
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	if !g.walkImmediately(dir) {
		g.clearAutoWalk()
		return false
	}
	g.autoWalk.Directions = g.autoWalk.Directions[1:]
	return true
}

func (g *Game) completeAutoWalkAction() bool {
	if !g.autoWalk.Active {
		return false
	}

	plan := g.autoWalk
	g.clearAutoWalk()

	switch plan.Action {
	case movement.QueuedActionPickupItem:
		if g.client.Character.X != plan.ActionTile.X || g.client.Character.Y != plan.ActionTile.Y {
			return false
		}
		if !g.hasPickupItemUIDAtTile(plan.ActionTile.X, plan.ActionTile.Y, plan.ItemUID) {
			return false
		}
		if plan.ItemUID <= 0 {
			return false
		}
		g.sendPickupItem(plan.ItemUID)
		return true
	case movement.QueuedActionFaceTile:
		dir, ok := movement.DirectionTowardTile(g.client.Character.X, g.client.Character.Y, plan.ActionTile.X, plan.ActionTile.Y)
		if !ok {
			return false
		}
		return g.faceOnly(dir)
	default:
		return false
	}
}

func (g *Game) queueAutoWalkToTile(tileX, tileY int) bool {
	start := movement.TileCoord{X: g.client.Character.X, Y: g.client.Character.Y}
	goals := []movement.TileCoord{{X: tileX, Y: tileY}}
	path, ok := movement.FindPath(start, goals, g.autoWalkVisitLimit(start, goals), g.canStepTo)
	if !ok || len(path) == 0 {
		g.clearAutoWalk()
		return false
	}
	g.autoWalk = autoWalkPlan{Active: true, Directions: path}
	return true
}

func (g *Game) queueAutoWalkToInteraction(tileX, tileY int) bool {
	goals := make([]movement.TileCoord, 0, 4)
	for dir := 0; dir < 4; dir++ {
		nextX, nextY := movement.NextTileInDirection(tileX, tileY, dir)
		if !g.canStepTo(nextX, nextY) {
			continue
		}
		goals = append(goals, movement.TileCoord{X: nextX, Y: nextY})
	}
	if len(goals) == 0 {
		g.clearAutoWalk()
		return false
	}
	start := movement.TileCoord{X: g.client.Character.X, Y: g.client.Character.Y}
	path, ok := movement.FindPath(start, goals, g.autoWalkVisitLimit(start, goals), g.canStepTo)
	if !ok {
		g.clearAutoWalk()
		return false
	}
	if len(path) == 0 {
		dir, ok := movement.DirectionTowardTile(g.client.Character.X, g.client.Character.Y, tileX, tileY)
		if !ok {
			return false
		}
		return g.faceOnly(dir)
	}
	g.autoWalk = autoWalkPlan{
		Active:     true,
		Directions: path,
		Action:     movement.QueuedActionFaceTile,
		ActionTile: movement.TileCoord{X: tileX, Y: tileY},
	}
	return true
}

func (g *Game) queueAutoPickup(tileX, tileY int, itemUID int) bool {
	if withinItemInteractionRange(g.client.Character.X, g.client.Character.Y, tileX, tileY) {
		if !g.hasPickupItemUIDAtTile(tileX, tileY, itemUID) {
			return false
		}
		g.sendPickupItem(itemUID)
		return true
	}
	start := movement.TileCoord{X: g.client.Character.X, Y: g.client.Character.Y}
	goals := []movement.TileCoord{{X: tileX, Y: tileY}}
	path, ok := movement.FindPath(start, goals, g.autoWalkVisitLimit(start, goals), g.canStepTo)
	if !ok {
		g.clearAutoWalk()
		return false
	}
	if len(path) == 0 {
		if !g.hasPickupItemUIDAtTile(tileX, tileY, itemUID) {
			return false
		}
		g.sendPickupItem(itemUID)
		return true
	}
	g.autoWalk = autoWalkPlan{
		Active:     true,
		Directions: path,
		Action:     movement.QueuedActionPickupItem,
		ActionTile: movement.TileCoord{X: tileX, Y: tileY},
		ItemUID:    itemUID,
	}
	return true
}

func (g *Game) findPickupItemAtTile(tileX, tileY, preferredUID int) (int, bool) {
	for _, item := range g.client.NearbyItems {
		if item.X != tileX || item.Y != tileY {
			continue
		}
		if preferredUID != 0 && item.UID != preferredUID {
			continue
		}
		return item.UID, true
	}
	if preferredUID != 0 {
		for _, item := range g.client.NearbyItems {
			if item.X == tileX && item.Y == tileY {
				return item.UID, true
			}
		}
	}
	return 0, false
}

func (g *Game) hasPickupItemUIDAtTile(tileX, tileY, itemUID int) bool {
	if itemUID <= 0 {
		return false
	}
	for _, item := range g.client.NearbyItems {
		if item.UID != itemUID {
			continue
		}
		return item.X == tileX && item.Y == tileY
	}
	return false
}

func (g *Game) autoWalkVisitLimit(start movement.TileCoord, goals []movement.TileCoord) int {
	if g.mapRenderer.Map == nil {
		return 2_048
	}
	return movement.AutoWalkVisitLimit(g.mapRenderer.Map.Width, g.mapRenderer.Map.Height, start, goals)
}
