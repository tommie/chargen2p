package chargen2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// ErrNoDataReceived is returned if Conn.Recv saw no data.
var ErrNoDataReceived = errors.New("no data received")

// A Conn can receive and send random data.
type Conn struct {
	conn NetConn
	r    io.Reader
	now  func() time.Time
}

// A NetConn is what we can wrap. `*net.TCPConn` implements this.
type NetConn interface {
	io.Closer
	io.Reader
	io.Writer
	CloseWrite() error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// NewConn wraps a new connection.
func NewConn(conn NetConn) *Conn {
	return &Conn{conn: conn, r: graphRandReader, now: time.Now}
}

// Dial is a helper to dial and create a new connection.
func Dial(ctx context.Context, network, addr string) (*Conn, error) {
	var d net.Dialer
	return dial(ctx, network, addr, &d)
}

func dial(ctx context.Context, network, addr string, d NetDialer) (*Conn, error) {
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	nc, ok := conn.(NetConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("connection is not a chargen2p.NetConn")
	}
	return NewConn(nc), nil
}

type NetDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// Send sends n bytes of data, closes the send-side and measures how
// long it took. Returns the actual number of bytes written, and the
// duration.
func (c *Conn) Send(ctx context.Context, n int) (int, time.Duration, error) {
	if t, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(t)
	}

	lr := io.LimitReader(c.r, int64(n))
	buf := make([]byte, bufSize)
	start := c.now()
	nw, err := io.CopyBuffer(c.conn, lr, buf)
	if err != nil {
		return 0, 0, err
	}

	if err := c.conn.CloseWrite(); err != nil {
		return int(nw), 0, err
	}

	return int(nw), c.now().Sub(start), nil
}

// Recv receives data until EOF and measures how long it took. Returns
// the number of bytes received, the time it took, and the time until
// the first block was received. Note that the later depends on the
// receiver buffer side, and is just an approximation of latency.
func (c *Conn) Recv(ctx context.Context) (int, time.Duration, time.Duration, error) {
	if t, ok := ctx.Deadline(); ok {
		c.conn.SetReadDeadline(t)
	}

	cw := countingWriter{now: c.now}
	buf := make([]byte, bufSize)
	start := c.now()
	nr, err := io.CopyBuffer(&cw, c.conn, buf)
	if err != nil {
		return 0, 0, 0, err
	}

	if nr == 0 {
		return 0, 0, 0, ErrNoDataReceived
	}

	return int(nr), c.now().Sub(start), cw.FirstTime.Sub(start), nil
}

type countingWriter struct {
	now       func() time.Time
	n         int
	FirstTime time.Time
}

func (w *countingWriter) Write(bs []byte) (int, error) {
	if w.n == 0 {
		w.FirstTime = w.now()
	}
	w.n += len(bs)
	return len(bs), nil
}
