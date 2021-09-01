package main

import (
	"io"
	"log"
	"testing"
)

func TestWorkAsIntendedWithH2(t *testing.T) {
	resp, err := defaultClient.Get("https://www.facebook.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
}

func TestWorkAsIntendedWithHTTP11(t *testing.T) {
	resp, err := defaultClient.Get("https://nexa.polito.it")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
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
