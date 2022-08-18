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
	// Unfortunately ja3er.com is down. So we're now using as the default
	// destination a google website. Not as useful as ja3er.com.
	url := flag.String("url", "https://google.com/robots.txt", "the URL to get")
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
