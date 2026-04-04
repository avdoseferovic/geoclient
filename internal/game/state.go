package game

import (
	"log/slog"
	"slices"
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
	ID          int
	Name        string
	Title       string
	GuildName   string
	GuildRank   string
	GuildTag    string
	Home        string
	Partner     string
	MapID       int
	X, Y        int
	Direction   protocol.Direction
	Gender      protocol.Gender
	Admin       protocol.AdminLevel
	ClassID     int
	Level       int
	Experience  int
	Usage       int
	StatPoints  int
	SkillPoints int
	Karma       int
	HP, MaxHP   int
	TP, MaxTP   int
	MaxSP       int
	Weight      eonet.Weight
	BaseStats   CharacterBaseStats
	CombatStats CharacterCombatStats
}

type CharacterBaseStats struct {
	Str int
	Int int
	Wis int
	Agi int
	Con int
	Cha int
}

type CharacterCombatStats struct {
	MinDamage int
	MaxDamage int
	Accuracy  int
	Evade     int
	Armor     int
}

type InventoryItem struct {
	ID     int
	Amount int
}

type AccountCreateProfile struct {
	FullName string
	Location string
	Email    string
}

type CharacterCreateProfile struct {
	Name      string
	Gender    protocol.Gender
	HairStyle int
	HairColor int
	Skin      int
}

type CombatIndicatorKind int

const (
	CombatIndicatorDamage CombatIndicatorKind = iota
	CombatIndicatorMiss
	CombatIndicatorHeal
)

const (
	CombatIndicatorDuration = 45
	AttackAnimationDuration = 18
	HitAnimationDuration    = 10
	NpcDeathAnimationTicks  = 24
)

type CombatIndicator struct {
	Text     string
	Kind     CombatIndicatorKind
	Ticks    int
	MaxTicks int
}

func (i CombatIndicator) Progress() float64 {
	if i.MaxTicks <= 0 {
		return 1
	}

	progress := 1.0 - float64(i.Ticks)/float64(i.MaxTicks)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

type CombatState struct {
	Indicators []CombatIndicator
	AttackTick int
	HitTick    int
}

func (s *CombatState) Reset() {
	s.Indicators = nil
	s.AttackTick = 0
	s.HitTick = 0
}

func (s *CombatState) Tick() {
	if s.AttackTick > 0 {
		s.AttackTick--
	}
	if s.HitTick > 0 {
		s.HitTick--
	}

	if len(s.Indicators) == 0 {
		return
	}

	next := s.Indicators[:0]
	for _, indicator := range s.Indicators {
		indicator.Ticks--
		if indicator.Ticks > 0 {
			next = append(next, indicator)
		}
	}
	s.Indicators = next
}

func (s *CombatState) StartAttack() {
	s.AttackTick = AttackAnimationDuration
}

func (s *CombatState) StartHit() {
	s.HitTick = HitAnimationDuration
}

func (s *CombatState) AddIndicator(kind CombatIndicatorKind, text string) {
	if text == "" {
		return
	}

	s.Indicators = append(s.Indicators, CombatIndicator{
		Text:     text,
		Kind:     kind,
		Ticks:    CombatIndicatorDuration,
		MaxTicks: CombatIndicatorDuration,
	})
	if len(s.Indicators) > 4 {
		s.Indicators = s.Indicators[len(s.Indicators)-4:]
	}
	if kind == CombatIndicatorDamage {
		s.StartHit()
	}
}

// NearbyCharacter is a renderable character on the map.
type NearbyCharacter struct {
	PlayerID  int
	Name      string
	GuildTag  string
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
	Combat   CombatState
}

// WalkDuration controls how long tile-to-tile travel takes in the local client.
const WalkDuration = 24

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
	Dead      bool
	DeathTick int
	Hidden    bool
	Combat    CombatState
}

// NPC idle and walk timings are tuned separately from character travel speed.
const (
	NpcIdleInterval = 6
	NpcWalkDuration = 24
)

// StartWalk begins a walk from the current position to (destX, destY).
// Matches the original client: X,Y is set to destination immediately.
func (n *NearbyNPC) StartWalk(destX, destY, dir int) {
	n.Dead = false
	n.DeathTick = 0
	n.Hidden = false
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
	n.Combat.Tick()
	if n.Dead {
		n.DeathTick++
		return
	}

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

func (n *NearbyNPC) StartDeath() {
	n.Dead = true
	n.DeathTick = 0
	n.Hidden = false
	n.Walking = false
	n.WalkTick = 0
	n.Combat.StartHit()
}

func (n *NearbyNPC) DeathProgress() float64 {
	if !n.Dead {
		return 0
	}
	progress := float64(n.DeathTick) / float64(NpcDeathAnimationTicks)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func (n *NearbyNPC) DeathComplete() bool {
	return n.Dead && n.DeathTick >= NpcDeathAnimationTicks
}

func (n *NearbyNPC) WalkProgress() float64 {
	if !n.Walking {
		return 0
	}
	return float64(n.WalkTick) / float64(NpcWalkDuration)
}

func (n *NearbyNPC) WalkFrame() int {
	ticksPerFrame := NpcWalkDuration / 4
	frame := n.WalkTick / ticksPerFrame
	if frame > 3 {
		frame = 3
	}
	return frame
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
	Inventory   []InventoryItem
	Equipment   server.EquipmentPaperdoll

	// Lookup callback set by the UI layer.
	ItemTypeFunc func(int) int

	// UI state
	Username string
	Password string

	PendingAccountCreate   *AccountCreateProfile
	PendingCharacterCreate *CharacterCreateProfile

	// Events channel for UI updates
	Events chan Event
}

type EventType int

const (
	EventStateChanged EventType = iota
	EventError
	EventChat
	EventCharacterList
	EventAccountCreated
	EventEnterGame
	EventWarp
)

type Event struct {
	Type    EventType
	Message string
	Data    any
}

type ChatChannel int

const (
	ChatChannelMap ChatChannel = iota
	ChatChannelGroup
	ChatChannelGlobal
	ChatChannelSystem
)

func (c ChatChannel) Label() string {
	switch c {
	case ChatChannelMap:
		return "Map"
	case ChatChannelGroup:
		return "Group"
	case ChatChannelGlobal:
		return "Global"
	case ChatChannelSystem:
		return "System"
	default:
		return "Unknown"
	}
}

type ChatMessage struct {
	Channel ChatChannel
	Text    string
}

type UISnapshot struct {
	PlayerID    int
	Character   Character
	Inventory   []InventoryItem
	Equipment   server.EquipmentPaperdoll
	NearbyChars []NearbyCharacter
	NearbyNpcs  []NearbyNPC
	NearbyItems []NearbyItem
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

func (c *Client) EmitCritical(evt Event) {
	c.Events <- evt
}

func (c *Client) Lock()   { c.mu.Lock() }
func (c *Client) Unlock() { c.mu.Unlock() }

func (c *Client) SetBus(bus *net.PacketBus) {
	c.mu.Lock()
	c.Bus = bus
	c.mu.Unlock()
}

func (c *Client) GetBus() *net.PacketBus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Bus
}

func (c *Client) UISnapshot() UISnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := UISnapshot{
		PlayerID:    c.PlayerID,
		Character:   c.Character,
		Inventory:   slices.Clone(c.Inventory),
		NearbyChars: slices.Clone(c.NearbyChars),
		NearbyNpcs:  slices.Clone(c.NearbyNpcs),
		NearbyItems: slices.Clone(c.NearbyItems),
	}
	snapshot.Equipment = c.Equipment
	snapshot.Equipment.Ring = slices.Clone(c.Equipment.Ring)
	snapshot.Equipment.Armlet = slices.Clone(c.Equipment.Armlet)
	snapshot.Equipment.Bracer = slices.Clone(c.Equipment.Bracer)
	return snapshot
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	bus := c.Bus
	c.Bus = nil
	c.mu.Unlock()

	if bus != nil {
		if err := bus.Close(); err != nil {
			slog.Debug("close packet bus", "err", err)
		}
	}
	c.SetState(StateInitial)
}
