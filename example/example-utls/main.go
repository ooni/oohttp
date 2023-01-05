package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/oohttp/example/internal/utlsx"
)

// newTransport returns a new http.Transport using the provided tls dialer.
func newTransport(f func(conn net.Conn, config *tls.Config) oohttp.TLSConn) http.RoundTripper {
	return utlsx.NewOOHTTPTransport(
		oohttp.ProxyFromEnvironment,
		f,
		nil, // we're using f to wrap dialed connections
	)
}

// newClient creates a new http.Client using the given transport.
func newClient(txp http.RoundTripper) *http.Client {
	return &http.Client{Transport: txp}
}

func main() {
	// The default URL we use should return us the JA3 fingerprint
	// we're using to communicate with the server. We expect such a
	// fingerprint to change when we use the `-utls` flag.
	url := flag.String("url", "https://ja3er.com/json", "the URL to get")
	utls := flag.Bool("utls", false, "try using uTLS")
	flag.Parse()
	var ffun func(conn net.Conn, config *tls.Config) oohttp.TLSConn
	if *utls {
		ffun = (&utlsx.FactoryWithParrot{}).NewUTLSConn
	}
	resp, err := newClient(newTransport(ffun)).Get(*url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}
