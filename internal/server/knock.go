package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

type ServerKnockResponse struct {
	Type   uint32
	Source uint32
	Size   uint32
	Data   []byte
}

var (
	ErrSocketCreate = errors.New("failed to create socket")
	ErrSocketWrite  = errors.New("socket write failed")
	ErrSocketRead   = errors.New("socket read failed")
	ErrTimeout      = errors.New("connection timeout")
)

// Knock imitates legacy "MasterServer" behavior, by connecting directly to the server
// and "knocking" to verify it's a real server. This is a blocking operation.
func Knock(ip string, port int, version string) (*ServerKnockResponse, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, ErrTimeout
	}
	defer conn.Close()
	pokePayload := []byte("MasterServer")
	pokeMsg := uint32(1000) // RoRnet: MSG2_HELLO
	if version == "RoRnet_2.40" {
		pokeMsg = 1025
	}
	pokeSource := uint32(5000)
	pokeStreamID := uint32(0)
	pokePayloadLen := uint32(len(pokePayload))

	var bin []byte
	if version != "" {
		bin = make([]byte, 20+len(pokePayload))
		binary.LittleEndian.PutUint32(bin[0:], pokeMsg)
		binary.LittleEndian.PutUint32(bin[4:], pokeSource)
		binary.LittleEndian.PutUint32(bin[8:], pokeStreamID)
		binary.LittleEndian.PutUint32(bin[12:], pokePayloadLen)
		copy(bin[16:], pokePayload)
	} else {
		bin = make([]byte, 16+len(pokePayload))
		binary.LittleEndian.PutUint32(bin[0:], pokeMsg)
		binary.LittleEndian.PutUint32(bin[4:], pokeSource)
		binary.LittleEndian.PutUint32(bin[8:], pokePayloadLen)
		copy(bin[12:], pokePayload)
	}

	_, err = conn.Write(bin)
	if err != nil {
		return nil, ErrSocketWrite
	}

	buffer := make([]byte, 2048)
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return nil, ErrSocketRead
	}

	if n < 12 {
		return nil, errors.New("invalid response size")
	}

	resp := &ServerKnockResponse{
		Type:   binary.LittleEndian.Uint32(buffer[0:4]),
		Source: binary.LittleEndian.Uint32(buffer[4:8]),
		Size:   binary.LittleEndian.Uint32(buffer[8:12]),
	}

	if n > 12 {
		resp.Data = buffer[12:n]
	}

	if resp.Type > 1000 && resp.Type < 2000 {
		return resp, nil
	}

	return nil, errors.New("unexpected response type")
}
