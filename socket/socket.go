// Copyright (C) 2022 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package socket

import (
	"bufio"
	"errors"
	"net"
	"syscall"
	"time"
)

// ErrRefused is returned when net.Dial is unable to establish a
// connection because the host refuses. In such a case, this can
// be attributed to the host blocking inbound connections.
var ErrRefused = errors.New("socket: connection refused")

// ErrTimeout is returned when net.DialTimeout or net.Dial has
// exceeded the time limit. This could potentially be attributed to
// a slow connection.
var ErrTimeout = errors.New("socket: connection timed out")

// ErrHostUnknown is returned when net.Dial is unable to resolve the
// host.
var ErrHostUnknown = errors.New("socket: unknown host")

// ErrReset happens when an existing connection was forcibly closed
// by the host.
var ErrReset = errors.New("socket: connection reset by peer")

// ErrFailed happens when the connection goes down without the peer
// explicitly closing the connection.
var ErrFailed = errors.New("socket: connection failed")

type Client interface {
	Read() (string, error)
	Write(d string) error
	Close()
}

type client struct {
	sock net.Conn
}

func Conn(addr string) (Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	c := client{
		sock: conn,
	}

	if err != nil {
		logError(err)
		if serr, ok := err.(net.Error); ok && serr.Timeout() {
			return &c, ErrTimeout
		}
		// We want to find exactly what type of error we encounter but
		// doing so with *OpError opens a can of worms with syscalls
		// under Linux being different to those under Windows. So we
		// try to mostly scratch the surface for debugging.
		switch e := err.(type) {
		case *net.OpError:
			if e.Op == "dial" {
				return &c, ErrHostUnknown
			} else if e.Op == "read" {
				return &c, ErrRefused
			}
		case syscall.Errno:
			if e == syscall.ECONNREFUSED {
				return &c, ErrRefused
			} else if e == syscall.ECONNABORTED {
				return &c, ErrFailed
			} else if e == syscall.ECONNRESET {
				return &c, ErrReset
			}
		}
		return &c, ErrFailed
	}

	return &c, nil
}

func (c *client) Read() (string, error) {
	return "", nil
}

func (c *client) Write(d string) error {
	w := bufio.NewWriter(c.sock)
	_, err := w.WriteString(d)
	if err == nil {
		err = w.Flush()
	}
	return err
}

func (c *client) Close() {
	c.sock.Close()
}

func logError(err error) {
}

func logDebug() {

}
