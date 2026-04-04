package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/charselect"
	"github.com/avdo/eoweb/internal/ui/hud"
	"github.com/avdo/eoweb/internal/ui/login"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

type overlayState struct {
	ticks               int
	authMode            login.AuthMode
	loginUsername       []rune
	loginPassword       []rune
	loginPassword2      []rune
	loginEmail          []rune
	loginAddress        []rune
	loginFocus          int
	loginSubmitting     bool
	previewDirection    int
	selectedCharacter   int
	rosterScroll        int
	selectingCharacter  bool
	characterCreate     charselect.CreateForm
	characterCreateOpen bool
	statusMessage       string
	activeMenuPanel     overlay.MenuPanel
	selectedInventory   int
	inventoryPage       int
	lastClickItem       int
	lastClickTick       int
	itemAmountPicker    itemAmountPickerState
}

type inventoryDragState struct {
	Active bool
	ItemID int
	Amount int
	Offset image.Point
}

func newOverlayState() overlayState {
	return overlayState{
		authMode:        login.ModeHome,
		characterCreate: charselect.NewCreateForm(),
	}
}

func (g *Game) updateOverlayState() {
	g.overlay.ticks++
	if g.client.GetState() == game.StateLoggedIn {
		charselect.NormalizeState(len(g.client.Characters), &g.overlay.selectedCharacter, &g.overlay.rosterScroll, &g.overlay.previewDirection)
		if len(g.client.Characters) == 0 {
			g.overlay.characterCreateOpen = true
		}
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

func (g *Game) drawOverlayScreen(screen *ebiten.Image) {
	theme := clientui.RetroTheme
	switch g.client.GetState() {
	case game.StateInitial:
		clientui.DrawBackdrop(screen, theme, g.overlay.ticks)
		login.DrawConnecting(screen, theme, g.screenW, g.screenH, g.overlay.ticks, g.overlay.statusMessage, g.connectError)
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
		g.drawItemAmountPicker(screen, theme)
	}
}

func (g *Game) drawGameHUD(screen *ebiten.Image, theme clientui.Theme) {
	snapshot := g.client.UISnapshot()
	layout := overlay.InGameHUDLayout(g.screenW, g.screenH)
	meterRects := overlay.TopStatusMeterRects(layout.StatusRect)
	tnlValue, tnlRange := overlay.TnlProgress(snapshot.Character.Level, snapshot.Character.Experience)
	hpFill := color.NRGBA{R: 153, G: 56, B: 48, A: 255}
	if hpRatio := overlay.StatRatio(snapshot.Character.HP, snapshot.Character.MaxHP); hpRatio < 0.25 && hpRatio > 0 {
		pulse := 0.5 + 0.5*math.Sin(float64(g.overlay.ticks)/4.0)
		hpFill.R = uint8(min(255, int(hpFill.R)+int(pulse*60)))
		hpFill.G = uint8(max(0, int(hpFill.G)-int(pulse*20)))
	}
	clientui.DrawMeter(screen, meterRects[0], theme, fmt.Sprintf("HP %d/%d", snapshot.Character.HP, snapshot.Character.MaxHP), overlay.StatRatio(snapshot.Character.HP, snapshot.Character.MaxHP), hpFill)
	clientui.DrawMeter(screen, meterRects[1], theme, fmt.Sprintf("TP %d/%d", snapshot.Character.TP, snapshot.Character.MaxTP), overlay.StatRatio(snapshot.Character.TP, snapshot.Character.MaxTP), color.NRGBA{R: 61, G: 104, B: 168, A: 255})
	clientui.DrawMeter(screen, meterRects[2], theme, fmt.Sprintf("TNL %d", tnlValue), overlay.StatRatio(tnlRange-tnlValue, tnlRange), color.NRGBA{R: 132, G: 108, B: 44, A: 255})

	clientui.DrawPanel(screen, layout.MenuRect, theme, clientui.PanelOptions{Accent: theme.Accent})
	for _, button := range overlay.HUDMenuButtons(layout) {
		clientui.DrawButton(screen, button.Rect, theme, button.Label, g.overlay.activeMenuPanel == button.Panel, false)
	}
	if g.overlay.activeMenuPanel != overlay.MenuPanelNone {
		g.drawActiveHUDPanel(screen, theme, layout.MenuPanelRect, snapshot)
	}
	if g.inventoryDrag.Active {
		g.drawInventoryDrag(screen)
	}
}

func (g *Game) drawActiveHUDPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) {
	switch g.overlay.activeMenuPanel {
	case overlay.MenuPanelStats:
		hud.DrawStats(screen, theme, rect, snapshot)
	case overlay.MenuPanelInventory:
		positions := g.inventoryGridPositions(snapshot.Inventory)
		g.overlay.inventoryPage = overlay.ClampInt(g.overlay.inventoryPage, 0, hud.InventoryGridPages-1)
		selected := overlay.ClampInt(g.overlay.selectedInventory, 0, max(0, len(snapshot.Inventory)-1))
		if len(snapshot.Inventory) > 0 {
			if pos, ok := positions[snapshot.Inventory[selected].ID]; ok && pos.Page != g.overlay.inventoryPage {
				for i, item := range snapshot.Inventory {
					if p, ok := positions[item.ID]; ok && p.Page == g.overlay.inventoryPage {
						selected = i
						break
					}
				}
			}
		}
		g.overlay.selectedInventory = selected
		hud.DrawInventory(screen, theme, rect, snapshot, positions, g.overlay.inventoryPage, selected, g.drawItemIcon, g.drawItemTooltip)
	case overlay.MenuPanelMap:
		hud.DrawMap(screen, theme, rect, snapshot, g.autoWalk.Active, autoWalkStatusLine(g.autoWalk), g.drawMinimap)
	case overlay.MenuPanelGuild:
		hud.DrawPaperdoll(screen, theme, rect, snapshot, g.drawItemSlot, g.drawItemTooltip)
	default:
		title, lines := g.activeHUDPanelContent(snapshot)
		clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.AccentMuted})
		for i, line := range lines {
			clientui.DrawTextWrappedCentered(screen, line, image.Rect(rect.Min.X+12, rect.Min.Y+22+i*28, rect.Max.X-12, rect.Min.Y+44+i*28), overlay.TernaryColor(i == 0, theme.Text, theme.TextDim))
		}
		clientui.DrawTextCentered(screen, "Click the lit menu tab or Esc to close.", image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Max.X-12, rect.Max.Y-12), theme.TextDim)
	}
}

func (g *Game) activeHUDPanelContent(snapshot game.UISnapshot) (string, []string) {
	switch g.overlay.activeMenuPanel {
	case overlay.MenuPanelStats:
		return "Stats Ledger", []string{
			fmt.Sprintf("Level %d • HP %d/%d • TP %d/%d", snapshot.Character.Level, snapshot.Character.HP, snapshot.Character.MaxHP, snapshot.Character.TP, snapshot.Character.MaxTP),
			fmt.Sprintf("Weight %d/%d • %d inventory entries", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max, len(snapshot.Inventory)),
		}
	case overlay.MenuPanelInventory:
		return "Inventory Satchel", []string{fmt.Sprintf("Satchel entries: %d • Ground nearby: %d", len(snapshot.Inventory), len(snapshot.NearbyItems))}
	case overlay.MenuPanelMap:
		return "Map Scroll", []string{
			fmt.Sprintf("Map %d • Position %d,%d", snapshot.Character.MapID, snapshot.Character.X, snapshot.Character.Y),
			autoWalkStatusLine(g.autoWalk),
		}
	case overlay.MenuPanelGuild:
		return "Paperdoll Wardrobe", []string{"Equipment slots render authoritative equipped item IDs."}
	default:
		return "Menu", []string{"No panel selected."}
	}
}

// --- Shared camera helpers ---

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
	return camSX + walkOffsetX, camSY + walkOffsetY
}

func playerCamOffsetSnapshot(snapshot game.UISnapshot) (float64, float64) {
	for _, ch := range snapshot.NearbyChars {
		if ch.PlayerID == snapshot.PlayerID && ch.Walking {
			return render.WalkOffset(ch.Direction, ch.WalkProgress())
		}
	}
	return 0, 0
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
