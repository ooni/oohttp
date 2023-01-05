package ja3x

import "testing"

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
