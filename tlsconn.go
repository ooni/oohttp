package http

import (
	"crypto/tls"
	"net"
)

// TLSConn is the interface representing a *tls.Conn compatible
// connection, which could possibly be different from a *tls.Conn
// as long as it implements the interface. You can use, for
// example, refraction-networking/utls instead of the stdlib.
type TLSConn interface {
	net.Conn
	ConnectionState() tls.ConnectionState
	Handshake() error
}
