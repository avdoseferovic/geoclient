package game

import (
	"testing"

	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func TestHandleTradeUseSubtractsOnlyTradedAmountFromStack(t *testing.T) {
	c := NewClient()
	c.Inventory = []InventoryItem{{ID: 1, Amount: 100}, {ID: 5, Amount: 3}}
	c.Trade = TradeState{
		State:        TradeStateOpen,
		PlayerItems:  []TradeItem{{ID: 1, Amount: 11}},
		PartnerItems: []TradeItem{{ID: 7, Amount: 1}},
	}

	err := handleTradeUse(c, packetReader(t, &server.TradeUseServerPacket{
		TradeData: []server.TradeItemData{
			{PlayerId: 1, Items: []eonet.Item{{Id: 1, Amount: 11}}},
			{PlayerId: 2, Items: []eonet.Item{{Id: 7, Amount: 1}}},
		},
	}))
	if err != nil {
		t.Fatalf("handleTradeUse returned error: %v", err)
	}

	if amount := inventoryAmount(c.Inventory, 1); amount != 89 {
		t.Fatalf("gold amount = %d, want 89", amount)
	}
	if amount := inventoryAmount(c.Inventory, 7); amount != 1 {
		t.Fatalf("partner item amount = %d, want 1", amount)
	}
	if c.Trade.State != TradeStateNone {
		t.Fatalf("trade state = %d, want %d", c.Trade.State, TradeStateNone)
	}
}
