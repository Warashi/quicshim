package quicshim

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/lucas-clemente/quic-go"
)

var (
	_ net.Listener = (*Listener)(nil)
)

type Listener struct {
	closeCh      chan struct{}
	streamCh     chan *StreamConn
	quicListener quic.Listener
}

func Listen(addr string, tlsConf *tls.Config, quicConf *quic.Config) (*Listener, error) {
	ql, err := quic.ListenAddr(addr, tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("quic.ListenAddr: %w", err)
	}
	l := &Listener{
		closeCh:      make(chan struct{}),
		streamCh:     make(chan *StreamConn),
		quicListener: ql,
	}
	go l.listenQUIC(context.Background())
	return l, nil
}

func (l *Listener) listenQUIC(ctx context.Context) error {
	ctx, cancel := cancelWhenClose(ctx, l.closeCh)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := l.quicListener.Accept(ctx)
		if err != nil {
			return fmt.Errorf("ql.Accept: %w", err)
		}
		go l.listenConn(ctx, conn)
	}
}

func (l *Listener) listenConn(ctx context.Context, conn quic.Connection) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-conn.Context().Done():
			return net.ErrClosed
		default:
		}

		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return fmt.Errorf("conn.AcceptStream: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-conn.Context().Done():
			return net.ErrClosed
		case l.streamCh <- &StreamConn{conn, stream}:
		}
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case <-l.closeCh:
		return nil, net.ErrClosed
	case conn := <-l.streamCh:
		return conn, nil
	}
}

func (l *Listener) Close() error {
	defer close(l.closeCh)
	if err := l.quicListener.Close(); err != nil {
		return fmt.Errorf("l.quicListener.Close: %w", err)
	}
	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.quicListener.Addr()
}
