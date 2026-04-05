package main

import (
	"github.com/avdo/eoweb/internal/movement"
	"github.com/avdo/eoweb/internal/render"
)

const (
	faceCooldownTicks      = 3
	walkStartCooldownTicks = 24
	attackCooldownTicks    = 30
	playerGhostHoldTicks   = 24
)

type stepBlocker int

const (
	stepBlockerNone stepBlocker = iota
	stepBlockerTile
	stepBlockerNPC
	stepBlockerPlayer
)

func (g *Game) resetMovementState() {
	g.walkCooldown = 0
	g.attackCooldown = 0
	g.isWalking = false
	g.moveHoldDir = -1
	g.moveHoldTicks = 0
}

func (g *Game) faceOrWalk(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 {
		return false
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	return g.stepToward(dir)
}

func (g *Game) faceOnly(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 {
		return false
	}
	g.facingDir = dir
	g.sendFace(dir)
	g.walkCooldown = faceCooldownTicks
	return true
}

func (g *Game) stepToward(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 || g.isWalking {
		return false
	}
	nextX, nextY := movement.NextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
	if !g.canArrowStepTo(nextX, nextY, dir) {
		// Walking into a closed door opens it
		if g.isDoorTile(nextX, nextY) && !g.client.IsDoorOpen(nextX, nextY) {
			g.sendDoorOpen(nextX, nextY)
			return g.faceOnly(dir)
		}
		return g.faceOnly(dir)
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	return g.walkImmediately(dir)
}

func (g *Game) walkImmediately(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 || g.isWalking {
		return false
	}
	nextX, nextY := movement.NextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
	if !g.canArrowStepTo(nextX, nextY, dir) {
		return false
	}
	g.facingDir = dir
	g.startLocalWalk(dir)
	g.walkCooldown = walkStartCooldownTicks
	return true
}

func (g *Game) faceOrAttack(dir int) bool {
	if dir < 0 {
		return false
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	if g.attackCooldown > 0 {
		return true
	}
	g.sendAttack()
	g.attackCooldown = attackCooldownTicks
	return true
}

func (g *Game) startLocalWalk(dir int) {
	g.isWalking = true
	g.sendWalk(dir)

	g.client.Lock()
	defer g.client.Unlock()
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID == g.client.PlayerID {
			g.client.NearbyChars[i].Walking = true
			g.client.NearbyChars[i].WalkTick = 0
			break
		}
	}
}

func (g *Game) tickAnimations() {
	g.client.Lock()
	defer g.client.Unlock()

	g.client.TickDoors()

	for i := range g.client.NearbyChars {
		ch := &g.client.NearbyChars[i]
		ch.Combat.Tick()
		if ch.TickWalk() {
			if ch.PlayerID == g.client.PlayerID {
				g.isWalking = false
			}
		}
	}
	nextNpcs := g.client.NearbyNpcs[:0]
	for i := range g.client.NearbyNpcs {
		npc := g.client.NearbyNpcs[i]
		npc.Tick()
		if npc.DeathComplete() {
			npc.Hidden = true
		}
		nextNpcs = append(nextNpcs, npc)
	}
	g.client.NearbyNpcs = nextNpcs
}

func (g *Game) updateMoveHold(dir int) {
	if dir < 0 {
		g.moveHoldDir = -1
		g.moveHoldTicks = 0
		return
	}
	if g.moveHoldDir != dir {
		g.moveHoldDir = dir
		g.moveHoldTicks = 1
		return
	}
	g.moveHoldTicks++
}

// --- Collision ---

func (g *Game) isMapTileInBounds(tileX, tileY int) bool {
	if g.mapRenderer.Map == nil {
		return false
	}
	if tileX < 0 || tileY < 0 {
		return false
	}
	return tileX <= g.mapRenderer.Map.Width && tileY <= g.mapRenderer.Map.Height
}

func (g *Game) canStepTo(tileX, tileY int) bool {
	if !g.isMapTileInBounds(tileX, tileY) {
		return false
	}
	return g.stepBlockerAt(tileX, tileY) == stepBlockerNone
}

func (g *Game) canArrowStepTo(tileX, tileY, dir int) bool {
	switch g.stepBlockerAt(tileX, tileY) {
	case stepBlockerNone:
		return true
	case stepBlockerPlayer:
		return g.playerPassThroughAllowed(dir)
	default:
		return false
	}
}

func (g *Game) playerPassThroughAllowed(dir int) bool {
	return dir >= 0 && g.moveHoldDir == dir && g.moveHoldTicks >= playerGhostHoldTicks
}

func (g *Game) stepBlockerAt(tileX, tileY int) stepBlocker {
	if !g.isMapTileInBounds(tileX, tileY) {
		return stepBlockerTile
	}
	if movement.BlockedTileSpec(g.tileSpecAt(tileX, tileY)) {
		// Open doors are walkable even though the warp tile spec blocks
		if !g.client.IsDoorOpen(tileX, tileY) {
			return stepBlockerTile
		}
	}
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return stepBlockerNPC
		}
	}
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID {
			continue
		}
		if ch.X == tileX && ch.Y == tileY {
			return stepBlockerPlayer
		}
	}
	return stepBlockerNone
}

func (g *Game) isDoorTile(tileX, tileY int) bool {
	return g.client.GetDoor(tileX, tileY) != nil
}

func (g *Game) tileSpecAt(tileX, tileY int) int {
	if g.mapRenderer.Map == nil {
		return -1
	}
	for _, row := range g.mapRenderer.Map.TileSpecRows {
		if row.Y != tileY {
			continue
		}
		for _, tile := range row.Tiles {
			if tile.X == tileX {
				return int(tile.TileSpec)
			}
		}
	}
	return -1
}

func (g *Game) isAttackTargetTile(tileX, tileY int) bool {
	return false
}

func (g *Game) getCursorType(tileX, tileY int) int {
	spec := g.tileSpecAt(tileX, tileY)
	// Wall (0) and Edge (18) hide the cursor entirely
	if spec == 0 || spec == 18 {
		return -1
	}
	// Closed doors show as interaction targets
	if g.isDoorTile(tileX, tileY) && !g.client.IsDoorOpen(tileX, tileY) {
		return 1
	}
	if movement.BlockedTileSpec(spec) {
		return 1
	}
	for _, ch := range g.client.NearbyChars {
		if ch.X == tileX && ch.Y == tileY {
			return 1
		}
	}
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return 1
		}
	}
	for _, item := range g.client.NearbyItems {
		if item.X == tileX && item.Y == tileY {
			return 2
		}
	}
	return 0
}

func (g *Game) playerCamOffset() (float64, float64) {
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID && ch.Walking {
			return render.WalkOffset(ch.Direction, ch.WalkProgress())
		}
	}
	return 0, 0
}
