package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

type ServerConnectResponse struct {
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

// Connect establishes a connection to a server at the specified IP and port.
// It sends a "poke" message to the server and waits for a response.
//
// Parameters:
//   - ip: The IP address of the server to connect to.
//   - port: The port number of the server to connect to.
//   - version: The version of the protocol to use for the connection.
// Returns:
//   - A pointer to a ServerConnectResponse containing the response from the server.
//   - An error if the connection fails or if the response is invalid.
func Connect(ip string, port int, version string) (*ServerConnectResponse, error) {
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

	resp := &ServerConnectResponse{
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
