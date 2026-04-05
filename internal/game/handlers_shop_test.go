package game

import (
	"testing"

	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func TestHandleShopOpenUpdatesShopStateAndEmitsEvent(t *testing.T) {
	c := NewClient()

	err := handleShopOpen(c, packetReader(t, &server.ShopOpenServerPacket{
		SessionId: 42,
		ShopName:  "Vendor Mira",
		TradeItems: []server.ShopTradeItem{
			{ItemId: 5, BuyPrice: 12, SellPrice: 6, MaxBuyAmount: 20},
		},
		CraftItems: []server.ShopCraftItem{
			{ItemId: 9, Ingredients: []eonet.CharItem{{Id: 1, Amount: 2}, {}, {}, {}}},
		},
	}))
	if err != nil {
		t.Fatalf("handleShopOpen returned error: %v", err)
	}

	if c.SessionID != 42 {
		t.Fatalf("session id = %d, want 42", c.SessionID)
	}
	if c.Shop.Name != "Vendor Mira" {
		t.Fatalf("shop name = %q, want %q", c.Shop.Name, "Vendor Mira")
	}
	if len(c.Shop.TradeItems) != 1 || c.Shop.TradeItems[0].ItemID != 5 {
		t.Fatalf("trade items = %#v", c.Shop.TradeItems)
	}
	if len(c.Shop.CraftItems) != 1 || c.Shop.CraftItems[0].ItemID != 9 {
		t.Fatalf("craft items = %#v", c.Shop.CraftItems)
	}

	evt := <-c.Events
	if evt.Type != EventShopOpened {
		t.Fatalf("event type = %v, want %v", evt.Type, EventShopOpened)
	}
}

func TestHandleShopBuyUpdatesGoldAndInventory(t *testing.T) {
	c := NewClient()
	c.Inventory = []InventoryItem{{ID: 1, Amount: 100}, {ID: 5, Amount: 2}}

	err := handleShopBuy(c, packetReader(t, &server.ShopBuyServerPacket{
		GoldAmount: 64,
		BoughtItem: eonet.Item{Id: 5, Amount: 3},
		Weight:     eonet.Weight{Current: 11, Max: 50},
	}))
	if err != nil {
		t.Fatalf("handleShopBuy returned error: %v", err)
	}

	if amount := inventoryAmount(c.Inventory, 1); amount != 64 {
		t.Fatalf("gold amount = %d, want 64", amount)
	}
	if amount := inventoryAmount(c.Inventory, 5); amount != 5 {
		t.Fatalf("item amount = %d, want 5", amount)
	}
	if c.Character.Weight.Current != 11 {
		t.Fatalf("weight current = %d, want 11", c.Character.Weight.Current)
	}
}

func TestHandleShopSellUpdatesGoldAndInventory(t *testing.T) {
	c := NewClient()
	c.Inventory = []InventoryItem{{ID: 1, Amount: 100}, {ID: 7, Amount: 4}}

	err := handleShopSell(c, packetReader(t, &server.ShopSellServerPacket{
		SoldItem:   server.ShopSoldItem{Id: 7, Amount: 1},
		GoldAmount: 118,
		Weight:     eonet.Weight{Current: 7, Max: 50},
	}))
	if err != nil {
		t.Fatalf("handleShopSell returned error: %v", err)
	}

	if amount := inventoryAmount(c.Inventory, 1); amount != 118 {
		t.Fatalf("gold amount = %d, want 118", amount)
	}
	if amount := inventoryAmount(c.Inventory, 7); amount != 1 {
		t.Fatalf("item amount = %d, want 1", amount)
	}
	if c.Character.Weight.Current != 7 {
		t.Fatalf("weight current = %d, want 7", c.Character.Weight.Current)
	}
}

func TestHandleShopCreateConsumesIngredientsAndAddsItem(t *testing.T) {
	c := NewClient()
	c.Inventory = []InventoryItem{{ID: 2, Amount: 5}, {ID: 3, Amount: 2}}

	err := handleShopCreate(c, packetReader(t, &server.ShopCreateServerPacket{
		CraftItemId: 9,
		Weight:      eonet.Weight{Current: 9, Max: 50},
		Ingredients: []eonet.Item{
			{Id: 2, Amount: 3},
			{Id: 3, Amount: 0},
			{},
			{},
		},
	}))
	if err != nil {
		t.Fatalf("handleShopCreate returned error: %v", err)
	}

	if amount := inventoryAmount(c.Inventory, 2); amount != 3 {
		t.Fatalf("ingredient amount = %d, want 3", amount)
	}
	if amount := inventoryAmount(c.Inventory, 3); amount != 0 {
		t.Fatalf("ingredient amount = %d, want 0", amount)
	}
	if amount := inventoryAmount(c.Inventory, 9); amount != 1 {
		t.Fatalf("crafted item amount = %d, want 1", amount)
	}
	if c.Character.Weight.Current != 9 {
		t.Fatalf("weight current = %d, want 9", c.Character.Weight.Current)
	}
}

func inventoryAmount(items []InventoryItem, itemID int) int {
	for _, item := range items {
		if item.ID == itemID {
			return item.Amount
		}
	}
	return 0
}
