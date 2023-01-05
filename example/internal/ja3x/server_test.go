package ja3x

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_serverHandleCommon(t *testing.T) {
	t.Run("with unparseable ClientHello", func(t *testing.T) {
		clientHello := make([]byte, 155)
		err := serverHandleCommon(nil, clientHello)
		if err == nil || err.Error() != "handshake is of wrong type, or not a handshake message" {
			t.Fatal("unexpected error", err)
		}
	})
}

// serverHandleV1MockedConn is a mocked conn for testing [serverHandleV1]
type serverHandleV1MockedConn struct {
	// read is called when reading
	read func(data []byte) (int, error)

	// write is called when writing
	write func(data []byte) (int, error)
}

var _ io.ReadWriter = &serverHandleV1MockedConn{}

// Read implements io.ReadWriter
func (c *serverHandleV1MockedConn) Read(p []byte) (n int, err error) {
	return c.read(p)
}

// Write implements io.ReadWriter
func (c *serverHandleV1MockedConn) Write(p []byte) (n int, err error) {
	return c.write(p)
}

func Test_serverHandleV1(t *testing.T) {
	t.Run("cannot read request", func(t *testing.T) {
		expected := errors.New("mocked error")
		conn := &serverHandleV1MockedConn{
			read: func(data []byte) (int, error) {
				return 0, expected
			},
		}
		err := serverHandleV1(conn, "deadbeef")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("cannot write response", func(t *testing.T) {
		expected := errors.New("mocked error")
		reader := strings.NewReader("GET / HTTP/1.1\r\n\r\n")
		conn := &serverHandleV1MockedConn{
			read: reader.Read,
			write: func(data []byte) (int, error) {
				return 0, expected
			},
		}
		err := serverHandleV1(conn, "deadbeef")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestNewServer(t *testing.T) {
	t.Run("with no ALPN arguments", func(t *testing.T) {
		srvr := NewServer()
		defer srvr.Close()
		if diff := cmp.Diff(srvr.alpn, []string{"h2", "http/1.1"}); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("with specific ALPN", func(t *testing.T) {
		expected := []string{"h2"}
		srvr := NewServer("h2")
		defer srvr.Close()
		if diff := cmp.Diff(srvr.alpn, expected); diff != "" {
			t.Fatal(diff)
		}
	})
}
