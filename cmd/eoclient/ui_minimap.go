package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/hud"
)

func (g *Game) drawMinimap(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	if g.mapRenderer.Map == nil {
		clientui.DrawTextWrappedCentered(screen, "Map data unavailable.", rect, theme.TextDim)
		return
	}
	cellSize := 4
	cols := max(9, (rect.Dx()-8)/cellSize)
	rows := max(9, (rect.Dy()-8)/cellSize)
	originX := rect.Min.X + (rect.Dx()-cols*cellSize)/2
	originY := rect.Min.Y + (rect.Dy()-rows*cellSize)/2
	centerX, centerY := g.minimapCenter(snapshot)
	halfCols := cols / 2
	halfRows := rows / 2
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			tileX := int(centerX) + col - halfCols
			tileY := int(centerY) + row - halfRows
			fill := hud.MinimapTileColor(g.tileStateAt(tileX, tileY), theme)
			clientui.FillRect(screen, float64(originX+col*cellSize), float64(originY+row*cellSize), float64(cellSize-1), float64(cellSize-1), fill)
		}
	}
	for _, item := range snapshot.NearbyItems {
		g.drawMinimapMarker(screen, originX, originY, cols, rows, cellSize, halfCols, halfRows, centerX, centerY, float64(item.X), float64(item.Y), color.NRGBA{R: 236, G: 192, B: 74, A: 255})
	}
	for _, npc := range snapshot.NearbyNpcs {
		if npc.Hidden || npc.Dead {
			continue
		}
		g.drawMinimapMarker(screen, originX, originY, cols, rows, cellSize, halfCols, halfRows, centerX, centerY, float64(npc.X), float64(npc.Y), color.NRGBA{R: 195, G: 84, B: 74, A: 255})
	}
	for _, ch := range snapshot.NearbyChars {
		if ch.PlayerID == snapshot.PlayerID {
			continue
		}
		g.drawMinimapMarker(screen, originX, originY, cols, rows, cellSize, halfCols, halfRows, centerX, centerY, float64(ch.X), float64(ch.Y), color.NRGBA{R: 116, G: 192, B: 255, A: 255})
	}
	playerPX := originX + halfCols*cellSize
	playerPY := originY + halfRows*cellSize
	clientui.FillRect(screen, float64(playerPX-1), float64(playerPY-1), float64(cellSize+1), float64(cellSize+1), theme.Accent)
}

func (g *Game) drawMinimapMarker(screen *ebiten.Image, originX, originY, cols, rows, cellSize, halfCols, halfRows int, centerX, centerY, worldX, worldY float64, fill color.Color) {
	col := int(worldX - centerX + float64(halfCols))
	row := int(worldY - centerY + float64(halfRows))
	if col < 0 || row < 0 || col >= cols || row >= rows {
		return
	}
	px := originX + col*cellSize + 1
	py := originY + row*cellSize + 1
	clientui.FillRect(screen, float64(px), float64(py), float64(max(1, cellSize-2)), float64(max(1, cellSize-2)), fill)
}

func (g *Game) minimapCenter(snapshot game.UISnapshot) (float64, float64) {
	camSX, camSY := g.currentCameraScreenPosition(snapshot)
	return (camSX/float64(render.HalfTileW) + camSY/float64(render.HalfTileH)) / 2,
		(camSY/float64(render.HalfTileH) - camSX/float64(render.HalfTileW)) / 2
}

func (g *Game) tileStateAt(tileX, tileY int) int {
	if g.mapRenderer.Map == nil || tileX < 0 || tileY < 0 || tileX > g.mapRenderer.Map.Width || tileY > g.mapRenderer.Map.Height {
		return 2
	}
	for _, row := range g.mapRenderer.Map.TileSpecRows {
		if row.Y != tileY {
			continue
		}
		for _, tile := range row.Tiles {
			if tile.X != tileX {
				continue
			}
			switch tile.TileSpec {
			case 0:
				return 0
			case 1, 16, 29:
				return 2
			default:
				return 1
			}
		}
	}
	return 0
}
