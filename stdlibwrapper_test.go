package http_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	oohttp "github.com/ooni/oohttp"
)

func TestStdlibWrapper(t *testing.T) {
	t.Run("we propagate the context correctly", func(t *testing.T) {
		var txp http.RoundTripper = &oohttp.StdlibTransport{
			Transport: &oohttp.Transport{},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we expect the request with context to fail immediately
		req, err := http.NewRequestWithContext(
			ctx, "GET", "https://google.com", nil)
		if err != nil {
			t.Fatal(err) // should not fail
		}
		resp, err := txp.RoundTrip(req)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("unexpected resp")
		}
	})
}
