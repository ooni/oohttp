package ja3x

import (
	"errors"
	"testing"
)

func Test_panicOnError(t *testing.T) {
	t.Run("calls panic on error", func(t *testing.T) {
		expected := errors.New("mocked error")
		errch := make(chan error)
		go func() {
			defer func() {
				errch <- recover().(error)
			}()
			panicOnError(expected)
		}()
		err := <-errch
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})
}

func Test_clientHelloDigest(t *testing.T) {
	t.Run("unmarshal fails for invalid ClientHello", func(t *testing.T) {
		data := make([]byte, 128)
		out, err := clientHelloDigest(data)
		if err == nil || err.Error() != "handshake is of wrong type, or not a handshake message" {
			t.Fatal("unexpected error", err)
		}
		if out != "" {
			t.Fatal("expected empty string")
		}
	})
}
