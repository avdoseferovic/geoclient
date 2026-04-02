package main

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/gfx"
	eonet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
	"github.com/avdo/eoweb/internal/render"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

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
	g := &Game{
		screenW:             cfg.defaultWidth,
		screenH:             cfg.defaultHeight,
		client:              game.NewClient(),
		handlers:            game.NewHandlerRegistry(),
		gfxLoad:             loader,
		itemDB:              itemDB,
		npcDB:               npcDB,
		inventoryLayout:     inventoryLayout,
		inventoryLayoutPath: cfg.layoutPath,
		mapsDir:             cfg.mapsDir,
		mapRenderer:         render.NewMapRendererWithReader(loader, cfg.assetReader),
		overlay:             newOverlayState(),
		connectArmed:        true,
		serverAddr:          cfg.serverAddr,
	}
	game.RegisterAllHandlers(g.handlers)

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

	connected    bool
	connectArmed bool
	connectError string

	// Chat
	chat ChatState

	// Movement
	walkCooldown   int // ticks until next walk allowed
	attackCooldown int
	facingDir      int  // current facing direction (0-3)
	isWalking      bool // true while local player walk animation is playing
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
	g.updateOverlayState()

	// Handle connection
	if !g.connected && g.connectArmed && g.client.GetState() == game.StateInitial && g.connectError == "" {
		g.overlay.statusMessage = "Contacting server..."
		go g.connect()
		g.connected = true
	}

	// Drain events
	for {
		select {
		case evt := <-g.client.Events:
			g.handleEvent(evt)
		default:
			goto done
		}
	}
done:

	// Handle input based on state
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

func (g *Game) connect() {
	slog.Info("connecting", "addr", g.serverAddr)
	conn, err := eonet.Dial(g.serverAddr)
	if err != nil {
		g.failConnection(fmt.Sprintf("Unable to reach server: %v", err), false)
		return
	}

	bus := eonet.NewPacketBus(conn)
	g.client.SetBus(bus)

	challenge := rand.Intn(11_092_110) + 1
	g.client.Challenge = challenge

	// Send init packet
	if err := bus.SendPacket(newInitPacket(challenge, g.client.Version)); err != nil {
		g.failConnection(fmt.Sprintf("Handshake failed: %v", err), true)
		return
	}

	// Start receive loop
	go g.recvLoop()
}

func (g *Game) recvLoop() {
	for {
		bus := g.client.GetBus()
		if bus == nil {
			return
		}

		action, family, reader, err := bus.Recv()
		if err != nil {
			g.failConnection(fmt.Sprintf("Connection lost: %v", err), true)
			return
		}

		if err := g.handlers.Dispatch(family, action, g.client, reader); err != nil {
			g.failConnection(fmt.Sprintf("Network flow interrupted: %v", err), true)
			return
		}
	}
}

func (g *Game) failConnection(message string, disconnectClient bool) {
	slog.Error("connection failure", "msg", message)
	g.client.EmitCritical(game.Event{
		Type:    game.EventError,
		Message: message,
		Data:    disconnectClient,
	})
}

func (g *Game) handleEvent(evt game.Event) {
	switch evt.Type {
	case game.EventError:
		slog.Error("game error", "msg", evt.Message)
		g.connectError = evt.Message
		g.connected = false
		g.connectArmed = false
		g.overlay.loginSubmitting = false
		g.overlay.selectingCharacter = false
		g.overlay.statusMessage = evt.Message
		disconnectClient, _ := evt.Data.(bool)
		if disconnectClient {
			g.client.Disconnect()
		}
	case game.EventChat:
		g.handleChatEvent(evt)
	case game.EventStateChanged:
		slog.Info("state changed", "state", evt.Message)
		if evt.Message == "Connected" {
			g.connectError = ""
			g.overlay.statusMessage = "Connected. Awaiting login."
		}
	case game.EventEnterGame:
		slog.Info("entered game")
		g.clearAutoWalk()
		g.overlay.activeMenuPanel = gameMenuPanelNone
		g.facingDir = int(g.client.Character.Direction)
		g.loadCurrentMap()
		g.overlay.loginSubmitting = false
		g.overlay.selectingCharacter = false
		g.overlay.statusMessage = ""
	case game.EventWarp:
		slog.Info("warping", "mapID", evt.Data)
		g.clearAutoWalk()
		g.loadCurrentMap()
	case game.EventCharacterList:
		g.overlay.loginSubmitting = false
		g.overlay.selectingCharacter = false
		if g.overlay.selectedCharacter >= len(g.client.Characters) {
			g.overlay.selectedCharacter = 0
		}
		g.overlay.statusMessage = "Choose a character."
	}
}

const faceCooldownTicks = 3 // Matches eoweb's 1 face tick at 20 TPS.
const walkStartCooldownTicks = 24
const attackCooldownTicks = 30

// walkAnimDuration must match game.WalkDuration for consistent timing.

func (g *Game) updateInGame() {
	// Advance all entity animations
	g.tickAnimations()

	// Chat input takes priority
	g.updateChat()
	if g.chat.Typing {
		return
	}

	// Cooldown timers
	if g.walkCooldown > 0 {
		g.walkCooldown--
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
		if g.overlay.activeMenuPanel != gameMenuPanelNone {
			g.overlay.activeMenuPanel = gameMenuPanelNone
			return
		}
		g.clearAutoWalk()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.handleInGameLeftClick() {
		return
	}

	// Attack with Ctrl
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		g.clearAutoWalk()
		if g.attackCooldown <= 0 {
			g.sendAttack()
			g.attackCooldown = attackCooldownTicks
		}
		return
	}

	// Arrow key input: face-then-walk
	dir := -1
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dir = 2 // Up
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dir = 0 // Down
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dir = 1 // Left
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dir = 3 // Right
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

	dir, ok := directionTowardTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY)
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

	if g.isAttackTargetTile(hover.TileX, hover.TileY) && isAdjacentTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
		return g.faceOrAttack(dir)
	}

	if hover.CursorType == 2 {
		if g.client.Character.X == hover.TileX && g.client.Character.Y == hover.TileY {
			if itemUID, found := g.findPickupItemAtTile(hover.TileX, hover.TileY, 0); found {
				g.sendPickupItem(itemUID)
				return true
			}
			return false
		}
		if itemUID, found := g.findPickupItemAtTile(hover.TileX, hover.TileY, 0); found {
			return g.queueAutoPickup(hover.TileX, hover.TileY, itemUID)
		}
		return g.queueAutoWalkToTile(hover.TileX, hover.TileY)
	}

	if hover.CursorType == 1 {
		if isAdjacentTile(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
			return g.faceOnly(dir)
		}
		return g.queueAutoWalkToInteraction(hover.TileX, hover.TileY)
	}

	if hover.TileX == g.client.Character.X && hover.TileY == g.client.Character.Y {
		return false
	}
	return g.queueAutoWalkToTile(hover.TileX, hover.TileY)
}

func (g *Game) faceOrWalk(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 {
		return false
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	return g.stepToward(dir)
}

func (g *Game) faceOnly(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 {
		return false
	}
	g.facingDir = dir
	g.sendFace(dir)
	g.walkCooldown = faceCooldownTicks
	return true
}

func (g *Game) stepToward(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 || g.isWalking {
		return false
	}
	nextX, nextY := nextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
	if !g.canArrowStepTo(nextX, nextY, dir) {
		return g.faceOnly(dir)
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	return g.walkImmediately(dir)
}

func (g *Game) walkImmediately(dir int) bool {
	if dir < 0 || g.walkCooldown > 0 || g.isWalking {
		return false
	}
	nextX, nextY := nextTileInDirection(g.client.Character.X, g.client.Character.Y, dir)
	if !g.canArrowStepTo(nextX, nextY, dir) {
		return false
	}
	g.facingDir = dir
	g.startLocalWalk(dir)
	g.walkCooldown = walkStartCooldownTicks
	return true
}

func (g *Game) faceOrAttack(dir int) bool {
	if dir < 0 {
		return false
	}
	if g.facingDir != dir {
		return g.faceOnly(dir)
	}
	if g.attackCooldown > 0 {
		return true
	}
	g.sendAttack()
	g.attackCooldown = attackCooldownTicks
	return true
}

func (g *Game) startLocalWalk(dir int) {
	g.isWalking = true
	g.sendWalk(dir)

	g.client.Lock()
	defer g.client.Unlock()
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID == g.client.PlayerID {
			g.client.NearbyChars[i].Walking = true
			g.client.NearbyChars[i].WalkTick = 0
			break
		}
	}
}

func (g *Game) tickAnimations() {
	g.client.Lock()
	defer g.client.Unlock()

	// Tick character walk animations
	for i := range g.client.NearbyChars {
		ch := &g.client.NearbyChars[i]
		ch.Combat.Tick()
		if ch.TickWalk() {
			if ch.PlayerID == g.client.PlayerID {
				g.isWalking = false
			}
		}
	}
	// Tick NPC animations (idle + walk)
	nextNpcs := g.client.NearbyNpcs[:0]
	for i := range g.client.NearbyNpcs {
		npc := g.client.NearbyNpcs[i]
		npc.Tick()
		if npc.DeathComplete() {
			npc.Hidden = true
		}
		nextNpcs = append(nextNpcs, npc)
	}
	g.client.NearbyNpcs = nextNpcs
}

func (g *Game) loadCurrentMap() {
	mapPath := fmt.Sprintf("%s/%05d.emf", g.mapsDir, g.client.Character.MapID)
	if err := g.mapRenderer.LoadMap(mapPath); err != nil {
		slog.Error("failed to load map", "path", mapPath, "err", err)
	}
}

func (g *Game) playerCamOffset() (float64, float64) {
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID && ch.Walking {
			return render.WalkOffset(ch.Direction, ch.WalkProgress())
		}
	}
	return 0, 0
}

func (g *Game) drawWorld(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	g.mapRenderer.CamX = float64(g.client.Character.X)
	g.mapRenderer.CamY = float64(g.client.Character.Y)

	// Smooth camera: offset by walk animation progress
	wox, woy := g.playerCamOffset()
	g.mapRenderer.CamOffX = wox
	g.mapRenderer.CamOffY = woy

	g.syncEntities()
	g.mapRenderer.DrawWithMid(screen, func() {
		// Tile cursor renders below actors so characters/NPCs can naturally cover it.
		camSX, camSY := render.IsoToScreen(g.mapRenderer.CamX, g.mapRenderer.CamY)
		camSX += wox
		camSY += woy
		g.drawCursor(screen, camSX, camSY)
	})
}

func (g *Game) drawCursor(screen *ebiten.Image, camSX, camSY float64) {
	hover := g.currentWorldHoverIntent()
	if !hover.Valid || hover.CursorType < 0 {
		return // wall/edge — no cursor
	}

	halfW := float64(g.screenW) / 2
	halfH := float64(g.screenH) / 2
	sx, sy := render.IsoToScreen(float64(hover.TileX), float64(hover.TileY))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	cursorImg, err := g.gfxLoad.GetImage(2, 24)
	if err != nil || cursorImg == nil {
		return
	}

	// Cursor sprite sheet: 3 states at 64px intervals, each 64x32
	tw := render.TileWidth
	th := render.TileHeight
	srcX := hover.CursorType * tw
	if srcX+tw > cursorImg.Bounds().Dx() {
		srcX = 0
	}
	sub := cursorImg.SubImage(image.Rect(srcX, 0, srcX+tw, th)).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-float64(tw)/2, sy-float64(th)/2)
	screen.DrawImage(sub, op)
}

func (g *Game) currentWorldHoverIntent() worldHoverIntent {
	tileX, tileY := g.hoveredTile(g.client.UISnapshot())
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

func (g *Game) isMapTileInBounds(tileX, tileY int) bool {
	if g.mapRenderer.Map == nil {
		return false
	}
	if tileX < 0 || tileY < 0 {
		return false
	}
	return tileX <= g.mapRenderer.Map.Width && tileY <= g.mapRenderer.Map.Height
}

func (g *Game) canStepTo(tileX, tileY int) bool {
	if !g.isMapTileInBounds(tileX, tileY) {
		return false
	}
	return g.stepBlockerAt(tileX, tileY) == stepBlockerNone
}

type stepBlocker int

const (
	stepBlockerNone stepBlocker = iota
	stepBlockerTile
	stepBlockerNPC
	stepBlockerPlayer
)

const playerGhostHoldTicks = 24

func (g *Game) canArrowStepTo(tileX, tileY, dir int) bool {
	switch g.stepBlockerAt(tileX, tileY) {
	case stepBlockerNone:
		return true
	case stepBlockerPlayer:
		return g.playerPassThroughAllowed(dir)
	default:
		return false
	}
}

func (g *Game) playerPassThroughAllowed(dir int) bool {
	return dir >= 0 && g.moveHoldDir == dir && g.moveHoldTicks >= playerGhostHoldTicks
}

func (g *Game) updateMoveHold(dir int) {
	if dir < 0 {
		g.moveHoldDir = -1
		g.moveHoldTicks = 0
		return
	}
	if g.moveHoldDir != dir {
		g.moveHoldDir = dir
		g.moveHoldTicks = 1
		return
	}
	g.moveHoldTicks++
}

func (g *Game) stepBlockerAt(tileX, tileY int) stepBlocker {
	if !g.isMapTileInBounds(tileX, tileY) {
		return stepBlockerTile
	}
	if blockedTileSpec(g.tileSpecAt(tileX, tileY)) {
		return stepBlockerTile
	}
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return stepBlockerNPC
		}
	}
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID {
			continue
		}
		if ch.X == tileX && ch.Y == tileY {
			return stepBlockerPlayer
		}
	}
	return stepBlockerNone
}

func (g *Game) tileSpecAt(tileX, tileY int) int {
	if g.mapRenderer.Map == nil {
		return 0
	}
	for _, row := range g.mapRenderer.Map.TileSpecRows {
		if row.Y != tileY {
			continue
		}
		for _, tile := range row.Tiles {
			if tile.X == tileX {
				return int(tile.TileSpec)
			}
		}
	}
	return 0
}

func blockedTileSpec(spec int) bool {
	switch spec {
	case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26:
		return true
	default:
		return false
	}
}

func (g *Game) isAttackTargetTile(tileX, tileY int) bool {
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID {
			continue
		}
		if ch.X == tileX && ch.Y == tileY {
			return true
		}
	}
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return true
		}
	}
	return false
}

func nextTileInDirection(tileX, tileY, dir int) (int, int) {
	switch dir {
	case 0:
		return tileX, tileY + 1
	case 1:
		return tileX - 1, tileY
	case 2:
		return tileX, tileY - 1
	case 3:
		return tileX + 1, tileY
	default:
		return tileX, tileY
	}
}

func directionTowardTile(fromX, fromY, toX, toY int) (int, bool) {
	dx := toX - fromX
	dy := toY - fromY
	if dx == 0 && dy == 0 {
		return -1, false
	}
	if absInt(dx) >= absInt(dy) {
		if dx < 0 {
			return 1, true
		}
		return 3, true
	}
	if dy < 0 {
		return 2, true
	}
	return 0, true
}

func isAdjacentTile(ax, ay, bx, by int) bool {
	return absInt(ax-bx)+absInt(ay-by) == 1
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (g *Game) getCursorType(tileX, tileY int) int {
	// Check if wall/edge tile — hide cursor
	spec := g.tileSpecAt(tileX, tileY)
	if spec == 1 || spec == 16 {
		return -1
	}
	if blockedTileSpec(spec) {
		return 1
	}

	// Check for character or NPC at this tile
	for _, ch := range g.client.NearbyChars {
		if ch.X == tileX && ch.Y == tileY {
			return 1
		}
	}
	for _, npc := range g.client.NearbyNpcs {
		if npc.Dead || npc.Hidden {
			continue
		}
		if npc.X == tileX && npc.Y == tileY {
			return 1
		}
	}

	// Check for items at this tile
	for _, item := range g.client.NearbyItems {
		if item.X == tileX && item.Y == tileY {
			return 2
		}
	}

	return 0 // default walkable cursor
}
