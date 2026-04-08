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
	g.client.Lock()
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
	charX, charY := g.client.Character.X, g.client.Character.Y
	g.client.Unlock()

	if err := bus.SendSequenced(&client.WalkPlayerClientPacket{
		WalkAction: client.WalkAction{
			Direction: eoproto.Direction(dir),
			Timestamp: 0,
			Coords:    eoproto.Coords{X: charX, Y: charY},
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
	g.client.Lock()
	g.client.Character.Direction = eoproto.Direction(dir)

	// Sync nearby chars entry
	for i := range g.client.NearbyChars {
		if g.client.NearbyChars[i].PlayerID == g.client.PlayerID {
			g.client.NearbyChars[i].Direction = dir
			break
		}
	}
	g.client.Unlock()

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

func (g *Game) sendTrainStat(statID int) {
	bus := g.client.GetBus()
	if bus == nil || statID < 1 || statID > 6 {
		return
	}
	if err := bus.SendSequenced(&client.StatSkillAddClientPacket{
		ActionType: client.Train_Stat,
		ActionTypeData: &client.StatSkillAddActionTypeDataStat{
			StatId: client.StatId(statID),
		},
	}); err != nil {
		slog.Error("send train stat failed", "statID", statID, "err", err)
	}
}

func (g *Game) sendTradeRequest(playerID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeRequestClientPacket{PlayerId: playerID}); err != nil {
		slog.Error("send trade request failed", "err", err)
	}
}

func (g *Game) sendTradeAccept(playerID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeAcceptClientPacket{PlayerId: playerID}); err != nil {
		slog.Error("send trade accept failed", "err", err)
	}
}

func (g *Game) sendTradeAdd(itemID, amount int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeAddClientPacket{
		AddItem: eonet.Item{Id: itemID, Amount: amount},
	}); err != nil {
		slog.Error("send trade add failed", "err", err)
	}
}

func (g *Game) sendTradeRemove(itemID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeRemoveClientPacket{ItemId: itemID}); err != nil {
		slog.Error("send trade remove failed", "err", err)
	}
}

func (g *Game) sendTradeAgree(agree bool) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeAgreeClientPacket{Agree: agree}); err != nil {
		slog.Error("send trade agree failed", "err", err)
	}
}

func (g *Game) sendTradeClose() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.TradeCloseClientPacket{}); err != nil {
		slog.Error("send trade close failed", "err", err)
	}
}

func (g *Game) sendPartyRequest(playerID int, requestType int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.PartyRequestClientPacket{
		RequestType: eonet.PartyRequestType(requestType),
		PlayerId:    playerID,
	}); err != nil {
		slog.Error("send party request failed", "err", err)
	}
}

func (g *Game) sendPartyAccept(playerID int, requestType int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.PartyAcceptClientPacket{
		RequestType:     eonet.PartyRequestType(requestType),
		InviterPlayerId: playerID,
	}); err != nil {
		slog.Error("send party accept failed", "err", err)
	}
}

func (g *Game) sendPartyRemove(playerID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.PartyRemoveClientPacket{
		PlayerId: playerID,
	}); err != nil {
		slog.Error("send party remove failed", "err", err)
	}
}

func (g *Game) sendPlayersRequest() {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.PlayersRequestClientPacket{}); err != nil {
		slog.Error("send players request failed", "err", err)
	}
}

func (g *Game) sendDoorOpen(x, y int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.DoorOpenClientPacket{
		Coords: eoproto.Coords{X: x, Y: y},
	}); err != nil {
		slog.Error("send door open failed", "err", err)
	}
}

func (g *Game) sendChestOpen(x, y int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.ChestOpenClientPacket{
		Coords: eoproto.Coords{X: x, Y: y},
	}); err != nil {
		slog.Error("send chest open failed", "err", err)
	}
}

func (g *Game) sendChestTake(x, y, itemID int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.ChestTakeClientPacket{
		Coords:     eoproto.Coords{X: x, Y: y},
		TakeItemId: itemID,
	}); err != nil {
		slog.Error("send chest take failed", "err", err)
	}
}

func (g *Game) sendChestAdd(x, y, itemID, amount int) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}
	if err := bus.SendSequenced(&client.ChestAddClientPacket{
		Coords:  eoproto.Coords{X: x, Y: y},
		AddItem: eonet.ThreeItem{Id: itemID, Amount: amount},
	}); err != nil {
		slog.Error("send chest add failed", "err", err)
	}
}

func (g *Game) sendNpcInteract(npcIndex int, npcID int) {
	bus := g.client.GetBus()
	if bus == nil || npcIndex <= 0 {
		return
	}

	var pkt eonet.Packet
	npcType := pub.Npc_Friendly
	if g.npcDB != nil {
		npcType = g.npcDB.Type(npcID)
	}

	switch npcType {
	case pub.Npc_Shop:
		pkt = &client.ShopOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Bank:
		pkt = &client.BankOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Inn:
		pkt = &client.CitizenOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Barber:
		pkt = &client.BarberOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Guild:
		pkt = &client.GuildOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Priest:
		pkt = &client.PriestOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Lawyer:
		pkt = &client.MarriageOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Trainer:
		pkt = &client.StatSkillOpenClientPacket{NpcIndex: npcIndex}
	case pub.Npc_Quest:
		pkt = &client.QuestUseClientPacket{NpcIndex: npcIndex}
	default:
		return
	}

	if err := bus.SendSequenced(pkt); err != nil {
		slog.Error("send npc interact failed", "npcIndex", npcIndex, "type", npcType, "err", err)
	}
}

func (g *Game) sendShopBuy(itemID, amount int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 || amount <= 0 {
		return
	}
	if err := bus.SendSequenced(&client.ShopBuyClientPacket{
		BuyItem:   eonet.Item{Id: itemID, Amount: amount},
		SessionId: g.client.SessionID,
	}); err != nil {
		slog.Error("send shop buy failed", "itemID", itemID, "amount", amount, "err", err)
	}
}

func (g *Game) sendShopSell(itemID, amount int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 || amount <= 0 {
		return
	}
	if err := bus.SendSequenced(&client.ShopSellClientPacket{
		SellItem:  eonet.Item{Id: itemID, Amount: amount},
		SessionId: g.client.SessionID,
	}); err != nil {
		slog.Error("send shop sell failed", "itemID", itemID, "amount", amount, "err", err)
	}
}

func (g *Game) sendShopCraft(itemID int) {
	bus := g.client.GetBus()
	if bus == nil || itemID <= 0 {
		return
	}
	if err := bus.SendSequenced(&client.ShopCreateClientPacket{
		CraftItemId: itemID,
		SessionId:   g.client.SessionID,
	}); err != nil {
		slog.Error("send shop craft failed", "itemID", itemID, "err", err)
	}
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
