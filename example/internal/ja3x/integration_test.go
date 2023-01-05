package ja3x

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/http2"
)

func TestServerWithSpecificALPN(t *testing.T) {
	srvr := NewServer("antani")
	defer srvr.Close()
	tlsConfig := srvr.ClientConfig()
	tlsConfig.NextProtos = []string{"antani"}
	conn, err := tls.Dial("tcp", srvr.Endpoint(), tlsConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if p := conn.ConnectionState().NegotiatedProtocol; p != "antani" {
		t.Fatal("unexpected negotiated protocol", p)
	}
}

func TestServerWithHTTP1ClientAndTLS(t *testing.T) {
	srvr := NewServer()
	defer srvr.Close()
	tlsConfig := srvr.ClientConfig()
	tlsConfig.NextProtos = []string{"http/1.1"}
	txp := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: false, // explicitly say no to HTTP2
	}
	req, err := http.NewRequest("GET", srvr.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal("unexpected status code", resp.StatusCode)
	}
	defer resp.Body.Close()
	if value := resp.Header.Get("X-ALPN"); value != "http/1.1" {
		t.Fatal("unexpected X-ALPN header value", value)
	}
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(respbody) != 32 {
		t.Fatal("not a valid JA3 response length")
	}
}

func TestServerWithHTTP2ClientAndTLS(t *testing.T) {
	srvr := NewServer()
	defer srvr.Close()
	tlsConfig := srvr.ClientConfig()
	tlsConfig.NextProtos = []string{"h2"}
	txp := &http2.Transport{
		TLSClientConfig: tlsConfig,
	}
	req, err := http.NewRequest("GET", srvr.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal("unexpected status code", resp.StatusCode)
	}
	defer resp.Body.Close()
	if value := resp.Header.Get("X-ALPN"); value != "h2" {
		t.Fatal("unexpected X-ALPN header value", value)
	}
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(respbody) != 32 {
		t.Fatal("not a valid JA3 response length")
	}
}

func TestServerWithHTTP1ClientAndTCP(t *testing.T) {
	srvr := NewServer()
	defer srvr.Close()
	parsed, err := url.Parse(srvr.URL())
	if err != nil {
		t.Fatal(err)
	}
	parsed.Scheme = "http" // forces using the cleartext HTTP protocol
	txp := &http.Transport{}
	req, err := http.NewRequest("GET", parsed.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, io.EOF) {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}

func TestServerWithEmptyALPN(t *testing.T) {
	srvr := NewServer()
	defer srvr.Close()
	tlsConfig := srvr.ClientConfig()
	tlsConfig.NextProtos = []string{} // explicitly empty so it will fail
	txp := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: false, // explicitly say no to HTTP2
	}
	req, err := http.NewRequest("GET", srvr.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, io.EOF) {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}

func TestServerWithInvalidALPN(t *testing.T) {
	srvr := NewServer()
	defer srvr.Close()
	tlsConfig := srvr.ClientConfig()
	tlsConfig.NextProtos = []string{"antani"} // explicitly invalid so it will fail
	txp := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: false, // explicitly say no to HTTP2
	}
	req, err := http.NewRequest("GET", srvr.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err == nil || !strings.HasSuffix(err.Error(), "tls: no application protocol") {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}
