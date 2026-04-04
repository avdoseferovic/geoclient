package main

import (
	"log/slog"

	eoproto "github.com/ethanmoffat/eolib-go/v3/protocol"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
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

func (g *Game) sendAccountCreateRequest() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	slog.Info("requesting account creation", "user", g.client.Username)
	if err := bus.SendSequenced(&client.AccountRequestClientPacket{
		Username: g.client.Username,
	}); err != nil {
		slog.Error("send account request failed", "err", err)
	}
}

func (g *Game) sendCharacterCreateRequest() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	slog.Info("requesting character creation")
	if err := bus.SendSequenced(&client.CharacterRequestClientPacket{}); err != nil {
		slog.Error("send character request failed", "err", err)
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

func (g *Game) sendDropItem(itemID, amount, tileX, tileY int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 || amount <= 0 {
		return
	}
	if err := bus.SendSequenced(&client.ItemDropClientPacket{
		Item: eonet.ThreeItem{
			Id:     itemID,
			Amount: amount,
		},
		Coords: client.ByteCoords{
			X: tileX + 1,
			Y: tileY + 1,
		},
	}); err != nil {
		slog.Error("send item drop failed", "itemID", itemID, "amount", amount, "x", tileX, "y", tileY, "err", err)
	}
}

func (g *Game) sendEquipItem(itemID int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 {
		return
	}
	subLoc := 0
	if g.itemDB != nil {
		if item, ok := g.itemDB.Get(itemID); ok {
			if isDualSlotType(item.Type) && dualSlotFirstOccupied(g.client.Equipment, item.Type) {
				subLoc = 1
			}
		}
	}
	slog.Info("equip item", "itemID", itemID, "subLoc", subLoc)
	if err := bus.SendSequenced(&client.PaperdollAddClientPacket{
		ItemId: itemID,
		SubLoc: subLoc,
	}); err != nil {
		slog.Error("send equip failed", "itemID", itemID, "err", err)
	}
}

func (g *Game) sendUnequipItem(itemID, absoluteSlot int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 {
		return
	}
	// Server expects SubLoc 0 (first slot) or 1 (second dual-slot).
	subLoc := 0
	if absoluteSlot == 11 || absoluteSlot == 13 || absoluteSlot == 15 {
		subLoc = 1 // Ring2, Armlet2, Bracer2
	}
	slog.Info("unequip item", "itemID", itemID, "subLoc", subLoc)
	if err := bus.SendSequenced(&client.PaperdollRemoveClientPacket{
		ItemId: itemID,
		SubLoc: subLoc,
	}); err != nil {
		slog.Error("send unequip failed", "itemID", itemID, "err", err)
	}
}

func isEquippableType(t pub.ItemType) bool {
	switch t {
	case pub.Item_Weapon, pub.Item_Shield, pub.Item_Armor, pub.Item_Hat,
		pub.Item_Boots, pub.Item_Gloves, pub.Item_Accessory, pub.Item_Belt,
		pub.Item_Necklace, pub.Item_Ring, pub.Item_Armlet, pub.Item_Bracer:
		return true
	}
	return false
}

func isDualSlotType(t pub.ItemType) bool {
	return t == pub.Item_Ring || t == pub.Item_Armlet || t == pub.Item_Bracer
}

func dualSlotFirstOccupied(eq server.EquipmentPaperdoll, t pub.ItemType) bool {
	switch t {
	case pub.Item_Ring:
		return len(eq.Ring) > 0 && eq.Ring[0] > 0
	case pub.Item_Armlet:
		return len(eq.Armlet) > 0 && eq.Armlet[0] > 0
	case pub.Item_Bracer:
		return len(eq.Bracer) > 0 && eq.Bracer[0] > 0
	}
	return false
}
