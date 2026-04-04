package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func handleNpcPlayer(c *Client, reader *data.EoReader) error {
	var pkt server.NpcPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		slog.Debug("npc player deserialize error", "err", err)
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	unknownNpcs := make(map[int]struct{})

	for _, pos := range pkt.Positions {
		npc := findNPC(c.NearbyNpcs, pos.NpcIndex)
		if npc == nil {
			unknownNpcs[pos.NpcIndex] = struct{}{}
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
		if npc == nil {
			unknownNpcs[attack.NpcIndex] = struct{}{}
		} else {
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

	requestNPCRangeLocked(c, unknownNpcs)

	return nil
}

func requestNPCRangeLocked(c *Client, indices map[int]struct{}) {
	if len(indices) == 0 || c == nil || c.Bus == nil {
		return
	}

	packet := &client.NpcRangeRequestClientPacket{}
	packet.NpcIndexes = make([]int, 0, len(indices))
	for index := range indices {
		packet.NpcIndexes = append(packet.NpcIndexes, index)
	}
	if err := c.Bus.SendSequenced(packet); err != nil {
		slog.Debug("npc range request failed", "count", len(packet.NpcIndexes), "err", err)
	}
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
