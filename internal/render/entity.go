package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
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
	ID           int
	GraphicID    int
	X, Y         int // origin position (during walk)
	DestX, DestY int // walk destination
	Direction    int
	IdleFrame    int // 0 or 1 for idle animation
	Walking      bool
	WalkFrame    int
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

type hatMaskType int

const (
	hatMaskStandard hatMaskType = iota
	hatMaskFaceMask
	hatMaskHideHair
)

// ItemEntity represents a dropped item on the map.
type ItemEntity struct {
	UID       int
	GraphicID int
	X, Y      int
}

type CursorEntity struct {
	X, Y int
	Type int
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

	drawX, drawY := CharacterDrawPosition(ch.Gender, frame, ch.Direction, sx, sy, w, h)
	maskType := hatMask(ch.Hat)

	renderCharacterBackBehind(screen, loader, ch, frame, drawX, drawY, w, h)
	renderCharacterWeaponBehind(screen, loader, ch, frame, drawX, drawY, w, h)
	if maskType != hatMaskHideHair {
		renderHairBehind(screen, loader, ch, frame, drawX, drawY, w, h)
	}
	renderImage(screen, sub, drawX, drawY, ch.Mirrored, 1.0, ch.HitProg)
	renderBoots(screen, loader, ch, frame, drawX, drawY, w, h)
	renderArmor(screen, loader, ch, frame, drawX, drawY, w, h)
	if maskType == hatMaskFaceMask {
		renderHat(screen, loader, ch, frame, drawX, drawY, w, h)
	}
	if maskType != hatMaskHideHair {
		renderHairFront(screen, loader, ch, frame, drawX, drawY, w, h)
	}
	if maskType != hatMaskFaceMask {
		renderHat(screen, loader, ch, frame, drawX, drawY, w, h)
	}
	renderShield(screen, loader, ch, frame, drawX, drawY, w, h)
	renderCharacterBackFront(screen, loader, ch, frame, drawX, drawY, w, h)
	renderCharacterWeaponFront(screen, loader, ch, frame, drawX, drawY, w, h)
	renderIndicators(screen, ch.Indicators, sx, drawY-30)
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

func renderHairBehind(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.HairStyle == 0 {
		return
	}

	baseID := (ch.HairStyle-1)*40 + ch.HairColor*4
	gfxFile := GenderGfx(ch.Gender, GfxFemaleHair, GfxMaleHair)

	// Hair has 4 sub-images: 1=down-right behind, 2=down-right front, 3=up-left behind, 4=up-left front.
	upLeft := isUpLeft(frame)
	hairID := baseID + 1
	if upLeft {
		hairID = baseID + 3
	}

	img, err := loader.GetImage(gfxFile, hairID)
	if err != nil || img == nil {
		return
	}

	offset := hairOffset(ch.Gender, frame)
	renderAttachment(screen, img, bodyX, bodyY, charW, charH, offset, ch.Mirrored, true, 1.0, ch.HitProg)
}

func renderHairFront(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.HairStyle == 0 {
		return
	}

	baseID := (ch.HairStyle-1)*40 + ch.HairColor*4
	gfxFile := GenderGfx(ch.Gender, GfxFemaleHair, GfxMaleHair)

	hairID := baseID + 2
	if isUpLeft(frame) {
		hairID = baseID + 4
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
	x, y := hatDrawPosition(bodyX, bodyY, charW, charH, img.Bounds().Dx(), img.Bounds().Dy(), offset, ch.Mirrored)
	renderImage(screen, img, x, y, ch.Mirrored, 1.0, ch.HitProg)
}

func hatDrawPosition(bodyX, bodyY float64, bodyW, bodyH, hatW, hatH int, offset attachmentOffset, mirrored bool) (float64, float64) {
	// Hats are anchored against the full 100x100 character frame in the reference client,
	// not against the cropped body sprite's top edge.
	frameTopX := bodyX - float64(halfCharacterFrameSize) + float64(bodyW)/2
	frameTopY := bodyY - float64(halfCharacterFrameSize) + float64(bodyH)/2
	x := frameTopX + float64(halfCharacterFrameSize-hatW/2) + offset.X
	y := frameTopY - float64(hatH)/2 + offset.Y
	if mirrored {
		attachmentRight := (x - bodyX) + float64(hatW)
		x = bodyX + float64(bodyW) - attachmentRight
	}
	return x, y
}

func renderShield(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Shield == 0 || isBackGraphic(ch.Shield) {
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

func renderCharacterBackBehind(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Shield == 0 || !isBackGraphic(ch.Shield) || isUpLeft(frame) {
		return
	}

	renderCharacterBack(screen, loader, ch, frame, bodyX, bodyY, charW, charH)
}

func renderCharacterBackFront(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	if ch.Shield == 0 || !isBackGraphic(ch.Shield) || !isUpLeft(frame) {
		return
	}

	renderCharacterBack(screen, loader, ch, frame, bodyX, bodyY, charW, charH)
}

func renderCharacterBack(screen *ebiten.Image, loader *gfx.Loader, ch *CharacterEntity, frame CharacterFrame, bodyX, bodyY float64, charW, charH int) {
	baseID := (ch.Shield - 1) * 50
	backID := baseID + backFrameNumber(frame) + 1

	img, err := loader.GetImage(GenderGfx(ch.Gender, GfxFemaleBack, GfxMaleBack), backID)
	if err != nil || img == nil {
		return
	}

	offset := backOffset(ch.Gender, frame)
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

func isBackGraphic(shield int) bool {
	switch shield {
	case 10, 11, 14, 15, 16, 18, 19:
		return true
	default:
		return false
	}
}

func hatMask(hat int) hatMaskType {
	switch hat {
	case 7, 8, 9, 10, 11, 12, 13, 14, 15, 32, 33, 48, 50:
		return hatMaskFaceMask
	case 16, 17, 18, 19, 20, 21, 25, 26, 28, 30, 31, 34, 35, 36, 37, 38, 40, 41, 44, 46, 47:
		return hatMaskHideHair
	default:
		return hatMaskStandard
	}
}

func backFrameNumber(frame CharacterFrame) int {
	switch frame {
	case FrameStandDown, FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4, FrameRaisedDown, FrameChairDown, FrameFloorDown:
		return 0
	case FrameStandUp, FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4, FrameRaisedUp, FrameChairUp, FrameFloorUp:
		return 1
	case FrameMeleeDown1, FrameMeleeDown2, FrameRangeDown:
		return 2
	case FrameMeleeUp1, FrameMeleeUp2, FrameRangeUp:
		return 3
	default:
		return 0
	}
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
	if npc.Walking {
		frameIdx = 5 + npc.WalkFrame
		if upLeft {
			frameIdx = 9 + npc.WalkFrame
		}
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

	drawX, drawY := NPCDrawPosition(npc.GraphicID, mirrored, sx, sy, w, h)
	renderImage(screen, img, drawX, drawY, mirrored, alpha, npc.HitProg)
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

func RenderCursor(screen *ebiten.Image, loader *gfx.Loader, cursor *CursorEntity, sx, sy float64) {
	if cursor == nil || cursor.Type < 0 {
		return
	}

	cursorImg, err := loader.GetImage(2, 24)
	if err != nil || cursorImg == nil {
		return
	}

	tw := TileWidth
	th := TileHeight
	srcX := cursor.Type * tw
	if srcX+tw > cursorImg.Bounds().Dx() {
		srcX = 0
	}
	sub := cursorImg.SubImage(image.Rect(srcX, 0, srcX+tw, th)).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-float64(tw)/2, sy-float64(th)/2)
	screen.DrawImage(sub, op)
}

func renderIndicators(screen *ebiten.Image, indicators []CombatIndicator, sx, baseY float64) {
	if len(indicators) == 0 {
		return
	}

	face := textv2.NewGoXFace(basicfont.Face7x13)
	ascent := float64(basicfont.Face7x13.Metrics().Ascent.Ceil())
	for i := range indicators {
		indicator := indicators[len(indicators)-1-i]
		rise := indicator.Progress * 18
		y := baseY - float64(i*14) - rise
		x := sx - float64(len(indicator.Text)*7)/2
		shadowOp := &textv2.DrawOptions{}
		shadowOp.GeoM.Translate(x+1, y-ascent+1)
		shadowOp.ColorScale.ScaleWithColor(color.NRGBA{A: 160})
		textv2.Draw(screen, indicator.Text, face, shadowOp)
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(x, y-ascent)
		op.ColorScale.ScaleWithColor(indicatorColor(indicator.Kind, indicator.Progress))
		textv2.Draw(screen, indicator.Text, face, op)
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
		case FrameChairDown:
			return attachmentOffset{1, -13}
		case FrameChairUp:
			return attachmentOffset{2, -13}
		case FrameFloorDown:
			return attachmentOffset{1, -8}
		case FrameFloorUp:
			return attachmentOffset{3, -8}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{0, -14}
		case FrameMeleeDown2:
			return attachmentOffset{-4, -9}
		case FrameMeleeUp2:
			return attachmentOffset{-4, -13}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{-1, -12}
		case FrameRangeDown:
			return attachmentOffset{5, -15}
		case FrameRangeUp:
			return attachmentOffset{3, -14}
		default:
			return attachmentOffset{-1, -14}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{0, -15}
	case FrameChairDown, FrameChairUp:
		return attachmentOffset{2, -12}
	case FrameFloorDown:
		return attachmentOffset{2, -7}
	case FrameFloorUp:
		return attachmentOffset{4, -7}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{1, -15}
	case FrameMeleeDown2:
		return attachmentOffset{-5, -11}
	case FrameMeleeUp2:
		return attachmentOffset{-3, -14}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{-1, -13}
	case FrameRangeDown:
		return attachmentOffset{4, -15}
	case FrameRangeUp:
		return attachmentOffset{2, -15}
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
		case FrameChairDown:
			return attachmentOffset{2, 24}
		case FrameChairUp:
			return attachmentOffset{3, 24}
		case FrameFloorDown:
			return attachmentOffset{2, 29}
		case FrameFloorUp:
			return attachmentOffset{4, 29}
		case FrameRangeDown:
			return attachmentOffset{6, 22}
		case FrameRangeUp:
			return attachmentOffset{4, 23}
		case FrameMeleeDown1, FrameMeleeUp1:
			return attachmentOffset{1, 23}
		case FrameMeleeDown2:
			return attachmentOffset{-3, 28}
		case FrameMeleeUp2:
			return attachmentOffset{-3, 24}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{0, 25}
		case FrameStandDown, FrameStandUp, FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
			FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
			return attachmentOffset{0, 23}
		}
		return attachmentOffset{0, 23}
	}

	switch frame {
	case FrameChairDown, FrameChairUp:
		return attachmentOffset{3, 25}
	case FrameFloorDown:
		return attachmentOffset{3, 30}
	case FrameFloorUp:
		return attachmentOffset{5, 30}
	case FrameRangeDown:
		return attachmentOffset{5, 22}
	case FrameRangeUp:
		return attachmentOffset{3, 22}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{2, 22}
	case FrameMeleeDown2:
		return attachmentOffset{-4, 26}
	case FrameMeleeUp2:
		return attachmentOffset{-2, 23}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{0, 24}
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{1, 22}
	case FrameStandDown, FrameStandUp, FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{0, 22}
	default:
		return attachmentOffset{0, 22}
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

func backOffset(gender int, frame CharacterFrame) attachmentOffset {
	if gender == 0 {
		switch frame {
		case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4,
			FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4,
			FrameMeleeDown1, FrameMeleeDown2, FrameMeleeUp1, FrameMeleeUp2:
			return attachmentOffset{0, -17}
		case FrameRaisedDown, FrameRaisedUp:
			return attachmentOffset{0, -15}
		case FrameChairDown:
			return attachmentOffset{2, -16}
		case FrameChairUp:
			return attachmentOffset{3, -16}
		case FrameFloorDown:
			return attachmentOffset{2, -11}
		case FrameFloorUp:
			return attachmentOffset{4, -11}
		case FrameRangeDown, FrameRangeUp:
			return attachmentOffset{1, -17}
		default:
			return attachmentOffset{0, -17}
		}
	}

	switch frame {
	case FrameWalkDown1, FrameWalkDown2, FrameWalkDown3, FrameWalkDown4:
		return attachmentOffset{1, -18}
	case FrameWalkUp1, FrameWalkUp2, FrameWalkUp3, FrameWalkUp4:
		return attachmentOffset{0, -18}
	case FrameMeleeDown1, FrameMeleeUp1:
		return attachmentOffset{2, -18}
	case FrameMeleeDown2, FrameMeleeUp2:
		return attachmentOffset{-2, -18}
	case FrameRaisedDown, FrameRaisedUp:
		return attachmentOffset{0, -16}
	case FrameChairDown, FrameChairUp:
		return attachmentOffset{3, -15}
	case FrameFloorDown:
		return attachmentOffset{3, -10}
	case FrameFloorUp:
		return attachmentOffset{5, -10}
	case FrameRangeDown, FrameRangeUp:
		return attachmentOffset{2, -18}
	default:
		return attachmentOffset{0, -18}
	}
}
