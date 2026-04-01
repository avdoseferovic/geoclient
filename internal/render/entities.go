package render

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

type entityCmd struct {
	x, y  int     // iso coords
	depth float64 // for sorting
	kind  int     // 0=item, 1=npc, 2=character
	index int     // index into the entity slice
}

// drawEntities renders all characters, NPCs, and items with correct isometric depth sorting.
func (r *MapRenderer) drawEntities(screen *ebiten.Image, camSX, camSY, halfW, halfH float64) {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	var ents []entityCmd

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
			depth: entityDepth(ch.X, ch.Y, 2),
			kind:  2, index: i,
		})
	}

	sort.Slice(ents, func(i, j int) bool {
		return ents[i].depth < ents[j].depth
	})

	for _, e := range ents {
		sx, sy := IsoToScreen(float64(e.x), float64(e.y))
		sx = sx - camSX + halfW
		sy = sy - camSY + halfH

		// Skip off-screen entities (generous bounds)
		if sx < -200 || sx > sw+200 || sy < -200 || sy > sh+200 {
			continue
		}

		switch e.kind {
		case 0:
			RenderItem(screen, r.Loader, &r.Items[e.index], sx, sy)
		case 1:
			RenderNPC(screen, r.Loader, &r.Npcs[e.index], sx, sy)
		case 2:
			RenderCharacter(screen, r.Loader, &r.Characters[e.index], sx, sy)
		}
	}
}

// entityDepth calculates depth for an entity at (x,y).
// kind offsets: items < npcs < characters for same tile.
func entityDepth(x, y, kind int) float64 {
	row := float64(x + y)
	return row*rdg + float64(kind)*tdg*3
}
