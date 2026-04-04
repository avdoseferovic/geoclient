package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

func (g *Game) drawItemTooltip(screen *ebiten.Image, theme clientui.Theme, mx, my, itemID, amount int) {
	if g.itemDB == nil || itemID <= 0 {
		return
	}
	item, ok := g.itemDB.Get(itemID)
	if !ok {
		return
	}
	lines := []string{item.Name}
	if amount > 1 {
		lines[0] = fmt.Sprintf("%s x%d", item.Name, amount)
	}
	lines = append(lines, g.itemDB.MetaLines(itemID)...)
	maxWidth := 0
	for _, line := range lines {
		if w := clientui.MeasureText(line); w > maxWidth {
			maxWidth = w
		}
	}
	rect := image.Rect(mx+12, my+12, mx+maxWidth+28, my+18+len(lines)*14)
	if rect.Max.X > g.screenW-8 {
		rect = rect.Add(image.Pt(-(rect.Dx() + 24), 0))
	}
	if rect.Max.Y > g.screenH-8 {
		rect = rect.Add(image.Pt(0, -(rect.Dy() + 24)))
	}
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Accent: theme.AccentMuted, Fill: overlay.Colorize(theme.PanelFill, 248)})
	for i, line := range lines {
		lineColor := theme.TextDim
		if i == 0 {
			lineColor = theme.Text
		}
		clientui.DrawText(screen, line, rect.Min.X+8, rect.Min.Y+14+i*14, lineColor)
	}
}

func (g *Game) drawItemSlot(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, label string, itemID int) {
	clientui.DrawInset(screen, rect, theme, true)
	contentRect := rect
	if label != "" && rect.Dy() >= 34 {
		clientui.DrawTextCentered(screen, label, image.Rect(rect.Min.X+1, rect.Max.Y-12, rect.Max.X-1, rect.Max.Y-1), theme.TextDim)
		contentRect = image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y-12)
	}
	if itemID == 0 {
		return
	}
	g.drawItemIcon(screen, image.Rect(contentRect.Min.X+3, contentRect.Min.Y+3, contentRect.Max.X-3, contentRect.Max.Y-3), itemID, 1)
}

func (g *Game) drawItemIcon(screen *ebiten.Image, rect image.Rectangle, itemID, amount int) {
	_ = amount
	if g.itemDB == nil || itemID <= 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	resourceID := g.itemDB.GridGraphicResourceID(itemID)
	if resourceID <= 0 {
		return
	}
	img, err := g.gfxLoad.GetImage(render.GfxItems, resourceID)
	if err != nil || img == nil {
		return
	}
	bounds := img.Bounds()
	iw, ih := bounds.Dx(), bounds.Dy()
	if iw <= 0 || ih <= 0 {
		return
	}
	scale := min(float64(rect.Dx())/float64(iw), float64(rect.Dy())/float64(ih))
	if scale > 1 {
		scale = 1
	}
	drawW := float64(iw) * scale
	drawH := float64(ih) * scale
	x := float64(rect.Min.X) + (float64(rect.Dx())-drawW)/2
	y := float64(rect.Min.Y) + (float64(rect.Dy())-drawH)/2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}
