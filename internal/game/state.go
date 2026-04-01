package game

import (
	"log/slog"
	"sync"

	"github.com/ethanmoffat/eolib-go/v3/protocol"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"

	"github.com/avdo/eoweb/internal/net"
)

type GameState int

const (
	StateInitial GameState = iota
	StateConnected
	StateLogin
	StateLoggedIn
	StateInGame
)

func (s GameState) String() string {
	switch s {
	case StateInitial:
		return "Initial"
	case StateConnected:
		return "Connected"
	case StateLogin:
		return "Login"
	case StateLoggedIn:
		return "LoggedIn"
	case StateInGame:
		return "InGame"
	default:
		return "Unknown"
	}
}

// Character holds local player info after entering the game.
type Character struct {
	ID        int
	Name      string
	MapID     int
	X, Y      int
	Direction protocol.Direction
	Level     int
	HP, MaxHP int
	TP, MaxTP int
}

// NearbyCharacter is a renderable character on the map.
type NearbyCharacter struct {
	PlayerID  int
	Name      string
	X, Y      int
	Direction int
	Gender    int
	Skin      int
	HairStyle int
	HairColor int
	Armor     int
	Boots     int
	Hat       int
	Weapon    int
	Shield    int
	SitState  int
	Level     int

	// Walk animation state
	Walking  bool
	WalkTick int // 0 to WalkDuration, increments each game tick
}

// WalkDuration is how many game ticks a walk animation lasts.
const WalkDuration = 16

// TickWalk advances walk animation by one tick. Returns true when walk completes.
func (c *NearbyCharacter) TickWalk() bool {
	if !c.Walking {
		return false
	}
	c.WalkTick++
	if c.WalkTick >= WalkDuration {
		c.Walking = false
		c.WalkTick = 0
		return true
	}
	return false
}

// WalkProgress returns 0.0 (just started) to 1.0 (arrived at destination).
func (c *NearbyCharacter) WalkProgress() float64 {
	if !c.Walking {
		return 0
	}
	return float64(c.WalkTick) / float64(WalkDuration)
}

// WalkFrame returns the current walk animation frame (0-3).
func (c *NearbyCharacter) WalkFrame() int {
	ticksPerFrame := WalkDuration / 4
	frame := c.WalkTick / ticksPerFrame
	if frame > 3 {
		frame = 3
	}
	return frame
}

// NearbyNPC is a renderable NPC on the map.
type NearbyNPC struct {
	Index     int
	ID        int // NPC type ID (from enf)
	X, Y      int // current position (destination during walk, like the original client)
	WalkFromX int // origin of current walk (only valid when Walking=true)
	WalkFromY int
	Direction int
	Walking   bool
	WalkTick  int
	IdleTick  int
}

const NpcIdleInterval = 30 // ~500ms at 60 TPS
const NpcWalkDuration = 16

// StartWalk begins a walk from the current position to (destX, destY).
// Matches the original client: X,Y is set to destination immediately.
func (n *NearbyNPC) StartWalk(destX, destY, dir int) {
	n.Direction = dir

	// No actual movement
	if n.X == destX && n.Y == destY {
		return
	}

	// Save origin for walk animation
	n.WalkFromX = n.X
	n.WalkFromY = n.Y
	// Update position to destination immediately (like original client)
	n.X = destX
	n.Y = destY
	n.Walking = true
	n.WalkTick = 0
}

// Tick advances NPC animation by one game tick.
func (n *NearbyNPC) Tick() {
	n.IdleTick++
	if !n.Walking {
		return
	}
	n.WalkTick++
	if n.WalkTick >= NpcWalkDuration {
		n.Walking = false
		n.WalkTick = 0
	}
}

func (n *NearbyNPC) WalkProgress() float64 {
	if !n.Walking {
		return 0
	}
	return float64(n.WalkTick) / float64(NpcWalkDuration)
}

func (n *NearbyNPC) IdleFrame() int {
	if (n.IdleTick/NpcIdleInterval)%2 == 0 {
		return 0
	}
	return 1
}

// NearbyItem is a renderable dropped item on the map.
type NearbyItem struct {
	UID       int
	ID        int // Item type ID (from eif)
	GraphicID int // Sprite graphic ID
	X, Y      int
	Amount    int
}

// Client holds all game state.
type Client struct {
	mu    sync.RWMutex
	Bus   *net.PacketBus
	State GameState

	// Connection
	PlayerID  int
	SessionID int
	Challenge int
	Version   eonet.Version

	// Character
	Character  Character
	Characters []server.CharacterSelectionListEntry

	// Nearby entities
	NearbyChars []NearbyCharacter
	NearbyNpcs  []NearbyNPC
	NearbyItems []NearbyItem

	// UI state
	Username string
	Password string

	// Events channel for UI updates
	Events chan Event
}

type EventType int

const (
	EventStateChanged EventType = iota
	EventError
	EventChat
	EventCharacterList
	EventEnterGame
	EventWarp
)

type Event struct {
	Type    EventType
	Message string
	Data    any
}

func NewClient() *Client {
	return &Client{
		State:   StateInitial,
		Version: eonet.Version{Major: 0, Minor: 0, Patch: 28},
		Events:  make(chan Event, 64),
	}
}

func (c *Client) SetState(state GameState) {
	c.mu.Lock()
	c.State = state
	c.mu.Unlock()
	slog.Info("state changed", "state", state)
	select {
	case c.Events <- Event{Type: EventStateChanged, Message: state.String()}:
	default:
	}
}

func (c *Client) GetState() GameState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

func (c *Client) Emit(evt Event) {
	select {
	case c.Events <- evt:
	default:
		slog.Warn("event channel full, dropping event", "type", evt.Type)
	}
}

func (c *Client) Lock()   { c.mu.Lock() }
func (c *Client) Unlock() { c.mu.Unlock() }

func (c *Client) Disconnect() {
	if c.Bus != nil {
		c.Bus.Close()
		c.Bus = nil
	}
	c.SetState(StateInitial)
}
