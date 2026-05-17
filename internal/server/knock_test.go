package server

import (
	"bytes"
	"carbon/domain"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// startKnockListener spins up a TCP listener that consumes one Knock request
// and responds according to `replyType` (use MSG2_MASTERINFO for success).
// If replyType is 0, the connection is closed without writing anything.
// Returns host, port, and a channel that receives the request bytes the
// listener read before responding.
func startKnockListener(t *testing.T, replyType uint32) (host string, port int, gotReq <-chan []byte, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	addr := ln.Addr().(*net.TCPAddr)
	reqCh := make(chan []byte, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read the 16-byte header + 12-byte padded payload = 28 bytes.
		buf := make([]byte, 28)
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _ = io.ReadFull(conn, buf)
		reqCh <- buf

		if replyType == 0 {
			return // hang up without writing
		}

		out := new(bytes.Buffer)
		binary.Write(out, binary.LittleEndian, replyType)   // Type
		binary.Write(out, binary.LittleEndian, uint32(0))   // Source
		binary.Write(out, binary.LittleEndian, uint32(0))   // Stream
		binary.Write(out, binary.LittleEndian, uint32(0))   // Size
		conn.Write(out.Bytes())
	}()

	return "127.0.0.1", addr.Port, reqCh, func() { _ = ln.Close() }
}

func makeServer(host string, port int) domain.Server {
	return domain.Server{Host: host, Port: port}
}

func TestKnock_Success(t *testing.T) {
	host, port, reqCh, closeFn := startKnockListener(t, MSG2_MASTERINFO)
	defer closeFn()

	if err := Knock(makeServer(host, port), "req-1"); err != nil {
		t.Errorf("Knock: %v", err)
	}

	select {
	case req := <-reqCh:
		// Verify the request encodes MSG2_HELLO and MAGIC_SOURCE in the header.
		var msg, source, stream, length uint32
		r := bytes.NewReader(req)
		binary.Read(r, binary.LittleEndian, &msg)
		binary.Read(r, binary.LittleEndian, &source)
		binary.Read(r, binary.LittleEndian, &stream)
		binary.Read(r, binary.LittleEndian, &length)
		if msg != MSG2_HELLO {
			t.Errorf("request msg = %d, want MSG2_HELLO=%d", msg, MSG2_HELLO)
		}
		if source != MAGIC_SOURCE {
			t.Errorf("request source = %d, want MAGIC_SOURCE=%d", source, MAGIC_SOURCE)
		}
		if length != uint32(len("MasterServer")) {
			t.Errorf("payload length = %d, want %d", length, len("MasterServer"))
		}
		// Payload is padded to 12 bytes — confirm "MasterServer" prefix.
		if !bytes.HasPrefix(req[16:], []byte("MasterServer")) {
			t.Errorf("payload prefix mismatch: %q", req[16:])
		}
	case <-time.After(2 * time.Second):
		t.Error("listener did not record a request")
	}
}

func TestKnock_UnexpectedResponseType(t *testing.T) {
	host, port, _, closeFn := startKnockListener(t, 9999) // not MSG2_MASTERINFO
	defer closeFn()

	err := Knock(makeServer(host, port), "req-2")
	if !errors.Is(err, ErrUnexpected) {
		t.Errorf("expected ErrUnexpected, got %v", err)
	}
}

func TestKnock_ConnectionRefused(t *testing.T) {
	// Bind a listener, close it, then attempt to knock the freed port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	ln.Close()

	err = Knock(makeServer("127.0.0.1", addr.Port), "req-3")
	if !errors.Is(err, ErrSocketCreate) {
		t.Errorf("expected ErrSocketCreate, got %v", err)
	}
}

func TestKnock_ServerHangsUp_ReadError(t *testing.T) {
	// Listener accepts then closes without responding — Read returns EOF.
	host, port, _, closeFn := startKnockListener(t, 0)
	defer closeFn()

	err := Knock(makeServer(host, port), "req-4")
	if !errors.Is(err, ErrSocketRead) {
		t.Errorf("expected ErrSocketRead, got %v", err)
	}
}

// guards against accidental constant changes — the protocol numbers are wire-level.
func TestProtocolConstantsAreStable(t *testing.T) {
	cases := map[string]uint32{
		"MSG2_HELLO":      MSG2_HELLO,
		"MSG2_MASTERINFO": MSG2_MASTERINFO,
		"MAGIC_SOURCE":    MAGIC_SOURCE,
	}
	expected := map[string]uint32{
		"MSG2_HELLO":      1025,
		"MSG2_MASTERINFO": 1034,
		"MAGIC_SOURCE":    5000,
	}
	for name, got := range cases {
		if got != expected[name] {
			t.Errorf("%s = %d, want %d (wire protocol — do not change without coordinating the RoRnet side)", name, got, expected[name])
		}
	}
}
