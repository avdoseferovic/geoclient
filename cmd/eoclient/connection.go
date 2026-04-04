package main

import (
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/avdo/eoweb/internal/game"
	eonet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/ui/login"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

func (g *Game) connect() {
	slog.Info("connecting", "addr", g.serverAddr)
	conn, err := eonet.Dial(g.serverAddr)
	if err != nil {
		g.failConnection(fmt.Sprintf("Unable to reach server: %v", err), false)
		return
	}

	bus := eonet.NewPacketBus(conn)
	g.client.SetBus(bus)

	challenge := rand.IntN(11_092_110) + 1
	g.client.Challenge = challenge

	if err := bus.SendPacket(newInitPacket(challenge, g.client.Version)); err != nil {
		g.failConnection(fmt.Sprintf("Handshake failed: %v", err), true)
		return
	}

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
		g.overlay.characterCreate.Submitting = false
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
	case game.EventAccountCreated:
		g.connectError = ""
		g.overlay.authMode = login.ModeSignIn
		g.overlay.loginSubmitting = true
		g.overlay.statusMessage = "Account created. Signing in..."
		g.sendLogin()
	case game.EventEnterGame:
		slog.Info("entered game")
		g.clearAutoWalk()
		g.resetMovementState()
		g.overlay.activeMenuPanel = overlay.MenuPanelNone
		g.facingDir = int(g.client.Character.Direction)
		g.loadCurrentMap()
		g.overlay.loginSubmitting = false
		g.overlay.selectingCharacter = false
		g.overlay.statusMessage = ""
	case game.EventWarp:
		slog.Info("warping", "mapID", evt.Data)
		g.clearAutoWalk()
		g.resetMovementState()
		g.facingDir = int(g.client.Character.Direction)
		g.loadCurrentMap()
	case game.EventCharacterList:
		g.overlay.loginSubmitting = false
		g.overlay.selectingCharacter = false
		g.overlay.characterCreate.Submitting = false
		g.overlay.characterCreateOpen = false
		g.resetCharacterCreateForm()
		if g.overlay.selectedCharacter >= len(g.client.Characters) {
			g.overlay.selectedCharacter = 0
		}
		g.overlay.statusMessage = overlay.TernaryString(len(g.client.Characters) == 0, "Create your first character.", "Choose a character.")
	}
}
