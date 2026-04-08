package main

import (
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/gfx"
	"github.com/avdo/eoweb/internal/movement"
	eonet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
	"github.com/avdo/eoweb/internal/render"
	"github.com/avdo/eoweb/internal/ui/overlay"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if maybeApplyDesktopBinaryUpdate() {
		return
	}

	cfg := loadRuntimeConfig()

	loader := gfx.NewLoaderWithReader(cfg.gfxDir, cfg.assetReader)
	itemDB, err := pubdata.LoadItemDBFromReader(cfg.assetReader, cfg.itemPubPath)
	if err != nil {
		slog.Warn("item metadata unavailable", "path", cfg.itemPubPath, "err", err)
	}
	npcDB, err := pubdata.LoadNPCDBFromReader(cfg.assetReader, cfg.npcPubPath)
	if err != nil {
		slog.Warn("npc metadata unavailable", "path", cfg.npcPubPath, "err", err)
	}
	inventoryLayout, err := loadInventoryLayout(cfg.layoutPath)
	if err != nil {
		slog.Warn("inventory layout unavailable", "path", cfg.layoutPath, "err", err)
		inventoryLayout = make(map[int]storedInventoryPos)
	}

	cl := game.NewClient()
	cl.MapsDir = cfg.mapsDir
	cl.ItemPubPath = cfg.itemPubPath
	cl.NpcPubPath = cfg.npcPubPath
	cl.SpellPubPath = cfg.spellPubPath
	cl.ClassPubPath = cfg.classPubPath
	cl.AssetReader = cfg.assetReader

	g := &Game{
		screenW:             cfg.defaultWidth,
		screenH:             cfg.defaultHeight,
		client:              cl,
		handlers:            game.NewHandlerRegistry(),
		gfxLoad:             loader,
		itemDB:              itemDB,
		npcDB:               npcDB,
		inventoryLayout:     inventoryLayout,
		inventoryLayoutPath: cfg.layoutPath,
		mapsDir:             cfg.mapsDir,
		mapRenderer:         render.NewMapRendererWithReader(loader, cfg.assetReader),
		overlay:             newOverlayState(),
		chat:                newChatState(),
		connectArmed:        true,
		serverAddr:          cfg.serverAddr,
		serverConfigKey:     cfg.serverConfigKey,
	}
	g.overlay.loginServerAddr = []rune(cfg.serverAddr)
	game.RegisterAllHandlers(g.handlers)
	if itemDB != nil {
		g.client.ItemTypeFunc = func(id int) int {
			if rec, ok := itemDB.Get(id); ok {
				return int(rec.Type)
			}
			return 0
		}
	}

	ebiten.SetWindowSize(cfg.defaultWidth, cfg.defaultHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(cfg.defaultWidth, cfg.defaultHeight, -1, -1)
	ebiten.SetWindowTitle(cfg.windowTitle)
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(g); err != nil {
		slog.Error("game exited", "err", err)
		os.Exit(1)
	}
}

type Game struct {
	screenW             int
	screenH             int
	client              *game.Client
	handlers            *game.HandlerRegistry
	gfxLoad             *gfx.Loader
	itemDB              *pubdata.ItemDB
	npcDB               *pubdata.NPCDB
	inventoryLayout     map[int]storedInventoryPos
	inventoryLayoutPath string
	mapsDir             string
	inventoryDrag       inventoryDragState
	mapRenderer         *render.MapRenderer
	overlay             overlayState
	autoWalk            autoWalkPlan
	serverAddr          string
	serverConfigKey     string

	connected    bool
	connectArmed bool
	connectError string

	chat           ChatState
	chatHistoryBuf *ebiten.Image
	chatHistoryW   int
	chatHistoryH   int

	walkCooldown   int
	faceCooldown   int
	attackCooldown int
	facingDir      int
	isWalking      bool
	moveHoldDir    int
	moveHoldTicks  int
}

type worldHoverIntent struct {
	TileX      int
	TileY      int
	CursorType int
	Valid      bool
}

func (g *Game) Update() error {
	g.client.InvalidateSnapshot()
	g.updateOverlayState()

	if !g.connected && g.connectArmed && g.client.GetState() == game.StateInitial && g.connectError == "" {
		g.overlay.statusMessage = "Contacting server..."
		go g.connect()
		g.connected = true
	}

	for {
		select {
		case evt := <-g.client.Events:
			g.handleEvent(evt)
		default:
			goto done
		}
	}
done:

	switch g.client.GetState() {
	case game.StateConnected:
		g.updateLogin()
	case game.StateLoggedIn:
		g.updateCharacterSelect()
	case game.StateInGame:
		g.updateInGame()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.client.GetState() != game.StateInGame {
		g.drawOverlayScreen(screen)
		return
	}
	g.drawWorld(screen)
	g.drawOverlayScreen(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW = outsideWidth
	g.screenH = outsideHeight
	return outsideWidth, outsideHeight
}

func (g *Game) updateInGame() {
	g.tickAnimations()

	// Handle dialog input first (before movement/combat)
	if g.updateDialogs() {
		return
	}

	g.updateChat()
	if g.chat.Typing {
		return
	}
	if g.updateItemAmountPicker() {
		return
	}

	if g.walkCooldown > 0 {
		g.walkCooldown--
	}
	if g.faceCooldown > 0 {
		g.faceCooldown--
	}
	if g.attackCooldown > 0 {
		g.attackCooldown--
	}
	if g.inventoryDrag.Active {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			g.finishInventoryDrag()
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.overlay.activeMenuPanel != overlay.MenuPanelNone {
			g.overlay.activeMenuPanel = overlay.MenuPanelNone
			return
		}
		g.clearAutoWalk()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.handleInGameLeftClick() {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.handleInGameRightClick()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		g.clearAutoWalk()
		if g.attackCooldown <= 0 {
			g.sendAttack()
			g.attackCooldown = attackCooldownTicks
		}
		return
	}

	dir := -1
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dir = 2
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dir = 0
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dir = 1
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dir = 3
	}
	g.updateMoveHold(dir)

	if dir >= 0 {
		g.clearAutoWalk()
		if g.isWalking || g.walkCooldown > 0 {
			return
		}
		g.faceOrWalk(dir)
		return
	}

	if g.isWalking {
		return
	}

	g.advanceAutoWalk()
}

func (g *Game) handleInGameLeftClick() bool {
	if g.handleInGameOverlayClick() {
		return true
	}

	hover := g.currentWorldHoverIntent()
	if !hover.Valid || hover.CursorType < 0 {
		g.clearAutoWalk()
		return false
	}

	dir, ok := movement.DirectionTowardTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY)
	if !ok {
		if hover.CursorType == 2 {
			if itemUID, found := g.findPickupItemAtTile(hover.TileX, hover.TileY, 0); found {
				g.clearAutoWalk()
				g.sendPickupItem(itemUID)
				return true
			}
		}
		return false
	}
	g.clearAutoWalk()

	if g.isAttackTargetTile(hover.TileX, hover.TileY) && movement.IsAdjacentTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
		return g.faceOrAttack(dir)
	}

	if hover.CursorType == 2 {
		if itemUID, found := g.findPickupItemAtTile(hover.TileX, hover.TileY, 0); found {
			if withinItemInteractionRange(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
				g.sendPickupItem(itemUID)
				return true
			}
			return g.queueAutoPickup(hover.TileX, hover.TileY, itemUID)
		}
		return g.queueAutoWalkToTile(hover.TileX, hover.TileY)
	}

	if hover.CursorType == 1 {
		if npc, ok := g.findNPCAtTile(hover.TileX, hover.TileY); ok && g.isInteractableNPC(npc.ID) {
			g.sendNpcInteract(npc.Index, npc.ID)
			return true
		}
		if movement.IsAdjacentTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
			// Check if it's a door tile — open it
			if g.isDoorTile(hover.TileX, hover.TileY) {
				g.sendDoorOpen(hover.TileX, hover.TileY)
				return true
			}
			// Check if it's a chest tile (spec 9) — open it
			if g.tileSpecAt(hover.TileX, hover.TileY) == 9 {
				g.sendChestOpen(hover.TileX, hover.TileY)
				g.overlay.chestX = hover.TileX
				g.overlay.chestY = hover.TileY
				return true
			}
			return g.faceOnly(dir)
		}
		return g.queueAutoWalkToInteraction(hover.TileX, hover.TileY)
	}

	if hover.TileX == g.client.Character.X && hover.TileY == g.client.Character.Y {
		return false
	}
	return g.queueAutoWalkToTile(hover.TileX, hover.TileY)
}

func (g *Game) findNPCAtTile(tileX, tileY int) (game.NearbyNPC, bool) {
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return npc, true
		}
	}
	return game.NearbyNPC{}, false
}

func (g *Game) isInteractableNPC(npcID int) bool {
	if g.npcDB == nil {
		return false
	}
	switch g.npcDB.Type(npcID) {
	case pub.Npc_Shop, pub.Npc_Inn, pub.Npc_Bank, pub.Npc_Barber,
		pub.Npc_Guild, pub.Npc_Priest, pub.Npc_Lawyer, pub.Npc_Trainer, pub.Npc_Quest:
		return true
	default:
		return false
	}
}

func (g *Game) tryOpenShopAt(tileX, tileY int) bool {
	npc, ok := g.findNPCAtTile(tileX, tileY)
	if !ok || !g.isInteractableNPC(npc.ID) || !movement.IsAdjacentTile(g.client.Character.X, g.client.Character.Y, tileX, tileY) {
		return false
	}
	g.sendNpcInteract(npc.Index, npc.ID)
	return true
}

func (g *Game) currentWorldHoverIntent() worldHoverIntent {
	snapshot := g.client.UISnapshot()
	mx, my := ebiten.CursorPosition()

	if g.worldHoverBlockedByHUD(mx, my) {
		return worldHoverIntent{Valid: false}
	}

	camSX, camSY := g.currentCameraScreenPosition(snapshot)
	halfW := float64(g.screenW) / 2
	halfH := float64(g.screenH) / 2

	// Check visual hits for entities first (depth-sorted)
	bestDepth := -1
	bestTileX, bestTileY := -1, -1
	bestCursor := -1

	// Characters
	for _, ch := range g.mapRenderer.Characters {
		rect := characterHoverRect(ch, camSX, camSY, halfW, halfH)
		if overlay.PointInRect(mx, my, rect) {
			if rect.Max.Y > bestDepth {
				bestDepth = rect.Max.Y
				bestTileX, bestTileY = ch.X, ch.Y
				bestCursor = 1 // Interact cursor for players (e.g. for context menu)
			}
		}
	}

	// NPCs
	for _, npc := range g.mapRenderer.Npcs {
		rect := g.npcHoverRect(npc, camSX, camSY, halfW, halfH)
		if overlay.PointInRect(mx, my, rect) {
			if rect.Max.Y > bestDepth {
				bestDepth = rect.Max.Y
				bestTileX, bestTileY = npc.DestX, npc.DestY
				bestCursor = 1 // Interaction cursor for NPCs
			}
		}
	}

	if bestDepth != -1 {
		return worldHoverIntent{
			TileX:      bestTileX,
			TileY:      bestTileY,
			CursorType: bestCursor,
			Valid:      true,
		}
	}

	// Fallback to ground-plane tile detection
	tileX, tileY := g.hoveredTile(snapshot)
	if !g.isMapTileInBounds(tileX, tileY) {
		return worldHoverIntent{TileX: tileX, TileY: tileY, CursorType: -1, Valid: false}
	}
	return worldHoverIntent{
		TileX:      tileX,
		TileY:      tileY,
		CursorType: g.getCursorType(tileX, tileY),
		Valid:      true,
	}
}

func (g *Game) loadCurrentMap() {
	mapPath := fmt.Sprintf("%s/%05d.emf", g.mapsDir, g.client.Character.MapID)
	if err := g.mapRenderer.LoadMap(mapPath); err != nil {
		slog.Error("failed to load map", "path", mapPath, "err", err)
	}
	g.loadDoors()
}

func (g *Game) loadDoors() {
	if g.mapRenderer.Map == nil {
		g.client.LoadDoors(nil)
		return
	}
	var doors []game.Door
	for _, row := range g.mapRenderer.Map.WarpRows {
		for _, tile := range row.Tiles {
			if tile.Warp.Door > 0 {
				doors = append(doors, game.Door{
					X:   tile.X,
					Y:   row.Y,
					Key: tile.Warp.Door,
				})
			}
		}
	}
	g.client.LoadDoors(doors)
}

func (g *Game) drawWorld(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	g.mapRenderer.CamX = float64(g.client.Character.X)
	g.mapRenderer.CamY = float64(g.client.Character.Y)

	wox, woy := g.playerCamOffset()
	g.mapRenderer.CamOffX = wox
	g.mapRenderer.CamOffY = woy

	g.syncEntities()
	hover := g.currentWorldHoverIntent()
	if !hover.Valid || hover.CursorType < 0 {
		g.mapRenderer.Cursor = nil
	} else {
		g.mapRenderer.CursorBuf = render.CursorEntity{
			X:    hover.TileX,
			Y:    hover.TileY,
			Type: hover.CursorType,
		}
		g.mapRenderer.Cursor = &g.mapRenderer.CursorBuf
	}
	g.mapRenderer.Draw(screen)
}

// Suppress unused import warnings — used transitively.
var (
	_ = eonet.Dial
	_ *pubdata.ItemDB
)
