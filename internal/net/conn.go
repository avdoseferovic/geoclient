package net

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/gorilla/websocket"
)

const writeDeadline = 10 * time.Second

// Conn wraps a WebSocket connection for EO protocol communication.
type Conn struct {
	ws   *websocket.Conn
	wsMu sync.Mutex
}

// Dial connects to an EO server via WebSocket.
func Dial(addr string) (*Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	return &Conn{ws: ws}, nil
}

// ReadPacket reads one EO packet from the WebSocket.
// Web clients include a 2-byte EO length prefix in the message which is stripped.
// Returns action + family + payload bytes.
func (c *Conn) ReadPacket() ([]byte, error) {
	_ = c.ws.SetReadDeadline(time.Time{})
	msgType, msg, err := c.ws.ReadMessage()
	if err != nil {
		return nil, err
	}
	if msgType != websocket.BinaryMessage {
		return nil, fmt.Errorf("expected binary message, got type %d", msgType)
	}
	// Strip 2-byte EO length prefix
	if len(msg) >= 4 {
		msg = msg[2:]
	}
	if len(msg) < 2 {
		return nil, fmt.Errorf("message too short: %d bytes", len(msg))
	}
	return msg, nil
}

// WritePacket writes a fully assembled packet with the 2-byte EO length prefix.
func (c *Conn) WritePacket(buf []byte) error {
	if len(buf) < 4 {
		return fmt.Errorf("packet too short: %d bytes", len(buf))
	}

	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(writeDeadline))
	return c.ws.WriteMessage(websocket.BinaryMessage, buf)
}

// Close closes the underlying WebSocket connection.
func (c *Conn) Close() error {
	return c.ws.Close()
}

// encodeLength encodes a packet length using EO number encoding.
func encodeLength(length int) []byte {
	return data.EncodeNumber(length)
}
