package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

type worldHoverKind int

const (
	worldHoverNone worldHoverKind = iota
	worldHoverCharacter
	worldHoverNPC
	worldHoverItem
)

type worldHoverTarget struct {
	Kind   worldHoverKind
	Title  string
	Lines  []string
	ItemID int
	Amount int
	Anchor image.Point
	Depth  int
}

func (g *Game) drawWorldHoverTooltip(screen *ebiten.Image, theme clientui.Theme, snapshot game.UISnapshot) {
	target, ok := g.currentWorldHoverTarget(snapshot)
	if !ok {
		return
	}

	mx, my := ebiten.CursorPosition()
	if target.Kind == worldHoverItem {
		g.drawItemTooltip(screen, theme, mx, my, target.ItemID, target.Amount)
		return
	}
	g.drawWorldNameplate(screen, theme, target)
}

func (g *Game) currentWorldHoverTarget(snapshot game.UISnapshot) (worldHoverTarget, bool) {
	mx, my := ebiten.CursorPosition()
	if g.worldHoverBlockedByHUD(mx, my) {
		return worldHoverTarget{}, false
	}
	intent := g.currentWorldHoverIntent()

	camSX, camSY := g.currentCameraScreenPosition(snapshot)
	halfW := float64(g.screenW) / 2
	halfH := float64(g.screenH) / 2

	best := worldHoverTarget{}
	hit := false

	for _, ch := range g.mapRenderer.Characters {
		rect := characterHoverRect(ch, camSX, camSY, halfW, halfH)
		if !overlay.PointInRect(mx, my, rect) {
			continue
		}

		target := worldHoverTarget{
			Kind:   worldHoverCharacter,
			Title:  overlay.FallbackString(strings.TrimSpace(ch.Name), "Unknown adventurer"),
			Anchor: image.Pt((rect.Min.X+rect.Max.X)/2, rect.Min.Y),
			Depth:  rect.Max.Y,
		}
		if !hit || target.Depth >= best.Depth {
			best = target
			hit = true
		}
	}

	for _, npc := range g.mapRenderer.Npcs {
		rect := g.npcHoverRect(npc, camSX, camSY, halfW, halfH)
		if !overlay.PointInRect(mx, my, rect) {
			continue
		}

		name := strings.TrimSpace(g.npcLabel(npc.ID))
		if name == "" {
			name = fmt.Sprintf("NPC %03d", npc.ID)
		}
		target := worldHoverTarget{
			Kind:   worldHoverNPC,
			Title:  name,
			Anchor: image.Pt((rect.Min.X+rect.Max.X)/2, rect.Min.Y),
			Depth:  rect.Max.Y,
		}
		if !hit || target.Depth >= best.Depth {
			best = target
			hit = true
		}
	}

	for _, item := range g.mapRenderer.Items {
		if !groundItemHoverAllowed(intent, item.X, item.Y) {
			continue
		}
		rect := g.itemHoverRect(item, camSX, camSY, halfW, halfH)
		if !overlay.PointInRect(mx, my, rect) {
			continue
		}
		itemID, amount := groundItemHoverData(snapshot, item.UID)
		if itemID <= 0 {
			continue
		}
		target := worldHoverTarget{
			Kind:   worldHoverItem,
			ItemID: itemID,
			Amount: amount,
			Depth:  rect.Max.Y,
		}
		if !hit || target.Depth >= best.Depth {
			best = target
			hit = true
		}
	}

	return best, hit
}

func groundItemHoverAllowed(intent worldHoverIntent, itemX, itemY int) bool {
	if !intent.Valid || intent.CursorType < 0 {
		return false
	}
	if intent.TileX != itemX || intent.TileY != itemY {
		return false
	}
	return intent.CursorType == 2
}

func (g *Game) drawWorldNameplate(screen *ebiten.Image, theme clientui.Theme, target worldHoverTarget) {
	title := strings.TrimSpace(target.Title)
	if title == "" {
		return
	}

	textW := clientui.MeasureText(title)
	x := target.Anchor.X - textW/2
	y := target.Anchor.Y - 2
	if x < 8 {
		x = 8
	}
	if x+textW > g.screenW-8 {
		x = g.screenW - 8 - textW
	}
	if y < 12 {
		y = 12
	}
	plateRect := image.Rect(x-6, y-12, x+textW+6, y+4)
	clientui.FillRect(screen, float64(plateRect.Min.X), float64(plateRect.Min.Y), float64(plateRect.Dx()), float64(plateRect.Dy()), color.NRGBA{A: 140})
	clientui.DrawText(screen, title, x+1, y+1, color.NRGBA{A: 180})
	clientui.DrawText(screen, title, x, y, theme.Text)
}

func (g *Game) worldHoverBlockedByHUD(mx, my int) bool {
	layout := overlay.InGameHUDLayout(g.screenW, g.screenH)
	chatPanelRect, _ := g.chatRects()
	if overlay.PointInRect(mx, my, chatPanelRect) ||
		overlay.PointInRect(mx, my, layout.StatusRect) ||
		overlay.PointInRect(mx, my, layout.MenuRect) ||
		overlay.PointInRect(mx, my, layout.InfoRect) {
		return true
	}
	return g.overlay.activeMenuPanel != overlay.MenuPanelNone && overlay.PointInRect(mx, my, layout.MenuPanelRect)
}

func characterHoverRect(ch render.CharacterEntity, camSX, camSY, halfW, halfH float64) image.Rectangle {
	sx, sy := render.IsoToScreen(float64(ch.X), float64(ch.Y))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	if ch.Walking {
		wox, woy := render.WalkOffset(ch.Direction, ch.WalkProg)
		sx += wox
		sy += woy
	}
	if ch.AttackProg > 0 {
		aox, aoy := hoverAttackOffset(ch.Direction, ch.AttackProg)
		sx += aox
		sy += aoy
	}

	width, height := characterFrameDimensions(ch.Frame)
	drawX, drawY := render.CharacterDrawPosition(ch.Gender, ch.Frame, ch.Direction, sx, sy, width, height)
	return image.Rect(
		int(drawX)-12,
		int(drawY)-14,
		int(drawX)+width+12,
		int(drawY)+height+8,
	)
}

func (g *Game) npcHoverRect(npc render.NpcEntity, camSX, camSY, halfW, halfH float64) image.Rectangle {
	sx, sy := render.IsoToScreen(float64(npc.X), float64(npc.Y))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	if npc.Walking && (npc.DestX != npc.X || npc.DestY != npc.Y) {
		destSX, destSY := render.IsoToScreen(float64(npc.DestX), float64(npc.DestY))
		origSX, origSY := render.IsoToScreen(float64(npc.X), float64(npc.Y))
		sx += (destSX - origSX) * npc.WalkProg
		sy += (destSY - origSY) * npc.WalkProg
	}
	if npc.AttackProg > 0 {
		aox, aoy := hoverAttackOffset(npc.Direction, npc.AttackProg)
		sx += aox
		sy += aoy
	}

	width := 40
	height := 64
	baseID := (npc.GraphicID-1)*40 + 1
	if npc.Direction == 1 || npc.Direction == 2 {
		baseID = (npc.GraphicID-1)*40 + 3
	}
	if img, err := g.gfxLoad.GetImage(render.GfxNPC, baseID); err == nil && img != nil {
		width = img.Bounds().Dx()
		height = img.Bounds().Dy()
	}

	mirrored := npc.Direction == 2 || npc.Direction == 3
	drawX, drawY := render.NPCDrawPosition(npc.GraphicID, mirrored, sx, sy, float64(width), float64(height))
	return image.Rect(
		int(drawX)-10,
		int(drawY)-12,
		int(drawX)+width+10,
		int(drawY)+height+6,
	)
}

func (g *Game) itemHoverRect(item render.ItemEntity, camSX, camSY, halfW, halfH float64) image.Rectangle {
	sx, sy := render.IsoToScreen(float64(item.X), float64(item.Y))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	width := 28
	height := 28
	if img, err := g.gfxLoad.GetImage(render.GfxItems, item.GraphicID); err == nil && img != nil {
		width = img.Bounds().Dx()
		height = img.Bounds().Dy()
	}

	drawX := sx - float64(width)/2
	drawY := sy - float64(height)/2
	return image.Rect(
		int(drawX)-4,
		int(drawY)-4,
		int(drawX)+width+4,
		int(drawY)+height+4,
	)
}

func groundItemHoverData(snapshot game.UISnapshot, uid int) (int, int) {
	for _, item := range snapshot.NearbyItems {
		if item.UID == uid {
			return item.ID, item.Amount
		}
	}
	return 0, 0
}

func (g *Game) npcLabel(id int) string {
	if g.npcDB == nil {
		return ""
	}
	return g.npcDB.Name(id)
}

func characterFrameDimensions(frame render.CharacterFrame) (int, int) {
	switch frame {
	case render.FrameWalkDown1, render.FrameWalkDown2, render.FrameWalkDown3, render.FrameWalkDown4,
		render.FrameWalkUp1, render.FrameWalkUp2, render.FrameWalkUp3, render.FrameWalkUp4:
		return render.CharWalkWidth, render.CharWalkHeight
	case render.FrameMeleeDown1, render.FrameMeleeDown2, render.FrameMeleeUp1, render.FrameMeleeUp2:
		return render.CharMeleeWidth, render.CharHeight
	case render.FrameRaisedDown, render.FrameRaisedUp:
		return render.CharWidth, render.CharRaisedHeight
	case render.FrameChairDown, render.FrameChairUp:
		return render.CharSitChairW, render.CharSitChairH
	case render.FrameFloorDown, render.FrameFloorUp:
		return render.CharSitGroundW, render.CharSitGroundH
	case render.FrameRangeDown, render.FrameRangeUp:
		return render.CharRangeWidth, render.CharHeight
	default:
		return render.CharWidth, render.CharHeight
	}
}

func hoverAttackOffset(dir int, progress float64) (float64, float64) {
	distance := 3.0 * (1.0 - progress)
	switch dir {
	case 0:
		return -distance, distance / 2
	case 1:
		return -distance, -distance / 2
	case 2:
		return distance, -distance / 2
	case 3:
		return distance, distance / 2
	default:
		return 0, 0
	}
}
