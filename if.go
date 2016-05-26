package taptun

import (
	"os"
)

// Interface is a TUN/TAP interface.
type Interface struct {
	isTAP bool
	file  *os.File
	name  string
}

// Provides thread-safe read and write operations that can be cancelled.
type Accessor interface {
	// io.Writer interface.
	Write(p []byte) (n int, err error)

	// io.Reader interface.
	Read(p []byte) (n int, err error)

	// Stops any pending reads and writes. Any subsequent read or write
	// operations on this accessor will return an EOF error.  Does not
	// close the underlying device.
	Stop() bool
}

// Wraps this Interface with a thread-safe Accessor.
func (ifce *Interface) Accessor() (Accessor, error) {
	return wrap(ifce)
}
