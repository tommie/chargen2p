package main

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNetListen(t *testing.T) {
	os.Setenv("LISTEN_PID", fmt.Sprint(os.Getpid()))
	defer os.Setenv("LISTEN_PID", "")

	t.Run("systemd", func(t *testing.T) {
		lf, err := newTestListenerFile()
		if err != nil {
			t.Fatalf("File failed: %v", err)
		}
		defer lf.Close()

		i := int(lf.Fd() - sdListenFDsStart)
		os.Setenv("LISTEN_FDS", fmt.Sprint(i+1))
		defer os.Setenv("LISTEN_FDS", "")

		l, err := netListen(fmt.Sprint("systemd:", i))
		if err != nil {
			t.Fatalf("netListen failed: %v", err)
		}
		l.Close()
	})

	t.Run("net", func(t *testing.T) {
		l, err := netListen("tcp:localhost:0")
		if err != nil {
			t.Fatalf("netListen failed: %v", err)
		}
		l.Close()
	})
}
func TestListenSystemd(t *testing.T) {
	os.Setenv("LISTEN_PID", fmt.Sprint(os.Getpid()))
	defer os.Setenv("LISTEN_PID", "")

	lf, err := newTestListenerFile()
	if err != nil {
		t.Fatalf("File failed: %v", err)
	}
	defer lf.Close()

	i := int(lf.Fd() - sdListenFDsStart)
	os.Setenv("LISTEN_FDS", fmt.Sprint(i+1))
	defer os.Setenv("LISTEN_FDS", "")

	l, err := listenSystemd(i)
	if err != nil {
		t.Fatalf("listenSystemd failed: %v", err)
	}
	l.Close()
}

func TestSystemdFDByIndex(t *testing.T) {
	os.Setenv("LISTEN_PID", fmt.Sprint(os.Getpid()))
	defer os.Setenv("LISTEN_PID", "")

	t.Run("good", func(t *testing.T) {
		os.Setenv("LISTEN_FDS", "1")
		defer os.Setenv("LISTEN_FDS", "")

		got, err := systemdFDByIndex(0)
		if err != nil {
			t.Fatalf("systemdFDByIndex failed: %v", err)
		}

		if want := sdListenFDsStart; got != want {
			t.Errorf("systemdFDByIndex: got %v, want %v", got, want)
		}
	})

	t.Run("oob", func(t *testing.T) {
		os.Setenv("LISTEN_FDS", "1")
		defer os.Setenv("LISTEN_FDS", "")

		_, err := systemdFDByIndex(1)
		if err == nil {
			t.Errorf("systemdFDByIndex err: got %v, want non-nil", err)
		} else if !strings.Contains(err.Error(), "index out") {
			t.Errorf("systemdFDByIndex err: got %v, want index OOB", err)
		}
	})
}

func TestParseSystemFD(t *testing.T) {
	t.Run("good", func(t *testing.T) {
		os.Setenv("LISTEN_PID", fmt.Sprint(os.Getpid()))
		defer os.Setenv("LISTEN_PID", "")
		os.Setenv("LISTEN_FDS", "1")
		defer os.Setenv("LISTEN_FDS", "")

		got, err := parseSystemdFD()
		if err != nil {
			t.Fatalf("parseSystemdFD failed: %v", err)
		}

		if want := []uintptr{sdListenFDsStart, sdListenFDsStart + 1}; !reflect.DeepEqual(got, want) {
			t.Errorf("parseSystemdFD: got %+v, want %+v", got, want)
		}
	})

	t.Run("nonint", func(t *testing.T) {
		os.Setenv("LISTEN_PID", fmt.Sprint(os.Getpid()))
		defer os.Setenv("LISTEN_PID", "")
		os.Setenv("LISTEN_FDS", "abc")
		defer os.Setenv("LISTEN_FDS", "")

		_, err := parseSystemdFD()
		if err == nil {
			t.Errorf("parseSystemdFD err: got %v, want non-nil", err)
		} else if !strings.Contains(err.Error(), "abc") {
			t.Errorf("parseSystemdFD err: got %v, want parse failure", err)
		}
	})

	t.Run("nopid", func(t *testing.T) {
		os.Setenv("LISTEN_PID", "")
		os.Setenv("LISTEN_FDS", "1")
		defer os.Setenv("LISTEN_FDS", "")

		_, err := parseSystemdFD()
		if err != errSystemdUnavailable {
			t.Errorf("parseSystemdFD err: got %v, want %v", err, errSystemdUnavailable)
		}
	})
}

func newTestListenerFile() (*os.File, error) {
	laddr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP(laddr.Network(), laddr)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	return l.File()
}
