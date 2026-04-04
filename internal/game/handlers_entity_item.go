package game

import (
	"fmt"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func handleItemAdd(c *Client, reader *data.EoReader) error {
	var pkt server.ItemAddServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item add: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems[i].ID = pkt.ItemId
		c.NearbyItems[i].GraphicID = pkt.ItemId
		c.NearbyItems[i].Amount = pkt.ItemAmount
		c.NearbyItems[i].X = pkt.Coords.X
		c.NearbyItems[i].Y = pkt.Coords.Y
		return nil
	}

	c.NearbyItems = append(c.NearbyItems, NearbyItem{UID: pkt.ItemIndex, ID: pkt.ItemId, GraphicID: pkt.ItemId, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.ItemAmount})
	return nil
}

func handleItemRemove(c *Client, reader *data.EoReader) error {
	var pkt server.ItemRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item remove: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems = append(c.NearbyItems[:i], c.NearbyItems[i+1:]...)
		break
	}
	return nil
}

func handleItemDrop(c *Client, reader *data.EoReader) error {
	var pkt server.ItemDropServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item drop: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.DroppedItem.Id, pkt.RemainingAmount)
	syncWeight(c, pkt.Weight)
	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.ItemIndex {
			continue
		}
		c.NearbyItems[i] = NearbyItem{UID: pkt.ItemIndex, ID: pkt.DroppedItem.Id, GraphicID: pkt.DroppedItem.Id, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.DroppedItem.Amount}
		return nil
	}
	c.NearbyItems = append(c.NearbyItems, NearbyItem{UID: pkt.ItemIndex, ID: pkt.DroppedItem.Id, GraphicID: pkt.DroppedItem.Id, X: pkt.Coords.X, Y: pkt.Coords.Y, Amount: pkt.DroppedItem.Amount})
	return nil
}

func handleItemJunk(c *Client, reader *data.EoReader) error {
	var pkt server.ItemJunkServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item junk: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.JunkedItem.Id, pkt.RemainingAmount)
	syncWeight(c, pkt.Weight)
	return nil
}

func handleItemGet(c *Client, reader *data.EoReader) error {
	var pkt server.ItemGetServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item get: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	syncWeight(c, pkt.Weight)
	addInventoryAmount(&c.Inventory, pkt.TakenItem.Id, pkt.TakenItem.Amount)
	for i := range c.NearbyItems {
		if c.NearbyItems[i].UID != pkt.TakenItemIndex {
			continue
		}
		c.NearbyItems = append(c.NearbyItems[:i], c.NearbyItems[i+1:]...)
		break
	}
	return nil
}

func handleItemObtain(c *Client, reader *data.EoReader) error {
	var pkt server.ItemObtainServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item obtain: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	addInventoryAmount(&c.Inventory, pkt.Item.Id, pkt.Item.Amount)
	c.Character.Weight.Current = pkt.CurrentWeight
	return nil
}

func handleItemKick(c *Client, reader *data.EoReader) error {
	var pkt server.ItemKickServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize item kick: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.Item.Id, pkt.Item.Amount)
	c.Character.Weight.Current = pkt.CurrentWeight
	return nil
}
