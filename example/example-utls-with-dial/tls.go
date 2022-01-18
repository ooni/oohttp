package main

import (
	"context"
	"crypto/tls"

	oohttp "github.com/ooni/oohttp"
	utls "github.com/refraction-networking/utls"
)

// utlsConnAdapter adapts utls.UConn to the oohttp.TLSConn interface.
type utlsConnAdapter struct {
	*utls.UConn
}

var _ oohttp.TLSConn = &utlsConnAdapter{}

// ConnectionState implements TLSConn's ConnectionState.
func (c *utlsConnAdapter) ConnectionState() tls.ConnectionState {
	ustate := c.UConn.ConnectionState()
	return tls.ConnectionState{
		Version:                     ustate.Version,
		HandshakeComplete:           ustate.HandshakeComplete,
		DidResume:                   ustate.DidResume,
		CipherSuite:                 ustate.CipherSuite,
		NegotiatedProtocol:          ustate.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  ustate.NegotiatedProtocolIsMutual,
		ServerName:                  ustate.ServerName,
		PeerCertificates:            ustate.PeerCertificates,
		VerifiedChains:              ustate.VerifiedChains,
		SignedCertificateTimestamps: ustate.SignedCertificateTimestamps,
		OCSPResponse:                ustate.OCSPResponse,
		TLSUnique:                   ustate.TLSUnique,
	}
}

// HandshakeContext implements TLSConn's HandshakeContext.
func (c *utlsConnAdapter) HandshakeContext(ctx context.Context) error {
	errch := make(chan error, 1)
	go func() {
		errch <- c.UConn.Handshake()
	}()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
