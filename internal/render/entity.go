package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

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

	AttackProg float64
	HitProg    float64
	Indicators []CombatIndicator
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
	AttackProg   float64
	HitProg      float64
	Dead         bool
	DeathProg    float64
	Indicators   []CombatIndicator
}

type CombatIndicator struct {
	Text     string
	Kind     int
	Progress float64
}

type attachmentOffset struct {
	X float64
	Y float64
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

	if ch.AttackProg > 0 {
		aox, aoy := attackOffset(ch.Direction, ch.AttackProg)
		sx += aox
		sy += aoy
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

	renderCharacterWeaponBehind(screen, loader, ch, frame, drawX, drawY, w, h)
	renderImage(screen, sub, drawX, drawY, ch.Mirrored, 1.0, ch.HitProg)
	renderBoots(screen, loader, ch, frame, drawX, drawY, w, h)
	renderArmor(screen, loader, ch, frame, drawX, drawY, w, h)
	renderHair(screen, loader, ch, frame, drawX, drawY, w, h)
	renderHat(screen, loader, ch, frame, drawX, drawY, w, h)
	renderShield(screen, loader, ch, frame, drawX, drawY, w, h)
	renderCharacterWeaponFront(screen, loader, ch, frame, drawX, drawY, w, h)
	renderIndicators(screen, ch.Indicators, sx, drawY-30)

	// Draw nameplate above character
	nameX := sx - float64(len(ch.Name)*3)
	nameY := drawY - 12
	ebitenutil.DebugPrintAt(screen, ch.Name, int(nameX), int(nameY))
}

func renderImage(screen, img *ebiten.Image, x, y float64, mirrored bool, alpha float32, hitProg float64) {
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	if mirrored {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(x+float64(img.Bounds().Dx()), y)
	} else {
		op.GeoM.Translate(x, y)
	}
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(alpha)
	}
	if hitProg > 0 {
		flash := float32(0.2 + 0.3*(1.0-hitProg))
		op.ColorScale.Scale(1.0+flash, 0.75, 0.75, 1)
	}
	screen.DrawImage(img, op)
}

func renderHair(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
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

	offset := hairOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderArmor(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
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

	offset := armorOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderBoots(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
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

	offset := bootsOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderHat(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Hat == 0 {
		return
	}

	baseID := (ch.Hat - 1) * 10
	hatID := baseID + 1
	if isUpLeft(frame) {
		hatID = baseID + 3
	}

	img, err := loader.GetImage(GenderGfx(ch.Gender, GfxFemaleHat, GfxMaleHat), hatID)
	if err != nil || img == nil {
		return
	}

	offset := hatOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderShield(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Shield == 0 {
		return
	}

	baseID := (ch.Shield - 1) * 50
	shieldID := baseID + int(frame) + 1
	img, err := loader.GetImage(GenderGfx(ch.Gender, GfxFemaleBack, GfxMaleBack), shieldID)
	if err != nil || img == nil {
		return
	}

	offset := shieldOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderCharacterWeaponBehind(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Weapon == 0 {
		return
	}

	weaponID, ok := weaponGraphicID(ch.Weapon, frame)
	if !ok {
		return
	}

	img, err := loader.GetImage(GenderGfx(ch.Gender, GfxFemaleWeapons, GfxMaleWeapons), weaponID)
	if err != nil || img == nil {
		return
	}

	offset := weaponOffset(ch.Direction, ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, false, 1.0, ch.HitProg)
}

func renderCharacterWeaponFront(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Weapon == 0 || frame != FrameMeleeDown2 {
		return
	}

	img, err := loader.GetImage(GenderGfx(ch.Gender, GfxFemaleWeapons, GfxMaleWeapons), (ch.Weapon-1)*100+17)
	if err != nil || img == nil {
		return
	}

	offset := weaponOffset(ch.Direction, ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, false, 1.0, ch.HitProg)
}

func renderAttachment(screen *ebiten.Image, img *ebiten.Image, bodyX, bodyY float64, bodyW, bodyH int, offset attachmentOffset, mirrored, mirrorOffset bool, alpha float32, hitProg float64) {
	if img == nil {
		return
	}

	imgW := img.Bounds().Dx()
	imgH := img.Bounds().Dy()
	x := bodyX + float64(bodyW-imgW)/2 + offset.X
	y := bodyY + float64(bodyH-imgH)/2 + offset.Y
	if mirrored && mirrorOffset {
		attachmentRight := (x - bodyX) + float64(imgW)
		x = bodyX + float64(bodyW) - attachmentRight
	}
	renderImage(screen, img, x, y, mirrored, alpha, hitProg)
}

func weaponGraphicID(weapon int, frame CharacterFrame) (int, bool) {
	frameMap := map[CharacterFrame]int{
		FrameStandDown:  0,
		FrameStandUp:    1,
		FrameWalkDown1:  2,
		FrameWalkDown2:  3,
		FrameWalkDown3:  4,
		FrameWalkDown4:  5,
		FrameWalkUp1:    6,
		FrameWalkUp2:    7,
		FrameWalkUp3:    8,
		FrameWalkUp4:    9,
		FrameRaisedDown: 10,
		FrameRaisedUp:   11,
		FrameMeleeDown1: 12,
		FrameMeleeDown2: 13,
		FrameMeleeUp1:   14,
		FrameMeleeUp2:   15,
	}

	index, ok := frameMap[frame]
	if !ok {
		return 0, false
	}
	return (weapon-1)*100 + index + 1, true
}

// RenderNPC draws an NPC at the given screen position.
func RenderNPC(screen *ebiten.Image, loader *gfx.Loader, npc *NpcEntity, sx, sy float64) {
	if npc.GraphicID <= 0 {
		return
	}

	if npc.Walking && npc.DestX != npc.X || npc.Walking && npc.DestY != npc.Y {
		// Interpolate from origin toward destination using actual tile delta
		destSX, destSY := IsoToScreen(float64(npc.DestX), float64(npc.DestY))
		origSX, origSY := IsoToScreen(float64(npc.X), float64(npc.Y))
		dx := (destSX - origSX) * npc.WalkProg
		dy := (destSY - origSY) * npc.WalkProg
		sx += dx
		sy += dy
	}
	if npc.AttackProg > 0 {
		aox, aoy := attackOffset(npc.Direction, npc.AttackProg)
		sx += aox
		sy += aoy
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
	alpha := float32(1.0)
	if npc.Dead {
		alpha = float32(1.0 - npc.DeathProg*0.85)
		if alpha < 0 {
			alpha = 0
		}
		sy -= npc.DeathProg * 8
	}

	renderImage(screen, img, sx-w/2, sy-h+HalfTileH, mirrored, alpha, npc.HitProg)
	renderIndicators(screen, npc.Indicators, sx, sy-h-10)
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

func renderIndicators(screen *ebiten.Image, indicators []CombatIndicator, sx, baseY float64) {
	if len(indicators) == 0 {
		return
	}

	face := basicfont.Face7x13
	for i := range indicators {
		indicator := indicators[len(indicators)-1-i]
		rise := indicator.Progress * 18
		y := baseY - float64(i*14) - rise
		x := sx - float64(len(indicator.Text)*7)/2
		text.Draw(screen, indicator.Text, face, int(x), int(y), indicatorColor(indicator.Kind, indicator.Progress))
	}
}

func indicatorColor(kind int, progress float64) color.Color {
	alpha := uint8(255)
	if progress > 0.65 {
		alpha = uint8(255 * (1.0 - (progress-0.65)/0.35))
	}
	if alpha < 48 {
		alpha = 48
	}

	switch kind {
	case 1:
		return color.NRGBA{R: 220, G: 220, B: 220, A: alpha}
	case 2:
		return color.NRGBA{R: 110, G: 255, B: 140, A: alpha}
	default:
		return color.NRGBA{R: 255, G: 110, B: 110, A: alpha}
	}
}

func attackOffset(dir int, progress float64) (float64, float64) {
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

func hairOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{0, -14}
		case FrameMeleeDown2:
			return attachmentOffset{-4, -9}
		case FrameMeleeUp2:
			return attachmentOffset{-4, -13}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{-1, -12}
		default:
			return attachmentOffset{-1, -14}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{0, -15}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{1, -15}
	case FrameMeleeDown2:
		return attachmentOffset{-5, -11}
	case FrameMeleeUp2:
		return attachmentOffset{-3, -14}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{-1, -13}
	default:
		return attachmentOffset{-1, -15}
	}
}

func bootsOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameWalkUp4:
			return attachmentOffset{0, 20}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{1, 21}
		case FrameMeleeDown2, FrameMeleeUp2:
			return attachmentOffset{-1, 21}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{0, 23}
		default:
			return attachmentOffset{0, 21}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{1, 19}
	case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{0, 19}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{2, 20}
	case FrameMeleeDown2, FrameMeleeUp2:
		return attachmentOffset{-2, 20}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{0, 22}
	default:
		return attachmentOffset{0, 20}
	}
}

func armorOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
			FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
			return attachmentOffset{0, -4}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{1, -3}
		case FrameMeleeDown2, FrameMeleeUp2:
			return attachmentOffset{-1, -3}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{0, -1}
		default:
			return attachmentOffset{0, -3}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{1, -5}
	case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{0, -5}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{2, -4}
	case FrameMeleeDown2, FrameMeleeUp2:
		return attachmentOffset{-2, -4}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{0, -2}
	default:
		return attachmentOffset{0, -4}
	}
}

func hatOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{1, -6}
		case FrameMeleeDown2:
			return attachmentOffset{-3, -1}
		case FrameMeleeUp2:
			return attachmentOffset{-3, -5}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{0, -4}
		default:
			return attachmentOffset{0, -6}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{1, -7}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{2, -7}
	case FrameMeleeDown2:
		return attachmentOffset{-4, -3}
	case FrameMeleeUp2:
		return attachmentOffset{-2, -6}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{0, -5}
	default:
		return attachmentOffset{0, -7}
	}
}

func weaponOffset(direction, gender int, frame CharacterFrame) attachmentOffset {
	base := baseWeaponOffset(gender, frame)
	if direction == 2 || direction == 3 {
		base.X = -base.X
		base.X += weaponMirrorNudge(direction, gender, frame)
	}
	return base
}

func baseWeaponOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
			FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
			return attachmentOffset{-9, -7}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{-8, -6}
		case FrameMeleeDown2, FrameMeleeUp2:
			return attachmentOffset{-10, -6}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{-9, -4}
		default:
			return attachmentOffset{-9, -6}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{-8, -8}
	case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{-9, -8}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{-7, -7}
	case FrameMeleeDown2, FrameMeleeUp2:
		return attachmentOffset{-11, -7}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{-9, -5}
	default:
		return attachmentOffset{-9, -7}
	}
}

func weaponMirrorNudge(direction, gender int, frame CharacterFrame) float64 {
	if direction == 3 {
		if gender == 0 {
			switch frame {
			case FrameMeleeDown2:
				return -2
			case FrameMeleeDown1, FrameRaisedDown:
				return -1
			default:
				return -1
			}
		}

		switch frame {
		case FrameMeleeDown2:
			return -2
		case FrameMeleeDown1, FrameRaisedDown:
			return -1
		default:
			return -1
		}
	}

	if direction == 2 {
		if gender == 0 {
			switch frame {
			case FrameMeleeUp2:
				return -2
			case FrameMeleeUp1, FrameRaisedUp:
				return -1
			default:
				return -1
			}
		}

		switch frame {
		case FrameMeleeUp2:
			return -2
		case FrameMeleeUp1, FrameRaisedUp:
			return -1
		default:
			return -1
		}
	}

	return 0
}

func shieldOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
			FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
			return attachmentOffset{-5, 4}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{-4, 5}
		case FrameMeleeDown2, FrameMeleeUp2:
			return attachmentOffset{-6, 5}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{-5, 7}
		default:
			return attachmentOffset{-5, 5}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{-4, 3}
	case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{-5, 3}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{-3, 4}
	case FrameMeleeDown2, FrameMeleeUp2:
		return attachmentOffset{-7, 4}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{-5, 6}
	default:
		return attachmentOffset{-5, 4}
	}
}
