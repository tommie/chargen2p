package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var errSystemdUnavailable = errors.New("systemd is not available")

// netListen splits the argument by colon and checks for Systemd file
// descriptors (e.g. "systemd:0") or creates a new listening socket
// (e.g. "tcp:localhost:80").
func netListen(networkAndAddr string) (net.Listener, error) {
	ss := strings.SplitN(networkAndAddr, ":", 2)
	if len(ss) != 2 {
		return nil, fmt.Errorf("invalid address: %s", networkAndAddr)
	}
	if ss[0] == "systemd" {
		i, err := strconv.ParseInt(ss[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %v", err)
		}
		return listenSystemd(int(i))
	}

	return net.Listen(ss[0], ss[1])
}

// listenSystemd looks up a Systemd-provided file descriptor by index.
//
// See https://www.freedesktop.org/software/systemd/man/sd_listen_fds.html.
func listenSystemd(i int) (net.Listener, error) {
	fd, err := systemdFDByIndex(i)
	if err != nil {
		return nil, err
	}

	f := os.NewFile(fd, "net+systemd")
	if f == nil {
		return nil, fmt.Errorf("invalid file descriptor: %d", fd)
	}
	return net.FileListener(f)
}

// systemdFDByIndex returns the FD associated with the given Systemd index.
func systemdFDByIndex(i int) (fd uintptr, err error) {
	fdrange, err := parseSystemdFD()
	if err != nil {
		return 0, err
	}
	if nfds := int(fdrange[1] - fdrange[0]); i < 0 || i >= nfds {
		return 0, fmt.Errorf("index out of bounds: %d (range [0, %d])", i, nfds-1)
	}
	return fdrange[0] + uintptr(i), nil
}

// parseSystemdFD interprets the process environment and returns the
// available range of FDs. The range is half-closed.
func parseSystemdFD() (fdrange []uintptr, err error) {
	if os.Getenv("LISTEN_PID") != strconv.FormatInt(int64(os.Getpid()), 10) {
		return nil, errSystemdUnavailable
	}
	nfds, err := strconv.ParseInt(os.Getenv("LISTEN_FDS"), 10, 32)
	if err != nil {
		return nil, err
	}
	return []uintptr{sdListenFDsStart, sdListenFDsStart + uintptr(nfds)}, nil
}

const sdListenFDsStart uintptr = 3
