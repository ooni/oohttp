package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	oohttp "github.com/ooni/oohttp"
	utls "github.com/refraction-networking/utls"
)

// defaultDialer is the default Dialer.
var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// tlsDialer is a dialer that uses TLS
type tlsDialer struct {
	// config is the optional config. In case it's not nil, we will
	// use its RootCAs value, if any, and we'll override the default
	// ServerName with its ServerName value, if any.
	config *tls.Config
}

// dial dials a TLS connection using refraction-networking/utls
// and returns a TLSConn compatible net.Conn.
func (d *tlsDialer) dial(ctx context.Context, network string, addr string) (net.Conn, error) {
	conn, err := defaultDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	sni, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	uconfig := &utls.Config{ServerName: sni}
	if d.config != nil {
		if d.config.RootCAs != nil {
			uconfig.RootCAs = d.config.RootCAs // as documented
		}
		if d.config.ServerName != "" {
			uconfig.ServerName = d.config.ServerName // as documented
		}
	}
	// Implementation note: using Firefox 55 ClientHello because that
	// avoids a bunch of issues, e.g., Brotli encrypted x509 certs.
	uconn := utls.UClient(conn, uconfig, utls.HelloFirefox_55)
	uadapter := &utlsConnAdapter{uconn}
	if err := uadapter.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	proto := uadapter.ConnectionState().NegotiatedProtocol
	log.Printf("negotiated protocol: %s", proto)
	return uadapter, nil
}

// newTransport returns a new http.Transport using the provided tls dialer.
func newTransport(f func(ctx context.Context, network, address string) (net.Conn, error)) http.RoundTripper {
	return &oohttp.StdlibTransport{
		Transport: &oohttp.Transport{
			Proxy:                 oohttp.ProxyFromEnvironment,
			DialContext:           defaultDialer.DialContext,
			DialTLSContext:        f,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// defaultTransport is the default http.RoundTripper.
var defaultTransport = newTransport((&tlsDialer{}).dial)
