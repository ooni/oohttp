package ja3x

import (
	"net"
	"sync"
)

// newConnWrapper creates a new [connWrapper].
func newConnWrapper(conn net.Conn) *connWrapper {
	return &connWrapper{
		Conn:  conn,
		hello: []byte{},
		mu:    sync.Mutex{},
	}
}

// connWrapper is a wrapper that extracts the raw TLS client hello.
type connWrapper struct {
	// Conn is the underlying conn.
	net.Conn

	// hello contains the client hello.
	hello []byte

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// Hello returns a copy of the bytes inside the client hello.
func (cw *connWrapper) Hello() (out []byte) {
	defer cw.mu.Unlock()
	cw.mu.Lock()
	out = append(out, cw.hello...)
	return
}

// Read implements net.Conn.
func (cw *connWrapper) Read(data []byte) (int, error) {
	count, err := cw.Conn.Read(data)
	if err != nil {
		return 0, err
	}
	defer cw.mu.Unlock()
	cw.mu.Lock()
	if len(cw.hello) <= 0 {
		cw.hello = append(cw.hello, data[:count]...) // makes a copy
	}
	return count, nil
}
