# Example of using TLSClientFactory

This example shows how you can use `Transport.TLSClientFactory` to force this
library to use `refraction-networking/utls` for proxied conns.

* [main.go](main.go) sets up an `oohttp.Transport` instance, uses the `Proxy` field
to configure a SOCKS5 proxy, and the `TLSClientFactory` to use uTLS when needed;

You should not change the `Transport.TLSClientFactory` while the transport
is being used. Doing that is likely to cause data races.

An alternative strategy could be to set the global `oohttp.TLSClientFactory`
field, which will cause all connections to use such a factory. If you
choose to override the global factory (as opposed to the per-Transport one), you
should do this only once before calling any HTTP code. Changing the global
factory while HTTP code is running causes data races.
