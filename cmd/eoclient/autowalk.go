package main

const maxAutoWalkVisitedTiles = 16_384

type tileCoord struct {
	X int
	Y int
}

type queuedWorldAction int

const (
	queuedWorldActionNone queuedWorldAction = iota
	queuedWorldActionPickupItem
	queuedWorldActionFaceTile
)

type autoWalkPlan struct {
	Active     bool
	Directions []int
	Action     queuedWorldAction
	ActionTile tileCoord
	ItemUID    int
}

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
	nextX, nextY := nextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
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
	case queuedWorldActionPickupItem:
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
	case queuedWorldActionFaceTile:
		dir, ok := directionTowardTile(g.client.Character.X, g.client.Character.Y, plan.ActionTile.X, plan.ActionTile.Y)
		if !ok {
			return false
		}
		return g.faceOnly(dir)
	default:
		return false
	}
}

func (g *Game) queueAutoWalkToTile(tileX, tileY int) bool {
	path, ok := g.findPath(tileCoord{X: g.client.Character.X, Y: g.client.Character.Y}, []tileCoord{{X: tileX, Y: tileY}})
	if !ok {
		g.clearAutoWalk()
		return false
	}
	if len(path) == 0 {
		g.clearAutoWalk()
		return false
	}
	g.autoWalk = autoWalkPlan{Active: true, Directions: path}
	return true
}

func (g *Game) queueAutoWalkToInteraction(tileX, tileY int) bool {
	goals := make([]tileCoord, 0, 4)
	for dir := 0; dir < 4; dir++ {
		nextX, nextY := nextTileInDirection(tileX, tileY, dir)
		if !g.canStepTo(nextX, nextY) {
			continue
		}
		goals = append(goals, tileCoord{X: nextX, Y: nextY})
	}
	if len(goals) == 0 {
		g.clearAutoWalk()
		return false
	}
	path, ok := g.findPath(tileCoord{X: g.client.Character.X, Y: g.client.Character.Y}, goals)
	if !ok {
		g.clearAutoWalk()
		return false
	}
	if len(path) == 0 {
		dir, ok := directionTowardTile(g.client.Character.X, g.client.Character.Y, tileX, tileY)
		if !ok {
			return false
		}
		return g.faceOnly(dir)
	}
	g.autoWalk = autoWalkPlan{
		Active:     true,
		Directions: path,
		Action:     queuedWorldActionFaceTile,
		ActionTile: tileCoord{X: tileX, Y: tileY},
	}
	return true
}

func (g *Game) queueAutoPickup(tileX, tileY int, itemUID int) bool {
	path, ok := g.findPath(tileCoord{X: g.client.Character.X, Y: g.client.Character.Y}, []tileCoord{{X: tileX, Y: tileY}})
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
		Action:     queuedWorldActionPickupItem,
		ActionTile: tileCoord{X: tileX, Y: tileY},
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

func (g *Game) findPath(start tileCoord, goals []tileCoord) ([]int, bool) {
	if len(goals) == 0 {
		return nil, false
	}

	goalSet := make(map[tileCoord]struct{}, len(goals))
	for _, goal := range goals {
		goalSet[goal] = struct{}{}
	}
	if _, ok := goalSet[start]; ok {
		return nil, true
	}

	visitLimit := g.autoWalkVisitLimit(start, goals)

	queue := []tileCoord{start}
	visited := map[tileCoord]struct{}{start: {}}
	previous := make(map[tileCoord]tileCoord, 32)
	previousDir := make(map[tileCoord]int, 32)

	for len(queue) > 0 && len(visited) <= visitLimit {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range orderedPathDirections(current, goals[0]) {
			nextX, nextY := nextTileInDirection(current.X, current.Y, dir)
			next := tileCoord{X: nextX, Y: nextY}
			if _, seen := visited[next]; seen {
				continue
			}
			if !g.canStepTo(next.X, next.Y) {
				continue
			}

			visited[next] = struct{}{}
			previous[next] = current
			previousDir[next] = dir
			if _, ok := goalSet[next]; ok {
				return rebuildPath(start, next, previous, previousDir), true
			}
			queue = append(queue, next)
		}
	}

	return nil, false
}

func (g *Game) autoWalkVisitLimit(start tileCoord, goals []tileCoord) int {
	if g.mapRenderer.Map == nil {
		return 2_048
	}

	mapTiles := (g.mapRenderer.Map.Width + 1) * (g.mapRenderer.Map.Height + 1)
	if mapTiles <= 0 {
		return 2_048
	}
	if mapTiles <= maxAutoWalkVisitedTiles {
		return mapTiles
	}

	minX, maxX := start.X, start.X
	minY, maxY := start.Y, start.Y
	nearestGoalDistance := -1
	for _, goal := range goals {
		if goal.X < minX {
			minX = goal.X
		}
		if goal.X > maxX {
			maxX = goal.X
		}
		if goal.Y < minY {
			minY = goal.Y
		}
		if goal.Y > maxY {
			maxY = goal.Y
		}

		distance := absInt(start.X-goal.X) + absInt(start.Y-goal.Y)
		if nearestGoalDistance < 0 || distance < nearestGoalDistance {
			nearestGoalDistance = distance
		}
	}

	detourMargin := max(12, nearestGoalDistance/2)
	searchWidth := maxX - minX + 1 + detourMargin*2
	searchHeight := maxY - minY + 1 + detourMargin*2
	searchArea := searchWidth * searchHeight
	if searchArea < 2_048 {
		searchArea = 2_048
	}
	if searchArea > maxAutoWalkVisitedTiles {
		searchArea = maxAutoWalkVisitedTiles
	}
	if searchArea > mapTiles {
		return mapTiles
	}
	return searchArea
}

func orderedPathDirections(from tileCoord, goal tileCoord) []int {
	primary, ok := directionTowardTile(from.X, from.Y, goal.X, goal.Y)
	if !ok {
		return []int{0, 1, 2, 3}
	}
	order := []int{primary}
	for _, dir := range []int{0, 1, 2, 3} {
		if dir == primary {
			continue
		}
		order = append(order, dir)
	}
	return order
}

func rebuildPath(start, goal tileCoord, previous map[tileCoord]tileCoord, previousDir map[tileCoord]int) []int {
	path := make([]int, 0, 16)
	for current := goal; current != start; current = previous[current] {
		path = append(path, previousDir[current])
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}
