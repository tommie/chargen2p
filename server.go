package chargen2p

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/sync/semaphore"
)

// A Server listens for incoming connections and handles processing
// them.
type Server struct {
	// Reporter is an optional result handler. Normally, only the
	// client is interested in the outcome, so this can be nil.
	Reporter Reporter

	// MaxConns is the maximum number of simultaneous connections.
	// This will create a backlog if we're over quota, which pushes
	// back through the kernel to TCP clients. The default is 10.
	MaxConns int

	// ConnContext is a factory function for creating a per-connection
	// context. This is useful to set a timeout.
	ConnContext func(context.Context, net.Conn) context.Context
}

// A Reporter hooks into various connection state
// changes. Implementations must be safe for concurrent use.
type Reporter interface {
	// ServedCharGen2P is called when a connection is closed. Either
	// there is throughput information, or an error.
	ServedCharGen2P(net.Conn, *ThroughputInfo, error)
}

// Serve starts accepting connections and forks off connection handlers.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	connContext := s.ConnContext
	if connContext == nil {
		connContext = func(ctx context.Context, _ net.Conn) context.Context {
			return ctx
		}
	}
	if s.MaxConns == 0 {
		s.MaxConns = 10
	}

	sem := semaphore.NewWeighted(int64(s.MaxConns))
	for {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func() {
			defer sem.Release(1)
			cctx, cancel := context.WithCancel(ctx)
			defer cancel()
			s.serveConn(connContext(cctx, conn), conn)
		}()
	}
}

// serveConn takes responsibility for the lifecycle of conn
// (including closing it). It counts received data, and then sends
// back the same amount.
func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	if tconn, ok := conn.(setNoDelayer); ok {
		tconn.SetNoDelay(true)
	}
	nc, ok := conn.(NetConn)
	if !ok {
		if s.Reporter != nil {
			s.Reporter.ServedCharGen2P(conn, nil, fmt.Errorf("connection is not a chargen2p.NetConn"))
		}
		return
	}

	s.handleConn(ctx, conn, NewConn(nc))
}

// handleConn counts received data, and then sends back the same
// amount.
func (s *Server) handleConn(ctx context.Context, conn net.Conn, c serverConn) {
	nr, rdur, _, err := c.Recv(ctx)
	if err != nil {
		if s.Reporter != nil {
			s.Reporter.ServedCharGen2P(conn, nil, err)
		}
		return
	}

	nw, wdur, err := c.Send(ctx, nr)
	if err != nil {
		if s.Reporter != nil {
			s.Reporter.ServedCharGen2P(conn, nil, err)
		}
		return
	}

	if s.Reporter != nil {
		s.Reporter.ServedCharGen2P(conn, &ThroughputInfo{
			NumWrittenBytes: nw,
			WriteDuration:   wdur,
			NumReadBytes:    nr,
			ReadDuration:    rdur,
		}, nil)
	}
}

type serverConn interface {
	Recv(context.Context) (int, time.Duration, time.Duration, error)
	Send(context.Context, int) (int, time.Duration, error)
}

// setNoDelayer allows the server to set flow control options on (TCP)
// connections.
type setNoDelayer interface {
	SetNoDelay(bool) error
}
