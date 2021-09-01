package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"testing"

	oohttp "github.com/ooni/oohttp"
)

// tlsDialerRecorder performs TLS dials and records the ALPN.
type tlsDialerRecorder struct {
	alpn map[string]int
	mu   sync.Mutex
}

// do is like dialTLSContext but also records the ALPN.
func (d *tlsDialerRecorder) do(ctx context.Context, network string, addr string) (net.Conn, error) {
	conn, err := dialTLSContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	tconn := conn.(oohttp.TLSConn)
	p := tconn.ConnectionState().NegotiatedProtocol
	d.mu.Lock()
	if d.alpn == nil {
		d.alpn = make(map[string]int)
	}
	d.alpn[p]++
	d.mu.Unlock()
	return conn, nil
}

func TestWorkAsIntendedWithH2(t *testing.T) {
	d := &tlsDialerRecorder{}
	clnt := newClient(newTransport(d.do))
	resp, err := clnt.Get("https://www.facebook.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if count := d.alpn["h2"]; count < 1 {
		t.Fatal("did not dial h2")
	}
}

func TestWorkAsIntendedWithHTTP11(t *testing.T) {
	d := &tlsDialerRecorder{}
	clnt := newClient(newTransport(d.do))
	resp, err := clnt.Get("https://nexa.polito.it")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if count := d.alpn["http/1.1"]; count < 1 {
		t.Fatal("did not dial http/1.1")
	}
}

func TestWorkAsIntendedWithHTTP(t *testing.T) {
	resp, err := defaultClient.Get("http://example.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
}
