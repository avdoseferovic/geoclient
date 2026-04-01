package main

import "github.com/avdo/eoweb/internal/render"

// syncEntities copies nearby entity data from game state into the map renderer.
func (g *Game) syncEntities() {
	g.client.Lock()
	defer g.client.Unlock()

	g.mapRenderer.Characters = g.mapRenderer.Characters[:0]
	for _, ch := range g.client.NearbyChars {
		frame := characterFrame(ch.Direction, ch.Walking, ch.WalkFrame())

		g.mapRenderer.Characters = append(g.mapRenderer.Characters, render.CharacterEntity{
			PlayerID:  ch.PlayerID,
			Name:      ch.Name,
			X:         ch.X,
			Y:         ch.Y,
			Direction: ch.Direction,
			Gender:    ch.Gender,
			Skin:      ch.Skin,
			HairStyle: ch.HairStyle,
			HairColor: ch.HairColor,
			Armor:     ch.Armor,
			Boots:     ch.Boots,
			Hat:       ch.Hat,
			Weapon:    ch.Weapon,
			Shield:    ch.Shield,
			Frame:     frame,
			Walking:   ch.Walking,
			WalkFrame: ch.WalkFrame(),
			WalkProg:  ch.WalkProgress(),
			Mirrored:  ch.Direction == 2 || ch.Direction == 3, // Up or Right
		})
	}

	g.mapRenderer.Npcs = g.mapRenderer.Npcs[:0]
	for _, npc := range g.client.NearbyNpcs {
		// During walk: render from origin, offset toward destination
		// When idle: render at current position (X,Y = destination)
		rx, ry := npc.X, npc.Y
		dx, dy := npc.X, npc.Y
		if npc.Walking {
			rx, ry = npc.WalkFromX, npc.WalkFromY
		}
		g.mapRenderer.Npcs = append(g.mapRenderer.Npcs, render.NpcEntity{
			Index:     npc.Index,
			GraphicID: npc.ID,
			X:         rx,
			Y:         ry,
			DestX:     dx,
			DestY:     dy,
			Direction: npc.Direction,
			IdleFrame: npc.IdleFrame(),
			Walking:   npc.Walking,
			WalkProg:  npc.WalkProgress(),
		})
	}

	g.mapRenderer.Items = g.mapRenderer.Items[:0]
	for _, item := range g.client.NearbyItems {
		g.mapRenderer.Items = append(g.mapRenderer.Items, render.ItemEntity{
			UID:       item.UID,
			GraphicID: item.GraphicID,
			X:         item.X,
			Y:         item.Y,
		})
	}
}

// characterFrame picks the correct animation frame based on direction and walk state.
// Directions: 0=Down, 1=Left, 2=Up, 3=Right
// Down(0) and Right(3) use the down-right sprite. Left(1) and Up(2) use the up-left sprite.
func characterFrame(dir int, walking bool, walkFrame int) render.CharacterFrame {
	upLeft := dir == 1 || dir == 2

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
