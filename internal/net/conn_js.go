//go:build js && wasm

package net

import (
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/ethanmoffat/eolib-go/v3/data"
)

const writeDeadline = 10 * time.Second

// Conn wraps a browser WebSocket for EO protocol communication.
type Conn struct {
	ws        js.Value
	callbacks []js.Func

	wsMu     sync.Mutex
	incoming chan []byte
	errs     chan error
	closed   chan struct{}
}

// Dial connects to an EO server via the browser WebSocket API.
func Dial(addr string) (*Conn, error) {
	ctor := js.Global().Get("WebSocket")
	if ctor.IsUndefined() || ctor.IsNull() {
		return nil, fmt.Errorf("browser websocket API unavailable")
	}

	ws := ctor.New(addr)
	ws.Set("binaryType", "arraybuffer")

	conn := &Conn{
		ws:       ws,
		incoming: make(chan []byte, 32),
		errs:     make(chan error, 1),
		closed:   make(chan struct{}),
	}

	opened := make(chan struct{}, 1)
	closedBeforeOpen := make(chan struct{}, 1)

	onOpen := js.FuncOf(func(this js.Value, args []js.Value) any {
		select {
		case opened <- struct{}{}:
		default:
		}
		return nil
	})
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		dataValue := args[0].Get("data")
		array := js.Global().Get("Uint8Array").New(dataValue)
		buf := make([]byte, array.Get("length").Int())
		js.CopyBytesToGo(buf, array)
		select {
		case conn.incoming <- buf:
		case <-conn.closed:
		}
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		conn.pushErr(fmt.Errorf("websocket error"))
		return nil
	})
	onClose := js.FuncOf(func(this js.Value, args []js.Value) any {
		select {
		case closedBeforeOpen <- struct{}{}:
		default:
		}
		conn.pushErr(fmt.Errorf("websocket closed"))
		conn.markClosed()
		return nil
	})

	conn.callbacks = []js.Func{onOpen, onMessage, onError, onClose}
	ws.Set("onopen", onOpen)
	ws.Set("onmessage", onMessage)
	ws.Set("onerror", onError)
	ws.Set("onclose", onClose)

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	select {
	case <-opened:
		return conn, nil
	case <-closedBeforeOpen:
		conn.releaseCallbacks()
		return nil, fmt.Errorf("websocket closed before open")
	case err := <-conn.errs:
		conn.releaseCallbacks()
		return nil, err
	case <-timer.C:
		conn.releaseCallbacks()
		return nil, fmt.Errorf("websocket open timeout")
	}
}

func (c *Conn) ReadPacket() ([]byte, error) {
	select {
	case msg := <-c.incoming:
		if len(msg) >= 4 {
			msg = msg[2:]
		}
		if len(msg) < 2 {
			return nil, fmt.Errorf("message too short: %d bytes", len(msg))
		}
		return msg, nil
	case err := <-c.errs:
		return nil, err
	}
}

func (c *Conn) WritePacket(buf []byte) error {
	if len(buf) < 4 {
		return fmt.Errorf("packet too short: %d bytes", len(buf))
	}

	c.wsMu.Lock()
	defer c.wsMu.Unlock()

	if c.ws.Get("readyState").Int() != 1 {
		return fmt.Errorf("websocket not open")
	}

	array := js.Global().Get("Uint8Array").New(len(buf))
	js.CopyBytesToJS(array, buf)
	c.ws.Call("send", array)
	return nil
}

func (c *Conn) Close() error {
	c.markClosed()
	if c.ws.Truthy() {
		c.ws.Call("close")
	}
	c.releaseCallbacks()
	return nil
}

func (c *Conn) pushErr(err error) {
	select {
	case c.errs <- err:
	default:
	}
}

func (c *Conn) markClosed() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

func (c *Conn) releaseCallbacks() {
	for _, callback := range c.callbacks {
		callback.Release()
	}
	c.callbacks = nil
}

// encodeLength encodes a packet length using EO number encoding.
func encodeLength(length int) []byte {
	return data.EncodeNumber(length)
}
