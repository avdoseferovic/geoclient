package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

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

	c.mu.Lock()
	c.Character.MapID = pkt.MapId
	c.SessionID = pkt.SessionId
	c.mu.Unlock()

	slog.Info("warp request", "mapID", pkt.MapId, "sessionID", pkt.SessionId)
	return nil
}

func handleWarpAgree(c *Client, reader *data.EoReader) error {
	var pkt server.WarpAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize warp agree: %w", err)
	}

	c.mu.Lock()
	replaceNearbyInfo(c, pkt.Nearby)
	c.mu.Unlock()

	slog.Debug("warp agree", "warpType", int(pkt.WarpType), "chars", len(pkt.Nearby.Characters), "npcs", len(pkt.Nearby.Npcs), "items", len(pkt.Nearby.Items))
	c.Emit(Event{Type: EventWarp, Data: c.Character.MapID})
	return nil
}

func handleTalkServer(c *Client, reader *data.EoReader) error {
	var pkt server.TalkServerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChatAllChannels(c, "[Server] "+pkt.Message)
	return nil
}

func handleRefreshReply(c *Client, reader *data.EoReader) error {
	var pkt server.RefreshReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize refresh: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	replaceNearbyInfo(c, pkt.Nearby)

	slog.Debug("refresh", "chars", len(c.NearbyChars), "npcs", len(c.NearbyNpcs))
	return nil
}

func handleRangeReplyEntity(c *Client, reader *data.EoReader) error {
	var pkt server.RangeReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize range reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	mergeNearbyInfo(c, pkt.Nearby)
	slog.Debug("range reply", "chars", len(pkt.Nearby.Characters), "npcs", len(pkt.Nearby.Npcs), "items", len(pkt.Nearby.Items))
	return nil
}
