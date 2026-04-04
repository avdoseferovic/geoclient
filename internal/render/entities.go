package render

import (
	"cmp"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

type entityCmd struct {
	x, y  int     // iso coords
	depth float64 // for sorting
	kind  int     // 0=item, 1=npc, 2=character, 3=cursor
	index int     // index into the entity slice
}

// buildEntityDrawCmds collects renderable entities with the same depth model as the reference client.
func (r *MapRenderer) buildEntityDrawCmds(screen *ebiten.Image, camSX, camSY, halfW, halfH float64) []entityCmd {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	ents := make([]entityCmd, 0, len(r.Items)+len(r.Npcs)+len(r.Characters)+1)

	for i := range r.Items {
		it := &r.Items[i]
		ents = append(ents, entityCmd{
			x: it.X, y: it.Y,
			depth: entityDepth(it.X, it.Y, 0),
			kind:  0, index: i,
		})
	}

	for i := range r.Npcs {
		npc := &r.Npcs[i]
		ents = append(ents, entityCmd{
			x: npc.X, y: npc.Y,
			depth: entityDepth(npc.X, npc.Y, 1),
			kind:  1, index: i,
		})
	}

	for i := range r.Characters {
		ch := &r.Characters[i]
		ents = append(ents, entityCmd{
			x: ch.X, y: ch.Y,
			depth: characterEntityDepth(ch),
			kind:  2, index: i,
		})
	}

	if r.Cursor != nil {
		ents = append(ents, entityCmd{
			x: r.Cursor.X, y: r.Cursor.Y,
			depth: entityDepth(r.Cursor.X, r.Cursor.Y, 3),
			kind:  3,
		})
	}
	filtered := ents[:0]

	for _, e := range ents {
		sx, sy := IsoToScreen(float64(e.x), float64(e.y))
		sx = sx - camSX + halfW
		sy = sy - camSY + halfH

		// Skip off-screen entities with generous bounds.
		if sx < -200 || sx > sw+200 || sy < -200 || sy > sh+200 {
			continue
		}
		filtered = append(filtered, e)
	}

	slices.SortFunc(filtered, func(a, b entityCmd) int {
		return cmp.Compare(a.depth, b.depth)
	})

	return filtered
}

func (r *MapRenderer) drawEntityCmd(screen *ebiten.Image, cmd entityCmd, camSX, camSY, halfW, halfH float64) {
	sx, sy := IsoToScreen(float64(cmd.x), float64(cmd.y))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	switch cmd.kind {
	case 0:
		RenderItem(screen, r.Loader, &r.Items[cmd.index], sx, sy)
	case 1:
		RenderNPC(screen, r.Loader, &r.Npcs[cmd.index], sx, sy)
	case 2:
		RenderCharacter(screen, r.Loader, &r.Characters[cmd.index], sx, sy)
	case 3:
		RenderCursor(screen, r.Loader, r.Cursor, sx, sy)
	}
}

func entityDepth(x, y, kind int) float64 {
	baseDepth := entityBaseDepth(kind)
	row := float64(y)*rdg + float64(x)*float64(worldDepthLayers)*tdg
	return baseDepth + row
}

func entityBaseDepth(kind int) float64 {
	switch kind {
	case 0: // items
		return 0.0 + tdg*3
	case 1: // npcs
		return 0.0 + tdg*6
	case 2: // characters
		return 0.0 + tdg*5
	case 3: // cursor
		return 0.0 + tdg*2
	default:
		return 0.0
	}
}

func characterEntityDepth(ch *CharacterEntity) float64 {
	if ch == nil || !ch.Walking {
		return entityDepth(ch.X, ch.Y, 2)
	}

	ox, oy, dx, dy := walkVector(ch.Direction)
	if dx == 0 && dy == 0 {
		return entityDepth(ch.X, ch.Y, 2)
	}

	fx := float64(ch.X+ox) + float64(dx)*ch.WalkProg
	fy := float64(ch.Y+oy) + float64(dy)*ch.WalkProg
	return entityBaseDepth(2) + fy*rdg + fx*float64(worldDepthLayers)*tdg
}

func walkVector(dir int) (originDX, originDY, moveDX, moveDY int) {
	switch dir {
	case 0: // Down: destination is one tile below origin.
		return 0, -1, 0, 1
	case 1: // Left
		return 1, 0, -1, 0
	case 2: // Up
		return 0, 1, 0, -1
	case 3: // Right
		return -1, 0, 1, 0
	default:
		return 0, 0, 0, 0
	}
}
