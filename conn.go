package quicshim

import (
	"net"

	"github.com/lucas-clemente/quic-go"
)

type StreamConn struct {
	conn quic.Connection
	quic.Stream
}

func (c *StreamConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *StreamConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
