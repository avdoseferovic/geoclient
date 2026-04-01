package main

import (
	"log/slog"

	eoproto "github.com/ethanmoffat/eolib-go/v3/protocol"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
)

func (g *Game) startLocalAttackAnimation() {
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID != g.client.PlayerID {
			continue
		}
		g.client.NearbyChars[i].Direction = int(g.client.Character.Direction)
		g.client.NearbyChars[i].Combat.StartAttack()
		return
	}
}

func newInitPacket(challenge int, version eonet.Version) *client.InitInitClientPacket {
	return &client.InitInitClientPacket{
		Challenge: challenge,
		Version:   version,
		Hdid:      "eoweb-go",
	}
}

func (g *Game) sendLogin() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	slog.Info("sending login", "user", g.client.Username)
	if err := bus.SendSequenced(&client.LoginRequestClientPacket{
		Username: g.client.Username,
		Password: g.client.Password,
	}); err != nil {
		slog.Error("send login failed", "err", err)
	}
}

func (g *Game) sendSelectCharacter(charID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	slog.Info("selecting character", "charID", charID)
	if err := bus.SendSequenced(&client.WelcomeRequestClientPacket{
		CharacterId: charID,
	}); err != nil {
		slog.Error("send welcome request failed", "err", err)
	}
}

func (g *Game) sendWalk(dir int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}

	// Update local position: 0=Down, 1=Left, 2=Up, 3=Right
	switch dir {
	case 0: // Down
		g.client.Character.Y++
	case 1: // Left
		g.client.Character.X--
	case 2: // Up
		g.client.Character.Y--
	case 3: // Right
		g.client.Character.X++
	}

	// Sync nearby chars entry for our player
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID == g.client.PlayerID {
			g.client.NearbyChars[i].X = g.client.Character.X
			g.client.NearbyChars[i].Y = g.client.Character.Y
			g.client.NearbyChars[i].Direction = dir
			break
		}
	}

	if err := bus.SendSequenced(&client.WalkPlayerClientPacket{
		WalkAction: client.WalkAction{
			Direction: eoproto.Direction(dir),
			Timestamp: 0,
			Coords:    eoproto.Coords{X: g.client.Character.X, Y: g.client.Character.Y},
		},
	}); err != nil {
		slog.Error("send walk failed", "err", err)
	}
}

func (g *Game) sendAttack() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	g.client.Lock()
	g.startLocalAttackAnimation()
	g.client.Unlock()
	if err := bus.SendSequenced(&client.AttackUseClientPacket{
		Direction: eoproto.Direction(g.client.Character.Direction),
		Timestamp: 0,
	}); err != nil {
		slog.Error("send attack failed", "err", err)
	}
}

func (g *Game) sendFace(dir int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	g.client.Character.Direction = eoproto.Direction(dir)

	// Sync nearby chars entry
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID == g.client.PlayerID {
			g.client.NearbyChars[i].Direction = dir
			break
		}
	}

	if err := bus.SendSequenced(&client.FacePlayerClientPacket{
		Direction: eoproto.Direction(dir),
	}); err != nil {
		slog.Error("send face failed", "err", err)
	}
}

func (g *Game) sendPickupItem(itemUID int) {
	bus := g.client.GetBus()
	if bus == nil || itemUID <= 0 {
		return
	}
	if err := bus.SendSequenced(&client.ItemGetClientPacket{ItemIndex: itemUID}); err != nil {
		slog.Error("send item pickup failed", "itemUID", itemUID, "err", err)
	}
}
