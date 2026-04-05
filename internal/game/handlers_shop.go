package game

import (
	"fmt"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterShopHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Shop, eonet.PacketAction_Open, handleShopOpen)
	reg.Register(eonet.PacketFamily_Shop, eonet.PacketAction_Buy, handleShopBuy)
	reg.Register(eonet.PacketFamily_Shop, eonet.PacketAction_Sell, handleShopSell)
	reg.Register(eonet.PacketFamily_Shop, eonet.PacketAction_Create, handleShopCreate)
}

func handleShopOpen(c *Client, reader *data.EoReader) error {
	var pkt server.ShopOpenServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize shop open: %w", err)
	}

	c.mu.Lock()
	c.SessionID = pkt.SessionId
	c.Shop = ShopState{
		Open:       true,
		Name:       pkt.ShopName,
		TradeItems: shopTradeItemsFromNet(pkt.TradeItems),
		CraftItems: shopCraftItemsFromNet(pkt.CraftItems),
	}
	c.mu.Unlock()

	c.Emit(Event{Type: EventShopOpened})
	return nil
}

func handleShopBuy(c *Client, reader *data.EoReader) error {
	var pkt server.ShopBuyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize shop buy: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, 1, pkt.GoldAmount)
	addInventoryAmount(&c.Inventory, pkt.BoughtItem.Id, pkt.BoughtItem.Amount)
	syncWeight(c, pkt.Weight)
	return nil
}

func handleShopSell(c *Client, reader *data.EoReader) error {
	var pkt server.ShopSellServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize shop sell: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, 1, pkt.GoldAmount)
	setInventoryAmount(&c.Inventory, pkt.SoldItem.Id, pkt.SoldItem.Amount)
	syncWeight(c, pkt.Weight)
	return nil
}

func handleShopCreate(c *Client, reader *data.EoReader) error {
	var pkt server.ShopCreateServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize shop create: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ingredient := range pkt.Ingredients {
		setInventoryAmount(&c.Inventory, ingredient.Id, ingredient.Amount)
	}
	addInventoryAmount(&c.Inventory, pkt.CraftItemId, 1)
	syncWeight(c, pkt.Weight)
	return nil
}

func shopTradeItemsFromNet(items []server.ShopTradeItem) []ShopTradeItem {
	result := make([]ShopTradeItem, len(items))
	for i, item := range items {
		result[i] = ShopTradeItem{
			ItemID:       item.ItemId,
			BuyPrice:     item.BuyPrice,
			SellPrice:    item.SellPrice,
			MaxBuyAmount: item.MaxBuyAmount,
		}
	}
	return result
}

func shopCraftItemsFromNet(items []server.ShopCraftItem) []ShopCraftItem {
	result := make([]ShopCraftItem, len(items))
	for i, item := range items {
		result[i] = ShopCraftItem{
			ItemID:      item.ItemId,
			Ingredients: append([]eonet.CharItem(nil), item.Ingredients...),
		}
	}
	return result
}
