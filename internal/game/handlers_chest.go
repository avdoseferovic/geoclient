package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterChestHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Chest, eonet.PacketAction_Open, handleChestOpen)
	reg.Register(eonet.PacketFamily_Chest, eonet.PacketAction_Reply, handleChestReply)
	reg.Register(eonet.PacketFamily_Chest, eonet.PacketAction_Get, handleChestGet)
	reg.Register(eonet.PacketFamily_Chest, eonet.PacketAction_Agree, handleChestAgree)
}

func handleChestOpen(c *Client, reader *data.EoReader) error {
	var pkt server.ChestOpenServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize chest open: %w", err)
	}

	c.mu.Lock()
	c.ChestItems = netThreeItemsToChest(pkt.Items)
	c.ChestOpen = true
	c.mu.Unlock()

	slog.Debug("chest opened", "items", len(pkt.Items))
	c.Emit(Event{Type: EventChestOpened, Data: c.ChestItems})
	return nil
}

func handleChestReply(c *Client, reader *data.EoReader) error {
	var pkt server.ChestReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize chest reply: %w", err)
	}

	c.mu.Lock()
	setInventoryAmount(&c.Inventory, pkt.AddedItemId, pkt.RemainingAmount)
	syncWeight(c, pkt.Weight)
	c.ChestItems = netThreeItemsToChest(pkt.Items)
	c.mu.Unlock()

	slog.Debug("chest deposit", "itemID", pkt.AddedItemId, "remaining", pkt.RemainingAmount)
	c.Emit(Event{Type: EventChestChanged})
	return nil
}

func handleChestGet(c *Client, reader *data.EoReader) error {
	var pkt server.ChestGetServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize chest get: %w", err)
	}

	c.mu.Lock()
	addInventoryAmount(&c.Inventory, pkt.TakenItem.Id, pkt.TakenItem.Amount)
	syncWeight(c, pkt.Weight)
	c.ChestItems = netThreeItemsToChest(pkt.Items)
	c.mu.Unlock()

	slog.Debug("chest withdraw", "itemID", pkt.TakenItem.Id, "amount", pkt.TakenItem.Amount)
	c.Emit(Event{Type: EventChestChanged})
	return nil
}

func handleChestAgree(c *Client, reader *data.EoReader) error {
	var pkt server.ChestAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize chest agree: %w", err)
	}

	c.mu.Lock()
	c.ChestItems = netThreeItemsToChest(pkt.Items)
	c.mu.Unlock()

	slog.Debug("chest updated by another player", "items", len(pkt.Items))
	c.Emit(Event{Type: EventChestChanged})
	return nil
}

func netThreeItemsToChest(items []eonet.ThreeItem) []ChestItem {
	result := make([]ChestItem, len(items))
	for i, item := range items {
		result[i] = ChestItem{ID: item.Id, Amount: item.Amount}
	}
	return result
}
