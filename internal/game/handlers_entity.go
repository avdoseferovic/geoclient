package game

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func init() {
	// Additional handlers registered in RegisterEntityHandlers
}

// RegisterEntityHandlers registers handlers for entity updates on the map.
func RegisterEntityHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Avatar, eonet.PacketAction_Agree, handleAvatarAgreeEntity)
	reg.Register(eonet.PacketFamily_Avatar, eonet.PacketAction_Reply, handleAvatarReplyEntity)
	reg.Register(eonet.PacketFamily_Walk, eonet.PacketAction_Player, handleWalkPlayerEntity)
	reg.Register(eonet.PacketFamily_Walk, eonet.PacketAction_Reply, handleWalkReplyEntity)
	reg.Register(eonet.PacketFamily_Avatar, eonet.PacketAction_Remove, handleAvatarRemoveEntity)
	reg.Register(eonet.PacketFamily_Cast, eonet.PacketAction_Reply, handleCastReplyEntity)
	reg.Register(eonet.PacketFamily_Cast, eonet.PacketAction_Spec, handleCastSpecEntity)
	reg.Register(eonet.PacketFamily_Cast, eonet.PacketAction_Accept, handleCastAcceptEntity)
	reg.Register(eonet.PacketFamily_Face, eonet.PacketAction_Player, handleFacePlayer)
	reg.Register(eonet.PacketFamily_Players, eonet.PacketAction_Agree, handlePlayersAgree)
	reg.Register(eonet.PacketFamily_Npc, eonet.PacketAction_Accept, handleNpcAcceptEntity)
	reg.Register(eonet.PacketFamily_Npc, eonet.PacketAction_Player, handleNpcPlayer)
	reg.Register(eonet.PacketFamily_Npc, eonet.PacketAction_Reply, handleNpcReplyEntity)
	reg.Register(eonet.PacketFamily_Npc, eonet.PacketAction_Spec, handleNpcSpecEntity)
	reg.Register(eonet.PacketFamily_Recover, eonet.PacketAction_Player, handleRecoverPlayerEntity)
	reg.Register(eonet.PacketFamily_Warp, eonet.PacketAction_Request, handleWarpRequest)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Server, handleTalkServer)
	reg.Register(eonet.PacketFamily_Refresh, eonet.PacketAction_Reply, handleRefreshReply)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Add, handleItemAdd)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Remove, handleItemRemove)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Drop, handleItemDrop)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Junk, handleItemJunk)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Get, handleItemGet)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Obtain, handleItemObtain)
	reg.Register(eonet.PacketFamily_Item, eonet.PacketAction_Kick, handleItemKick)
	reg.Register(eonet.PacketFamily_Paperdoll, eonet.PacketAction_Reply, handlePaperdollReply)
	reg.Register(eonet.PacketFamily_Paperdoll, eonet.PacketAction_Agree, handlePaperdollAgree)
	reg.Register(eonet.PacketFamily_Paperdoll, eonet.PacketAction_Remove, handlePaperdollRemove)
}

func findCharacter(chars []NearbyCharacter, playerID int) *NearbyCharacter {
	for i := range chars {
		if chars[i].PlayerID == playerID {
			return &chars[i]
		}
	}
	return nil
}

func findNPC(npcs []NearbyNPC, npcIndex int) *NearbyNPC {
	for i := range npcs {
		if npcs[i].Index == npcIndex {
			return &npcs[i]
		}
	}
	return nil
}

func resetNPCTransientState(npc *NearbyNPC) {
	if npc == nil {
		return
	}
	npc.Dead = false
	npc.DeathTick = 0
	npc.Hidden = false
	npc.Walking = false
	npc.WalkTick = 0
	npc.Combat.Reset()
}

func applyEquipmentChange(ch *NearbyCharacter, eq server.EquipmentChange) {
	if ch == nil {
		return
	}
	ch.Boots = eq.Boots
	ch.Armor = eq.Armor
	ch.Hat = eq.Hat
	ch.Weapon = eq.Weapon
	ch.Shield = eq.Shield
}

func addDamageIndicator(state *CombatState, damage int) {
	if state == nil {
		return
	}
	if damage <= 0 {
		state.AddIndicator(CombatIndicatorMiss, "MISS")
		return
	}
	state.AddIndicator(CombatIndicatorDamage, strconv.Itoa(damage))
}

func addHealIndicator(state *CombatState, amount int) {
	if state == nil || amount <= 0 {
		return
	}
	state.AddIndicator(CombatIndicatorHeal, "+"+strconv.Itoa(amount))
}

func syncLocalVitalsFromCharacter(c *Client, ch server.CharacterMapInfo) {
	if ch.PlayerId != c.PlayerID {
		return
	}
	c.Character.X = ch.Coords.X
	c.Character.Y = ch.Coords.Y
	c.Character.Direction = ch.Direction
	c.Character.Name = ch.Name
	c.Character.Level = ch.Level
	c.Character.HP = ch.Hp
	c.Character.MaxHP = ch.MaxHp
	c.Character.TP = ch.Tp
	c.Character.MaxTP = ch.MaxTp
}

func visibleEquipmentFromCharacterMapInfo(ch server.CharacterMapInfo) (boots, armor, hat, weapon, shield int) {
	return ch.Equipment.Boots, ch.Equipment.Armor, ch.Equipment.Hat, ch.Equipment.Weapon, ch.Equipment.Shield
}

func handleAvatarAgreeEntity(c *Client, reader *data.EoReader) error {
	var pkt server.AvatarAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize avatar agree: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ch := findCharacter(c.NearbyChars, pkt.Change.PlayerId)
	if ch == nil {
		return nil
	}

	switch pkt.Change.ChangeType {
	case server.AvatarChange_Equipment:
		if update, ok := pkt.Change.ChangeTypeData.(*server.ChangeTypeDataEquipment); ok {
			applyEquipmentChange(ch, update.Equipment)
		}
	case server.AvatarChange_Hair:
		if update, ok := pkt.Change.ChangeTypeData.(*server.ChangeTypeDataHair); ok {
			ch.HairStyle = update.HairStyle
			ch.HairColor = update.HairColor
		}
	case server.AvatarChange_HairColor:
		if update, ok := pkt.Change.ChangeTypeData.(*server.ChangeTypeDataHairColor); ok {
			ch.HairColor = update.HairColor
		}
	}

	return nil
}

func handleAvatarReplyEntity(c *Client, reader *data.EoReader) error {
	var pkt server.AvatarReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize avatar reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if attacker := findCharacter(c.NearbyChars, pkt.PlayerId); attacker != nil {
		attacker.Direction = int(pkt.Direction)
		attacker.Combat.StartAttack()
	}

	if victim := findCharacter(c.NearbyChars, pkt.VictimId); victim != nil {
		addDamageIndicator(&victim.Combat, pkt.Damage)
	}

	if pkt.VictimId == c.PlayerID && pkt.Damage > 0 {
		c.Character.HP -= pkt.Damage
		if c.Character.HP < 0 {
			c.Character.HP = 0
		}
	}

	return nil
}

func handleWalkPlayerEntity(c *Client, reader *data.EoReader) error {
	var pkt server.WalkPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize walk player: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyChars {
		if c.NearbyChars[i].PlayerID == pkt.PlayerId {
			c.NearbyChars[i].X = pkt.Coords.X
			c.NearbyChars[i].Y = pkt.Coords.Y
			c.NearbyChars[i].Direction = int(pkt.Direction)
			// Start walk animation for remote players
			c.NearbyChars[i].Walking = true
			c.NearbyChars[i].WalkTick = 0
			break
		}
	}
	return nil
}

func handleWalkReplyEntity(c *Client, _ *data.EoReader) error {
	// Walk reply confirms our own movement - already handled client-side
	return nil
}

func handleAvatarRemoveEntity(c *Client, reader *data.EoReader) error {
	var pkt server.AvatarRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize avatar remove: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyChars {
		if c.NearbyChars[i].PlayerID == pkt.PlayerId {
			c.NearbyChars = append(c.NearbyChars[:i], c.NearbyChars[i+1:]...)
			slog.Debug("character left", "playerID", pkt.PlayerId)
			break
		}
	}
	return nil
}

func handleFacePlayer(c *Client, reader *data.EoReader) error {
	var pkt server.FacePlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize face: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyChars {
		if c.NearbyChars[i].PlayerID == pkt.PlayerId {
			c.NearbyChars[i].Direction = int(pkt.Direction)
			break
		}
	}
	return nil
}

func handlePlayersAgree(c *Client, reader *data.EoReader) error {
	var pkt server.PlayersAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize players agree: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Add new characters that appeared
	for _, ch := range pkt.Nearby.Characters {
		found := false
		for _, existing := range c.NearbyChars {
			if existing.PlayerID == ch.PlayerId {
				found = true
				break
			}
		}
		if !found {
			boots, armor, hat, weapon, shield := visibleEquipmentFromCharacterMapInfo(ch)
			c.NearbyChars = append(c.NearbyChars, NearbyCharacter{
				PlayerID:  ch.PlayerId,
				Name:      ch.Name,
				X:         ch.Coords.X,
				Y:         ch.Coords.Y,
				Direction: int(ch.Direction),
				Gender:    int(ch.Gender),
				Skin:      ch.Skin,
				HairStyle: ch.HairStyle,
				HairColor: ch.HairColor,
				Armor:     armor,
				Boots:     boots,
				Hat:       hat,
				Weapon:    weapon,
				Shield:    shield,
				SitState:  int(ch.SitState),
				Level:     ch.Level,
			})
			syncLocalVitalsFromCharacter(c, ch)
			slog.Debug("character appeared", "name", ch.Name, "id", ch.PlayerId)
		} else {
			syncLocalVitalsFromCharacter(c, ch)
		}
	}

	// Add new NPCs
	for _, npc := range pkt.Nearby.Npcs {
		found := false
		for _, existing := range c.NearbyNpcs {
			if existing.Index == npc.Index {
				found = true
				break
			}
		}
		if !found {
			c.NearbyNpcs = append(c.NearbyNpcs, NearbyNPC{
				Index:     npc.Index,
				ID:        npc.Id,
				X:         npc.Coords.X,
				Y:         npc.Coords.Y,
				Direction: int(npc.Direction),
			})
		} else if existing := findNPC(c.NearbyNpcs, npc.Index); existing != nil {
			resetNPCTransientState(existing)
			existing.ID = npc.Id
			existing.X = npc.Coords.X
			existing.Y = npc.Coords.Y
			existing.Direction = int(npc.Direction)
		}
	}

	return nil
}

func handleNpcPlayer(c *Client, reader *data.EoReader) error {
	var pkt server.NpcPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		slog.Debug("npc player deserialize error", "err", err)
		return nil // Don't propagate — partial packets are common
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, pos := range pkt.Positions {
		npc := findNPC(c.NearbyNpcs, pos.NpcIndex)
		if npc == nil {
			c.NearbyNpcs = append(c.NearbyNpcs, NearbyNPC{
				Index:     pos.NpcIndex,
				X:         pos.Coords.X,
				Y:         pos.Coords.Y,
				Direction: int(pos.Direction),
			})
			continue
		}
		if npc.Hidden || npc.Dead {
			resetNPCTransientState(npc)
			npc.X = pos.Coords.X
			npc.Y = pos.Coords.Y
			npc.WalkFromX = pos.Coords.X
			npc.WalkFromY = pos.Coords.Y
			npc.Direction = int(pos.Direction)
			continue
		}
		npc.StartWalk(pos.Coords.X, pos.Coords.Y, int(pos.Direction))
	}

	for _, attack := range pkt.Attacks {
		npc := findNPC(c.NearbyNpcs, attack.NpcIndex)
		if npc != nil {
			npc.Direction = int(attack.Direction)
			npc.Combat.StartAttack()
		}

		victim := findCharacter(c.NearbyChars, attack.PlayerId)
		if victim != nil {
			addDamageIndicator(&victim.Combat, attack.Damage)
		}

		if attack.PlayerId == c.PlayerID && attack.Damage > 0 {
			c.Character.HP -= attack.Damage
			if c.Character.HP < 0 {
				c.Character.HP = 0
			}
		}
	}

	return nil
}

func handleNpcReplyEntity(c *Client, reader *data.EoReader) error {
	var pkt server.NpcReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize npc reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if attacker := findCharacter(c.NearbyChars, pkt.PlayerId); attacker != nil {
		attacker.Direction = int(pkt.PlayerDirection)
		attacker.Combat.StartAttack()
	}

	if npc := findNPC(c.NearbyNpcs, pkt.NpcIndex); npc != nil {
		addDamageIndicator(&npc.Combat, pkt.Damage)
	}

	return nil
}

func handleCastReplyEntity(c *Client, reader *data.EoReader) error {
	var pkt server.CastReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize cast reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if caster := findCharacter(c.NearbyChars, pkt.CasterId); caster != nil {
		caster.Direction = int(pkt.CasterDirection)
		caster.Combat.StartAttack()
	}

	if npc := findNPC(c.NearbyNpcs, pkt.NpcIndex); npc != nil {
		addDamageIndicator(&npc.Combat, pkt.Damage)
	}

	if pkt.CasterId == c.PlayerID && pkt.CasterTp != nil {
		c.Character.TP = *pkt.CasterTp
	}

	return nil
}

func handleNpcSpecEntity(c *Client, reader *data.EoReader) error {
	var pkt server.NpcSpecServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize npc spec: %w", err)
	}
	handleKilledNPC(c, pkt.NpcKilledData)
	return nil
}

func handleNpcAcceptEntity(c *Client, reader *data.EoReader) error {
	var pkt server.NpcAcceptServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize npc accept: %w", err)
	}
	handleKilledNPC(c, pkt.NpcKilledData)
	return nil
}

func handleCastSpecEntity(c *Client, reader *data.EoReader) error {
	var pkt server.CastSpecServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize cast spec: %w", err)
	}
	if pkt.CasterTp != nil {
		c.mu.Lock()
		c.Character.TP = *pkt.CasterTp
		c.mu.Unlock()
	}
	handleKilledNPC(c, pkt.NpcKilledData)
	return nil
}

func handleCastAcceptEntity(c *Client, reader *data.EoReader) error {
	var pkt server.CastAcceptServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize cast accept: %w", err)
	}
	if pkt.CasterTp != nil {
		c.mu.Lock()
		c.Character.TP = *pkt.CasterTp
		c.mu.Unlock()
	}
	handleKilledNPC(c, pkt.NpcKilledData)
	return nil
}

func handleKilledNPC(c *Client, killed server.NpcKilledData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if killer := findCharacter(c.NearbyChars, killed.KillerId); killer != nil {
		killer.Direction = int(killed.KillerDirection)
		killer.Combat.StartAttack()
	}

	npc := findNPC(c.NearbyNpcs, killed.NpcIndex)
	if npc == nil {
		return
	}

	addDamageIndicator(&npc.Combat, killed.Damage)
	npc.StartDeath()
}

func handleRecoverPlayerEntity(c *Client, reader *data.EoReader) error {
	var pkt server.RecoverPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize recover player: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	previousHP := c.Character.HP
	c.Character.HP = pkt.Hp
	c.Character.TP = pkt.Tp

	if previousHP > 0 && pkt.Hp > previousHP {
		if ch := findCharacter(c.NearbyChars, c.PlayerID); ch != nil {
			addHealIndicator(&ch.Combat, pkt.Hp-previousHP)
		}
	}

	return nil
}

func handleWarpRequest(c *Client, reader *data.EoReader) error {
	var pkt server.WarpRequestServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize warp request: %w", err)
	}

	bus := c.GetBus()
	if bus == nil {
		return fmt.Errorf("warp failed: connection bus missing during warp accept")
	}
	if err := bus.SendSequenced(&client.WarpAcceptClientPacket{
		MapId:     pkt.MapId,
		SessionId: pkt.SessionId,
	}); err != nil {
		return fmt.Errorf("warp failed while sending warp accept: %w", err)
	}

	c.Character.MapID = pkt.MapId
	c.SessionID = pkt.SessionId

	c.mu.Lock()
	c.NearbyChars = nil
	c.NearbyNpcs = nil
	c.NearbyItems = nil
	c.mu.Unlock()

	slog.Info("warp request", "mapID", pkt.MapId, "sessionID", pkt.SessionId)

	c.Emit(Event{Type: EventWarp, Data: pkt.MapId})
	return nil
}

func handleTalkServer(c *Client, reader *data.EoReader) error {
	var pkt server.TalkServerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	c.Emit(Event{Type: EventChat, Message: "[Server] " + pkt.Message})
	return nil
}

func handleRefreshReply(c *Client, reader *data.EoReader) error {
	var pkt server.RefreshReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize refresh: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.NearbyChars = nil
	for _, ch := range pkt.Nearby.Characters {
		syncLocalVitalsFromCharacter(c, ch)
		boots, armor, hat, weapon, shield := visibleEquipmentFromCharacterMapInfo(ch)
		nc := NearbyCharacter{
			PlayerID: ch.PlayerId, Name: ch.Name, GuildTag: ch.GuildTag,
			X: ch.Coords.X, Y: ch.Coords.Y,
			Direction: int(ch.Direction), Gender: int(ch.Gender),
			Skin: ch.Skin, HairStyle: ch.HairStyle, HairColor: ch.HairColor,
			Armor: armor, Boots: boots,
			Hat: hat, Weapon: weapon,
			Shield: shield, SitState: int(ch.SitState), Level: ch.Level,
		}
		c.NearbyChars = append(c.NearbyChars, nc)
	}

	c.NearbyNpcs = nil
	for _, npc := range pkt.Nearby.Npcs {
		c.NearbyNpcs = append(c.NearbyNpcs, NearbyNPC{
			Index: npc.Index, ID: npc.Id,
			X: npc.Coords.X, Y: npc.Coords.Y, Direction: int(npc.Direction),
		})
	}

	c.NearbyItems = nil
	for _, item := range pkt.Nearby.Items {
		c.NearbyItems = append(c.NearbyItems, NearbyItem{
			UID: item.Uid, ID: item.Id, GraphicID: item.Id,
			X: item.Coords.X, Y: item.Coords.Y, Amount: item.Amount,
		})
	}

	slog.Debug("refresh", "chars", len(c.NearbyChars), "npcs", len(c.NearbyNpcs))
	return nil
}

func handleItemAdd(c *Client, reader *data.EoReader) error {
	var pkt server.ItemAddServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item add: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems[i].ID = pkt.ItemId
		c.NearbyItems[i].GraphicID = pkt.ItemId
		c.NearbyItems[i].Amount = pkt.ItemAmount
		c.NearbyItems[i].X = pkt.Coords.X
		c.NearbyItems[i].Y = pkt.Coords.Y
		return nil
	}

	c.NearbyItems = append(c.NearbyItems, NearbyItem{UID: pkt.ItemIndex, ID: pkt.ItemId, GraphicID: pkt.ItemId, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.ItemAmount})
	return nil
}

func handleItemRemove(c *Client, reader *data.EoReader) error {
	var pkt server.ItemRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item remove: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems = append(c.NearbyItems[:i], c.NearbyItems[i+1:]...)
		break
	}
	return nil
}

func handleItemDrop(c *Client, reader *data.EoReader) error {
	var pkt server.ItemDropServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item drop: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.DroppedItem.Id, pkt.RemainingAmount)
	syncWeight(c, pkt.Weight)
	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems[i] = NearbyItem{UID: pkt.ItemIndex, ID: pkt.DroppedItem.Id, GraphicID: pkt.DroppedItem.Id, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.DroppedItem.Amount}
		return nil
	}
	c.NearbyItems = append(c.NearbyItems, NearbyItem{UID: pkt.ItemIndex, ID: pkt.DroppedItem.Id, GraphicID: pkt.DroppedItem.Id, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.DroppedItem.Amount})
	return nil
}

func handleItemJunk(c *Client, reader *data.EoReader) error {
	var pkt server.ItemJunkServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item junk: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.JunkedItem.Id, pkt.RemainingAmount)
	syncWeight(c, pkt.Weight)
	return nil
}

func handleItemGet(c *Client, reader *data.EoReader) error {
	var pkt server.ItemGetServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item get: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	syncWeight(c, pkt.Weight)
	addInventoryAmount(&c.Inventory, pkt.TakenItem.Id, pkt.TakenItem.Amount)
	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.TakenItemIndex {
			continue
		}
		c.NearbyItems = append(c.NearbyItems[:i], c.NearbyItems[i+1:]...)
		break
	}
	return nil
}

func handleItemObtain(c *Client, reader *data.EoReader) error {
	var pkt server.ItemObtainServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item obtain: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	addInventoryAmount(&c.Inventory, pkt.Item.Id, pkt.Item.Amount)
	c.Character.Weight.Current = pkt.CurrentWeight
	return nil
}

func handleItemKick(c *Client, reader *data.EoReader) error {
	var pkt server.ItemKickServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item kick: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.Item.Id, pkt.Item.Amount)
	c.Character.Weight.Current = pkt.CurrentWeight
	return nil
}

func handlePaperdollReply(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	syncPaperdollDetails(c, pkt.Details)
	c.Equipment = pkt.Equipment
	return nil
}

func handlePaperdollAgree(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll agree: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.ItemId, pkt.RemainingAmount)
	applyPaperdollSubLoc(&c.Equipment, pkt.SubLoc, pkt.ItemId)
	applyPaperdollAvatarChange(c, pkt.Change)
	applyEquipmentChangeStats(c, pkt.Stats)
	return nil
}

func handlePaperdollRemove(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll remove: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	addInventoryAmount(&c.Inventory, pkt.ItemId, 1)
	applyPaperdollSubLoc(&c.Equipment, pkt.SubLoc, 0)
	applyPaperdollAvatarChange(c, pkt.Change)
	applyEquipmentChangeStats(c, pkt.Stats)
	return nil
}

func applyPaperdollAvatarChange(c *Client, change server.AvatarChange) {
	if ch := findCharacter(c.NearbyChars, change.PlayerId); ch != nil {
		switch change.ChangeType {
		case server.AvatarChange_Equipment:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataEquipment); ok {
				applyEquipmentChange(ch, update.Equipment)
			}
		case server.AvatarChange_Hair:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataHair); ok {
				ch.HairStyle = update.HairStyle
				ch.HairColor = update.HairColor
			}
		case server.AvatarChange_HairColor:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataHairColor); ok {
				ch.HairColor = update.HairColor
			}
		}
	}
}

func applyPaperdollSubLoc(equipment *server.EquipmentPaperdoll, subLoc, itemID int) {
	if equipment == nil {
		return
	}

	switch subLoc {
	case 1:
		equipment.Boots = itemID
	case 2:
		equipment.Accessory = itemID
	case 3:
		equipment.Gloves = itemID
	case 4:
		equipment.Belt = itemID
	case 5:
		equipment.Armor = itemID
	case 6:
		equipment.Necklace = itemID
	case 7:
		equipment.Hat = itemID
	case 8:
		equipment.Shield = itemID
	case 9:
		equipment.Weapon = itemID
	case 10, 11:
		index := subLoc - 10
		for len(equipment.Ring) <= index {
			equipment.Ring = append(equipment.Ring, 0)
		}
		equipment.Ring[index] = itemID
	case 12, 13:
		index := subLoc - 12
		for len(equipment.Armlet) <= index {
			equipment.Armlet = append(equipment.Armlet, 0)
		}
		equipment.Armlet[index] = itemID
	case 14, 15:
		index := subLoc - 14
		for len(equipment.Bracer) <= index {
			equipment.Bracer = append(equipment.Bracer, 0)
		}
		equipment.Bracer[index] = itemID
	}
}
