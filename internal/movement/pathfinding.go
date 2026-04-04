package movement

// TileCoord represents an (X, Y) tile position on the map.
type TileCoord struct {
	X int
	Y int
}

// QueuedWorldAction identifies the action to perform after an auto-walk completes.
type QueuedWorldAction int

const (
	QueuedActionNone QueuedWorldAction = iota
	QueuedActionPickupItem
	QueuedActionFaceTile
)

// MaxAutoWalkVisitedTiles caps BFS search to prevent unbounded exploration.
const MaxAutoWalkVisitedTiles = 16_384

// AutoWalkPlan holds the state for queued auto-walk navigation.
type AutoWalkPlan struct {
	Active     bool
	Directions []int
	Action     QueuedWorldAction
	ActionTile TileCoord
	ItemUID    int
}

// CanStepFunc is called by the pathfinder to check if a tile is walkable.
type CanStepFunc func(x, y int) bool

// OrderedPathDirections returns directions sorted by greedy preference toward the goal.
func OrderedPathDirections(from TileCoord, goal TileCoord) [4]int {
	primary, ok := DirectionTowardTile(from.X, from.Y, goal.X, goal.Y)
	if !ok {
		return [4]int{0, 1, 2, 3}
	}
	var order [4]int
	order[0] = primary
	idx := 1
	for _, dir := range [4]int{0, 1, 2, 3} {
		if dir != primary {
			order[idx] = dir
			idx++
		}
	}
	return order
}

// RebuildPath traces back from goal to start through the BFS parent map.
func RebuildPath(start, goal TileCoord, previous map[TileCoord]TileCoord, previousDir map[TileCoord]int) []int {
	path := make([]int, 0, 16)
	for current := goal; current != start; current = previous[current] {
		path = append(path, previousDir[current])
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

// FindPath runs BFS from start to any of the goals, using canStep to check walkability.
func FindPath(start TileCoord, goals []TileCoord, visitLimit int, canStep CanStepFunc) ([]int, bool) {
	if len(goals) == 0 {
		return nil, false
	}

	goalSet := make(map[TileCoord]struct{}, len(goals))
	for _, goal := range goals {
		goalSet[goal] = struct{}{}
	}
	if _, ok := goalSet[start]; ok {
		return nil, true
	}

	queue := []TileCoord{start}
	visited := map[TileCoord]struct{}{start: {}}
	previous := make(map[TileCoord]TileCoord, 32)
	previousDir := make(map[TileCoord]int, 32)

	for len(queue) > 0 && len(visited) <= visitLimit {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range OrderedPathDirections(current, goals[0]) {
			nextX, nextY := NextTileInDirection(current.X, current.Y, dir)
			next := TileCoord{X: nextX, Y: nextY}
			if _, seen := visited[next]; seen {
				continue
			}
			if !canStep(next.X, next.Y) {
				continue
			}

			visited[next] = struct{}{}
			previous[next] = current
			previousDir[next] = dir
			if _, ok := goalSet[next]; ok {
				return RebuildPath(start, next, previous, previousDir), true
			}
			queue = append(queue, next)
		}
	}

	return nil, false
}

// AutoWalkVisitLimit calculates the BFS budget based on map size and goal distance.
func AutoWalkVisitLimit(mapWidth, mapHeight int, start TileCoord, goals []TileCoord) int {
	mapTiles := (mapWidth + 1) * (mapHeight + 1)
	if mapTiles <= 0 {
		return 2_048
	}
	if mapTiles <= MaxAutoWalkVisitedTiles {
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

		distance := AbsInt(start.X-goal.X) + AbsInt(start.Y-goal.Y)
		if nearestGoalDistance < 0 || distance < nearestGoalDistance {
			nearestGoalDistance = distance
		}
	}

	detourMargin := max(12, nearestGoalDistance/2)
	searchWidth := maxX - minX + 1 + detourMargin*2
	searchHeight := maxY - minY + 1 + detourMargin*2
	searchArea := min(max(searchWidth*searchHeight, 2_048), MaxAutoWalkVisitedTiles)
	if searchArea > mapTiles {
		return mapTiles
	}
	return searchArea
}
