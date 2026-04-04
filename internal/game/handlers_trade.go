package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterTradeHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Request, handleTradeRequest)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Open, handleTradeOpen)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Reply, handleTradeReply)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Spec, handleTradeSpec)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Agree, handleTradeAgree)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Use, handleTradeUse)
	reg.Register(eonet.PacketFamily_Trade, eonet.PacketAction_Close, handleTradeClose)
}

func handleTradeRequest(c *Client, reader *data.EoReader) error {
	var pkt server.TradeRequestServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		slog.Error("trade request deserialize failed", "err", err)
		return fmt.Errorf("deserialize trade request: %w", err)
	}

	c.mu.Lock()
	c.Trade.State = TradeStatePending
	c.Trade.PartnerID = pkt.PartnerPlayerId
	c.Trade.PartnerName = pkt.PartnerPlayerName
	c.mu.Unlock()

	emitChat(c, ChatChannelSystem, fmt.Sprintf("%s wants to trade with you.", pkt.PartnerPlayerName))
	slog.Info("trade request", "from", pkt.PartnerPlayerName)
	c.Emit(Event{Type: EventTradeRequested})
	return nil
}

func handleTradeOpen(c *Client, reader *data.EoReader) error {
	var pkt server.TradeOpenServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize trade open: %w", err)
	}

	c.mu.Lock()
	c.Trade.State = TradeStateOpen
	c.Trade.PartnerID = pkt.PartnerPlayerId
	c.Trade.PartnerName = pkt.PartnerPlayerName
	c.Trade.PlayerItems = nil
	c.Trade.PartnerItems = nil
	c.Trade.PlayerAgreed = false
	c.Trade.PartnerAgreed = false
	c.mu.Unlock()

	slog.Info("trade opened", "partner", pkt.PartnerPlayerName)
	c.Emit(Event{Type: EventTradeOpened})
	return nil
}

func handleTradeReply(c *Client, reader *data.EoReader) error {
	var pkt server.TradeReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize trade reply: %w", err)
	}

	c.mu.Lock()
	applyTradeData(c, pkt.TradeData)
	c.Trade.PlayerAgreed = false
	c.Trade.PartnerAgreed = false
	c.mu.Unlock()

	c.Emit(Event{Type: EventTradeUpdated})
	return nil
}

func handleTradeSpec(c *Client, reader *data.EoReader) error {
	var pkt server.TradeSpecServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize trade spec: %w", err)
	}

	c.mu.Lock()
	c.Trade.PlayerAgreed = pkt.Agree
	c.mu.Unlock()

	c.Emit(Event{Type: EventTradeUpdated})
	return nil
}

func handleTradeAgree(c *Client, reader *data.EoReader) error {
	var pkt server.TradeAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize trade agree: %w", err)
	}

	c.mu.Lock()
	c.Trade.PartnerAgreed = pkt.Agree
	c.mu.Unlock()

	c.Emit(Event{Type: EventTradeUpdated})
	return nil
}

func handleTradeUse(c *Client, reader *data.EoReader) error {
	var pkt server.TradeUseServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize trade use: %w", err)
	}

	c.mu.Lock()
	applyTradeData(c, pkt.TradeData)
	// Complete the trade: remove our items, add partner's items
	for _, item := range c.Trade.PlayerItems {
		setInventoryAmount(&c.Inventory, item.ID, 0)
	}
	for _, item := range c.Trade.PartnerItems {
		addInventoryAmount(&c.Inventory, item.ID, item.Amount)
	}
	c.Trade = TradeState{}
	c.mu.Unlock()

	emitChat(c, ChatChannelSystem, "Trade completed.")
	slog.Info("trade completed")
	c.Emit(Event{Type: EventTradeClosed})
	return nil
}

func handleTradeClose(c *Client, reader *data.EoReader) error {
	c.mu.Lock()
	c.Trade = TradeState{}
	c.mu.Unlock()

	emitChat(c, ChatChannelSystem, "Trade cancelled.")
	slog.Info("trade closed")
	c.Emit(Event{Type: EventTradeClosed})
	return nil
}

func applyTradeData(c *Client, tradeData []server.TradeItemData) {
	if len(tradeData) >= 1 {
		c.Trade.PlayerItems = tradeItemsFromNet(tradeData[0].Items)
	}
	if len(tradeData) >= 2 {
		c.Trade.PartnerItems = tradeItemsFromNet(tradeData[1].Items)
	}
}

func tradeItemsFromNet(items []eonet.Item) []TradeItem {
	result := make([]TradeItem, len(items))
	for i, item := range items {
		result[i] = TradeItem{ID: item.Id, Amount: item.Amount}
	}
	return result
}
