package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

// defaultClient is the default http.Client
var defaultClient = &http.Client{Transport: defaultTransport}

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
