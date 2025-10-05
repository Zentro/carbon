package server

import (
	"bytes"
	"carbon/domain"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"
)

const (
	// RoRnet protocol constants
	MSG2_HELLO      = 1025
	MSG2_MASTERINFO = 1034
	MAGIC_SOURCE    = 5000
)

// RoRnetPacket represents a packet in the RoRnet protocol.
type RoRnetPacket struct {
	Type   uint32
	Source uint32
	Stream uint32
	Size   uint32
	Data   []byte
}

// PokePacket represents the structure of a "poke" packet.
type PokePacket struct {
	Msg      uint32
	Source   uint32
	StreamID uint32
	Length   uint32
	Payload  [12]byte
}

var (
	ErrSocketCreate = errors.New("failed to create socket")
	ErrSocketWrite  = errors.New("socket write failed")
	ErrSocketRead   = errors.New("socket read failed")
	ErrTimeout      = errors.New("connection timeout")
	ErrUnexpected   = errors.New("unexpected response from server")
)

func Knock(server domain.Server, request_id string) error {
	slog.Info("knocking server",
		"request_id", request_id,
		"server_id", server.ID(),
		"host", server.Host,
		"port", server.Port)

	// Create a TCP socket and connect to the server.
	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		slog.Error("failed to create socket",
			"request_id", request_id,
			"server_id", server.ID(),
			"error", err,
			"host", server.Host,
			"port", server.Port)
		return ErrSocketCreate
	}
	defer conn.Close()

	// Prepare the poke payload (extended header format only)
	pokePayload := "MasterServer"
	pokeMsg := uint32(MSG2_HELLO)
	pokeSource := uint32(MAGIC_SOURCE)
	pokePayloadLen := uint32(len(pokePayload))
	pokeStreamID := uint32(0)

	// Build the binary packet (extended header format)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, pokeMsg)
	binary.Write(buf, binary.LittleEndian, pokeSource)
	binary.Write(buf, binary.LittleEndian, pokeStreamID)
	binary.Write(buf, binary.LittleEndian, pokePayloadLen)

	// Pad payload to 12 bytes as per pack format "a12"
	paddedPayload := make([]byte, 12)
	copy(paddedPayload, pokePayload)
	buf.Write(paddedPayload)

	// Write to socket
	w, err := conn.Write(buf.Bytes())
	if err != nil {
		slog.Error("failed to write to socket",
			"request_id", request_id,
			"server_id", server.ID(),
			"error", err,
			"host", server.Host,
			"port", server.Port)
		return ErrSocketWrite
	}

	slog.Debug("wrote to socket",
		"bytes", w,
		"host", server.Host,
		"port", server.Port)

	r := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(r)
	if err != nil {
		slog.Error("failed to read from socket",
			"request_id", request_id,
			"server_id", server.ID(),
			"error", err,
			"host", server.Host,
			"port", server.Port)
		return ErrSocketRead
	}

	if n == 0 {
		slog.Error("no data received from server",
			"request_id", request_id,
			"server_id", server.ID(),
			"host", server.Host,
			"port", server.Port)
		return ErrTimeout
	}

	packet := RoRnetPacket{}
	reader := bytes.NewReader(r[:n])
	// Extended format: type, source, stream, size, data
	binary.Read(reader, binary.LittleEndian, &packet.Type)
	binary.Read(reader, binary.LittleEndian, &packet.Source)
	binary.Read(reader, binary.LittleEndian, &packet.Stream)
	binary.Read(reader, binary.LittleEndian, &packet.Size)

	if packet.Size > 0 && int(packet.Size) <= len(r)-16 {
		packet.Data = make([]byte, packet.Size)
		reader.Read(packet.Data)
	}

	if packet.Type != MSG2_MASTERINFO {
		slog.Error("unexpected response from server",
			"request_id", request_id,
			"server_id", server.ID(),
			"expected_type", MSG2_MASTERINFO,
			"received_type", packet.Type,
			"host", server.Host,
			"port", server.Port)
		return ErrUnexpected
	}

	return nil
}
