# Example of using TLSClientFactory

This example shows how you can use `TLSClientFactory` to force this
library to use `refraction-networking/utls` for proxied conns.

* [main.go](main.go) sets up an `oohttp.Transport` instance and uses
the `Proxy` field to configure a SOCKS5 proxy;

* [utls.go](utls.go) contains the code to replace `tls` with `utls`
when using a proxy through the `TLSClientFactory`.
