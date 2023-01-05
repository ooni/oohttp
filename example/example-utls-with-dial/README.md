# Example using refraction-networking/utls

This example shows how you can use this library to combine `net/http` and
`refraction-networking/utls` into the same product.

The strategy we're using here is to override `DialTLSContext` to force
using our own TLS dialer that uses uTLS (this is what OONI does).
