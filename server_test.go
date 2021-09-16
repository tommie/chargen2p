package chargen2p

import (
	"context"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	ctx := context.Background()

	var r recordReporter
	s := Server{Reporter: &r, MaxConns: 1}
	l := fakeNetListener{NumAccept: 2}
	err := s.Serve(ctx, &l)
	if err != io.EOF {
		t.Fatalf("Serve err: got %v, want %v", err, io.EOF)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if want := []error{nil, nil}; !reflect.DeepEqual(r.Errs, want) {
		t.Errorf("Errs: got %+v, want %+v", r.Errs, want)
	}
}

func TestServerHandleConn(t *testing.T) {
	ctx := context.Background()

	var r recordReporter
	s := Server{Reporter: &r, MaxConns: 1}
	conn := &fakeConn{NumRecv: 1024}
	s.handleConn(ctx, nil, conn)

	r.mu.Lock()
	defer r.mu.Unlock()
	if want := []error{nil}; !reflect.DeepEqual(r.Errs, want) {
		t.Errorf("Errs: got %+v, want %+v", r.Errs, want)
	}

	if want := 1024; conn.NumSend != want {
		t.Errorf("NumSend: got %v, want %v", conn.NumSend, want)
	}
}

type recordReporter struct {
	Errs []error
	mu   sync.Mutex
}

func (r *recordReporter) ServedCharGen2P(conn net.Conn, ti *ThroughputInfo, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Errs = append(r.Errs, err)
}

type fakeNetListener struct {
	net.Listener

	NumAccept int
}

func (l *fakeNetListener) Accept() (net.Conn, error) {
	if l.NumAccept == 0 {
		return nil, io.EOF
	}
	l.NumAccept--
	return &fakeServerNetConn{fakeNetConn: fakeNetConn{NumRead: 1024}}, nil
}

type fakeServerNetConn struct {
	net.Conn
	fakeNetConn
}

func (c *fakeServerNetConn) Close() error                 { return nil }
func (c *fakeServerNetConn) Read(bs []byte) (int, error)  { return c.fakeNetConn.Read(bs) }
func (c *fakeServerNetConn) Write(bs []byte) (int, error) { return c.fakeNetConn.Write(bs) }
func (c *fakeServerNetConn) SetReadDeadline(t time.Time) error {
	return c.fakeNetConn.SetReadDeadline(t)
}
func (c *fakeServerNetConn) SetWriteDeadline(t time.Time) error {
	return c.fakeNetConn.SetWriteDeadline(t)
}

type fakeConn struct {
	NumRecv int

	NumSend int
}

func (c *fakeConn) Send(_ context.Context, n int) (int, time.Duration, error) {
	c.NumSend += n
	return 0, 0, nil
}
func (c *fakeConn) Recv(_ context.Context) (int, time.Duration, time.Duration, error) {
	return c.NumRecv, 0, 0, nil
}
