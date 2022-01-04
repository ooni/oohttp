package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"net"
	"time"

	"github.com/armon/go-socks5"
	oohttp "github.com/ooni/oohttp"
)

// startProxyServer starts a SOCKS5 proxy server at the given endpoint.
func startProxyServer(endpoint string, ch chan<- interface{}) {
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}
	close(ch) // signal the main goroutine it can now continue
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

// useProxy uses the oohttp library to possibly use refraction-networking/utls
// when communicating with a remote TLS endpoint through the given proxy.
func useProxy(URL, proxy string) {
	w := &oohttp.StdlibTransport{
		Transport: &oohttp.Transport{
			Proxy: func(*oohttp.Request) (*url.URL, error) {
				return &url.URL{Scheme: "socks5", Host: proxy}, nil
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	clnt := &http.Client{Transport: w}
	resp, err := clnt.Get(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	r := io.LimitReader(resp.Body, 1<<22)
	data, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(data))
}

func main() {
	proxy := flag.String("proxy", "127.0.0.1:54321", "where the proxy shold listen")
	// The default URL we use should return us the JA3 fingerprint
	// we're using to communicate with the server. We expect such a
	// fingerprint to change when we use the `-utls` flag.
	url := flag.String("url", "https://ja3er.com/json", "the URL to get")
	utls := flag.Bool("utls", false, "try using uTLS")
	flag.Parse()
	if *utls {
		useUTLS()
	}
	ch := make(chan interface {})
	go startProxyServer(*proxy, ch)
	<-ch // wait for the listener to be listening
	useProxy(*url, *proxy)
}
