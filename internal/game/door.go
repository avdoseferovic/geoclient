package game

// Door represents a door on the map that can be opened.
type Door struct {
	X, Y      int
	Key       int
	Open      bool
	OpenTicks int
}

// DoorOpenTicks is how long a door stays open before auto-closing.
const DoorOpenTicks = 25

// LoadDoors extracts door positions from the map renderer's warp data.
// Called by the UI layer after loading a map.
func (c *Client) LoadDoors(doors []Door) {
	c.mu.Lock()
	c.Doors = doors
	c.mu.Unlock()
}

// GetDoor returns a pointer to the door at the given coordinates, or nil.
func (c *Client) GetDoor(x, y int) *Door {
	for i := range c.Doors {
		if c.Doors[i].X == x && c.Doors[i].Y == y {
			return &c.Doors[i]
		}
	}
	return nil
}

// IsDoorOpen returns true if there is an open door at the given tile.
func (c *Client) IsDoorOpen(x, y int) bool {
	for _, d := range c.Doors {
		if d.X == x && d.Y == y && d.Open {
			return true
		}
	}
	return false
}

// TickDoors decrements open door timers and auto-closes expired doors.
func (c *Client) TickDoors() {
	for i := range c.Doors {
		if !c.Doors[i].Open {
			continue
		}
		c.Doors[i].OpenTicks--
		if c.Doors[i].OpenTicks <= 0 {
			c.Doors[i].Open = false
			c.Doors[i].OpenTicks = 0
		}
	}
}
