package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tommie/chargen2p"
)

var (
	addr = flag.String("addr", "localhost:19", "TCP address to connect to.")
)

func main() {
	flag.Parse()

	if err := run(context.Background()); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(11)
	}
}

func run(ctx context.Context) error {
	ti, err := chargen2p.MeasureThroughput(ctx, "tcp", *addr)
	if err != nil {
		return err
	}

	fmt.Printf("Uploaded %d bytes in %s: %.3f Mbps\n", ti.NumWrittenBytes, ti.WriteDuration, float64(ti.NumWrittenBytes)/(float64(ti.WriteDuration)/float64(time.Second))*8/1e6)
	fmt.Printf("Downloaded %d bytes in %s: %.3f Mbps\n", ti.NumReadBytes, ti.ReadDuration, float64(ti.NumReadBytes)/(float64(ti.ReadDuration)/float64(time.Second))*8/1e6)

	return nil
}
