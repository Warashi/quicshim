package quicshim

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"golang.org/x/sync/singleflight"
)

type Dialer struct {
	tlsConfig    *tls.Config
	quicConfig   *quic.Config
	singleflight singleflight.Group
	connections  sync.Map
}

func NewDialer(tlsConf *tls.Config, quicConf *quic.Config) *Dialer {
	return &Dialer{tlsConfig: tlsConf, quicConfig: quicConf}
}

func (d *Dialer) open(ctx context.Context, conn quic.Connection) (net.Conn, error) {
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("conn.OpenStreamSync: %w", err)
	}
	return &StreamConn{conn, stream}, nil
}

func (d *Dialer) getConnection(ctx context.Context, addr string) (quic.Connection, error) {
	if conn, ok := d.connections.Load(addr); ok {
		return conn.(quic.Connection), nil
	}
	conn, err := quic.DialAddrContext(ctx, addr, d.tlsConfig, d.quicConfig)
	if err != nil {
		return nil, fmt.Errorf("quic.DialAddr: %w", err)
	}
	d.connections.Store(addr, conn)
	return conn, nil
}

func (d *Dialer) Dial(_network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), _network, addr)
}

func (d *Dialer) DialContext(ctx context.Context, _network, addr string) (net.Conn, error) {
	conn, err, _ := d.singleflight.Do(addr, func() (interface{}, error) { return d.getConnection(ctx, addr) })
	if err != nil {
		return nil, fmt.Errorf("d.getConnection: %w", err)
	}
	return d.open(ctx, conn.(quic.Connection))
}
