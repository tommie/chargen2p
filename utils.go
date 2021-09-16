package chargen2p

import (
	"io"
	"math/rand"
)

// bufSize is how much data we generate per call to the kernel.
const bufSize = 1024 * 1024

// graphRandReader is like math/rand.Read, but only generates
// graphical ASCII characters.
var graphRandReader io.Reader = graphRandReaderStruct{}

type graphRandReaderStruct struct{}

func (graphRandReaderStruct) Read(bs []byte) (int, error) {
	n, err := rand.Read(bs)
	if err != nil {
		return 0, err
	}
	for i := 0; i < n; i++ {
		bs[i] = 33 + bs[i]%(127-33)
	}
	for i := 0; i < n; i += 80 {
		bs[i] = '\n'
	}
	return n, nil
}
