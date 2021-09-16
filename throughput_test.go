package chargen2p

import (
	"context"
	"net"
	"testing"
)

func TestMeasureThroughput(t *testing.T) {
	ctx := context.Background()

	var s Server

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	go s.Serve(ctx, l)

	got, err := MeasureThroughput(ctx, l.Addr().Network(), l.Addr().String(), WithMaxBytes(128*bufSize), WithMinIterations(4), WithTolerance(0.5), WithDialer(&net.Dialer{}))
	if err != nil {
		t.Fatalf("MeasureThroughput failed: %v", err)
	}

	if got.DialDuration == 0 {
		t.Errorf("MeasureThroughput DialDuration: got %+v, want >0", got)
	}
	if got.NumWrittenBytes == 0 {
		t.Errorf("MeasureThroughput NumWrittenBytes: got %+v, want >0", got)
	}
	if got.WriteDuration == 0 {
		t.Errorf("MeasureThroughput WriteDuration: got %+v, want >0", got)
	}
	if got.NumReadBytes == 0 {
		t.Errorf("MeasureThroughput NumReadBytes: got %+v, want >0", got)
	}
	if got.ReadDuration == 0 {
		t.Errorf("MeasureThroughput ReadDuration: got %+v, want >0", got)
	}
	if got.ReadLatency == 0 {
		t.Errorf("MeasureThroughput ReadLatency: got %+v, want >0", got)
	}
	if got.WorstAccuracy == 0 {
		t.Errorf("MeasureThroughput WorstAccuracy: got %+v, want >0", got)
	}
}
