package game

import (
	"strconv"

	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
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
	reg.Register(eonet.PacketFamily_Range, eonet.PacketAction_Reply, handleRangeReplyEntity)
	reg.Register(eonet.PacketFamily_Warp, eonet.PacketAction_Request, handleWarpRequest)
	reg.Register(eonet.PacketFamily_Warp, eonet.PacketAction_Agree, handleWarpAgree)
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

func mergeNearbyInfo(c *Client, nearby server.NearbyInfo) {
	for _, ch := range nearby.Characters {
		syncLocalVitalsFromCharacter(c, ch)
		boots, armor, hat, weapon, shield := visibleEquipmentFromCharacterMapInfo(ch)
		existing := findCharacter(c.NearbyChars, ch.PlayerId)
		if existing == nil {
			c.NearbyChars = append(c.NearbyChars, NearbyCharacter{
				PlayerID:  ch.PlayerId,
				Name:      ch.Name,
				GuildTag:  ch.GuildTag,
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
			continue
		}

		existing.Name = ch.Name
		existing.GuildTag = ch.GuildTag
		existing.X = ch.Coords.X
		existing.Y = ch.Coords.Y
		existing.Direction = int(ch.Direction)
		existing.Gender = int(ch.Gender)
		existing.Skin = ch.Skin
		existing.HairStyle = ch.HairStyle
		existing.HairColor = ch.HairColor
		existing.Armor = armor
		existing.Boots = boots
		existing.Hat = hat
		existing.Weapon = weapon
		existing.Shield = shield
		existing.SitState = int(ch.SitState)
		existing.Level = ch.Level
	}

	for _, npc := range nearby.Npcs {
		existing := findNPC(c.NearbyNpcs, npc.Index)
		if existing == nil {
			c.NearbyNpcs = append(c.NearbyNpcs, NearbyNPC{
				Index:     npc.Index,
				ID:        npc.Id,
				X:         npc.Coords.X,
				Y:         npc.Coords.Y,
				Direction: int(npc.Direction),
			})
			continue
		}

		existing.ID = npc.Id
		existing.X = npc.Coords.X
		existing.Y = npc.Coords.Y
		existing.Direction = int(npc.Direction)
	}

	for _, item := range nearby.Items {
		found := false
		for i := range c.NearbyItems {
			if c.NearbyItems[i].UID != item.Uid {
				continue
			}
			c.NearbyItems[i].ID = item.Id
			c.NearbyItems[i].GraphicID = item.Id
			c.NearbyItems[i].X = item.Coords.X
			c.NearbyItems[i].Y = item.Coords.Y
			c.NearbyItems[i].Amount = item.Amount
			found = true
			break
		}
		if !found {
			c.NearbyItems = append(c.NearbyItems, NearbyItem{
				UID:       item.Uid,
				ID:        item.Id,
				GraphicID: item.Id,
				X:         item.Coords.X,
				Y:         item.Coords.Y,
				Amount:    item.Amount,
			})
		}
	}
}

func replaceNearbyInfo(c *Client, nearby server.NearbyInfo) {
	c.NearbyChars = nil
	c.NearbyNpcs = nil
	c.NearbyItems = nil
	mergeNearbyInfo(c, nearby)
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
