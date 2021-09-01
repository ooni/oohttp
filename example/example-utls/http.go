package main

import (
	"context"
	"log"
	"net"
	"time"

	oohttp "github.com/ooni/oohttp"
	utls "github.com/refraction-networking/utls"
)

// defaultDialer is the default Dialer.
var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// defaultTransport is the default http.RoundTripper.
var defaultTransport = &oohttp.StdlibTransport{
	Transport: &oohttp.Transport{
		Proxy:       oohttp.ProxyFromEnvironment,
		DialContext: defaultDialer.DialContext,
		DialTLSContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			conn, err := defaultDialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			sni, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			uconfig := &utls.Config{ServerName: sni}
			uconn := utls.UClient(conn, uconfig, utls.HelloFirefox_55)
			uadapter := &utlsConnAdapter{uconn}
			if err := uadapter.HandshakeContext(ctx); err != nil {
				conn.Close()
				return nil, err
			}
			proto := uadapter.ConnectionState().NegotiatedProtocol
			log.Printf("negotiated protocol: %s", proto)
			return uadapter, nil
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}
