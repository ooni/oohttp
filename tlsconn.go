package http

import (
	"context"
	"crypto/tls"
	"net"
)

// TLSConn is the interface representing a *tls.Conn compatible
// connection, which could possibly be different from a *tls.Conn
// as long as it implements the interface. You can use, for
// example, refraction-networking/utls instead of the stdlib.
type TLSConn interface {
	// net.Conn is the underlying interface
	net.Conn

	// ConnectionState returns the ConnectionState according
	// to the standard library.
	ConnectionState() tls.ConnectionState

	// HandshakeContext performs an TLS handshake bounded
	// in time by the given context.
	HandshakeContext(ctx context.Context) error
}

// TLSClientFactory is the factory used when creating connections
// using a proxy inside of the HTTP library. By default, this is
// the tls.Client function. You'll need to override this factory if
// you want to use refraction-networking/utls for proxied conns.
var TLSClientFactory = tls.Client
