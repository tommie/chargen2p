package chargen2p

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

var _ NetConn = &net.TCPConn{}
var _ NetConn = &net.UnixConn{}

func TestDialTCP(t *testing.T) {
	ctx := context.Background()

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %v", err)
		}
		conn.Close()
	}()

	conn, err := Dial(ctx, l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	conn.Close()
}

func TestSend(t *testing.T) {
	ctx := context.Background()

	var nconn fakeNetConn
	conn := NewConn(&nconn)
	defer conn.Close()

	var now int
	conn.now = func() time.Time {
		now++
		return time.Date(2006, 1, 2, 15, 16, now, 0, time.UTC)
	}

	n, dur, err := conn.Send(ctx, 1024)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if want := 1; nconn.NumCloseWrite != want {
		t.Errorf("NumCloseWrite: got %v, want %v", nconn.NumCloseWrite, want)
	}
	if want := 1024; nconn.NumWrite != want {
		t.Errorf("NumWrite: got %v, want %v", nconn.NumWrite, want)
	}

	if want := 1024; n != want {
		t.Errorf("Send n: got %v, want %v", n, want)
	}

	if want := 1 * time.Second; dur != want {
		t.Errorf("Send dur: got %v, want %v", dur, want)
	}
}

func TestRecv(t *testing.T) {
	ctx := context.Background()

	nconn := fakeNetConn{NumRead: 1024}
	conn := NewConn(&nconn)
	defer conn.Close()

	var now int
	conn.now = func() time.Time {
		now++
		return time.Date(2006, 1, 2, 15, 16, now, 0, time.UTC)
	}

	n, dur, lat, err := conn.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if want := 0; nconn.NumRead != want {
		t.Errorf("NumRead: got %v, want %v", nconn.NumRead, want)
	}

	if want := 1024; n != want {
		t.Errorf("Recv n: got %v, want %v", n, want)
	}

	if want := 1 * time.Second; lat != want {
		t.Errorf("Recv lat: got %v, want %v", lat, want)
	}

	if want := 2 * time.Second; dur != want {
		t.Errorf("Recv dur: got %v, want %v", dur, want)
	}
}

func TestDeadline(t *testing.T) {
	want := time.Now().Add(1 * time.Hour)
	ctx, cancel := context.WithDeadline(context.Background(), want)
	defer cancel()

	t.Run("send", func(t *testing.T) {
		var nconn fakeNetConn
		conn := NewConn(&nconn)
		defer conn.Close()

		_, _, err := conn.Send(ctx, 1024)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if nconn.WriteDeadline != want {
			t.Errorf("WriteDeadline: got %v, want %v", nconn.WriteDeadline, want)
		}
	})

	t.Run("recv", func(t *testing.T) {
		nconn := fakeNetConn{NumRead: 1024}
		conn := NewConn(&nconn)
		defer conn.Close()

		_, _, _, err := conn.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}

		if nconn.ReadDeadline != want {
			t.Errorf("ReadDeadline: got %v, want %v", nconn.ReadDeadline, want)
		}
	})
}

type fakeNetConn struct {
	NumRead int

	NumCloseWrite int
	NumWrite      int
	ReadDeadline  time.Time
	WriteDeadline time.Time
}

func (*fakeNetConn) Close() error { return nil }
func (c *fakeNetConn) CloseWrite() error {
	c.NumCloseWrite++
	return nil
}
func (c *fakeNetConn) Read(bs []byte) (int, error) {
	if c.NumRead == 0 {
		return 0, io.EOF
	}
	n := len(bs)
	if n > c.NumRead {
		n = c.NumRead
	}
	c.NumRead -= n
	return n, nil
}
func (c *fakeNetConn) Write(bs []byte) (int, error) {
	c.NumWrite += len(bs)
	return len(bs), nil
}
func (c *fakeNetConn) SetReadDeadline(t time.Time) error {
	c.ReadDeadline = t
	return nil
}
func (c *fakeNetConn) SetWriteDeadline(t time.Time) error {
	c.WriteDeadline = t
	return nil
}
