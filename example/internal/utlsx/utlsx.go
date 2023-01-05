// Package utlsx contains UTLS extensions.
package utlsx

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/url"
	"time"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/oohttp/example/internal/runtimex"
	utls "github.com/refraction-networking/utls"
)

// connAdapter adapts utls.UConn to the oohttp.TLSConn interface.
type connAdapter struct {
	*utls.UConn
}

var _ oohttp.TLSConn = &connAdapter{}

// ConnectionState implements oohttp.TLSConn.
func (c *connAdapter) ConnectionState() tls.ConnectionState {
	cs := c.UConn.ConnectionState()
	return tls.ConnectionState{
		Version:                     cs.Version,
		HandshakeComplete:           cs.HandshakeComplete,
		DidResume:                   cs.DidResume,
		CipherSuite:                 cs.CipherSuite,
		NegotiatedProtocol:          cs.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  cs.NegotiatedProtocolIsMutual,
		ServerName:                  cs.ServerName,
		PeerCertificates:            cs.PeerCertificates,
		VerifiedChains:              cs.VerifiedChains,
		SignedCertificateTimestamps: cs.SignedCertificateTimestamps,
		OCSPResponse:                cs.OCSPResponse,
		TLSUnique:                   cs.TLSUnique,
	}
}

// HandshakeContext implements oohttp.TLSConn.
func (c *connAdapter) HandshakeContext(ctx context.Context) error {
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

// DefaultClientHelloID is the default [utls.ClientHelloID].
var DefaultClientHelloID = &utls.HelloFirefox_55

// ConnFactory is a factory for creating UTLS connections.
type ConnFactory interface {
	// NewUTLSConn creates a new UTLS connection. The conn and config arguments MUST
	// NOT be nil. We will honour the following fields of config:
	//
	// - RootCAs
	//
	// - NextProtos
	//
	// - ServerName
	//
	// - InsecureSkipVerify
	//
	// - DynamicRecordSizingDisabled
	//
	// However, some of these fields values MAY be ignored depending on the parrot we use.
	NewUTLSConn(conn net.Conn, config *tls.Config) oohttp.TLSConn
}

// FactoryWithParrot implements ConnFactory.
type FactoryWithParrot struct {
	// Parrot is the OPTIONAL parrot.
	Parrot *utls.ClientHelloID
}

// NewUTLSConn implements ConnFactory.
func (f *FactoryWithParrot) NewUTLSConn(conn net.Conn, config *tls.Config) oohttp.TLSConn {
	parrot := f.Parrot
	if parrot == nil {
		parrot = DefaultClientHelloID
	}
	uConfig := &utls.Config{
		RootCAs:                     config.RootCAs,
		NextProtos:                  config.NextProtos,
		ServerName:                  config.ServerName,
		InsecureSkipVerify:          config.InsecureSkipVerify,
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
	}
	return &connAdapter{utls.UClient(conn, uConfig, *parrot)}
}

// DefaultNetDialer is the default [net.Dialer].
var DefaultNetDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// TLSDialer is a dialer that uses UTLS
type TLSDialer struct {
	// Config is the OPTIONAL Config. In case it's not nil, we will
	// pass this config to [Factory] rather than a default one.
	Config *tls.Config

	// Parrot is the OPTIONAL parrot to use.
	Parrot *utls.ClientHelloID

	// beforeHandshakeFunc is a function called before the
	// TLS handshake, which is only useful for testing.
	beforeHandshakeFunc func()
}

// DialTLSContext dials a TLS connection using UTLS.
func (d *TLSDialer) DialTLSContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	conn, err := DefaultNetDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	sni, _, err := net.SplitHostPort(addr)
	runtimex.PanicOnError(err, "net.SplitHostPort failed") // cannot fail after successful dial
	config := &tls.Config{ServerName: sni}
	if d.Config != nil {
		config = d.Config // as documented
	}
	if d.beforeHandshakeFunc != nil {
		d.beforeHandshakeFunc() // useful for testing
	}
	uadapter := (&FactoryWithParrot{d.Parrot}).NewUTLSConn(conn, config)
	if err := uadapter.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	proto := uadapter.ConnectionState().NegotiatedProtocol
	log.Printf("negotiated protocol: %s", proto)
	return uadapter, nil
}

// TLSFactoryFunc is the type of the [ConnFactory.NewUTLSConn] func.
type TLSFactoryFunc func(conn net.Conn, config *tls.Config) oohttp.TLSConn

// ProxyFunc is the the type of the [http.ProxyFromEnvironment] func.
type ProxyFunc func(*oohttp.Request) (*url.URL, error)

// TLSDialFunc is the type of the [TLSDialer.DialTLSContext] func.
type TLSDialFunc func(ctx context.Context, network string, addr string) (net.Conn, error)

// NewOOHTTPTransport creates a new OOHTTP transport using the given funcs. Each
// function MAY be nil. In such a case, we'll use a reasonable default.
func NewOOHTTPTransport(
	proxy ProxyFunc, tlsFactory TLSFactoryFunc, tlsDialer TLSDialFunc) *oohttp.StdlibTransport {
	return &oohttp.StdlibTransport{
		Transport: &oohttp.Transport{
			Proxy:                 proxy,
			DialContext:           DefaultNetDialer.DialContext,
			DialTLSContext:        tlsDialer,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientFactory:      tlsFactory,
		},
	}
}
