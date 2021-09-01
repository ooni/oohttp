package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

// newClient creates a new http.Client using the given transport.
func newClient(txp http.RoundTripper) *http.Client {
	return &http.Client{Transport: txp}
}

// defaultClient is the default http.Client
var defaultClient = newClient(defaultTransport)

func main() {
	url := flag.String("url", "", "The URL to retrieve")
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
	fmt.Printf("body length: %d bytes\n", len(data))
}
