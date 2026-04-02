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
	GfxMapObjects = 4
)

// GFX file IDs for each map layer (9 layers).
var layerGfxIDs = [9]int{3, GfxMapObjects, 5, 6, 6, 7, 3, 22, 5}

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

type mapDrawState struct {
	camSX        float64
	camSY        float64
	halfW        float64
	halfH        float64
	underlayCmds []drawCmd
	overlayCmds  []drawCmd
}

// Draw renders the map centered on (CamX, CamY).
func (r *MapRenderer) Draw(screen *ebiten.Image) {
	r.DrawWithMid(screen, nil)
}

func (r *MapRenderer) DrawWithMid(screen *ebiten.Image, mid func()) {
	if r.Map == nil {
		return
	}

	state := r.buildDrawState(screen)

	// Draw base layers first, then entities, then object/top layers over them.
	drawCmds(screen, state.underlayCmds)
	if mid != nil {
		mid()
	}

	// Draw entities on top of ground/shadow layers, sorted by Y for depth
	r.drawEntities(screen, state.camSX, state.camSY, state.halfW, state.halfH)
	drawCmds(screen, state.overlayCmds)
}

func shouldDrawTileAboveEntities(layerIdx, imgHeight int) bool {
	// Keep normal objects/walls under entities.
	// Only very tall object-layer sprites, such as trees, should cover them.
	return layerIdx == 1 && imgHeight >= 100
}

func drawCmds(screen *ebiten.Image, cmds []drawCmd) {
	for _, cmd := range cmds {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(cmd.sx, cmd.sy)
		if cmd.alpha > 0 {
			op.ColorScale.ScaleAlpha(cmd.alpha)
		}
		screen.DrawImage(cmd.img, op)
	}
}

func (r *MapRenderer) buildDrawState(screen *ebiten.Image) mapDrawState {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	state := mapDrawState{
		halfW: float64(sw) / 2,
		halfH: float64(sh) / 2,
	}

	state.camSX, state.camSY = IsoToScreen(r.CamX, r.CamY)
	state.camSX += r.CamOffX
	state.camSY += r.CamOffY

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
				sx = sx - state.camSX + state.halfW
				sy = sy - state.camSY + state.halfH

				imgW := float64(img.Bounds().Dx())
				imgH := float64(img.Bounds().Dy())
				sx -= imgW / 2
				sy -= imgH - HalfTileH

				if sx+imgW < 0 || sx > float64(sw) || sy+imgH < 0 || sy > float64(sh) {
					continue
				}

				depth := layerDepth(layerIdx, x, y)
				var alpha float32
				if layerIdx == 7 {
					alpha = 0.2
				}
				target := &state.underlayCmds
				if shouldDrawTileAboveEntities(layerIdx, int(imgH)) {
					target = &state.overlayCmds
				}
				*target = append(*target, drawCmd{img: img, sx: sx, sy: sy, depth: depth, alpha: alpha})
			}
		}

		if layerIdx == 0 && r.Map.FillTile > 0 {
			fillImg, err := r.Loader.GetImage(gfxFileID, r.Map.FillTile)
			if err == nil && fillImg != nil {
				r.drawFillTiles(screen, fillImg, state.camSX, state.camSY, state.halfW, state.halfH, &state.underlayCmds)
			}
		}
	}

	sort.Slice(state.underlayCmds, func(i, j int) bool {
		return state.underlayCmds[i].depth < state.underlayCmds[j].depth
	})
	sort.Slice(state.overlayCmds, func(i, j int) bool {
		return state.overlayCmds[i].depth < state.overlayCmds[j].depth
	})

	return state
}

func (r *MapRenderer) drawFillTiles(screen *ebiten.Image, fillImg *ebiten.Image, camSX, camSY, halfW, halfH float64, cmds *[]drawCmd) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	imgW := float64(fillImg.Bounds().Dx())
	imgH := float64(fillImg.Bounds().Dy())

	for y := 0; y <= r.Map.Height; y++ {
		for x := 0; x <= r.Map.Width; x++ {
			// Skip tiles that have explicit ground graphics
			if r.HasTileAt(0, x, y) {
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

func (r *MapRenderer) HasTileAt(layerIdx, x, y int) bool {
	if r == nil || r.Map == nil {
		return false
	}
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
