// Command chargen2pd implements a 2-phase character generator
// service. It first receives and discards data until the receiving
// side is closed, and then sends back the same amount of random data.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"github.com/tommie/chargen2p"
)

var (
	listenAddr      = flag.String("listen-addr", "tcp:localhost:19", "TCP address to listen for connections on. Use `systemd:0` for Systemd socket activation.")
	maxConns        = flag.Uint("max-conns", 10, "Maximum number of simultaneous connections.")
	connTimeoutSecs = flag.Int("conn-timeout", 10, "Connection timeout in seconds.")

	logPerConnection = flag.Bool("log-per-connection", true, "Whether to log statistics for each connection.")
	standaloneLog    = flag.Bool("standalone-log", true, "Log to stderr, with time prefix.")
)

func main() {
	flag.Parse()

	if err := run(context.Background()); err != nil {
		log.Print(err)
		os.Exit(11)
	}
}

func run(ctx context.Context) error {
	if !*standaloneLog {
		log.SetFlags(0)
		log.SetOutput(os.Stdout)
	}

	l, err := netListen(*listenAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("Listening for connections on %s...", l.Addr().String())

	s := chargen2p.Server{
		Reporter: logReporter(*logPerConnection),
		MaxConns: int(*maxConns),
		ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
			// The parent context will be cancelled for us.
			cctx, _ := context.WithTimeout(ctx, time.Duration(*connTimeoutSecs)*time.Second)
			return cctx
		},
	}

	return s.Serve(ctx, l)
}

type logReporter bool

func (r logReporter) ServedCharGen2P(conn net.Conn, ti *chargen2p.ThroughputInfo, err error) {
	if !r {
		return
	}

	if err != nil {
		log.Printf("%s: %v", conn.RemoteAddr().String(), err)
		return
	}

	log.Printf("%s: Received %d bytes in %s, sent %d bytes in %s.", conn.RemoteAddr().String(), ti.NumReadBytes, ti.ReadDuration, ti.NumWrittenBytes, ti.WriteDuration)
}
