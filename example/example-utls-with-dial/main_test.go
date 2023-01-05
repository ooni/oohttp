package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/oohttp/example/internal/ja3x"
	"github.com/ooni/oohttp/example/internal/utlsx"
)

// tlsDialerRecorder performs TLS dials and records the ALPN.
type tlsDialerRecorder struct {
	alpn   map[string]int
	config *tls.Config
	mu     sync.Mutex
}

// do is like dialTLSContext but also records the ALPN.
func (d *tlsDialerRecorder) do(ctx context.Context, network string, addr string) (net.Conn, error) {
	child := &utlsx.TLSDialer{
		Config: d.config,
	}
	conn, err := child.DialTLSContext(ctx, network, addr)
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
	srvr := ja3x.NewServer("h2")
	defer srvr.Close()
	d := &tlsDialerRecorder{
		alpn:   map[string]int{},
		config: srvr.ClientConfig(),
		mu:     sync.Mutex{},
	}
	clnt := newClient(newTransport(d.do))
	resp, err := clnt.Get(srvr.URL())
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
	srvr := ja3x.NewServer("http/1.1")
	defer srvr.Close()
	d := &tlsDialerRecorder{
		alpn:   map[string]int{},
		config: srvr.ClientConfig(),
		mu:     sync.Mutex{},
	}
	clnt := newClient(newTransport(d.do))
	resp, err := clnt.Get(srvr.URL())
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
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0xdeadbeef"))
	}))
	defer srvr.Close()
	resp, err := defaultClient.Get(srvr.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
}
