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
	chatPanelRect image.Rectangle
	chatInputRect image.Rectangle
}

type hudMenuButton struct {
	Panel gameMenuPanel
	Label string
	Rect  image.Rectangle
}

const (
	loginFocusUsername = iota
	loginFocusPassword
	characterSelectRosterRowHeight   = 56
	characterSelectRosterTopPadding  = 26
	characterSelectRosterBottomSpace = 12
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
		g.drawGameHUD(screen, theme)
		g.drawChat(screen, theme)
	}
}

func (g *Game) drawConnectingDialog(screen *ebiten.Image, theme clientui.Theme) {
	rect := centeredRect(360, 128)
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
	clientui.DrawTextCentered(screen, "Endless Offline Native Client", image.Rect(0, screenHeight-38, screenWidth, screenHeight-18), theme.TextDim)
}

func (g *Game) drawLoginDialog(screen *ebiten.Image, theme clientui.Theme) {
	rect := centeredRect(404, 236)
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
	layout := characterSelectDialogLayout()
	rect := layout.dialog
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Character Hall", Accent: theme.Accent})

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
	clientui.DrawText(screen, characterSelectEquipmentLine(selected), metaRect.Min.X+12, metaRect.Min.Y+40, theme.Text)
	if selected.Admin > 0 {
		clientui.DrawTextf(screen, metaRect.Min.X+12, metaRect.Min.Y+62, theme.Accent, "Admin Rank %d", selected.Admin)
	}
	clientui.DrawTextCentered(screen, characterSelectPreviewDirectionLabel(g.overlay.previewDirection), image.Rect(previewArtRect.Min.X+8, previewArtRect.Max.Y-24, previewArtRect.Max.X-8, previewArtRect.Max.Y-8), theme.TextDim)

	clientui.DrawInset(screen, rosterRect, theme, false)
	clientui.DrawText(screen, "Roster", rosterRect.Min.X+12, rosterRect.Min.Y+18, theme.TextDim)
	visibleRows := characterSelectVisibleRows(rosterRect)
	start, end := characterSelectVisibleRange(len(g.client.Characters), g.overlay.rosterScroll, visibleRows)

	for rowIndex, i := 0, start; i < end; i, rowIndex = i+1, rowIndex+1 {
		ch := g.client.Characters[i]
		row := image.Rect(rosterRect.Min.X+10, rosterRect.Min.Y+characterSelectRosterTopPadding+rowIndex*characterSelectRosterRowHeight, rosterRect.Max.X-10, rosterRect.Min.Y+characterSelectRosterTopPadding+48+rowIndex*characterSelectRosterRowHeight)
		active := i == selectedIndex
		drawCharacterSelectionCard(screen, row, theme, i+1, ch, active)
	}
	if start > 0 {
		clientui.DrawText(screen, "^ more above ^", rosterRect.Min.X+12, rosterRect.Max.Y-24, theme.TextDim)
	}
	if end < len(g.client.Characters) {
		label := "v more below v"
		clientui.DrawText(screen, label, rosterRect.Max.X-clientui.MeasureText(label)-12, rosterRect.Max.Y-24, theme.TextDim)
	}

	status := "Arrow keys choose • Click hero selects • Preview rotates"
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
	clientui.DrawTextCentered(screen, "Choose a hero to enter the map.", image.Rect(rect.Min.X+22, rect.Min.Y+22, rect.Max.X-22, rect.Min.Y+40), theme.TextDim)
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

func characterSelectPreviewDirectionLabel(dir int) string {
	return [...]string{"Facing South", "Facing West", "Facing North", "Facing East"}[clampInt(dir, 0, 3)]
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

func drawCharacterSelectionCard(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, slot int, ch server.CharacterSelectionListEntry, active bool) {
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
	badge := image.Rect(rect.Min.X+8, rect.Min.Y+9, rect.Min.X+34, rect.Min.Y+39)
	clientui.DrawInset(screen, badge, theme, active)
	clientui.DrawTextf(screen, badge.Min.X+8, badge.Min.Y+19, ternaryColor(active, theme.Accent, theme.TextDim), "%d", slot)
	clientui.DrawText(screen, ch.Name, rect.Min.X+44, rect.Min.Y+18, nameColor)
	clientui.DrawTextf(screen, rect.Min.X+44, rect.Min.Y+36, metaColor, "LVL %d • %s", ch.Level, characterSelectSummaryLine(ch))
	if ch.Admin > 0 {
		adminLabel := fmt.Sprintf("GM%d", ch.Admin)
		clientui.DrawText(screen, adminLabel, rect.Max.X-clientui.MeasureText(adminLabel)-10, rect.Min.Y+18, theme.Accent)
	}
}

func characterSelectSummaryLine(entry server.CharacterSelectionListEntry) string {
	gender := "Adventurer"
	if entry.Gender == 1 {
		gender = "Male"
	}
	if entry.Gender == 0 {
		gender = "Female"
	}
	return fmt.Sprintf("%s • Skin %d • Hair %d/%d", gender, entry.Skin, entry.HairStyle, entry.HairColor)
}

func characterSelectEquipmentLine(entry server.CharacterSelectionListEntry) string {
	return fmt.Sprintf("Gear A:%d B:%d H:%d S:%d W:%d", entry.Equipment.Armor, entry.Equipment.Boots, entry.Equipment.Hat, entry.Equipment.Shield, entry.Equipment.Weapon)
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
	layout := inGameHUDLayout()
	statusRect := layout.statusRect
	clientui.DrawPanel(screen, statusRect, theme, clientui.PanelOptions{Title: "Status", Accent: theme.Accent})

	name := snapshot.Character.Name
	if name == "" {
		name = "Wayfarer"
	}
	clientui.DrawText(screen, name, statusRect.Min.X+14, statusRect.Min.Y+26, theme.Text)
	clientui.DrawTextf(screen, statusRect.Min.X+14, statusRect.Min.Y+42, theme.TextDim, "Map %d  Pos %d,%d", snapshot.Character.MapID, snapshot.Character.X, snapshot.Character.Y)
	clientui.DrawMeter(screen, image.Rect(statusRect.Min.X+14, statusRect.Min.Y+52, statusRect.Max.X-14, statusRect.Min.Y+68), theme, "HP", statRatio(snapshot.Character.HP, snapshot.Character.MaxHP), color.NRGBA{R: 153, G: 56, B: 48, A: 255})
	clientui.DrawMeter(screen, image.Rect(statusRect.Min.X+14, statusRect.Min.Y+74, statusRect.Max.X-14, statusRect.Min.Y+90), theme, "TP", statRatio(snapshot.Character.TP, snapshot.Character.MaxTP), color.NRGBA{R: 61, G: 104, B: 168, A: 255})

	menuRect := layout.menuRect
	clientui.DrawPanel(screen, menuRect, theme, clientui.PanelOptions{Title: "Menu", Accent: theme.Accent})
	for _, button := range hudMenuButtons(layout) {
		clientui.DrawButton(screen, button.Rect, theme, button.Label, g.overlay.activeMenuPanel == button.Panel, false)
	}
	if g.overlay.activeMenuPanel != gameMenuPanelNone {
		g.drawActiveHUDPanel(screen, theme, layout.menuPanelRect, snapshot)
	}

	infoRect := layout.infoRect
	clientui.DrawPanel(screen, infoRect, theme, clientui.PanelOptions{Title: "Field Notes", Accent: theme.AccentMuted})
	hoverX, hoverY := g.hoveredTile(snapshot)
	clientui.DrawTextf(screen, infoRect.Min.X+14, infoRect.Min.Y+26, theme.TextDim, "Facing: %s", [4]string{"Down", "Left", "Up", "Right"}[g.facingDir])
	clientui.DrawTextf(screen, infoRect.Min.X+14, infoRect.Min.Y+42, theme.TextDim, "Tile: %d,%d", hoverX, hoverY)
	clientui.DrawTextf(screen, infoRect.Min.X+14, infoRect.Min.Y+58, theme.TextDim, "FPS: %.0f", ebiten.ActualFPS())
	clientui.DrawText(screen, "Click-to-move • Enter chat", infoRect.Min.X+14, infoRect.Min.Y+80, theme.Text)
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
		float64(mx)-float64(screenWidth)/2+camSX,
		float64(my)-float64(screenHeight)/2+camSY+render.HalfTileH,
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

	selected := clampInt(g.overlay.selectedInventory, 0, len(items)-1)
	visibleRows := inventoryVisibleRows(rect)
	start := clampInt(selected-visibleRows/2, 0, max(0, len(items)-visibleRows))
	listRect := image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Max.X-10, rect.Min.Y+36+visibleRows*18)
	if !pointInRect(mx, my, listRect) {
		return true
	}

	row := (my - listRect.Min.Y) / 18
	index := start + row
	if index >= 0 && index < len(items) {
		g.overlay.selectedInventory = index
	}
	return true
}

func (g *Game) drawStatsHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Stats Ledger", Accent: theme.AccentMuted})
	sections := []image.Rectangle{
		image.Rect(rect.Min.X+10, rect.Min.Y+22, rect.Max.X-10, rect.Min.Y+76),
		image.Rect(rect.Min.X+10, rect.Min.Y+80, rect.Max.X-10, rect.Min.Y+134),
		image.Rect(rect.Min.X+10, rect.Min.Y+138, rect.Max.X-10, rect.Max.Y-16),
	}
	for _, section := range sections {
		clientui.DrawInset(screen, section, theme, false)
	}

	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+14, theme.Text, "%s • Lvl %d", fallbackString(snapshot.Character.Name, "Wayfarer"), snapshot.Character.Level)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+28, theme.TextDim, "Class %d • Admin %d • EXP %d", snapshot.Character.ClassID, snapshot.Character.Admin, snapshot.Character.Experience)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+42, theme.TextDim, "HP %d/%d • TP %d/%d • SP %d", snapshot.Character.HP, snapshot.Character.MaxHP, snapshot.Character.TP, snapshot.Character.MaxTP, snapshot.Character.MaxSP)

	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+14, theme.Text, "Stats %d/%d pts", snapshot.Character.StatPoints, snapshot.Character.SkillPoints)
	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+28, theme.TextDim, "STR %d  INT %d  WIS %d", snapshot.Character.BaseStats.Str, snapshot.Character.BaseStats.Int, snapshot.Character.BaseStats.Wis)
	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+42, theme.TextDim, "AGI %d  CON %d  CHA %d", snapshot.Character.BaseStats.Agi, snapshot.Character.BaseStats.Con, snapshot.Character.BaseStats.Cha)

	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+14, theme.Text, "Combat & field")
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+28, theme.TextDim, "Dmg %d-%d  Hit %d  Eva %d  Arm %d", snapshot.Character.CombatStats.MinDamage, snapshot.Character.CombatStats.MaxDamage, snapshot.Character.CombatStats.Accuracy, snapshot.Character.CombatStats.Evade, snapshot.Character.CombatStats.Armor)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+42, theme.TextDim, "Weight %d/%d  Karma %d  Usage %d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max, snapshot.Character.Karma, snapshot.Character.Usage)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+56, theme.TextDim, "Players %d  NPCs %d  Ground %d  Bag %d", max(0, len(snapshot.NearbyChars)-1), len(snapshot.NearbyNpcs), len(snapshot.NearbyItems), len(snapshot.Inventory))
}

func (g *Game) drawInventoryHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Inventory Satchel", Accent: theme.AccentMuted})
	items := snapshot.Inventory
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Weight %d/%d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max)
	clientui.DrawTextf(screen, rect.Max.X-74, rect.Min.Y+24, theme.TextDim, "%d items", len(items))

	if len(items) == 0 {
		clientui.DrawTextWrappedCentered(screen, "No inventory items reported yet.", image.Rect(rect.Min.X+12, rect.Min.Y+58, rect.Max.X-12, rect.Min.Y+86), theme.Text)
		g.drawEquipmentSummary(screen, theme, image.Rect(rect.Min.X+10, rect.Min.Y+98, rect.Max.X-10, rect.Max.Y-18), snapshot)
		return
	}

	selected := clampInt(g.overlay.selectedInventory, 0, len(items)-1)
	g.overlay.selectedInventory = selected
	visibleRows := inventoryVisibleRows(rect)
	start := clampInt(selected-visibleRows/2, 0, max(0, len(items)-visibleRows))
	listRect := image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Max.X-10, rect.Min.Y+36+visibleRows*18)
	clientui.DrawInset(screen, listRect, theme, false)
	for row := 0; row < visibleRows && start+row < len(items); row++ {
		item := items[start+row]
		entry := image.Rect(listRect.Min.X+4, listRect.Min.Y+4+row*18, listRect.Max.X-4, listRect.Min.Y+20+row*18)
		active := start+row == selected
		if active {
			ebitenutil.DrawRect(screen, float64(entry.Min.X), float64(entry.Min.Y), float64(entry.Dx()), float64(entry.Dy()), colorize(theme.AccentMuted, 96))
		}
		clientui.DrawTextf(screen, entry.Min.X+6, entry.Min.Y+13, ternaryColor(active, theme.Text, theme.TextDim), "Item %03d", item.ID)
		amountLabel := fmt.Sprintf("x%d", item.Amount)
		clientui.DrawText(screen, amountLabel, entry.Max.X-clientui.MeasureText(amountLabel)-6, entry.Min.Y+13, ternaryColor(active, theme.Accent, theme.Text))
	}
	if start > 0 {
		clientui.DrawText(screen, "^", listRect.Max.X-12, listRect.Min.Y+12, theme.TextDim)
	}
	if start+visibleRows < len(items) {
		clientui.DrawText(screen, "v", listRect.Max.X-12, listRect.Max.Y-6, theme.TextDim)
	}

	selectedItem := items[selected]
	detailRect := image.Rect(rect.Min.X+10, listRect.Max.Y+6, rect.Max.X-10, listRect.Max.Y+34)
	clientui.DrawInset(screen, detailRect, theme, true)
	clientui.DrawTextf(screen, detailRect.Min.X+8, detailRect.Min.Y+14, theme.Text, "Selected: Item %03d", selectedItem.ID)
	clientui.DrawTextf(screen, detailRect.Min.X+8, detailRect.Min.Y+28, theme.TextDim, "Count %d", selectedItem.Amount)

	g.drawEquipmentSummary(screen, theme, image.Rect(rect.Min.X+10, detailRect.Max.Y+6, rect.Max.X-10, rect.Max.Y-16), snapshot)
}

func (g *Game) drawEquipmentSummary(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawInset(screen, rect, theme, false)
	clientui.DrawText(screen, "Paperdoll", rect.Min.X+8, rect.Min.Y+14, theme.Text)
	rows := []string{
		fmt.Sprintf("W:%d S:%d A:%d H:%d", snapshot.Equipment.Weapon, snapshot.Equipment.Shield, snapshot.Equipment.Armor, snapshot.Equipment.Hat),
		fmt.Sprintf("B:%d N:%d G:%d Bt:%d", snapshot.Equipment.Belt, snapshot.Equipment.Necklace, snapshot.Equipment.Gloves, snapshot.Equipment.Boots),
		fmt.Sprintf("R:%s Ar:%s Br:%s", paperdollPair(snapshot.Equipment.Ring), paperdollPair(snapshot.Equipment.Armlet), paperdollPair(snapshot.Equipment.Bracer)),
	}
	for i, row := range rows {
		clientui.DrawText(screen, row, rect.Min.X+8, rect.Min.Y+28+i*14, ternaryColor(i == 0, theme.TextDim, theme.TextDim))
	}
}

func (g *Game) drawMapHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Map Scroll", Accent: theme.AccentMuted})
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Map %d", snapshot.Character.MapID)
	clientui.DrawTextf(screen, rect.Max.X-86, rect.Min.Y+24, theme.TextDim, "%d,%d", snapshot.Character.X, snapshot.Character.Y)
	mapRect := image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Max.X-10, rect.Max.Y-26)
	clientui.DrawInset(screen, mapRect, theme, true)
	g.drawMinimap(screen, theme, mapRect, snapshot)
	clientui.DrawTextCentered(screen, autoWalkStatusLine(g.autoWalk), image.Rect(rect.Min.X+12, rect.Max.Y-24, rect.Max.X-12, rect.Max.Y-10), theme.TextDim)
}

func (g *Game) drawPaperdollHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Paperdoll Wardrobe", Accent: theme.AccentMuted})
	name := fallbackString(snapshot.Character.Name, "Wayfarer")
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.Text, "%s • Class %d", name, snapshot.Character.ClassID)
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+40, theme.TextDim, "%s • %s", fallbackString(snapshot.Character.Title, "No title"), paperdollGuildLine(snapshot))

	leftCol := image.Rect(rect.Min.X+10, rect.Min.Y+54, rect.Min.X+102, rect.Max.Y-14)
	rightCol := image.Rect(leftCol.Max.X+6, rect.Min.Y+54, rect.Max.X-10, rect.Max.Y-14)
	clientui.DrawInset(screen, leftCol, theme, false)
	clientui.DrawInset(screen, rightCol, theme, false)

	slotLines := []string{
		fmt.Sprintf("Hat     %d", snapshot.Equipment.Hat),
		fmt.Sprintf("Armor   %d", snapshot.Equipment.Armor),
		fmt.Sprintf("Weapon  %d", snapshot.Equipment.Weapon),
		fmt.Sprintf("Shield  %d", snapshot.Equipment.Shield),
		fmt.Sprintf("Boots   %d", snapshot.Equipment.Boots),
		fmt.Sprintf("Gloves  %d", snapshot.Equipment.Gloves),
		fmt.Sprintf("Belt    %d", snapshot.Equipment.Belt),
		fmt.Sprintf("Neck    %d", snapshot.Equipment.Necklace),
		fmt.Sprintf("Acc     %d", snapshot.Equipment.Accessory),
	}
	for i, line := range slotLines {
		clientui.DrawText(screen, line, leftCol.Min.X+8, leftCol.Min.Y+14+i*14, ternaryColor(i < 4, theme.Text, theme.TextDim))
	}

	clientui.DrawText(screen, "Jewelry", rightCol.Min.X+8, rightCol.Min.Y+14, theme.Text)
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+28, theme.TextDim, "Rings   %s", paperdollPair(snapshot.Equipment.Ring))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+42, theme.TextDim, "Armlets %s", paperdollPair(snapshot.Equipment.Armlet))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+56, theme.TextDim, "Bracers %s", paperdollPair(snapshot.Equipment.Bracer))
	clientui.DrawText(screen, "Identity", rightCol.Min.X+8, rightCol.Min.Y+78, theme.Text)
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+92, theme.TextDim, "Guild %s", fallbackString(snapshot.Character.GuildName, "None"))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+106, theme.TextDim, "Rank  %s", fallbackString(snapshot.Character.GuildRank, "None"))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+120, theme.TextDim, "Home  %s", fallbackString(snapshot.Character.Home, "Unknown"))
	clientui.DrawTextf(screen, rightCol.Min.X+8, rightCol.Min.Y+134, theme.TextDim, "Bond  %s", fallbackString(snapshot.Character.Partner, "Solo"))
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

func inventoryVisibleRows(rect image.Rectangle) int {
	return 4
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
	layout := characterSelectDialogLayout()
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
	layout := inGameHUDLayout()
	mx, my := ebiten.CursorPosition()
	if pointInRect(mx, my, layout.chatInputRect) {
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
	return pointInRect(mx, my, layout.chatPanelRect) ||
		pointInRect(mx, my, layout.statusRect) ||
		pointInRect(mx, my, layout.menuRect) ||
		pointInRect(mx, my, layout.menuPanelRect) ||
		pointInRect(mx, my, layout.infoRect)
}

func characterSelectDialogLayout() characterSelectLayout {
	dialog := centeredRect(556, 308)
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

func inGameHUDLayout() gameHUDLayout {
	chatPanelRect := image.Rect(10, screenHeight-150, 392, screenHeight-10)
	return gameHUDLayout{
		statusRect:    image.Rect(10, 10, 184, 108),
		menuRect:      image.Rect(screenWidth-158, 10, screenWidth-10, 92),
		menuPanelRect: image.Rect(screenWidth-230, 102, screenWidth-10, 360),
		infoRect:      image.Rect(screenWidth-166, screenHeight-104, screenWidth-10, screenHeight-10),
		chatPanelRect: chatPanelRect,
		chatInputRect: image.Rect(chatPanelRect.Min.X+12, chatPanelRect.Max.Y-36, chatPanelRect.Max.X-12, chatPanelRect.Max.Y-12),
	}
}

func hudMenuButtons(layout gameHUDLayout) []hudMenuButton {
	panels := []struct {
		panel gameMenuPanel
		label string
	}{
		{panel: gameMenuPanelStats, label: "Stats"},
		{panel: gameMenuPanelInventory, label: "Inventory"},
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
			Rect:  image.Rect(layout.menuRect.Min.X+12+col*66, layout.menuRect.Min.Y+22+row*26, layout.menuRect.Min.X+72+col*66, layout.menuRect.Min.Y+44+row*26),
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

func centeredRect(width, height int) image.Rectangle {
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
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
