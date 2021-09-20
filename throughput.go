package chargen2p

import (
	"context"
	"fmt"
	"math"
	"net"
	"time"
)

// MeasureThroughput dials, sends and receives a Conn. Reports summary
// statistics.
func MeasureThroughput(ctx context.Context, network, addr string, opts ...MeasureThroughputOpt) (*ThroughputInfo, error) {
	mto := &measureThroughputOpts{
		maxBytes:  1024 * 1024 * 1024,
		maxDur:    10 * time.Second,
		minIters:  8,
		tolerance: 0.05,
		netDialer: &net.Dialer{},
	}
	for _, opt := range opts {
		if err := opt(mto); err != nil {
			return nil, err
		}
	}

	ti := &ThroughputInfo{}
	n := bufSize
	if n > mto.maxBytes/mto.minIters {
		n = mto.maxBytes / mto.minIters
	}
	i := 0
	for ; ti.NumReadBytes+n <= mto.maxBytes; i++ {
		dstart := time.Now()
		c, err := dial(ctx, network, addr, mto.netDialer)
		if err != nil {
			return nil, err
		}
		dend := time.Now()

		nw, wdur, err := c.Send(ctx, n)
		if err != nil {
			return nil, err
		}

		nr, rdur, frdur, err := c.Recv(ctx)
		if err != nil {
			return nil, err
		}

		ti.DialDuration += dend.Sub(dstart)
		ti.NumWrittenBytes += nw
		ti.WriteDuration += wdur
		ti.NumReadBytes += nr
		ti.ReadDuration += rdur
		ti.ReadLatency += frdur

		if i > 1 {
			// This is a simple relative error tolerance
			// check. Real-world use will determine if it needs to be
			// more statistically robust.
			ti.WorstAccuracy = math.Abs(float64(ti.NumWrittenBytes)/float64(ti.WriteDuration)/(float64(ti.NumWrittenBytes-nw)/float64(ti.WriteDuration-wdur)) - 1)
			if v := math.Abs(float64(ti.NumReadBytes)/float64(ti.ReadDuration)/(float64(ti.NumReadBytes-nr)/float64(ti.ReadDuration-rdur)) - 1); v > ti.WorstAccuracy {
				ti.WorstAccuracy = v
			}

			if ti.WorstAccuracy <= mto.tolerance {
				ti.ReadLatency /= time.Duration(i)
				return ti, nil
			}
		}

		// The number of iterations matter, so we cap per-iteration
		// size to something reasonable. We don't want to hit a
		// server-side timeout either.
		wdt := time.Duration(2 * n * int(ti.WriteDuration) / int(ti.NumWrittenBytes))
		rdt := time.Duration(2 * n * int(ti.ReadDuration) / int(ti.NumReadBytes))
		if n < mto.maxBytes/mto.minIters && (i == 0 || wdt+rdt < mto.maxDur) {
			n *= 2
		}
	}

	ti.ReadLatency /= time.Duration(i)
	return ti, fmt.Errorf("couldn't reach tolerance within limits: got %.2f, want %.2f", ti.WorstAccuracy, mto.tolerance)
}

// A MeasureThroughputOpt can be provided to MeasureThroughput.
type MeasureThroughputOpt func(*measureThroughputOpts) error

// WithMaxBytes sets a cap on the total number of bytes to send. The
// hope is to never hit this hard limit, but that the tolerance
// criterion is fulfilled first. The default is 1 GiB.
func WithMaxBytes(n int) MeasureThroughputOpt {
	return func(opts *measureThroughputOpts) error {
		opts.maxBytes = n
		return nil
	}
}

// WithMaxDuration sets a cap on the duration of a single
// connection. This should be at most what the connection timeout is
// on the server. The default is 10s.
func WithMaxDuration(d time.Duration) MeasureThroughputOpt {
	return func(opts *measureThroughputOpts) error {
		opts.maxDur = d
		return nil
	}
}

// WithMinIterations adjusts the number of bytes per iteration so that
// `WithMaxBytes` is not hit until, at least, this many
// iterations. The default is 8.
func WithMinIterations(n int) MeasureThroughputOpt {
	return func(opts *measureThroughputOpts) error {
		opts.minIters = n
		return nil
	}
}

// WithTolerance sets the tolerated relative error of the last
// iteration. If the latest send/receive measurement is within this
// tolerance, measurements stop. A value in [0, 1]. The default is 5%.
func WithTolerance(frac float64) MeasureThroughputOpt {
	return func(opts *measureThroughputOpts) error {
		opts.tolerance = frac
		return nil
	}
}

func WithDialer(d NetDialer) MeasureThroughputOpt {
	return func(opts *measureThroughputOpts) error {
		opts.netDialer = d
		return nil
	}
}

type measureThroughputOpts struct {
	maxBytes  int
	maxDur    time.Duration
	minIters  int
	tolerance float64
	netDialer NetDialer
}

// A ThroughputInfo contains various measurements from a throughput
// test.
type ThroughputInfo struct {
	// DialDuration is how long connecting to the remote peer took.
	DialDuration time.Duration

	// NumWrittenBytes is the number of (payload) bytes sent. Protocol
	// headers are not included.
	NumWrittenBytes int

	// WriteDuration is how long it took to send the data, including
	// closing the send endpoint.
	WriteDuration time.Duration

	// NumReadBytes is the number of (payload) bytes
	// received. Protocol headers are not included.
	NumReadBytes int

	// ReadDuration is how long it took to receive the data, counting
	// from after closing the send endpoint.
	ReadDuration time.Duration

	// ReadLatency is how long after closing the send endpoint the
	// first received block was ready. This is a rough estimate of
	// latency, but depends on the internal receive buffer size.
	ReadLatency time.Duration

	// WorstAccuracy describes the worst acceptable relative error for
	// the write and read throughput of the last iteration.
	// `DialDuration` and `FirstReadLatency` are not considered as
	// they normally have much higher variances. A value in [0, 1].
	WorstAccuracy float64
}
