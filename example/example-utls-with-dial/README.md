# Example using refraction-networking/utls

This example shows how you can use this library to combine `net/http` and
`refraction-networking/utls` into the same product.

* [http.go](http.go) shows how to interface this library with `net/http` so
the rest of your codebase uses `net/http`;

* [tls.go](tls.go) shows how to wrap `utls.UConn` to be compatible with
the `TLSConn` interface defined by this library;

* [main.go](main.go) is the rest of the codebase using `net/http`.

The strategy we're using here is to override `DialTLSContext` to force
using our own TLS dialer that uses uTLS (this is what OONI does).
