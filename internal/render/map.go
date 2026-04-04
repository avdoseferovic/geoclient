package render

import (
	"cmp"
	"image"
	"math"
	"path/filepath"
	"slices"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eomap "github.com/ethanmoffat/eolib-go/v3/protocol/map"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/assets"
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
	Reader assets.Reader

	// Camera position in isometric coordinates
	CamX, CamY float64

	// Screen-space camera offset (for smooth walk animation)
	CamOffX, CamOffY float64

	// Entities to render
	Characters []CharacterEntity
	Npcs       []NpcEntity
	Items      []ItemEntity
	Cursor     *CursorEntity
}

func NewMapRenderer(loader *gfx.Loader) *MapRenderer {
	return NewMapRendererWithReader(loader, assets.NewOSReader())
}

func NewMapRendererWithReader(loader *gfx.Loader, reader assets.Reader) *MapRenderer {
	return &MapRenderer{Loader: loader, Reader: reader}
}

// LoadMap reads and parses an EMF file.
func (r *MapRenderer) LoadMap(path string) error {
	fileData, err := r.Reader.ReadFile(filepath.Clean(path))
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
}

type tileOffset struct {
	x float64
	y float64
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
	entities := r.buildEntityDrawCmds(screen, state.camSX, state.camSY, state.halfW, state.halfH)

	drawMergedWorld(screen, state.underlayCmds, entities, r, state.camSX, state.camSY, state.halfW, state.halfH)
	if mid != nil {
		mid()
	}
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

func drawMergedWorld(screen *ebiten.Image, tiles []drawCmd, entities []entityCmd, r *MapRenderer, camSX, camSY, halfW, halfH float64) {
	tileIndex := 0
	entityIndex := 0

	for tileIndex < len(tiles) || entityIndex < len(entities) {
		if entityIndex >= len(entities) || (tileIndex < len(tiles) && tiles[tileIndex].depth <= entities[entityIndex].depth) {
			drawCmds(screen, tiles[tileIndex:tileIndex+1])
			tileIndex++
			continue
		}
		r.drawEntityCmd(screen, entities[entityIndex], camSX, camSY, halfW, halfH)
		entityIndex++
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
				img = normalizeStaticTileImage(layerIdx, img)

				imgW := float64(img.Bounds().Dx())
				imgH := float64(img.Bounds().Dy())
				sx, sy := tileDrawPosition(layerIdx, x, y, imgW, imgH, state.camSX, state.camSY, state.halfW, state.halfH)

				if sx+imgW < 0 || sx > float64(sw) || sy+imgH < 0 || sy > float64(sh) {
					continue
				}

				depth := layerDepth(layerIdx, x, y)
				var alpha float32
				if layerIdx == 7 {
					alpha = 0.2
				}
				state.underlayCmds = append(state.underlayCmds, drawCmd{img: img, sx: sx, sy: sy, depth: depth, alpha: alpha})
			}
		}

		if layerIdx == 0 && r.Map.FillTile > 0 {
			fillImg, err := r.Loader.GetImage(gfxFileID, r.Map.FillTile)
			if err == nil && fillImg != nil {
				fillImg = normalizeStaticTileImage(0, fillImg)
				r.drawFillTiles(screen, fillImg, state.camSX, state.camSY, state.halfW, state.halfH, &state.underlayCmds)
			}
		}
	}

	slices.SortFunc(state.underlayCmds, func(a, b drawCmd) int {
		return cmp.Compare(a.depth, b.depth)
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

			sx, sy := tileDrawPosition(0, x, y, imgW, imgH, camSX, camSY, halfW, halfH)

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

func normalizeStaticTileImage(layer int, img *ebiten.Image) *ebiten.Image {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	if layer == 0 && w > TileWidth {
		return img.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Min.X+TileWidth, b.Min.Y+TileHeight)).(*ebiten.Image)
	}

	if (layer == 3 || layer == 4) && w > 120 {
		frameWidth := w / 4
		return img.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Min.X+frameWidth, b.Min.Y+h)).(*ebiten.Image)
	}

	return img
}

const (
	tdg              = 0.00000001 // gap between tiles on same layer
	rdg              = 0.001      // gap between rows
	worldDepthLayers = 13
)

var layerBaseDepths = [...]float64{
	-3.0 + tdg*1, // Ground
	0.0 + tdg*4,  // Objects
	0.0 + tdg*7,  // Overlay
	0.0 + tdg*8,  // Down Wall
	-rdg + tdg*9, // Right Wall
	0.0 + tdg*10, // Roof
	0.0 + tdg*1,  // Top
	-1.0 + tdg*1, // Shadow
	1.0 + tdg*1,  // Overlay2
}

func layerDepth(layer, x, y int) float64 {
	baseDepth := 0.0
	if layer >= 0 && layer < len(layerBaseDepths) {
		baseDepth = layerBaseDepths[layer]
	}

	row := float64(y)*rdg + float64(x)*float64(len(layerBaseDepths))*tdg
	return baseDepth + row
}

func tileDrawPosition(layer, x, y int, imgW, imgH, camSX, camSY, halfW, halfH float64) (float64, float64) {
	sx, sy := IsoToScreen(float64(x), float64(y))
	sx = sx - camSX + halfW - float64(HalfTileW)
	sy = sy - camSY + halfH - float64(HalfTileH)

	offset := tileLayerOffset(layer, imgW, imgH)
	return sx + offset.x, sy + offset.y
}

func tileLayerOffset(layer int, imgW, imgH float64) tileOffset {
	switch layer {
	case 7: // Shadow
		return tileOffset{x: -24, y: -12}
	case 1, 2, 8: // Objects, Overlay, Overlay2
		return tileOffset{x: -2 - imgW/2 + float64(HalfTileW), y: -2 - imgH + float64(TileHeight)}
	case 3: // Down Wall
		return tileOffset{x: -32 + float64(HalfTileW), y: -1 - (imgH - float64(TileHeight))}
	case 4: // Right Wall
		return tileOffset{x: float64(HalfTileW), y: -1 - (imgH - float64(TileHeight))}
	case 5: // Roof
		return tileOffset{x: 0, y: -float64(TileWidth)}
	case 6: // Top
		return tileOffset{x: 0, y: -float64(TileHeight)}
	default:
		return tileOffset{}
	}
}
