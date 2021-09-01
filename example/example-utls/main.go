package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	url := flag.String("url", "", "The URL to retrieve")
	flag.Parse()
	clnt := &http.Client{Transport: defaultTransport}
	resp, err := clnt.Get(*url)
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
