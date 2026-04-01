package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/avdo/eoweb/internal/gfx"
)

// WalkWidthFactor is the pixel offset per walk frame horizontally.
const WalkWidthFactor = 8 // HalfTileW / 4

// WalkHeightFactor is the pixel offset per walk frame vertically.
const WalkHeightFactor = 4 // HalfTileH / 4

// CharacterEntity represents a character to render on the map.
type CharacterEntity struct {
	PlayerID  int
	Name      string
	X, Y      int
	Direction int // 0=down, 1=up, 2=left, 3=right
	Gender    int // 0=female, 1=male
	Skin      int
	HairStyle int
	HairColor int

	// Equipment graphic IDs (from pub file spec1)
	Armor  int
	Boots  int
	Hat    int
	Weapon int
	Shield int

	// Animation state
	Frame     CharacterFrame
	Walking   bool
	WalkFrame int     // 0-3 current walk animation frame
	WalkProg  float64 // 0.0 to 1.0 interpolation within current frame
	Mirrored  bool    // true for Right(3) and Up(2) directions
}

// NpcEntity represents an NPC to render on the map.
type NpcEntity struct {
	Index        int
	GraphicID    int
	X, Y         int // origin position (during walk)
	DestX, DestY int // walk destination
	Direction    int
	IdleFrame    int // 0 or 1 for idle animation
	Walking      bool
	WalkProg     float64 // 0.0 to 1.0
}

// ItemEntity represents a dropped item on the map.
type ItemEntity struct {
	UID       int
	GraphicID int
	X, Y      int
}

// WalkOffset returns the pixel offset for a walking character based on direction and progress.
// progress is 0.0 (just started) to 1.0 (arrived at destination tile).
func WalkOffset(dir int, progress float64) (float64, float64) {
	// Full tile offset per direction: 0=Down, 1=Left, 2=Up, 3=Right
	var dx, dy float64
	switch dir {
	case 0: // Down: -x, +y
		dx, dy = -HalfTileW, HalfTileH
	case 1: // Left: -x, -y
		dx, dy = -HalfTileW, -HalfTileH
	case 2: // Up: +x, -y
		dx, dy = HalfTileW, -HalfTileH
	case 3: // Right: +x, +y
		dx, dy = HalfTileW, HalfTileH
	}
	// Invert: offset is from destination back toward origin
	return -dx * (1.0 - progress), -dy * (1.0 - progress)
}

// WalkForwardOffset returns the pixel offset for an entity walking FROM origin TOWARD destination.
// progress is 0.0 (at origin) to 1.0 (at destination).
func WalkForwardOffset(dir int, progress float64) (float64, float64) {
	var dx, dy float64
	switch dir {
	case 0: // Down: -x, +y
		dx, dy = -HalfTileW, HalfTileH
	case 1: // Left: -x, -y
		dx, dy = -HalfTileW, -HalfTileH
	case 2: // Up: +x, -y
		dx, dy = HalfTileW, -HalfTileH
	case 3: // Right: +x, +y
		dx, dy = HalfTileW, HalfTileH
	}
	return dx * progress, dy * progress
}

// RenderCharacter draws a character at the given screen position.
func RenderCharacter(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, sx, sy float64) {
	// Apply walk offset using total progress
	if ch.Walking {
		wox, woy := WalkOffset(ch.Direction, ch.WalkProg)
		sx += wox
		sy += woy
	}

	frame := ch.Frame
	gfxID := skinGfxID(frame)

	img, err := loader.GetImage(GfxSkinSprites, gfxID)
	if err != nil || img == nil {
		return
	}

	w, h := frameSize(frame)
	fc := frameCount(frame)
	fn := frameNumber(frame)
	upLeft := isUpLeft(frame)

	// Calculate source rectangle within the sprite sheet
	// Layout: [Female DownRight frames | Female UpLeft frames | Male DownRight frames | Male UpLeft frames]
	// Each direction block has fc sub-frames side by side
	startX := 0
	if ch.Gender == 1 { // male
		startX = w * fc * 2
	}
	if upLeft {
		startX += w * fc
	}
	srcX := startX + w*fn
	srcY := ch.Skin * h

	// Bounds check
	imgBounds := img.Bounds()
	if srcX+w > imgBounds.Dx() || srcY+h > imgBounds.Dy() {
		return
	}

	sub := img.SubImage(image.Rect(srcX, srcY, srcX+w, srcY+h)).(*ebiten.Image)

	// Position: sx,sy is the tile center (top of diamond).
	// Character feet should be at tile center: offset so bottom-center of sprite aligns.
	drawX := sx - float64(w)/2
	drawY := sy - float64(h) + HalfHalfTileH

	op := &ebiten.DrawImageOptions{}
	if ch.Mirrored {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(drawX+float64(w), drawY)
	} else {
		op.GeoM.Translate(drawX, drawY)
	}
	screen.DrawImage(sub, op)

	// Draw hair
	renderHair(screen, loader, ch, frame, sx, sy, w, h)

	// Draw armor
	renderArmor(screen, loader, ch, frame, sx, sy)

	// Draw boots
	renderBoots(screen, loader, ch, frame, sx, sy)

	// Draw nameplate above character
	nameX := sx - float64(len(ch.Name)*3)
	nameY := drawY - 12
	ebitenutil.DebugPrintAt(screen, ch.Name, int(nameX), int(nameY))
}

func renderHair(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, sx, sy float64, charW, charH int) {
	if ch.HairStyle == 0 {
		return
	}

	baseID := (ch.HairStyle-1)*40 + ch.HairColor*4
	gfxFile := GenderGfx(ch.Gender, GfxFemaleHair, GfxMaleHair)

	// Hair has 4 sub-images: 1=down-right, 2=down-right behind, 3=up-left, 4=up-left behind
	upLeft := isUpLeft(frame)
	hairID := baseID + 1 // front down-right
	if upLeft {
		hairID = baseID + 3 // front up-left
	}

	img, err := loader.GetImage(gfxFile, hairID)
	if err != nil || img == nil {
		return
	}

	hw := float64(img.Bounds().Dx())
	hh := float64(img.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-hw/2, sy-float64(charH)+HalfTileH-hh/2+float64(charH)/2-14)
	screen.DrawImage(img, op)
}

func renderArmor(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, sx, sy float64) {
	if ch.Armor == 0 {
		return
	}

	baseID := (ch.Armor - 1) * 50
	gfxFile := GenderGfx(ch.Gender, GfxFemaleArmor, GfxMaleArmor)

	// Armor frame index = CharacterFrame enum value + 1
	armorID := baseID + int(frame) + 1

	img, err := loader.GetImage(gfxFile, armorID)
	if err != nil || img == nil {
		return
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	_, charH := frameSize(frame)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-w/2, sy-float64(charH)+HalfTileH+(float64(charH)-h)/2-3)
	screen.DrawImage(img, op)
}

func renderBoots(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, sx, sy float64) {
	if ch.Boots == 0 {
		return
	}

	baseID := (ch.Boots - 1) * 40
	gfxFile := GenderGfx(ch.Gender, GfxFemaleShoes, GfxMaleShoes)

	// Boots use same frame mapping as armor (simplified)
	bootsID := baseID + int(frame) + 1
	if bootsID > baseID+16 {
		bootsID = baseID + 1
	}

	img, err := loader.GetImage(gfxFile, bootsID)
	if err != nil || img == nil {
		return
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	_, charH := frameSize(frame)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-w/2, sy-float64(charH)+HalfTileH+(float64(charH)-h)/2+21)
	screen.DrawImage(img, op)
}

// RenderNPC draws an NPC at the given screen position.
func RenderNPC(screen *ebiten.Image, loader *gfx.Loader, npc *NpcEntity, sx, sy float64) {
	if npc.Walking && npc.DestX != npc.X || npc.Walking && npc.DestY != npc.Y {
		// Interpolate from origin toward destination using actual tile delta
		destSX, destSY := IsoToScreen(float64(npc.DestX), float64(npc.DestY))
		origSX, origSY := IsoToScreen(float64(npc.X), float64(npc.Y))
		dx := (destSX - origSX) * npc.WalkProg
		dy := (destSY - origSY) * npc.WalkProg
		sx += dx
		sy += dy
	}

	baseID := (npc.GraphicID - 1) * 40

	// Standing frames: 1=down frame1, 2=down frame2, 3=up frame1, 4=up frame2
	upLeft := npc.Direction == 1 || npc.Direction == 2
	mirrored := npc.Direction == 2 || npc.Direction == 3

	frameIdx := 1 // standing down frame 1
	if upLeft {
		frameIdx = 3 // standing up frame 1
	}

	// Idle animation disabled — requires ENF metadata (animatedStanding flag)
	// to know which NPCs have a second standing frame. Without it, toggling
	// to a non-existent frame causes flickering.

	gfxID := baseID + frameIdx

	img, err := loader.GetImage(GfxNPC, gfxID)
	if err != nil || img == nil {
		return
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	if mirrored {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(sx+w/2, sy-h+HalfTileH)
	} else {
		op.GeoM.Translate(sx-w/2, sy-h+HalfTileH)
	}
	screen.DrawImage(img, op)
}

// RenderItem draws a dropped item at the given screen position.
func RenderItem(screen *ebiten.Image, loader *gfx.Loader, item *ItemEntity, sx, sy float64) {
	img, err := loader.GetImage(GfxItems, item.GraphicID)
	if err != nil || img == nil {
		return
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-w/2, sy-h/2)
	screen.DrawImage(img, op)
}
