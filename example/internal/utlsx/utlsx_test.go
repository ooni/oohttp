package utlsx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/oohttp/example/internal/ja3x"
	utls "github.com/refraction-networking/utls"
)

func TestTLSDialerWorkingAsIntended(t *testing.T) {
	srvr := ja3x.NewServer("h2", "http/1.1")
	defer srvr.Close()
	dialer := &TLSDialer{
		Config:              srvr.ClientConfig(),
		Parrot:              &utls.HelloFirefox_105,
		beforeHandshakeFunc: nil,
	}
	ctx := context.Background()
	conn, err := dialer.DialTLSContext(ctx, "tcp", srvr.Endpoint())
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestTLSDialerWithHandshakeError(t *testing.T) {
	srvr := ja3x.NewServer("h2", "http/1.1")
	defer srvr.Close()
	ctx, cancel := context.WithCancel(context.Background())
	dialer := &TLSDialer{
		Config: srvr.ClientConfig(),
		Parrot: &utls.HelloFirefox_105,
		beforeHandshakeFunc: func() {
			cancel() // make sure the handshake fails
		},
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", srvr.Endpoint())
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestTLSDialerWithDialError(t *testing.T) {
	srvr := ja3x.NewServer("h2", "http/1.1")
	defer srvr.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause immediate failure
	dialer := &TLSDialer{
		Config:              srvr.ClientConfig(),
		Parrot:              &utls.HelloFirefox_105,
		beforeHandshakeFunc: nil,
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", srvr.Endpoint())
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("unexpected err", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestNewOOHTTPTransportWithCustomProxy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	acceptch := make(chan error)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptch <- err
			return
		}
		conn.Close()
		acceptch <- nil
	}()
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
	}
	txp := NewOOHTTPTransport(
		func(r *oohttp.Request) (*url.URL, error) {
			return proxyURL, nil
		},
		nil, // use default
		nil, // use default
	)
	req, err := http.NewRequest("GET", "https://google.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("unexpected err", err)
	}
	if resp != nil {
		t.Fatal("expected nil resp")
	}
	if err := <-acceptch; err != nil {
		t.Fatal("accept failed")
	}
}

const (
	// defaultJA3 is the default JA3
	defaultJA3 = "0ffee3ba8e615ad22535e7f771690a28"

	// customJA3 is the custom JA3
	customJA3 = "579ccef312d18482fc42e2b822ca2430"
)

func TestJA3SignaturesDifference(t *testing.T) {
	if defaultJA3 == customJA3 {
		t.Fatal("the two JA3 signatures must be different")
	}
}

func TestNewOOHTTPTransportWithTLSFactoryFunc(t *testing.T) {
	// doit is the common function implementing all of this test's test cases.
	doit := func(parrot *utls.ClientHelloID, expectedJA3 string, alpn ...string) error {
		srvr := ja3x.NewServer(alpn...)
		defer srvr.Close()
		factory := &FactoryWithParrot{
			Parrot: parrot, // may or may not be nil
		}
		txp := NewOOHTTPTransport(
			nil, // default proxy function
			factory.NewUTLSConn,
			nil, // default TLS dialing
		)
		txp.Transport.TLSClientConfig = srvr.ClientConfig()
		txp.Transport.TLSClientConfig.NextProtos = alpn
		req, err := http.NewRequest("GET", srvr.URL(), nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := txp.RoundTrip(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		switch {
		case len(alpn) <= 0:
			return errors.New("expected at least one ALPN")
		case alpn[0] == "h2" && resp.Header.Get("X-ALPN") != "h2":
			return fmt.Errorf("expected ALPN h2 but got %s", resp.Header.Get("X-ALPN"))
		case alpn[0] == "http/1.1" && resp.Header.Get("X-ALPN") != "http/1.1":
			return fmt.Errorf("expected ALPN http/1.1 but got %s", resp.Header.Get("X-ALPN"))
		default:
			// all seems good in ALPN terms
		}
		if got := string(data); got != expectedJA3 {
			return fmt.Errorf("expected JA3 %s but got %s", expectedJA3, got)
		}
		return nil
	}

	t.Run("with http/1.1", func(t *testing.T) {
		t.Run("with default parrot", func(t *testing.T) {
			err := doit(nil, defaultJA3, "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with custom parrot", func(t *testing.T) {
			err := doit(&utls.HelloFirefox_105, customJA3, "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("with h2 and http/1.1", func(t *testing.T) {
		t.Run("with default parrot", func(t *testing.T) {
			err := doit(nil, defaultJA3, "h2", "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with custom parrot", func(t *testing.T) {
			err := doit(&utls.HelloFirefox_105, customJA3, "h2", "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})
	})
}

func TestNewOOHTTPTransportWithTLSDialerFunc(t *testing.T) {
	// doit is the common function implementing all of this test's test cases.
	doit := func(parrot *utls.ClientHelloID, expectedJA3 string, alpn ...string) error {
		srvr := ja3x.NewServer(alpn...)
		defer srvr.Close()
		dialer := &TLSDialer{
			Config:              srvr.ClientConfig(),
			Parrot:              parrot,
			beforeHandshakeFunc: nil,
		}
		dialer.Config.NextProtos = alpn
		txp := NewOOHTTPTransport(
			nil,                   // default proxy function
			nil,                   // no factory
			dialer.DialTLSContext, // but use a custom dialer
		)
		req, err := http.NewRequest("GET", srvr.URL(), nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := txp.RoundTrip(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		switch {
		case len(alpn) <= 0:
			return errors.New("expected at least one ALPN")
		case alpn[0] == "h2" && resp.Header.Get("X-ALPN") != "h2":
			return fmt.Errorf("expected ALPN h2 but got %s", resp.Header.Get("X-ALPN"))
		case alpn[0] == "http/1.1" && resp.Header.Get("X-ALPN") != "http/1.1":
			return fmt.Errorf("expected ALPN http/1.1 but got %s", resp.Header.Get("X-ALPN"))
		default:
			// all seems good in ALPN terms
		}
		if got := string(data); got != expectedJA3 {
			return fmt.Errorf("expected JA3 %s but got %s", expectedJA3, got)
		}
		return nil
	}

	t.Run("with http/1.1", func(t *testing.T) {
		t.Run("with default parrot", func(t *testing.T) {
			err := doit(nil, defaultJA3, "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with custom parrot", func(t *testing.T) {
			err := doit(&utls.HelloFirefox_105, customJA3, "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("with h2 and http/1.1", func(t *testing.T) {
		t.Run("with default parrot", func(t *testing.T) {
			err := doit(nil, defaultJA3, "h2", "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with custom parrot", func(t *testing.T) {
			err := doit(&utls.HelloFirefox_105, customJA3, "h2", "http/1.1")
			if err != nil {
				t.Fatal(err)
			}
		})
	})
}
