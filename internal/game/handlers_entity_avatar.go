package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

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
			c.NearbyChars[i].Walking = true
			c.NearbyChars[i].WalkTick = 0
			break
		}
	}
	return nil
}

func handleWalkReplyEntity(c *Client, _ *data.EoReader) error {
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
