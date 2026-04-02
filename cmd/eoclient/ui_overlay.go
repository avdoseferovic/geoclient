package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
)

type overlayState struct {
	ticks              int
	loginUsername      []rune
	loginPassword      []rune
	loginFocus         int
	loginSubmitting    bool
	previewDirection   int
	selectedCharacter  int
	rosterScroll       int
	selectingCharacter bool
	statusMessage      string
	activeMenuPanel    gameMenuPanel
	selectedInventory  int
	inventoryPage      int
}

type inventoryDragState struct {
	Active bool
	ItemID int
	Amount int
	Offset image.Point
}

type gameMenuPanel int

const (
	gameMenuPanelNone gameMenuPanel = iota
	gameMenuPanelStats
	gameMenuPanelInventory
	gameMenuPanelMap
	gameMenuPanelGuild
)

type characterSelectLayout struct {
	dialog         image.Rectangle
	previewRect    image.Rectangle
	previewArtRect image.Rectangle
	namePlateRect  image.Rectangle
	metaRect       image.Rectangle
	rosterRect     image.Rectangle
	joinButtonRect image.Rectangle
}

type gameHUDLayout struct {
	statusRect    image.Rectangle
	menuRect      image.Rectangle
	menuPanelRect image.Rectangle
	infoRect      image.Rectangle
}

type hudMenuButton struct {
	Panel gameMenuPanel
	Label string
	Rect  image.Rectangle
}

const (
	loginFocusUsername = iota
	loginFocusPassword
	characterSelectRosterRowHeight   = 48
	characterSelectRosterTopPadding  = 12
	characterSelectRosterBottomSpace = 10
	inventoryGridCols                = 8
	inventoryGridRows                = 8
	inventoryGridPages               = 2
	inventoryCellSize                = 22
)

func newOverlayState() overlayState {
	return overlayState{}
}

func (g *Game) updateOverlayState() {
	g.overlay.ticks++
	if g.client.GetState() == game.StateLoggedIn {
		g.normalizeCharacterSelectState()
	}

	if g.client.GetState() == game.StateInitial && g.connectError != "" {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.connected = false
			g.connectArmed = true
			g.connectError = ""
			g.overlay.statusMessage = "Reconnecting..."
		}
	}
}

func (g *Game) updateLogin() {
	if g.overlay.loginSubmitting {
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.overlay.loginFocus = (g.overlay.loginFocus + 1) % 2
	}

	field := &g.overlay.loginUsername
	if g.overlay.loginFocus == loginFocusPassword {
		field = &g.overlay.loginPassword
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(*field) > 0 {
		*field = (*field)[:len(*field)-1]
	}

	for _, r := range ebiten.AppendInputChars(nil) {
		if r < 32 || r > 126 {
			continue
		}
		if len(*field) >= 24 {
			continue
		}
		*field = append(*field, r)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.submitLogin()
	}
}

func (g *Game) submitLogin() {
	username := strings.TrimSpace(string(g.overlay.loginUsername))
	password := string(g.overlay.loginPassword)
	if username == "" || password == "" {
		g.connectError = "Enter both account name and password"
		return
	}

	g.client.Username = username
	g.client.Password = password
	g.overlay.loginSubmitting = true
	g.connectError = ""
	g.overlay.statusMessage = "Sending credentials..."
	g.sendLogin()
}

func (g *Game) updateCharacterSelect() {
	count := len(g.client.Characters)
	if count == 0 || g.overlay.selectingCharacter {
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.handleCharacterSelectClick() {
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.overlay.selectedCharacter = (g.overlay.selectedCharacter - 1 + count) % count
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.overlay.selectedCharacter = (g.overlay.selectedCharacter + 1) % count
	}
	g.normalizeCharacterSelectState()
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.selectCurrentCharacter()
	}
}

func (g *Game) drawOverlayScreen(screen *ebiten.Image) {
	theme := clientui.RetroTheme

	switch g.client.GetState() {
	case game.StateInitial:
		clientui.DrawBackdrop(screen, theme, g.overlay.ticks)
		g.drawConnectingDialog(screen, theme)
	case game.StateConnected:
		clientui.DrawBackdrop(screen, theme, g.overlay.ticks)
		g.drawLoginDialog(screen, theme)
	case game.StateLoggedIn:
		clientui.DrawBackdrop(screen, theme, g.overlay.ticks)
		g.drawCharacterSelectDialog(screen, theme)
	case game.StateInGame:
		snapshot := g.client.UISnapshot()
		g.drawGameHUD(screen, theme)
		g.drawChat(screen, theme)
		g.drawWorldHoverTooltip(screen, theme, snapshot)
	}
}

func (g *Game) drawConnectingDialog(screen *ebiten.Image, theme clientui.Theme) {
	rect := centeredRect(360, 128, g.screenW, g.screenH)
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Gate Link", Accent: theme.Accent})
	status := "Contacting server"
	if g.connectError != "" {
		status = "Connection failed"
	}
	clientui.DrawTextCentered(screen, status, image.Rect(rect.Min.X+24, rect.Min.Y+26, rect.Max.X-24, rect.Min.Y+54), theme.Text)

	line := g.overlay.statusMessage
	if line == "" {
		line = dotPulse("Opening relay", g.overlay.ticks)
	}
	if g.connectError != "" {
		line = g.connectError
	}
	clientui.DrawTextCentered(screen, line, image.Rect(rect.Min.X+28, rect.Min.Y+56, rect.Max.X-28, rect.Min.Y+82), ternaryColor(g.connectError == "", theme.TextDim, theme.Danger))

	footer := "Please wait"
	if g.connectError != "" {
		footer = "Press Enter or R to try again"
	}
	clientui.DrawTextCentered(screen, footer, image.Rect(rect.Min.X+20, rect.Max.Y-34, rect.Max.X-20, rect.Max.Y-14), theme.TextDim)
	spinnerRect := image.Rect(rect.Min.X+rect.Dx()/2-44, rect.Min.Y+88, rect.Min.X+rect.Dx()/2+44, rect.Min.Y+104)
	drawPulseBar(screen, spinnerRect, theme, g.overlay.ticks)
	clientui.DrawTextCentered(screen, "Endless Offline Native Client", image.Rect(0, g.screenH-38, g.screenW, g.screenH-18), theme.TextDim)
}

func (g *Game) drawLoginDialog(screen *ebiten.Image, theme clientui.Theme) {
	rect := centeredRect(404, 236, g.screenW, g.screenH)
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Adventurer Sign-In", Accent: theme.Accent})

	clientui.DrawText(screen, "Account", rect.Min.X+30, rect.Min.Y+54, theme.TextDim)
	userRect := image.Rect(rect.Min.X+30, rect.Min.Y+62, rect.Max.X-30, rect.Min.Y+90)
	clientui.DrawInset(screen, userRect, theme, g.overlay.loginFocus == loginFocusUsername)
	userText := string(g.overlay.loginUsername)
	if g.overlay.loginFocus == loginFocusUsername && g.overlay.ticks%40 < 20 {
		userText += "_"
	}
	clientui.DrawText(screen, userText, userRect.Min.X+10, userRect.Min.Y+18, theme.Text)

	clientui.DrawText(screen, "Password", rect.Min.X+30, rect.Min.Y+118, theme.TextDim)
	passRect := image.Rect(rect.Min.X+30, rect.Min.Y+126, rect.Max.X-30, rect.Min.Y+154)
	clientui.DrawInset(screen, passRect, theme, g.overlay.loginFocus == loginFocusPassword)
	passText := strings.Repeat("*", len(g.overlay.loginPassword))
	if g.overlay.loginFocus == loginFocusPassword && g.overlay.ticks%40 < 20 {
		passText += "_"
	}
	clientui.DrawText(screen, passText, passRect.Min.X+10, passRect.Min.Y+18, theme.Text)

	buttonRect := image.Rect(rect.Min.X+rect.Dx()/2-66, rect.Max.Y-50, rect.Min.X+rect.Dx()/2+66, rect.Max.Y-20)
	clientui.DrawButton(screen, buttonRect, theme, ternaryString(g.overlay.loginSubmitting, "Signing In", "Enter Realm"), true, g.overlay.loginSubmitting)

	statusColor := theme.TextDim
	status := ""
	if g.overlay.loginSubmitting {
		status = "Awaiting server response..."
	}
	if g.connectError != "" {
		status = g.connectError
		statusColor = theme.Danger
	}
	if status != "" {
		clientui.DrawTextWrappedCentered(screen, status, image.Rect(rect.Min.X+24, rect.Min.Y+172, rect.Max.X-24, rect.Min.Y+196), statusColor)
	}
}

func (g *Game) drawCharacterSelectDialog(screen *ebiten.Image, theme clientui.Theme) {
	layout := characterSelectDialogLayout(g.screenW, g.screenH)
	rect := layout.dialog
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Characters", Accent: theme.Accent})

	if len(g.client.Characters) == 0 {
		clientui.DrawTextCentered(screen, "No characters were returned by the server.", image.Rect(rect.Min.X+24, rect.Min.Y+92, rect.Max.X-24, rect.Min.Y+116), theme.Danger)
		clientui.DrawTextCentered(screen, "Create one in your server tools, then reconnect.", image.Rect(rect.Min.X+24, rect.Min.Y+122, rect.Max.X-24, rect.Min.Y+146), theme.TextDim)
		return
	}

	selectedIndex := clampInt(g.overlay.selectedCharacter, 0, len(g.client.Characters)-1)
	selected := g.client.Characters[selectedIndex]
	previewRect := layout.previewRect
	rosterRect := layout.rosterRect
	previewArtRect := layout.previewArtRect
	namePlateRect := layout.namePlateRect
	metaRect := layout.metaRect

	clientui.DrawInset(screen, previewRect, theme, false)
	clientui.DrawInset(screen, previewArtRect, theme, true)
	clientui.DrawInset(screen, namePlateRect, theme, false)
	clientui.DrawInset(screen, metaRect, theme, false)

	drawCharacterPreviewBackdrop(screen, previewArtRect, theme)
	g.drawCharacterSelectionPreview(screen, previewArtRect, selected)
	clientui.DrawText(screen, selected.Name, namePlateRect.Min.X+12, namePlateRect.Min.Y+18, theme.Text)
	levelLabel := fmt.Sprintf("LVL %d", selected.Level)
	clientui.DrawText(screen, levelLabel, namePlateRect.Max.X-clientui.MeasureText(levelLabel)-12, namePlateRect.Min.Y+18, theme.Accent)
	clientui.DrawText(screen, characterSelectSummaryLine(selected), metaRect.Min.X+12, metaRect.Min.Y+18, theme.TextDim)
	clientui.DrawTextf(screen, metaRect.Min.X+12, metaRect.Min.Y+40, theme.TextDim, "%d equipped", equippedCharacterSelectItemCount(selected))

	clientui.DrawInset(screen, rosterRect, theme, false)
	visibleRows := characterSelectVisibleRows(rosterRect)
	start, end := characterSelectVisibleRange(len(g.client.Characters), g.overlay.rosterScroll, visibleRows)

	for rowIndex, i := 0, start; i < end; i, rowIndex = i+1, rowIndex+1 {
		ch := g.client.Characters[i]
		row := image.Rect(rosterRect.Min.X+10, rosterRect.Min.Y+characterSelectRosterTopPadding+rowIndex*characterSelectRosterRowHeight, rosterRect.Max.X-10, rosterRect.Min.Y+characterSelectRosterTopPadding+40+rowIndex*characterSelectRosterRowHeight)
		active := i == selectedIndex
		drawCharacterSelectionCard(screen, row, theme, ch, active)
	}

	status := "Arrow keys choose • Enter joins"
	statusColor := theme.TextDim
	if g.overlay.selectingCharacter {
		status = g.overlay.statusMessage
	}
	if g.connectError != "" {
		status = g.connectError
		statusColor = theme.Danger
	}
	clientui.DrawText(screen, status, rect.Min.X+24, rect.Max.Y-28, statusColor)
	clientui.DrawButton(screen, layout.joinButtonRect, theme, ternaryString(g.overlay.selectingCharacter, "Joining", "Enter World"), true, g.overlay.selectingCharacter)
}

func (g *Game) normalizeCharacterSelectState() {
	count := len(g.client.Characters)
	if count == 0 {
		g.overlay.previewDirection = 0
		g.overlay.selectedCharacter = 0
		g.overlay.rosterScroll = 0
		return
	}
	g.overlay.previewDirection = clampInt(g.overlay.previewDirection, 0, 3)
	g.overlay.selectedCharacter = clampInt(g.overlay.selectedCharacter, 0, count-1)
	visibleRows := characterSelectVisibleRows(image.Rect(0, 0, 236, 210))
	maxScroll := max(0, count-visibleRows)
	targetScroll := g.overlay.selectedCharacter - visibleRows/2
	g.overlay.rosterScroll = clampInt(targetScroll, 0, maxScroll)
}

func (g *Game) drawCharacterSelectionPreview(screen *ebiten.Image, rect image.Rectangle, entry server.CharacterSelectionListEntry) {
	preview := characterSelectPreviewEntity(entry, g.overlay.previewDirection, g.overlay.ticks)
	previewX := float64(rect.Min.X + rect.Dx()/2)
	previewY := float64(rect.Min.Y + rect.Dy() - 20 - characterSelectPreviewBobOffset(g.overlay.ticks))
	render.RenderCharacter(screen, g.gfxLoad, &preview, previewX, previewY)
}

func characterSelectPreviewEntity(entry server.CharacterSelectionListEntry, dir, ticks int) render.CharacterEntity {
	return render.CharacterEntity{
		Name:      "",
		Direction: dir,
		Gender:    int(entry.Gender),
		Skin:      entry.Skin,
		HairStyle: entry.HairStyle,
		HairColor: entry.HairColor,
		Armor:     entry.Equipment.Armor,
		Boots:     entry.Equipment.Boots,
		Hat:       entry.Equipment.Hat,
		Weapon:    entry.Equipment.Weapon,
		Shield:    entry.Equipment.Shield,
		Frame:     characterSelectPreviewFrame(dir, ticks),
		Mirrored:  dir == 2 || dir == 3,
	}
}

func characterSelectPreviewFrame(dir, ticks int) render.CharacterFrame {
	upLeft := dir == 1 || dir == 2
	phase := (ticks / 16) % 6
	if upLeft {
		switch phase {
		case 1:
			return render.FrameWalkUp1
		case 2:
			return render.FrameWalkUp2
		case 4:
			return render.FrameWalkUp3
		case 5:
			return render.FrameWalkUp4
		default:
			return render.FrameStandUp
		}
	}
	switch phase {
	case 1:
		return render.FrameWalkDown1
	case 2:
		return render.FrameWalkDown2
	case 4:
		return render.FrameWalkDown3
	case 5:
		return render.FrameWalkDown4
	default:
		return render.FrameStandDown
	}
}

func characterSelectPreviewBobOffset(ticks int) int {
	return [...]int{0, 1, 2, 1}[(ticks/10)%4]
}

func drawCharacterPreviewBackdrop(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme) {
	for y := rect.Min.Y + 3; y < rect.Max.Y-3; y++ {
		t := float64(y-(rect.Min.Y+3)) / float64(max(1, rect.Dy()-6))
		fill := characterSelectBlendColors(theme.PanelFillAlt, color.NRGBA{R: 24, G: 20, B: 18, A: 255}, t)
		ebitenutil.DrawRect(screen, float64(rect.Min.X+3), float64(y), float64(rect.Dx()-6), 1, fill)
	}
	for x := rect.Min.X + 8; x < rect.Max.X-8; x += 22 {
		ebitenutil.DrawRect(screen, float64(x), float64(rect.Min.Y+8), 1, float64(rect.Dy()-16), colorize(theme.AccentMuted, 30))
	}
	stage := image.Rect(rect.Min.X+44, rect.Max.Y-30, rect.Max.X-44, rect.Max.Y-12)
	ebitenutil.DrawRect(screen, float64(stage.Min.X), float64(stage.Min.Y), float64(stage.Dx()), float64(stage.Dy()), colorize(theme.AccentMuted, 90))
	clientui.DrawBorder(screen, stage, theme.BorderDark, theme.BorderMid, theme.Accent)
}

func drawCharacterSelectionCard(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, ch server.CharacterSelectionListEntry, active bool) {
	fill := theme.PanelFillAlt
	accent := theme.BorderMid
	nameColor := theme.Text
	metaColor := theme.TextDim
	if active {
		fill = color.NRGBA{R: 86, G: 62, B: 34, A: 255}
		accent = theme.Accent
		metaColor = theme.Text
	}

	ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
	clientui.DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, accent)
	clientui.DrawText(screen, ch.Name, rect.Min.X+12, rect.Min.Y+16, nameColor)
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+32, metaColor, "Lvl %d • %s", ch.Level, characterSelectSummaryLine(ch))
}

func characterSelectSummaryLine(entry server.CharacterSelectionListEntry) string {
	gender := "Adventurer"
	if entry.Gender == 1 {
		gender = "Male"
	}
	if entry.Gender == 0 {
		gender = "Female"
	}
	return gender
}

func equippedCharacterSelectItemCount(entry server.CharacterSelectionListEntry) int {
	count := 0
	values := []int{
		entry.Equipment.Armor,
		entry.Equipment.Boots,
		entry.Equipment.Hat,
		entry.Equipment.Shield,
		entry.Equipment.Weapon,
	}
	for _, value := range values {
		if value > 0 {
			count++
		}
	}
	return count
}

func characterSelectBlendColors(a, b color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.NRGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}

func characterSelectVisibleRows(rosterRect image.Rectangle) int {
	usableHeight := rosterRect.Dy() - characterSelectRosterTopPadding - characterSelectRosterBottomSpace
	if usableHeight <= 0 {
		return 1
	}
	return max(1, usableHeight/characterSelectRosterRowHeight)
}

func characterSelectVisibleRange(count, scroll, visibleRows int) (int, int) {
	if count <= 0 {
		return 0, 0
	}
	if visibleRows <= 0 {
		visibleRows = 1
	}
	start := clampInt(scroll, 0, max(0, count-visibleRows))
	end := min(count, start+visibleRows)
	return start, end
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func (g *Game) drawGameHUD(screen *ebiten.Image, theme clientui.Theme) {
	snapshot := g.client.UISnapshot()
	layout := inGameHUDLayout(g.screenW, g.screenH)
	statusRect := layout.statusRect
	meterRects := topStatusMeterRects(statusRect)
	tnlValue, tnlRange := tnlProgress(snapshot.Character.Level, snapshot.Character.Experience)
	clientui.DrawMeter(screen, meterRects[0], theme, fmt.Sprintf("HP %d/%d", snapshot.Character.HP, snapshot.Character.MaxHP), statRatio(snapshot.Character.HP, snapshot.Character.MaxHP), color.NRGBA{R: 153, G: 56, B: 48, A: 255})
	clientui.DrawMeter(screen, meterRects[1], theme, fmt.Sprintf("TP %d/%d", snapshot.Character.TP, snapshot.Character.MaxTP), statRatio(snapshot.Character.TP, snapshot.Character.MaxTP), color.NRGBA{R: 61, G: 104, B: 168, A: 255})
	clientui.DrawMeter(screen, meterRects[2], theme, fmt.Sprintf("TNL %d", tnlValue), statRatio(tnlRange-tnlValue, tnlRange), color.NRGBA{R: 132, G: 108, B: 44, A: 255})

	menuRect := layout.menuRect
	clientui.DrawPanel(screen, menuRect, theme, clientui.PanelOptions{Accent: theme.Accent})
	for _, button := range hudMenuButtons(layout) {
		clientui.DrawButton(screen, button.Rect, theme, button.Label, g.overlay.activeMenuPanel == button.Panel, false)
	}
	if g.overlay.activeMenuPanel != gameMenuPanelNone {
		g.drawActiveHUDPanel(screen, theme, layout.menuPanelRect, snapshot)
	}
	if g.inventoryDrag.Active {
		g.drawInventoryDrag(screen)
	}
}

func (g *Game) drawActiveHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	switch g.overlay.activeMenuPanel {
	case gameMenuPanelStats:
		g.drawStatsHUDPanel(screen, theme, rect, snapshot)
	case gameMenuPanelInventory:
		g.drawInventoryHUDPanel(screen, theme, rect, snapshot)
	case gameMenuPanelMap:
		g.drawMapHUDPanel(screen, theme, rect, snapshot)
	case gameMenuPanelGuild:
		g.drawPaperdollHUDPanel(screen, theme, rect, snapshot)
	default:
		title, lines := g.activeHUDPanelContent(snapshot)
		clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.AccentMuted})
		for i, line := range lines {
			clientui.DrawTextWrappedCentered(screen, line, image.Rect(rect.Min.X+12, rect.Min.Y+22+i*28, rect.Max.X-12, rect.Min.Y+44+i*28), ternaryColor(i == 0, theme.Text, theme.TextDim))
		}
		clientui.DrawTextCentered(screen, "Click the lit menu tab or Esc to close.", image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Max.X-12, rect.Max.Y-12), theme.TextDim)
	}
}

func (g *Game) activeHUDPanelContent(snapshot game.UISnapshot) (string, []string) {
	switch g.overlay.activeMenuPanel {
	case gameMenuPanelStats:
		return "Stats Ledger", []string{
			fmt.Sprintf("Level %d • HP %d/%d • TP %d/%d", snapshot.Character.Level, snapshot.Character.HP, snapshot.Character.MaxHP, snapshot.Character.TP, snapshot.Character.MaxTP),
			fmt.Sprintf("Weight %d/%d • %d inventory entries", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max, len(snapshot.Inventory)),
			"Expanded combat and carry notes are now live here.",
		}
	case gameMenuPanelInventory:
		return "Inventory Satchel", []string{
			fmt.Sprintf("Satchel entries: %d • Ground nearby: %d", len(snapshot.Inventory), len(snapshot.NearbyItems)),
			"This panel now renders server-authoritative inventory counts.",
			"Equipment slots fall back to raw item IDs when metadata is absent.",
		}
	case gameMenuPanelMap:
		return "Map Scroll", []string{
			fmt.Sprintf("Map %d • Position %d,%d", snapshot.Character.MapID, snapshot.Character.X, snapshot.Character.Y),
			autoWalkStatusLine(g.autoWalk),
			"Route planning stays client-side and sends normal one-step walks.",
		}
	case gameMenuPanelGuild:
		return "Paperdoll Wardrobe", []string{
			"Equipment slots now render authoritative equipped item IDs.",
			"Identity fields come from welcome and paperdoll detail packets.",
			"This panel intentionally blocks world clicks while open.",
		}
	default:
		return "Menu", []string{"No panel selected."}
	}
}

func (g *Game) hoveredTile(snapshot game.UISnapshot) (int, int) {
	camSX, camSY := g.currentCameraScreenPosition(snapshot)
	mx, my := ebiten.CursorPosition()
	return render.ScreenToIso(
		float64(mx)-float64(g.screenW)/2+camSX,
		float64(my)-float64(g.screenH)/2+camSY+render.HalfTileH,
	)
}

func (g *Game) currentCameraScreenPosition(snapshot game.UISnapshot) (float64, float64) {
	camX := float64(snapshot.Character.X)
	camY := float64(snapshot.Character.Y)
	walkOffsetX, walkOffsetY := playerCamOffsetSnapshot(snapshot)
	camSX, camSY := render.IsoToScreen(camX, camY)
	camSX += walkOffsetX
	camSY += walkOffsetY
	return camSX, camSY
}

func playerCamOffsetSnapshot(snapshot game.UISnapshot) (float64, float64) {
	for _, ch := range snapshot.NearbyChars {
		if ch.PlayerID == snapshot.PlayerID && ch.Walking {
			return render.WalkOffset(ch.Direction, ch.WalkProgress())
		}
	}
	return 0, 0
}

func (g *Game) handleHUDPanelClick(rect image.Rectangle, mx, my int) bool {
	if g.overlay.activeMenuPanel != gameMenuPanelInventory {
		return true
	}

	items := g.client.UISnapshot().Inventory
	if len(items) == 0 {
		g.overlay.selectedInventory = 0
		return true
	}

	pageTabs := inventoryPageTabRects(rect)
	for i, tabRect := range pageTabs {
		if pointInRect(mx, my, tabRect) {
			g.overlay.inventoryPage = i
			return true
		}
	}

	gridRect := inventoryGridRect(rect)
	if !pointInRect(mx, my, gridRect) {
		return true
	}

	positions := g.inventoryGridPositions(items)
	cellX := (mx - gridRect.Min.X) / inventoryCellSize
	cellY := (my - gridRect.Min.Y) / inventoryCellSize
	for index, item := range items {
		pos, ok := positions[item.ID]
		if !ok || pos.Page != g.overlay.inventoryPage {
			continue
		}
		if cellX >= pos.X && cellX < pos.X+pos.W && cellY >= pos.Y && cellY < pos.Y+pos.H {
			g.overlay.selectedInventory = index
			itemRect := image.Rect(
				gridRect.Min.X+pos.X*inventoryCellSize+1,
				gridRect.Min.Y+pos.Y*inventoryCellSize+1,
				gridRect.Min.X+(pos.X+pos.W)*inventoryCellSize,
				gridRect.Min.Y+(pos.Y+pos.H)*inventoryCellSize,
			)
			g.inventoryDrag = inventoryDragState{
				Active: true,
				ItemID: item.ID,
				Amount: item.Amount,
				Offset: image.Pt(mx-itemRect.Min.X, my-itemRect.Min.Y),
			}
			return true
		}
	}
	return true
}

func (g *Game) drawStatsHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Stats", Accent: theme.AccentMuted})
	sections := []image.Rectangle{
		image.Rect(rect.Min.X+10, rect.Min.Y+22, rect.Max.X-10, rect.Min.Y+72),
		image.Rect(rect.Min.X+10, rect.Min.Y+76, rect.Max.X-10, rect.Min.Y+126),
		image.Rect(rect.Min.X+10, rect.Min.Y+130, rect.Max.X-10, rect.Max.Y-16),
	}
	for _, section := range sections {
		clientui.DrawInset(screen, section, theme, false)
	}

	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+14, theme.Text, "%s • Lvl %d", fallbackString(snapshot.Character.Name, "Wayfarer"), snapshot.Character.Level)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+28, theme.TextDim, "Class %d • EXP %d", snapshot.Character.ClassID, snapshot.Character.Experience)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+42, theme.TextDim, "HP %d/%d • TP %d/%d", snapshot.Character.HP, snapshot.Character.MaxHP, snapshot.Character.TP, snapshot.Character.MaxTP)

	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+14, theme.Text, "Stats %d/%d pts", snapshot.Character.StatPoints, snapshot.Character.SkillPoints)
	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+28, theme.TextDim, "STR %d  INT %d  WIS %d", snapshot.Character.BaseStats.Str, snapshot.Character.BaseStats.Int, snapshot.Character.BaseStats.Wis)
	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+42, theme.TextDim, "AGI %d  CON %d  CHA %d", snapshot.Character.BaseStats.Agi, snapshot.Character.BaseStats.Con, snapshot.Character.BaseStats.Cha)

	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+14, theme.Text, "Combat")
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+28, theme.TextDim, "Dmg %d-%d  Hit %d  Eva %d  Arm %d", snapshot.Character.CombatStats.MinDamage, snapshot.Character.CombatStats.MaxDamage, snapshot.Character.CombatStats.Accuracy, snapshot.Character.CombatStats.Evade, snapshot.Character.CombatStats.Armor)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+42, theme.TextDim, "Weight %d/%d  Karma %d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max, snapshot.Character.Karma)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+56, theme.TextDim, "Bag %d  Players %d  NPCs %d", len(snapshot.Inventory), max(0, len(snapshot.NearbyChars)-1), len(snapshot.NearbyNpcs))
}

func (g *Game) drawInventoryHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Inventory", Accent: theme.AccentMuted})
	items := snapshot.Inventory
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Weight %d/%d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max)
	itemsLabel := fmt.Sprintf("%d items", len(items))
	clientui.DrawText(screen, itemsLabel, rect.Max.X-12-clientui.MeasureText(itemsLabel), rect.Min.Y+24, theme.TextDim)

	if len(items) == 0 {
		clientui.DrawTextWrappedCentered(screen, "No inventory items reported yet.", image.Rect(rect.Min.X+12, rect.Min.Y+58, rect.Max.X-12, rect.Max.Y-86), theme.Text)
		return
	}

	positions := g.inventoryGridPositions(items)
	g.overlay.inventoryPage = clampInt(g.overlay.inventoryPage, 0, inventoryGridPages-1)
	selected := clampInt(g.overlay.selectedInventory, 0, len(items)-1)
	selectedPos, ok := positions[items[selected].ID]
	if ok && selectedPos.Page != g.overlay.inventoryPage {
		for i, item := range items {
			pos, ok := positions[item.ID]
			if ok && pos.Page == g.overlay.inventoryPage {
				selected = i
				break
			}
		}
	}
	g.overlay.selectedInventory = selected

	for i, tabRect := range inventoryPageTabRects(rect) {
		clientui.DrawButton(screen, tabRect, theme, fmt.Sprintf("Pack %d", i+1), g.overlay.inventoryPage == i, false)
	}

	gridRect := inventoryGridRect(rect)
	clientui.DrawInset(screen, gridRect, theme, false)
	mx, my := ebiten.CursorPosition()
	tooltipItemID := 0
	tooltipAmount := 0
	for row := 0; row <= inventoryGridRows; row++ {
		y := gridRect.Min.Y + row*inventoryCellSize
		ebitenutil.DrawRect(screen, float64(gridRect.Min.X), float64(y), float64(gridRect.Dx()), 1, colorize(theme.BorderMid, 64))
	}
	for col := 0; col <= inventoryGridCols; col++ {
		x := gridRect.Min.X + col*inventoryCellSize
		ebitenutil.DrawRect(screen, float64(x), float64(gridRect.Min.Y), 1, float64(gridRect.Dy()), colorize(theme.BorderMid, 64))
	}

	for index, item := range items {
		pos, ok := positions[item.ID]
		if !ok || pos.Page != g.overlay.inventoryPage {
			continue
		}
		entry := image.Rect(
			gridRect.Min.X+pos.X*inventoryCellSize+1,
			gridRect.Min.Y+pos.Y*inventoryCellSize+1,
			gridRect.Min.X+(pos.X+pos.W)*inventoryCellSize,
			gridRect.Min.Y+(pos.Y+pos.H)*inventoryCellSize,
		)
		active := index == selected
		if active {
			ebitenutil.DrawRect(screen, float64(entry.Min.X), float64(entry.Min.Y), float64(entry.Dx()), float64(entry.Dy()), colorize(theme.AccentMuted, 84))
		}
		g.drawItemIcon(screen, entry, item.ID, item.Amount)
		if pointInRect(mx, my, entry) {
			tooltipItemID = item.ID
			tooltipAmount = item.Amount
			ebitenutil.DrawRect(screen, float64(entry.Min.X), float64(entry.Min.Y), float64(entry.Dx()), float64(entry.Dy()), colorize(theme.Accent, 38))
		}
	}

	if tooltipItemID != 0 {
		g.drawItemTooltip(screen, theme, mx, my, tooltipItemID, tooltipAmount)
	}
}

func (g *Game) drawEquipmentSummary(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawInset(screen, rect, theme, false)
	clientui.DrawText(screen, "Paperdoll", rect.Min.X+8, rect.Min.Y+14, theme.Text)
	slots := []struct {
		label  string
		itemID int
		icon   image.Rectangle
	}{
		{"Weapon", snapshot.Equipment.Weapon, image.Rect(rect.Min.X+8, rect.Min.Y+24, rect.Min.X+36, rect.Min.Y+52)},
		{"Shield", snapshot.Equipment.Shield, image.Rect(rect.Min.X+42, rect.Min.Y+24, rect.Min.X+70, rect.Min.Y+52)},
		{"Armor", snapshot.Equipment.Armor, image.Rect(rect.Min.X+76, rect.Min.Y+24, rect.Min.X+104, rect.Min.Y+52)},
		{"Hat", snapshot.Equipment.Hat, image.Rect(rect.Min.X+110, rect.Min.Y+24, rect.Min.X+138, rect.Min.Y+52)},
		{"Boots", snapshot.Equipment.Boots, image.Rect(rect.Min.X+144, rect.Min.Y+24, rect.Min.X+172, rect.Min.Y+52)},
		{"Gloves", snapshot.Equipment.Gloves, image.Rect(rect.Min.X+178, rect.Min.Y+24, rect.Min.X+206, rect.Min.Y+52)},
		{"Belt", snapshot.Equipment.Belt, image.Rect(rect.Min.X+212, rect.Min.Y+24, rect.Min.X+240, rect.Min.Y+52)},
		{"Neck", snapshot.Equipment.Necklace, image.Rect(rect.Min.X+246, rect.Min.Y+24, rect.Min.X+274, rect.Min.Y+52)},
	}
	for _, slot := range slots {
		g.drawItemSlot(screen, theme, slot.icon, slot.label, slot.itemID)
	}
	clientui.DrawTextf(screen, rect.Min.X+8, rect.Min.Y+66, theme.TextDim, "Rings %s", paperdollPair(snapshot.Equipment.Ring))
	clientui.DrawTextf(screen, rect.Min.X+8, rect.Min.Y+80, theme.TextDim, "Armlets %s", paperdollPair(snapshot.Equipment.Armlet))
	clientui.DrawTextf(screen, rect.Min.X+8, rect.Min.Y+94, theme.TextDim, "Bracers %s", paperdollPair(snapshot.Equipment.Bracer))
}

func (g *Game) drawMapHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Map", Accent: theme.AccentMuted})
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Map %d", snapshot.Character.MapID)
	clientui.DrawTextf(screen, rect.Max.X-86, rect.Min.Y+24, theme.TextDim, "%d,%d", snapshot.Character.X, snapshot.Character.Y)
	mapRect := image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Max.X-10, rect.Max.Y-26)
	clientui.DrawInset(screen, mapRect, theme, true)
	g.drawMinimap(screen, theme, mapRect, snapshot)
	if g.autoWalk.Active {
		clientui.DrawTextCentered(screen, autoWalkStatusLine(g.autoWalk), image.Rect(rect.Min.X+12, rect.Max.Y-24, rect.Max.X-12, rect.Max.Y-10), theme.TextDim)
	}
}

func (g *Game) drawPaperdollHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Paperdoll", Accent: theme.AccentMuted})
	name := fallbackString(snapshot.Character.Name, "Wayfarer")
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.Text, "%s • C%d", truncateLabel(name, 14), snapshot.Character.ClassID)
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+40, theme.TextDim, "%s • %s", truncateLabel(fallbackString(snapshot.Character.Title, "No title"), 10), truncateLabel(paperdollGuildLine(snapshot), 10))

	leftCol := image.Rect(rect.Min.X+10, rect.Min.Y+54, rect.Min.X+132, rect.Max.Y-14)
	rightCol := image.Rect(leftCol.Max.X+6, rect.Min.Y+54, rect.Max.X-10, rect.Max.Y-14)
	clientui.DrawInset(screen, leftCol, theme, false)
	clientui.DrawInset(screen, rightCol, theme, false)

	mx, my := ebiten.CursorPosition()
	tooltipItemID := 0
	slotRects := []struct {
		label  string
		itemID int
		rect   image.Rectangle
	}{
		{"Hat", snapshot.Equipment.Hat, image.Rect(leftCol.Min.X+42, leftCol.Min.Y+8, leftCol.Min.X+82, leftCol.Min.Y+44)},
		{"Neck", snapshot.Equipment.Necklace, image.Rect(leftCol.Min.X+84, leftCol.Min.Y+12, leftCol.Min.X+118, leftCol.Min.Y+36)},
		{"Weapon", snapshot.Equipment.Weapon, image.Rect(leftCol.Min.X+6, leftCol.Min.Y+52, leftCol.Min.X+40, leftCol.Min.Y+110)},
		{"Armor", snapshot.Equipment.Armor, image.Rect(leftCol.Min.X+42, leftCol.Min.Y+52, leftCol.Min.X+82, leftCol.Min.Y+110)},
		{"Shield", snapshot.Equipment.Shield, image.Rect(leftCol.Min.X+84, leftCol.Min.Y+52, leftCol.Min.X+118, leftCol.Min.Y+110)},
		{"Gloves", snapshot.Equipment.Gloves, image.Rect(leftCol.Min.X+6, leftCol.Min.Y+118, leftCol.Min.X+40, leftCol.Min.Y+154)},
		{"Belt", snapshot.Equipment.Belt, image.Rect(leftCol.Min.X+42, leftCol.Min.Y+118, leftCol.Min.X+82, leftCol.Min.Y+146)},
		{"Boots", snapshot.Equipment.Boots, image.Rect(leftCol.Min.X+42, leftCol.Min.Y+154, leftCol.Min.X+82, leftCol.Min.Y+190)},
		{"", snapshot.Equipment.Accessory, image.Rect(leftCol.Min.X+10, leftCol.Min.Y+156, leftCol.Min.X+34, leftCol.Min.Y+180)},
	}
	for _, slot := range slotRects {
		g.drawItemSlot(screen, theme, slot.rect, slot.label, slot.itemID)
		if slot.itemID > 0 && pointInRect(mx, my, slot.rect) {
			tooltipItemID = slot.itemID
		}
	}

	clientui.DrawText(screen, "Guild", rightCol.Min.X+8, rightCol.Min.Y+14, theme.Text)
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+30, theme.TextDim, "%s", truncateLabel(fallbackString(snapshot.Character.GuildName, "No guild"), 12))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+44, theme.TextDim, "%s", truncateLabel(fallbackString(snapshot.Character.GuildRank, "No rank"), 12))
	clientui.DrawText(screen, "Jewelry", rightCol.Min.X+8, rightCol.Min.Y+74, theme.Text)
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+90, theme.TextDim, "Rings %s", paperdollPair(snapshot.Equipment.Ring))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+104, theme.TextDim, "Arm %s", paperdollPair(snapshot.Equipment.Armlet))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+118, theme.TextDim, "Brac %s", paperdollPair(snapshot.Equipment.Bracer))
	clientui.DrawText(screen, "Loadout", rightCol.Min.X+8, rightCol.Min.Y+148, theme.Text)
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+164, theme.TextDim, "%d equipped", equippedItemCount(snapshot.Equipment))
	if tooltipItemID != 0 {
		g.drawItemTooltip(screen, theme, mx, my, tooltipItemID, 1)
	}
}

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
			fill := minimapTileColor(g.tileStateAt(tileX, tileY), theme)
			ebitenutil.DrawRect(screen, float64(originX+col*cellSize), float64(originY+row*cellSize), float64(cellSize-1), float64(cellSize-1), fill)
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
		marker := color.NRGBA{R: 116, G: 192, B: 255, A: 255}
		if ch.PlayerID == snapshot.PlayerID {
			continue
		}
		g.drawMinimapMarker(screen, originX, originY, cols, rows, cellSize, halfCols, halfRows, centerX, centerY, float64(ch.X), float64(ch.Y), marker)
	}

	playerPX := originX + halfCols*cellSize
	playerPY := originY + halfRows*cellSize
	ebitenutil.DrawRect(screen, float64(playerPX-1), float64(playerPY-1), float64(cellSize+1), float64(cellSize+1), theme.Accent)
}

func (g *Game) drawMinimapMarker(screen *ebiten.Image, originX, originY, cols, rows, cellSize, halfCols, halfRows int, centerX, centerY, worldX, worldY float64, fill color.Color) {
	col := int(worldX - centerX + float64(halfCols))
	row := int(worldY - centerY + float64(halfRows))
	if col < 0 || row < 0 || col >= cols || row >= rows {
		return
	}
	px := originX + col*cellSize + 1
	py := originY + row*cellSize + 1
	ebitenutil.DrawRect(screen, float64(px), float64(py), float64(max(1, cellSize-2)), float64(max(1, cellSize-2)), fill)
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

func minimapTileColor(tileState int, theme clientui.Theme) color.Color {
	switch tileState {
	case 2:
		return color.NRGBA{R: 38, G: 34, B: 42, A: 255}
	case 1:
		return colorize(theme.AccentMuted, 120)
	default:
		return color.NRGBA{R: 91, G: 108, B: 88, A: 255}
	}
}

type inventoryGridPos struct {
	Page int
	X    int
	Y    int
	W    int
	H    int
}

func inventoryPageTabRects(rect image.Rectangle) []image.Rectangle {
	return []image.Rectangle{
		image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Min.X+74, rect.Min.Y+56),
		image.Rect(rect.Min.X+80, rect.Min.Y+36, rect.Min.X+144, rect.Min.Y+56),
	}
}

func inventoryGridRect(rect image.Rectangle) image.Rectangle {
	width := inventoryGridCols * inventoryCellSize
	height := inventoryGridRows * inventoryCellSize
	x := rect.Min.X + (rect.Dx()-width)/2
	return image.Rect(x, rect.Min.Y+62, x+width, rect.Min.Y+62+height)
}

func (g *Game) inventoryGridPositions(items []game.InventoryItem) map[int]inventoryGridPos {
	positions := make(map[int]inventoryGridPos, len(items))
	changed := false
	occupied := make([][][]bool, inventoryGridPages)
	for page := 0; page < inventoryGridPages; page++ {
		occupied[page] = make([][]bool, inventoryGridRows)
		for row := 0; row < inventoryGridRows; row++ {
			occupied[page][row] = make([]bool, inventoryGridCols)
		}
	}

	validItems := make(map[int]bool, len(items))
	for _, item := range items {
		validItems[item.ID] = true
	}
	for itemID := range g.inventoryLayout {
		if !validItems[itemID] {
			delete(g.inventoryLayout, itemID)
			changed = true
		}
	}

	for _, item := range items {
		stored, ok := g.inventoryLayout[item.ID]
		if !ok {
			continue
		}
		w, h := g.inventoryItemSize(item.ID)
		if stored.Page < 0 || stored.Page >= inventoryGridPages || stored.X < 0 || stored.Y < 0 || stored.X+w > inventoryGridCols || stored.Y+h > inventoryGridRows {
			delete(g.inventoryLayout, item.ID)
			changed = true
			continue
		}
		if !inventoryFits(occupied[stored.Page], stored.X, stored.Y, w, h) {
			delete(g.inventoryLayout, item.ID)
			changed = true
			continue
		}
		inventoryOccupy(occupied[stored.Page], stored.X, stored.Y, w, h)
		positions[item.ID] = inventoryGridPos{Page: stored.Page, X: stored.X, Y: stored.Y, W: w, H: h}
	}

	for _, item := range items {
		if _, ok := positions[item.ID]; ok {
			continue
		}
		w, h := g.inventoryItemSize(item.ID)
		for page := 0; page < inventoryGridPages; page++ {
			placed := false
			for y := 0; y <= inventoryGridRows-h; y++ {
				for x := 0; x <= inventoryGridCols-w; x++ {
					if !inventoryFits(occupied[page], x, y, w, h) {
						continue
					}
					inventoryOccupy(occupied[page], x, y, w, h)
					positions[item.ID] = inventoryGridPos{Page: page, X: x, Y: y, W: w, H: h}
					g.inventoryLayout[item.ID] = storedInventoryPos{Page: page, X: x, Y: y}
					changed = true
					placed = true
					break
				}
				if placed {
					break
				}
			}
			if placed {
				break
			}
		}
	}
	if changed {
		g.saveInventoryLayout()
	}

	return positions
}

func (g *Game) saveInventoryLayout() {
	if err := saveInventoryLayout(g.inventoryLayoutPath, g.inventoryLayout); err != nil {
		g.overlay.statusMessage = "Inventory layout save failed"
	}
}

func (g *Game) moveInventoryItem(itemID, page, x, y int, items []game.InventoryItem) bool {
	w, h := g.inventoryItemSize(itemID)
	if page < 0 || page >= inventoryGridPages || x < 0 || y < 0 || x+w > inventoryGridCols || y+h > inventoryGridRows {
		return false
	}

	layout := make(map[int]storedInventoryPos, len(g.inventoryLayout))
	for id, pos := range g.inventoryLayout {
		if id == itemID {
			continue
		}
		layout[id] = pos
	}
	layout[itemID] = storedInventoryPos{Page: page, X: x, Y: y}

	old := g.inventoryLayout
	g.inventoryLayout = layout
	positions := g.inventoryGridPositions(items)
	pos, ok := positions[itemID]
	if !ok || pos.Page != page || pos.X != x || pos.Y != y {
		g.inventoryLayout = old
		return false
	}
	g.saveInventoryLayout()
	return true
}

func (g *Game) finishInventoryDrag() {
	if !g.inventoryDrag.Active {
		return
	}
	defer func() {
		g.inventoryDrag = inventoryDragState{}
	}()

	layout := inGameHUDLayout(g.screenW, g.screenH)
	panelRect := layout.menuPanelRect
	if g.overlay.activeMenuPanel != gameMenuPanelInventory {
		return
	}

	mx, my := ebiten.CursorPosition()
	for i, tabRect := range inventoryPageTabRects(panelRect) {
		if pointInRect(mx, my, tabRect) {
			g.overlay.inventoryPage = i
			return
		}
	}

	gridRect := inventoryGridRect(panelRect)
	if !pointInRect(mx, my, gridRect) {
		return
	}

	x := (mx - g.inventoryDrag.Offset.X - gridRect.Min.X) / inventoryCellSize
	y := (my - g.inventoryDrag.Offset.Y - gridRect.Min.Y) / inventoryCellSize
	items := g.client.UISnapshot().Inventory
	if g.moveInventoryItem(g.inventoryDrag.ItemID, g.overlay.inventoryPage, x, y, items) {
		g.overlay.statusMessage = "Inventory layout saved"
	}
}

func (g *Game) drawInventoryDrag(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	rect := image.Rect(mx-g.inventoryDrag.Offset.X, my-g.inventoryDrag.Offset.Y, mx-g.inventoryDrag.Offset.X+48, my-g.inventoryDrag.Offset.Y+48)
	g.drawItemIcon(screen, rect, g.inventoryDrag.ItemID, g.inventoryDrag.Amount)
}

func inventoryFits(grid [][]bool, x, y, w, h int) bool {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if grid[yy][xx] {
				return false
			}
		}
	}
	return true
}

func inventoryOccupy(grid [][]bool, x, y, w, h int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			grid[yy][xx] = true
		}
	}
}

func (g *Game) inventoryItemSize(itemID int) (int, int) {
	if g.itemDB == nil {
		return 1, 1
	}
	item, ok := g.itemDB.Get(itemID)
	if !ok {
		return 1, 1
	}
	switch item.Size {
	case 1:
		return 1, 2
	case 2:
		return 1, 3
	case 3:
		return 1, 4
	case 4:
		return 2, 1
	case 5:
		return 2, 2
	case 6:
		return 2, 3
	case 7:
		return 2, 4
	default:
		return 1, 1
	}
}

func (g *Game) itemLabel(itemID int) string {
	if g.itemDB == nil {
		return ""
	}
	return g.itemDB.Name(itemID)
}

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

	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Accent: theme.AccentMuted, Fill: colorize(theme.PanelFill, 248)})
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
	iw := bounds.Dx()
	ih := bounds.Dy()
	if iw <= 0 || ih <= 0 {
		return
	}

	scaleX := float64(rect.Dx()) / float64(iw)
	scaleY := float64(rect.Dy()) / float64(ih)
	scale := minFloat(scaleX, scaleY)
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

func paperdollValue(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func truncateLabel(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func paperdollPair(values []int) string {
	left, right := 0, 0
	if len(values) > 0 {
		left = values[0]
	}
	if len(values) > 1 {
		right = values[1]
	}
	return fmt.Sprintf("%d/%d", left, right)
}

func equippedItemCount(equipment server.EquipmentPaperdoll) int {
	count := 0
	values := []int{equipment.Boots, equipment.Accessory, equipment.Gloves, equipment.Belt, equipment.Armor, equipment.Necklace, equipment.Hat, equipment.Shield, equipment.Weapon}
	for _, value := range values {
		if value > 0 {
			count++
		}
	}
	for _, group := range [][]int{equipment.Ring, equipment.Armlet, equipment.Bracer} {
		for _, value := range group {
			if value > 0 {
				count++
			}
		}
	}
	return count
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func paperdollGuildLine(snapshot game.UISnapshot) string {
	if strings.TrimSpace(snapshot.Character.GuildName) == "" {
		return "No guild"
	}
	if strings.TrimSpace(snapshot.Character.GuildTag) == "" {
		return snapshot.Character.GuildName
	}
	return fmt.Sprintf("[%s] %s", snapshot.Character.GuildTag, snapshot.Character.GuildName)
}

func (g *Game) handleCharacterSelectClick() bool {
	layout := characterSelectDialogLayout(g.screenW, g.screenH)
	mx, my := ebiten.CursorPosition()
	if !pointInRect(mx, my, layout.dialog) {
		return false
	}
	if pointInRect(mx, my, layout.joinButtonRect) {
		g.selectCurrentCharacter()
		return true
	}
	if pointInRect(mx, my, layout.previewArtRect) {
		if mx < layout.previewArtRect.Min.X+layout.previewArtRect.Dx()/2 {
			g.overlay.previewDirection = (g.overlay.previewDirection + 3) % 4
		} else {
			g.overlay.previewDirection = (g.overlay.previewDirection + 1) % 4
		}
		return true
	}
	if !pointInRect(mx, my, layout.rosterRect) {
		return true
	}
	visibleRows := characterSelectVisibleRows(layout.rosterRect)
	start, end := characterSelectVisibleRange(len(g.client.Characters), g.overlay.rosterScroll, visibleRows)
	for rowIndex, i := 0, start; i < end; i, rowIndex = i+1, rowIndex+1 {
		row := image.Rect(layout.rosterRect.Min.X+10, layout.rosterRect.Min.Y+characterSelectRosterTopPadding+rowIndex*characterSelectRosterRowHeight, layout.rosterRect.Max.X-10, layout.rosterRect.Min.Y+characterSelectRosterTopPadding+48+rowIndex*characterSelectRosterRowHeight)
		if pointInRect(mx, my, row) {
			g.overlay.selectedCharacter = i
			g.normalizeCharacterSelectState()
			return true
		}
	}
	return true
}

func (g *Game) selectCurrentCharacter() {
	if g.overlay.selectingCharacter || len(g.client.Characters) == 0 {
		return
	}
	charID := g.client.Characters[g.overlay.selectedCharacter].Id
	g.overlay.selectingCharacter = true
	g.overlay.statusMessage = "Entering the world..."
	g.sendSelectCharacter(charID)
}

func (g *Game) handleInGameOverlayClick() bool {
	layout := inGameHUDLayout(g.screenW, g.screenH)
	mx, my := ebiten.CursorPosition()
	chatPanelRect, chatInputRect := g.chatRects()
	if pointInRect(mx, my, chatInputRect) {
		g.chat.Typing = true
		return true
	}
	for _, button := range hudMenuButtons(layout) {
		if !pointInRect(mx, my, button.Rect) {
			continue
		}
		if g.overlay.activeMenuPanel == button.Panel {
			g.overlay.activeMenuPanel = gameMenuPanelNone
		} else {
			g.overlay.activeMenuPanel = button.Panel
		}
		return true
	}
	if g.overlay.activeMenuPanel != gameMenuPanelNone {
		if pointInRect(mx, my, layout.menuPanelRect) && g.handleHUDPanelClick(layout.menuPanelRect, mx, my) {
			return true
		}
		return true
	}
	return pointInRect(mx, my, chatPanelRect) ||
		pointInRect(mx, my, layout.statusRect) ||
		pointInRect(mx, my, layout.menuRect) ||
		pointInRect(mx, my, layout.menuPanelRect) ||
		pointInRect(mx, my, layout.infoRect)
}

func characterSelectDialogLayout(sw, sh int) characterSelectLayout {
	dialog := centeredRect(556, 308, sw, sh)
	previewRect := image.Rect(dialog.Min.X+24, dialog.Min.Y+44, dialog.Min.X+280, dialog.Max.Y-54)
	previewArtRect := image.Rect(previewRect.Min.X+14, previewRect.Min.Y+16, previewRect.Max.X-14, previewRect.Min.Y+134)
	namePlateRect := image.Rect(previewRect.Min.X+14, previewArtRect.Max.Y+8, previewRect.Max.X-14, previewArtRect.Max.Y+42)
	metaRect := image.Rect(previewRect.Min.X+14, namePlateRect.Max.Y+10, previewRect.Max.X-14, previewRect.Max.Y-14)
	rosterRect := image.Rect(previewRect.Max.X+16, dialog.Min.Y+44, dialog.Max.X-24, dialog.Max.Y-54)
	joinButtonRect := image.Rect(dialog.Max.X-148, dialog.Max.Y-44, dialog.Max.X-24, dialog.Max.Y-18)
	return characterSelectLayout{
		dialog:         dialog,
		previewRect:    previewRect,
		previewArtRect: previewArtRect,
		namePlateRect:  namePlateRect,
		metaRect:       metaRect,
		rosterRect:     rosterRect,
		joinButtonRect: joinButtonRect,
	}
}

func inGameHUDLayout(sw, sh int) gameHUDLayout {
	statusW := 390
	menuHeight := 74
	const edgeInset = 6
	return gameHUDLayout{
		statusRect:    image.Rect((sw-statusW)/2, edgeInset, (sw-statusW)/2+statusW, edgeInset+16),
		menuRect:      image.Rect(sw-158, sh-menuHeight-edgeInset, sw-10, sh-edgeInset),
		menuPanelRect: image.Rect(sw-230, sh-menuHeight-254-edgeInset, sw-10, sh-menuHeight-8-edgeInset),
		infoRect:      image.Rectangle{},
	}
}

func topStatusMeterRects(rect image.Rectangle) [3]image.Rectangle {
	const gap = 8
	inner := image.Rect(rect.Min.X+12, rect.Min.Y, rect.Max.X-12, rect.Max.Y)
	width := (inner.Dx() - gap*2) / 3
	top := rect.Min.Y
	return [3]image.Rectangle{
		image.Rect(inner.Min.X, top, inner.Min.X+width, top+16),
		image.Rect(inner.Min.X+width+gap, top, inner.Min.X+width*2+gap, top+16),
		image.Rect(inner.Min.X+(width+gap)*2, top, inner.Max.X, top+16),
	}
}

func hudMenuButtons(layout gameHUDLayout) []hudMenuButton {
	panels := []struct {
		panel gameMenuPanel
		label string
	}{
		{panel: gameMenuPanelStats, label: "Stats"},
		{panel: gameMenuPanelInventory, label: "Bag"},
		{panel: gameMenuPanelMap, label: "Map"},
		{panel: gameMenuPanelGuild, label: "Doll"},
	}
	buttons := make([]hudMenuButton, 0, len(panels))
	for i, entry := range panels {
		col := i % 2
		row := i / 2
		buttons = append(buttons, hudMenuButton{
			Panel: entry.panel,
			Label: entry.label,
			Rect:  image.Rect(layout.menuRect.Min.X+12+col*66, layout.menuRect.Min.Y+16+row*26, layout.menuRect.Min.X+72+col*66, layout.menuRect.Min.Y+38+row*26),
		})
	}
	return buttons
}

func autoWalkStatusLine(plan autoWalkPlan) string {
	if !plan.Active {
		return "Auto-walk idle."
	}
	steps := len(plan.Directions)
	if steps == 0 {
		return "Auto-walk finishing interaction."
	}
	return fmt.Sprintf("Auto-walk queued: %d step(s)", steps)
}

func pointInRect(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x < rect.Max.X && y >= rect.Min.Y && y < rect.Max.Y
}

func centeredRect(width, height, sw, sh int) image.Rectangle {
	x := (sw - width) / 2
	y := (sh - height) / 2
	return image.Rect(x, y, x+width, y+height)
}

func drawPulseBar(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, ticks int) {
	clientui.DrawInset(screen, rect, theme, false)
	segmentW := 12
	for i := 0; i < 5; i++ {
		alpha := uint8(50)
		if i == (ticks/12)%5 {
			alpha = 180
		}
		fill := colorize(theme.Accent, alpha)
		r := image.Rect(rect.Min.X+6+i*(segmentW+6), rect.Min.Y+4, rect.Min.X+6+i*(segmentW+6)+segmentW, rect.Max.Y-4)
		ebitenutil.DrawRect(screen, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), fill)
	}
}

func dotPulse(prefix string, ticks int) string {
	return prefix + strings.Repeat(".", (ticks/20)%4)
}

func statRatio(value, maxValue int) float64 {
	if maxValue <= 0 {
		return 1
	}
	if value <= 0 {
		return 0
	}
	return float64(value) / float64(maxValue)
}

func expForLevel(level int) int {
	if level <= 0 {
		return 0
	}
	return int(float64(level*level*level) * 133.1)
}

func tnlProgress(level, experience int) (remaining int, rangeValue int) {
	currentLevelExp := expForLevel(level)
	nextLevelExp := expForLevel(level + 1)
	if nextLevelExp <= currentLevelExp {
		return 0, 1
	}
	progressRange := nextLevelExp - currentLevelExp
	remaining = nextLevelExp - experience
	if remaining < 0 {
		remaining = 0
	}
	if remaining > progressRange {
		remaining = progressRange
	}
	return remaining, progressRange
}

func ternaryString(ok bool, left, right string) string {
	if ok {
		return left
	}
	return right
}

func ternaryColor(ok bool, left, right color.Color) color.Color {
	if ok {
		return left
	}
	return right
}

func colorize(base color.NRGBA, alpha uint8) color.NRGBA {
	base.A = alpha
	return base
}
