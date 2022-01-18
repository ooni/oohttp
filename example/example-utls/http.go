package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	oohttp "github.com/ooni/oohttp"
)

// newTransport returns a new http.Transport using the provided tls dialer.
func newTransport(f func(conn net.Conn, config *tls.Config) oohttp.TLSConn) http.RoundTripper {
	return &oohttp.StdlibTransport{
		Transport: &oohttp.Transport{
			Proxy:                 oohttp.ProxyFromEnvironment,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientFactory:      f,
		},
	}
}
