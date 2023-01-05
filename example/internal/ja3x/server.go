package ja3x

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/martian/v3/mitm"
	"github.com/ooni/oohttp/example/internal/runtimex"
	"golang.org/x/net/http2"
)

// serverTLSConfigMITM creates a MITM TLS configuration.
func serverTLSConfigMITM() (*x509.Certificate, *rsa.PrivateKey, *mitm.Config) {
	cert, privkey, err := mitm.NewAuthority("ja3x", "OONI", time.Hour)
	runtimex.PanicOnError(err, "mitm.NewAuthority failed")
	config, err := mitm.NewConfig(cert, privkey)
	runtimex.PanicOnError(err, "mitm.NewConfig failed")
	return cert, privkey, config
}

// serverConfigALPN configures the ALPN or a default ALPN.
func serverConfigALPN(alpn []string) []string {
	if len(alpn) > 0 {
		return alpn
	}
	return []string{"h2", "http/1.1"}
}

// NewServer creates a new [Server] instance.
func NewServer(alpn ...string) *Server {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	runtimex.PanicOnError(err, "net.Listen failed")
	cert, privkey, config := serverTLSConfigMITM()
	ctx, cancel := context.WithCancel(context.Background())
	endpoint := listener.Addr().String()
	server := &Server{
		alpn:     serverConfigALPN(alpn),
		cancel:   cancel,
		cert:     cert,
		config:   config,
		done:     make(chan bool),
		endpoint: endpoint,
		listener: listener,
		privkey:  privkey,
		once:     sync.Once{},
	}
	go server.mainloop(ctx)
	return server
}

// Server is an HTTP server that returns JA3 information. This server
// only supports HTTP/1.1 and HTTP/2.
//
// This server will use its own internal CA. You should use [Server.ClientConfig]
// to obtain a suitable [tls.Config] or [Server.CertPool] to obtain a suitable
// [x509.CertPool]. If you use [Server.CertPool] keep in mind that you also need
// to specify a ServerName in the [tls.Config] for an handshake to complete
// successfully. By default [Server.Config] uses "example.com" as the server
// name but you can override that.
//
// Please, use the [NewServer] factory to create a new instance: the
// zero-value [Server] is invalid and will panic if used.
type Server struct {
	alpn     []string
	cancel   context.CancelFunc
	cert     *x509.Certificate
	config   *mitm.Config
	done     chan bool
	endpoint string
	listener net.Listener
	once     sync.Once
	privkey  *rsa.PrivateKey
}

// Endpoint returns the server's TCP endpoint.
func (p *Server) Endpoint() string {
	return p.endpoint
}

// CertPool returns the internal CA as a cert pool.
func (p *Server) CertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(p.cert)
	return pool
}

// ClientConfig returns a [tls.Config] where the ServerName
// is set to "example.com" and the RootCAs is set to the value returned by
// the [Server.CertPool] method.
func (p *Server) ClientConfig() *tls.Config {
	return &tls.Config{
		ServerName: "example.com",
		RootCAs:    p.CertPool(),
	}
}

// Close terminates a running [Server] instance.
func (p *Server) Close() (err error) {
	p.once.Do(func() {
		p.cancel()
		err = p.listener.Close()
		<-p.done
	})
	return err
}

// URL returns the server's URL, which is guaranteed to contain
// an explicit port even in the very unlikely case in which
// the randomly selected port is 443/tcp.
func (p *Server) URL() string {
	u := &url.URL{
		Scheme: "https",
		Host:   p.endpoint,
		Path:   "/",
	}
	return u.String()
}

// mainloop is the main loop of a [Server].
func (p *Server) mainloop(ctx context.Context) {
	defer close(p.done)
	for {
		canContinue, _ := p.serveConn(ctx)
		if !canContinue {
			break
		}
	}
}

// serveConn accepts and services a client connection
func (p *Server) serveConn(ctx context.Context) (bool, error) {
	conn, err := p.listener.Accept()
	if err != nil {
		return !errors.Is(err, net.ErrClosed), err
	}
	done := make(chan bool)
	go func() {
		// make sure we close the connection both in case of
		// normal shutdown and when the ctx is done
		defer conn.Close()
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}()
	err = p.setupTLSAndHandle(conn)
	close(done)
	return true, err // we can continue running
}

// errInvalidALPN indicates we were passed an invalid ALPN.
var errInvalidALPN = errors.New("ja3x: invalid ALPN")

// setupTLSAndHandle setups a TLS connection and handles the request.
func (p *Server) setupTLSAndHandle(tcpConn net.Conn) error {
	cw := newConnWrapper(tcpConn)
	tlsConn := tls.Server(cw, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			sni, alpn := info.ServerName, info.SupportedProtos
			log.Printf("ja3x: got connection request for %s %s", sni, alpn)
			return p.config.TLSForHost(sni).GetCertificate(info)
		},
		NextProtos: p.alpn,
	})
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("ja3x: unexpected TLS handshake failure: %v", err)
		return err
	}
	hello := cw.Hello()
	return serverHandleCommon(tlsConn, hello)
}

func serverHandleCommon(tlsConn *tls.Conn, hello []byte) error {
	digest, err := clientHelloDigest(hello)
	if err != nil {
		log.Printf("ja3x: cannot parse client hello: %v", err)
		return err
	}
	switch alpn := tlsConn.ConnectionState().NegotiatedProtocol; alpn {
	case "http/1.1":
		return serverHandleV1(tlsConn, digest)
	case "h2":
		return serverHandleV2(tlsConn, digest)
	default:
		log.Printf("ja3x: got invalid ALPN: '%s'", alpn)
		return errInvalidALPN
	}
}

// serverHandleV1 handles an HTTP/1.1 client request.
func serverHandleV1(conn io.ReadWriter, digest string) error {
	rbio := bufio.NewReader(conn)
	req, err := http.ReadRequest(rbio)
	if err != nil {
		log.Printf("ja3x: cannot read request: %v", err)
		return err
	}
	// TODO(bassosimone): read the request body when it becomes useful to do so
	resp := &http.Response{
		StatusCode: 200,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: map[string][]string{
			"X-ALPN": {"http/1.1"}, // allows to easily verify we're using http/1.1
		},
		Body:    io.NopCloser(strings.NewReader(digest)),
		Request: req,
	}
	if err := resp.Write(conn); err != nil {
		log.Printf("ja3x: cannot write response: %s", err)
		return err
	}
	return nil
}

// serverHandleV2 handles an HTTP/2 client request.
func serverHandleV2(conn net.Conn, digest string) error {
	options := &http2.ServeConnOpts{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-ALPN", "h2") // allows to easily verify we're using HTTP/2
			w.Write([]byte(digest))
		}),
	}
	srv := &http2.Server{}
	srv.ServeConn(conn, options)
	return nil
}
