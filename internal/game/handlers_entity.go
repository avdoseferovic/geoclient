package game

import (
	"fmt"
	"log/slog"

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
	reg.Register(eonet.PacketFamily_Walk, eonet.PacketAction_Player, handleWalkPlayerEntity)
	reg.Register(eonet.PacketFamily_Walk, eonet.PacketAction_Reply, handleWalkReplyEntity)
	reg.Register(eonet.PacketFamily_Avatar, eonet.PacketAction_Remove, handleAvatarRemoveEntity)
	reg.Register(eonet.PacketFamily_Face, eonet.PacketAction_Player, handleFacePlayer)
	reg.Register(eonet.PacketFamily_Players, eonet.PacketAction_Agree, handlePlayersAgree)
	reg.Register(eonet.PacketFamily_Npc, eonet.PacketAction_Player, handleNpcPlayer)
	reg.Register(eonet.PacketFamily_Warp, eonet.PacketAction_Request, handleWarpRequest)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Server, handleTalkServer)
	reg.Register(eonet.PacketFamily_Refresh, eonet.PacketAction_Reply, handleRefreshReply)
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
				Armor:     ch.Equipment.Armor,
				Boots:     ch.Equipment.Boots,
				Hat:       ch.Equipment.Hat,
				Weapon:    ch.Equipment.Weapon,
				Shield:    ch.Equipment.Shield,
				SitState:  int(ch.SitState),
				Level:     ch.Level,
			})
			slog.Debug("character appeared", "name", ch.Name, "id", ch.PlayerId)
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
		for i := range c.NearbyNpcs {
			if c.NearbyNpcs[i].Index == pos.NpcIndex {
				c.NearbyNpcs[i].StartWalk(pos.Coords.X, pos.Coords.Y, int(pos.Direction))
				break
			}
		}
	}

	return nil
}

func handleWarpRequest(c *Client, reader *data.EoReader) error {
	var pkt server.WarpRequestServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize warp request: %w", err)
	}

	c.Character.MapID = pkt.MapId
	c.SessionID = pkt.SessionId

	c.mu.Lock()
	c.NearbyChars = nil
	c.NearbyNpcs = nil
	c.NearbyItems = nil
	c.mu.Unlock()

	slog.Info("warp request", "mapID", pkt.MapId, "sessionID", pkt.SessionId)

	if c.Bus != nil {
		c.Bus.SendSequenced(&client.WarpAcceptClientPacket{
			MapId:     pkt.MapId,
			SessionId: pkt.SessionId,
		})
	}

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
		c.NearbyChars = append(c.NearbyChars, NearbyCharacter{
			PlayerID: ch.PlayerId, Name: ch.Name,
			X: ch.Coords.X, Y: ch.Coords.Y,
			Direction: int(ch.Direction), Gender: int(ch.Gender),
			Skin: ch.Skin, HairStyle: ch.HairStyle, HairColor: ch.HairColor,
			Armor: ch.Equipment.Armor, Boots: ch.Equipment.Boots,
			Hat: ch.Equipment.Hat, Weapon: ch.Equipment.Weapon,
			Shield: ch.Equipment.Shield, SitState: int(ch.SitState), Level: ch.Level,
		})
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
