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
)

// newClient creates a new http.Client using the given transport.
func newClient(txp http.RoundTripper) *http.Client {
	return &http.Client{Transport: txp}
}

func main() {
	// Unfortunately ja3er.com is down. So we're now using as the default
	// destination a google website. Not as useful as ja3er.com.
	url := flag.String("url", "https://google.com/robots.txt", "the URL to get")
	utls := flag.Bool("utls", false, "try using uTLS")
	flag.Parse()
	var ffun func(conn net.Conn, config *tls.Config) oohttp.TLSConn
	if *utls {
		ffun = utlsFactory
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
