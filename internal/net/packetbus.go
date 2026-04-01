package net

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/encrypt"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
)

// PacketBus handles reading/writing EO protocol packets with encryption and sequencing.
type PacketBus struct {
	conn      *Conn
	mu        sync.Mutex
	Sequencer *Sequencer

	EncodeMultiple int // for encrypting outgoing packets
	DecodeMultiple int // for decrypting incoming packets
}

// NewPacketBus creates a new PacketBus over a WebSocket connection.
func NewPacketBus(conn *Conn) *PacketBus {
	return &PacketBus{
		conn:      conn,
		Sequencer: NewSequencer(),
	}
}

// SetEncryption sets the encode/decode multiples for packet encryption.
func (pb *PacketBus) SetEncryption(encodeMultiple, decodeMultiple int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.EncodeMultiple = encodeMultiple
	pb.DecodeMultiple = decodeMultiple
}

// Send writes a raw packet with the given action/family and payload.
func (pb *PacketBus) Send(action eonet.PacketAction, family eonet.PacketFamily, payload []byte) error {
	packetSize := 2 + len(payload)
	lengthBytes := data.EncodeNumber(packetSize)

	buf := make([]byte, 0, 2+packetSize)
	buf = append(buf, lengthBytes[0], lengthBytes[1])
	buf = append(buf, byte(action), byte(family))
	buf = append(buf, payload...)

	// Encrypt if needed (skip init packets 0xFF 0xFF)
	if buf[2] != 0xFF || buf[3] != 0xFF {
		pb.mu.Lock()
		mult := pb.EncodeMultiple
		pb.mu.Unlock()
		if mult != 0 {
			encrypted := encryptPacket(buf[2:], mult)
			copy(buf[2:], encrypted)
		}
	}

	slog.Debug("packet send", "action", int(action), "family", int(family), "len", len(buf))
	return pb.conn.WritePacket(buf)
}

// SendPacket serializes and sends an eolib Packet (no sequence byte).
func (pb *PacketBus) SendPacket(pkt eonet.Packet) error {
	writer := data.NewEoWriter()
	if err := pkt.Serialize(writer); err != nil {
		return fmt.Errorf("serialize: %w", err)
	}
	return pb.Send(pkt.Action(), pkt.Family(), writer.Array())
}

// SendSequenced serializes and sends a packet with a sequence byte prepended.
func (pb *PacketBus) SendSequenced(pkt eonet.Packet) error {
	writer := data.NewEoWriter()
	if err := pkt.Serialize(writer); err != nil {
		return fmt.Errorf("serialize: %w", err)
	}

	seqBytes := data.EncodeNumber(pb.Sequencer.NextSequence())

	payload := make([]byte, 0, 1+len(writer.Array()))
	payload = append(payload, seqBytes[0])
	payload = append(payload, writer.Array()...)

	return pb.Send(pkt.Action(), pkt.Family(), payload)
}

// Recv reads the next packet from the connection.
// Returns action, family, and an EoReader over the payload.
func (pb *PacketBus) Recv() (eonet.PacketAction, eonet.PacketFamily, *data.EoReader, error) {
	packetBuf, err := pb.conn.ReadPacket()
	if err != nil {
		return 0, 0, nil, err
	}

	if len(packetBuf) < 2 {
		return 0, 0, nil, fmt.Errorf("packet too short: %d bytes", len(packetBuf))
	}

	// Decrypt if needed (skip init packets 0xFF 0xFF)
	if packetBuf[0] != 0xFF || packetBuf[1] != 0xFF {
		pb.mu.Lock()
		mult := pb.DecodeMultiple
		pb.mu.Unlock()
		if mult != 0 {
			decrypted := decryptPacket(packetBuf, mult)
			copy(packetBuf, decrypted)
		}
	}

	action := eonet.PacketAction(packetBuf[0])
	family := eonet.PacketFamily(packetBuf[1])

	slog.Debug("packet recv", "action", int(action), "family", int(family), "len", len(packetBuf))

	reader := data.NewEoReader(packetBuf[2:])
	return action, family, reader, nil
}

// Close closes the underlying connection.
func (pb *PacketBus) Close() error {
	return pb.conn.Close()
}

// encryptPacket applies client-side EO packet encryption.
// Order: SwapMultiples → FlipMsb → Interleave
func encryptPacket(buf []byte, multiple int) []byte {
	result, _ := encrypt.SwapMultiples(buf, multiple)
	result = encrypt.FlipMsb(result)
	result = encrypt.Interleave(result)
	return result
}

// decryptPacket applies client-side EO packet decryption (reverse of server encrypt).
// Order: Deinterleave → FlipMsb → SwapMultiples
func decryptPacket(buf []byte, multiple int) []byte {
	result := encrypt.Deinterleave(buf)
	result = encrypt.FlipMsb(result)
	result, _ = encrypt.SwapMultiples(result, multiple)
	return result
}
