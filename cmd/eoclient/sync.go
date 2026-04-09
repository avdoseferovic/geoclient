package main

import (
	"github.com/avdoseferovic/geoclient/internal/game"
	"github.com/avdoseferovic/geoclient/internal/render"
)

// syncEntities copies nearby entity data from game state into the map renderer.
func (g *Game) syncEntities() {
	g.client.Lock()
	defer g.client.Unlock()

	// Sync open door state for rendering
	if g.mapRenderer.OpenDoors == nil {
		g.mapRenderer.OpenDoors = make(map[[2]int]bool)
	}
	for k := range g.mapRenderer.OpenDoors {
		delete(g.mapRenderer.OpenDoors, k)
	}
	for _, d := range g.client.Doors {
		if d.Open {
			g.mapRenderer.OpenDoors[[2]int{d.X, d.Y}] = true
		}
	}

	g.mapRenderer.Characters = g.mapRenderer.Characters[:0]
	for _, ch := range g.client.NearbyChars {
		frame := characterFrame(ch.Direction, ch.Walking, ch.WalkFrame(), ch.Combat.AttackTick)

		g.mapRenderer.Characters = append(g.mapRenderer.Characters, render.CharacterEntity{
			PlayerID:   ch.PlayerID,
			Name:       ch.Name,
			X:          ch.X,
			Y:          ch.Y,
			Direction:  ch.Direction,
			Gender:     ch.Gender,
			Skin:       ch.Skin,
			HairStyle:  ch.HairStyle,
			HairColor:  ch.HairColor,
			Armor:      ch.Armor,
			Boots:      ch.Boots,
			Hat:        ch.Hat,
			Weapon:     ch.Weapon,
			Shield:     ch.Shield,
			Frame:      frame,
			Walking:    ch.Walking,
			WalkFrame:  ch.WalkFrame(),
			WalkProg:   ch.WalkProgress(),
			Mirrored:   ch.Direction == 2 || ch.Direction == 3, // Up or Right
			AttackProg: combatProgress(ch.Combat.AttackTick, game.AttackAnimationDuration),
			HitProg:    combatProgress(ch.Combat.HitTick, game.HitAnimationDuration),
			Indicators: syncIndicators(ch.Combat.Indicators),
		})
	}

	g.mapRenderer.Npcs = g.mapRenderer.Npcs[:0]
	for _, npc := range g.client.NearbyNpcs {
		if npc.Hidden {
			continue
		}
		// During walk: render from origin, offset toward destination
		// When idle: render at current position (X,Y = destination)
		rx, ry := npc.X, npc.Y
		dx, dy := npc.X, npc.Y
		if npc.Walking {
			rx, ry = npc.WalkFromX, npc.WalkFromY
		}
		g.mapRenderer.Npcs = append(g.mapRenderer.Npcs, render.NpcEntity{
			Index:      npc.Index,
			ID:         npc.ID,
			GraphicID:  g.npcGraphicID(npc.ID),
			X:          rx,
			Y:          ry,
			DestX:      dx,
			DestY:      dy,
			Direction:  npc.Direction,
			IdleFrame:  npc.IdleFrame(),
			Walking:    npc.Walking,
			WalkFrame:  npc.WalkFrame(),
			WalkProg:   npc.WalkProgress(),
			AttackProg: combatProgress(npc.Combat.AttackTick, game.AttackAnimationDuration),
			HitProg:    combatProgress(npc.Combat.HitTick, game.HitAnimationDuration),
			Dead:       npc.Dead,
			DeathProg:  npc.DeathProgress(),
			Indicators: syncIndicators(npc.Combat.Indicators),
		})
	}

	g.mapRenderer.Items = g.mapRenderer.Items[:0]
	for _, item := range g.client.NearbyItems {
		g.mapRenderer.Items = append(g.mapRenderer.Items, render.ItemEntity{
			UID:       item.UID,
			GraphicID: g.groundItemGraphicID(item),
			X:         item.X,
			Y:         item.Y,
		})
	}
}

func (g *Game) groundItemGraphicID(item game.NearbyItem) int {
	if g.itemDB != nil {
		if graphicID := g.itemDB.GraphicResourceID(item.ID, item.Amount); graphicID > 0 {
			return graphicID
		}
	}
	return item.GraphicID
}

func (g *Game) npcGraphicID(id int) int {
	if g.npcDB != nil {
		if graphicID := g.npcDB.GraphicID(id); graphicID > 0 {
			return graphicID
		}
	}
	return id
}

// characterFrame picks the correct animation frame based on direction and walk state.
// Directions: 0=Down, 1=Left, 2=Up, 3=Right
// Down(0) and Right(3) use the down-right sprite. Left(1) and Up(2) use the up-left sprite.
func syncIndicators(indicators []game.CombatIndicator) []render.CombatIndicator {
	if len(indicators) == 0 {
		return nil
	}

	result := make([]render.CombatIndicator, 0, len(indicators))
	for _, indicator := range indicators {
		result = append(result, render.CombatIndicator{
			Text:     indicator.Text,
			Kind:     int(indicator.Kind),
			Progress: indicator.Progress(),
		})
	}
	return result
}

func combatProgress(ticks, maxTicks int) float64 {
	if maxTicks <= 0 || ticks <= 0 {
		return 0
	}
	progress := 1.0 - float64(ticks)/float64(maxTicks)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func characterFrame(dir int, walking bool, walkFrame int, attackTick int) render.CharacterFrame {
	upLeft := dir == 1 || dir == 2
	if attackTick > 0 {
		if upLeft {
			if attackTick > game.AttackAnimationDuration/2 {
				return render.FrameMeleeUp1
			}
			return render.FrameMeleeUp2
		}
		if attackTick > game.AttackAnimationDuration/2 {
			return render.FrameMeleeDown1
		}
		return render.FrameMeleeDown2
	}

	if walking {
		base := render.FrameWalkDown1
		if upLeft {
			base = render.FrameWalkUp1
		}
		return base + render.CharacterFrame(walkFrame)
	}

	if upLeft {
		return render.FrameStandUp
	}
	return render.FrameStandDown
}
