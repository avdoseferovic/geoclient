package main

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/gfx"
	eonet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/render"
)

const (
	screenWidth  = 640
	screenHeight = 480
	serverAddr   = "ws://127.0.0.1:8078"
	gfxDir       = "gfx"
	mapsDir      = "maps"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	loader := gfx.NewLoader(gfxDir)
	g := &Game{
		client:      game.NewClient(),
		handlers:    game.NewHandlerRegistry(),
		gfxLoad:     loader,
		mapRenderer: render.NewMapRenderer(loader),
	}
	game.RegisterAllHandlers(g.handlers)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("EO Client")
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(g); err != nil {
		slog.Error("game exited", "err", err)
		os.Exit(1)
	}
}

type Game struct {
	client      *game.Client
	handlers    *game.HandlerRegistry
	gfxLoad     *gfx.Loader
	mapRenderer *render.MapRenderer

	connected    bool
	connectError string

	// Chat
	chat ChatState

	// Movement
	walkCooldown   int // ticks until next walk allowed
	attackCooldown int
	facingDir      int  // current facing direction (0-3)
	isWalking      bool // true while local player walk animation is playing
}

func (g *Game) Update() error {
	// Handle connection
	if !g.connected && g.client.GetState() == game.StateInitial && g.connectError == "" {
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
	switch g.client.GetState() {
	case game.StateInitial:
		g.drawConnecting(screen)
	case game.StateConnected:
		g.drawLogin(screen)
	case game.StateLoggedIn:
		g.drawCharacterSelect(screen)
	case game.StateInGame:
		g.drawInGame(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) connect() {
	slog.Info("connecting", "addr", serverAddr)
	conn, err := eonet.Dial(serverAddr)
	if err != nil {
		g.connectError = err.Error()
		slog.Error("connect failed", "err", err)
		g.connected = false
		return
	}

	bus := eonet.NewPacketBus(conn)
	g.client.Bus = bus

	challenge := rand.Intn(11_092_110) + 1
	g.client.Challenge = challenge

	// Send init packet
	if err := bus.SendPacket(newInitPacket(challenge, g.client.Version)); err != nil {
		g.connectError = err.Error()
		slog.Error("send init failed", "err", err)
		return
	}

	// Start receive loop
	go g.recvLoop()
}

func (g *Game) recvLoop() {
	for {
		bus := g.client.Bus
		if bus == nil {
			return
		}

		action, family, reader, err := bus.Recv()
		if err != nil {
			slog.Error("recv error", "err", err)
			g.client.Disconnect()
			g.connected = false
			return
		}

		if err := g.handlers.Dispatch(family, action, g.client, reader); err != nil {
			slog.Error("handler error", "family", int(family), "action", int(action), "err", err)
		}
	}
}

func (g *Game) handleEvent(evt game.Event) {
	switch evt.Type {
	case game.EventError:
		slog.Error("game error", "msg", evt.Message)
		g.connectError = evt.Message
	case game.EventChat:
		g.handleChatEvent(evt)
	case game.EventStateChanged:
		slog.Info("state changed", "state", evt.Message)
		if evt.Message == "Connected" {
			g.connectError = ""
		}
	case game.EventEnterGame:
		slog.Info("entered game")
		g.facingDir = int(g.client.Character.Direction)
		g.loadCurrentMap()
	case game.EventWarp:
		slog.Info("warping", "mapID", evt.Data)
		g.loadCurrentMap()
	}
}

func (g *Game) updateLogin() {
	// Auto-login for now (will be replaced with UI)
	if g.client.Username == "" {
		g.client.Username = "testbot1"
		g.client.Password = "testpass"
		g.sendLogin()
	}
}

func (g *Game) updateCharacterSelect() {
	// Auto-select first character
	if len(g.client.Characters) > 0 {
		charID := g.client.Characters[0].Id
		slog.Info("selecting character", "id", charID)
		g.sendSelectCharacter(charID)
		g.client.Characters = nil // prevent re-selecting
	}
}

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

	// Attack with Ctrl
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		if g.attackCooldown <= 0 {
			g.sendAttack()
			g.attackCooldown = 18
		}
		return
	}

	// Don't accept new input while walking
	if g.isWalking {
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

	if dir < 0 || g.walkCooldown > 0 {
		return
	}

	if g.facingDir != dir {
		// Different direction: just face, don't walk
		g.facingDir = dir
		g.sendFace(dir)
		g.walkCooldown = 8 // short cooldown for face
	} else {
		// Already facing this direction: walk
		g.startLocalWalk(dir)
		g.walkCooldown = game.WalkDuration
	}
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
		if ch.TickWalk() {
			if ch.PlayerID == g.client.PlayerID {
				g.isWalking = false
			}
		}
	}
	// Tick NPC animations (idle + walk)
	for i := range g.client.NearbyNpcs {
		g.client.NearbyNpcs[i].Tick()
	}
}

func (g *Game) loadCurrentMap() {
	mapPath := fmt.Sprintf("%s/%05d.emf", mapsDir, g.client.Character.MapID)
	if err := g.mapRenderer.LoadMap(mapPath); err != nil {
		slog.Error("failed to load map", "path", mapPath, "err", err)
	}
}

func (g *Game) drawConnecting(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 20, G: 20, B: 40, A: 255})
	msg := "Connecting to server..."
	if g.connectError != "" {
		msg = fmt.Sprintf("Error: %s\nPress R to retry", g.connectError)
	}
	ebitenutil.DebugPrintAt(screen, msg, 20, 20)
}

func (g *Game) drawLogin(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 20, G: 20, B: 40, A: 255})
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Logging in as '%s'...", g.client.Username), 20, 20)
}

func (g *Game) drawCharacterSelect(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 20, G: 20, B: 40, A: 255})
	ebitenutil.DebugPrintAt(screen, "Selecting character...", 20, 20)
}

func (g *Game) playerCamOffset() (float64, float64) {
	for _, ch := range g.client.NearbyChars {
		if ch.PlayerID == g.client.PlayerID && ch.Walking {
			return render.WalkOffset(ch.Direction, ch.WalkProgress())
		}
	}
	return 0, 0
}

func (g *Game) drawInGame(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	g.mapRenderer.CamX = float64(g.client.Character.X)
	g.mapRenderer.CamY = float64(g.client.Character.Y)

	// Smooth camera: offset by walk animation progress
	wox, woy := g.playerCamOffset()
	g.mapRenderer.CamOffX = wox
	g.mapRenderer.CamOffY = woy

	g.syncEntities()
	g.mapRenderer.Draw(screen)

	// Tile cursor
	camSX, camSY := render.IsoToScreen(g.mapRenderer.CamX, g.mapRenderer.CamY)
	camSX += wox
	camSY += woy
	g.drawCursor(screen, camSX, camSY)

	// HUD
	mx, my := ebiten.CursorPosition()
	isoX, isoY := render.ScreenToIso(
		float64(mx)-float64(screenWidth)/2+camSX,
		float64(my)-float64(screenHeight)/2+camSY+render.HalfTileH,
	)
	dir := [4]string{"Down", "Left", "Up", "Right"}[g.facingDir]
	hud := fmt.Sprintf("%s  Map:%d  Pos:(%d,%d)  Tile:(%d,%d)  Dir:%s  FPS:%.0f",
		g.client.Character.Name, g.client.Character.MapID,
		g.client.Character.X, g.client.Character.Y,
		int(isoX), int(isoY), dir, ebiten.ActualFPS(),
	)
	ebitenutil.DebugPrintAt(screen, hud, 4, 4)

	g.drawChat(screen)
}

func (g *Game) drawCursor(screen *ebiten.Image, camSX, camSY float64) {
	mx, my := ebiten.CursorPosition()
	halfW := float64(screenWidth) / 2
	halfH := float64(screenHeight) / 2

	worldX := float64(mx) - halfW + camSX
	worldY := float64(my) - halfH + camSY + render.HalfTileH
	ix, iy := render.ScreenToIso(worldX, worldY)

	// Don't show cursor outside map bounds
	if g.mapRenderer.Map != nil {
		if ix < 0 || iy < 0 || ix > g.mapRenderer.Map.Width || iy > g.mapRenderer.Map.Height {
			return
		}
	}

	// Determine cursor type: 0=walk, 1=interact (NPC/player/chair), 2=item
	cursorType := g.getCursorType(ix, iy)
	if cursorType < 0 {
		return // wall/edge — no cursor
	}

	sx, sy := render.IsoToScreen(float64(ix), float64(iy))
	sx = sx - camSX + halfW
	sy = sy - camSY + halfH

	cursorImg, err := g.gfxLoad.GetImage(2, 24)
	if err != nil || cursorImg == nil {
		return
	}

	// Cursor sprite sheet: 3 states at 64px intervals, each 64x32
	tw := render.TileWidth
	th := render.TileHeight
	srcX := cursorType * tw
	if srcX+tw > cursorImg.Bounds().Dx() {
		srcX = 0
	}
	sub := cursorImg.SubImage(image.Rect(srcX, 0, srcX+tw, th)).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-float64(tw)/2, sy-float64(th)/2)
	screen.DrawImage(sub, op)
}

func (g *Game) getCursorType(tileX, tileY int) int {
	// Check if wall/edge tile — hide cursor
	if g.mapRenderer.Map != nil {
		for _, row := range g.mapRenderer.Map.TileSpecRows {
			if row.Y != tileY {
				continue
			}
			for _, tile := range row.Tiles {
				if tile.X == tileX {
					spec := tile.TileSpec
					// MapTileSpec_Wall=1, MapTileSpec_Edge=16
					if spec == 1 || spec == 16 {
						return -1
					}
					// Interactive tiles (chairs, chests, boards, etc.)
					switch spec {
					case 4, 5, 6, 7, 8, 9, 10, 11, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26:
						return 1
					}
				}
			}
		}
	}

	// Check for character or NPC at this tile
	for _, ch := range g.client.NearbyChars {
		if ch.X == tileX && ch.Y == tileY {
			return 1
		}
	}
	for _, npc := range g.client.NearbyNpcs {
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
