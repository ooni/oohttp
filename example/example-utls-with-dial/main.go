package main

import (
	"context"
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
func newTransport(f func(ctx context.Context, network, address string) (net.Conn, error)) http.RoundTripper {
	return utlsx.NewOOHTTPTransport(
		oohttp.ProxyFromEnvironment,
		nil, // we're using f to directly create TLS connections
		f,
	)
}

// defaultTransport is the default http.RoundTripper.
var defaultTransport = newTransport((&utlsx.TLSDialer{}).DialTLSContext)

// newClient creates a new http.Client using the given transport.
func newClient(txp http.RoundTripper) *http.Client {
	return &http.Client{Transport: txp}
}

// defaultClient is the default http.Client
var defaultClient = newClient(defaultTransport)

func main() {
	// The default URL we use should return us the JA3 fingerprint
	// we're using to communicate with the server. We expect such a
	// fingerprint to change when we use the `-utls` flag.
	url := flag.String("url", "https://ja3er.com/json", "the URL to get")
	flag.Parse()
	resp, err := defaultClient.Get(*url)
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
