package render

import (
	"math"
	"os"
	"sort"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eomap "github.com/ethanmoffat/eolib-go/v3/protocol/map"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/gfx"
)

const (
	TileWidth     = 64
	TileHeight    = 32
	HalfTileW     = TileWidth / 2  // 32
	HalfTileH     = TileHeight / 2 // 16
	HalfHalfTileH = HalfTileH / 2  // 8
)

// GFX file IDs for each map layer (9 layers).
var layerGfxIDs = [9]int{3, 4, 5, 6, 6, 7, 3, 22, 5}

// MapRenderer handles isometric map rendering.
type MapRenderer struct {
	Map    *eomap.Emf
	Loader *gfx.Loader

	// Camera position in isometric coordinates
	CamX, CamY float64

	// Screen-space camera offset (for smooth walk animation)
	CamOffX, CamOffY float64

	// Entities to render
	Characters []CharacterEntity
	Npcs       []NpcEntity
	Items      []ItemEntity
}

func NewMapRenderer(loader *gfx.Loader) *MapRenderer {
	return &MapRenderer{Loader: loader}
}

// LoadMap reads and parses an EMF file.
func (r *MapRenderer) LoadMap(path string) error {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	reader := data.NewEoReader(fileData)
	emf := &eomap.Emf{}
	if err := emf.Deserialize(reader); err != nil {
		return err
	}
	r.Map = emf
	return nil
}

// SetMap sets a pre-loaded map.
func (r *MapRenderer) SetMap(emf *eomap.Emf) {
	r.Map = emf
}

// IsoToScreen converts isometric tile coordinates to screen pixel coordinates.
func IsoToScreen(ix, iy float64) (float64, float64) {
	sx := (ix - iy) * HalfTileW
	sy := (ix + iy) * HalfTileH
	return sx, sy
}

// ScreenToIso converts screen pixel coordinates to isometric tile coordinates (floored to int).
func ScreenToIso(sx, sy float64) (int, int) {
	ix := math.Floor((sx/HalfTileW + sy/HalfTileH) / 2)
	iy := math.Floor((sy/HalfTileH - sx/HalfTileW) / 2)
	return int(ix), int(iy)
}

type drawCmd struct {
	img   *ebiten.Image
	sx    float64
	sy    float64
	depth float64
	alpha float32 // 0 = use default (1.0)
}

// Draw renders the map centered on (CamX, CamY).
func (r *MapRenderer) Draw(screen *ebiten.Image) {
	if r.Map == nil {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	halfW, halfH := float64(sw)/2, float64(sh)/2

	// Camera screen offset (includes walk animation smoothing)
	camSX, camSY := IsoToScreen(r.CamX, r.CamY)
	camSX += r.CamOffX
	camSY += r.CamOffY

	var cmds []drawCmd

	// Draw each of the 9 graphic layers
	for layerIdx := 0; layerIdx < 9 && layerIdx < len(r.Map.GraphicLayers); layerIdx++ {
		layer := r.Map.GraphicLayers[layerIdx]
		gfxFileID := layerGfxIDs[layerIdx]

		for _, row := range layer.GraphicRows {
			y := row.Y
			for _, tile := range row.Tiles {
				x := tile.X
				graphicID := tile.Graphic
				if graphicID == 0 {
					continue
				}

				img, err := r.Loader.GetImage(gfxFileID, graphicID)
				if err != nil || img == nil {
					continue
				}

				sx, sy := IsoToScreen(float64(x), float64(y))
				// Offset from camera
				sx = sx - camSX + halfW
				sy = sy - camSY + halfH

				// Adjust for tile anchor (center-bottom of tile)
				imgW := float64(img.Bounds().Dx())
				imgH := float64(img.Bounds().Dy())
				sx -= imgW / 2
				sy -= imgH - HalfTileH

				// Skip off-screen tiles
				if sx+imgW < 0 || sx > float64(sw) || sy+imgH < 0 || sy > float64(sh) {
					continue
				}

				depth := layerDepth(layerIdx, x, y)
				var alpha float32
				if layerIdx == 7 { // Shadow layer
					alpha = 0.2
				}
				cmds = append(cmds, drawCmd{img: img, sx: sx, sy: sy, depth: depth, alpha: alpha})
			}
		}

		// Fill tile for ground layer
		if layerIdx == 0 && r.Map.FillTile > 0 {
			fillImg, err := r.Loader.GetImage(gfxFileID, r.Map.FillTile)
			if err == nil && fillImg != nil {
				r.drawFillTiles(screen, fillImg, camSX, camSY, halfW, halfH, &cmds)
			}
		}
	}

	// Sort by depth for correct draw order
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].depth < cmds[j].depth
	})

	// Draw all tiles
	for _, cmd := range cmds {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(cmd.sx, cmd.sy)
		if cmd.alpha > 0 {
			op.ColorScale.ScaleAlpha(cmd.alpha)
		}
		screen.DrawImage(cmd.img, op)
	}

	// Draw entities on top of ground/shadow layers, sorted by Y for depth
	r.drawEntities(screen, camSX, camSY, halfW, halfH)
}

func (r *MapRenderer) drawFillTiles(screen *ebiten.Image, fillImg *ebiten.Image, camSX, camSY, halfW, halfH float64, cmds *[]drawCmd) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	imgW := float64(fillImg.Bounds().Dx())
	imgH := float64(fillImg.Bounds().Dy())

	for y := 0; y <= r.Map.Height; y++ {
		for x := 0; x <= r.Map.Width; x++ {
			// Skip tiles that have explicit ground graphics
			if r.hasTileAt(0, x, y) {
				continue
			}

			sx, sy := IsoToScreen(float64(x), float64(y))
			sx = sx - camSX + halfW - imgW/2
			sy = sy - camSY + halfH - imgH + HalfTileH

			if sx+imgW < 0 || sx > float64(sw) || sy+imgH < 0 || sy > float64(sh) {
				continue
			}

			*cmds = append(*cmds, drawCmd{img: fillImg, sx: sx, sy: sy, depth: layerDepth(0, x, y)})
		}
	}
}

func (r *MapRenderer) hasTileAt(layerIdx, x, y int) bool {
	if layerIdx >= len(r.Map.GraphicLayers) {
		return false
	}
	layer := r.Map.GraphicLayers[layerIdx]
	for _, row := range layer.GraphicRows {
		if row.Y != y {
			continue
		}
		for _, tile := range row.Tiles {
			if tile.X == x && tile.Graphic != 0 {
				return true
			}
		}
	}
	return false
}

const (
	tdg = 0.00000001 // gap between tiles on same layer
	rdg = 0.001      // gap between rows
)

func layerDepth(layer, x, y int) float64 {
	var baseDepth float64
	switch layer {
	case 0: // Ground
		baseDepth = -3.0
	case 1: // Objects
		baseDepth = 0.0
	case 2: // Overlay
		baseDepth = 0.0
	case 3: // Down Wall
		baseDepth = 0.0
	case 4: // Right Wall
		baseDepth = -rdg
	case 5: // Roof
		baseDepth = 0.0
	case 6: // Top
		baseDepth = 0.0
	case 7: // Shadow
		baseDepth = -1.0
	case 8: // Overlay2
		baseDepth = 1.0
	}

	row := float64(x+y) * rdg
	return baseDepth + row + float64(layer)*tdg
}
