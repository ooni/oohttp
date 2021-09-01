# github.com/ooni/oohttp

This repository contains a fork of Go's standard library `net/http`
package including patches to allow using this HTTP code with
[github.com/refraction-networking/utls](
https://github.com/refraction-networking/utls).

## Motivation and maintenance

We created this package [because it simplifies testing URLs using
specific TLS Client Hello messages](https://github.com/ooni/probe/issues/1731). We
will continue to keep it up to date as long as it serves our goals.

## Limitations

1. This fork does not include a fork of `pprof` because such package
depends on the stdlib's `internal/profile` package. If your code uses
`http/pprof`, then you cannot switch to this fork.

2. This fork's `httptrace` package is partly broken because there
is no support for network events tracing, which requires the stdlib's
`internal/nettrace` package. If your code depends on network events
tracing, then you cannot switch to this fork.

3. This fork tracks the latest stable version of Go by merging
upstream changes into the `main` branch. This means that it _may_
not be working with earlier versions of Go. For example, when
writing this note we are at Go 1.16 and this package accordingly
uses `io.ReadAll`. If you are compiling using Go 1.15, you should
get build errors because `io.ReadAll` did not exist before Go 1.16.

## Usage

The follow diagram shows your typical app architecture when you're
using this library as an alternative HTTP library.

![architecture](oohttp.png)

From the diagram, it stems that we need to discuss two interfaces:

1. the interface between your code and this library;

2. the interface between this library and a TLS library.

### Interface between your code and this library

The simplest approach is to just replace

```Go
import "net/http"
```

with

```Go
import "github.com/ooni/oohttp"
```

everywhere in your codebase.

This approach is not practical when your code or a dependency of yours
already assumes `net/http`. In such a case, use
[stdlibwrapper.go](stdlibwrapper.go),
which provides you with an adapter implementing `net/http.Transport`. It
takes the stdlib's `net/http.Request` as input and returns the stdlib's
`net/http.Response` as output. But, internally, it uses the `Transport` defined
by this library, that is capable to interface with any TLS library
implementing the [TLSConn](tlsconn.go) interface:

```Go
// StdlibTransport is an adapter for integrating net/http dependend code.
// It looks like an http.RoundTripper but uses this fork internally.
type StdlibTransport struct {
	*Transport
}

// RoundTrip implements the http.RoundTripper interface.
func (txp *StdlibTransport) RoundTrip(stdReq *http.Request) (*http.Response, error) {
	// ...
}
```

See [example/example-utls/http.go](example/example-utls/http.go) for a real
world example where we use `StdlibTransport` to be `net/http` compatible.

### Interface between this library and any TLS library

You need to write a wrapper for your definition of the TLS connection that
implements the [TLSConn](tlsconn.go) interface:

```Go
// TLSConn is the interface representing a *tls.Conn compatible
// connection, which could possibly be different from a *tls.Conn
// as long as it implements the interface. You can use, for
// example, refraction-networking/utls instead of the stdlib.
type TLSConn interface {
	// net.Conn is the underlying interface
	net.Conn

	// ConnectionState returns the ConnectionState according
	// to the standard library.
	ConnectionState() tls.ConnectionState

	// HandshakeContext performs an TLS handshake bounded
	// in time by the given context.
	HandshakeContext(ctx context.Context) error
}
```

If you are using `crypto/tls` (which does not make much sense), then
your `tls.Conn` is already a valid `TLSConn`.

If you are using `refraction-networking/utls`, you need to write an
adapter. Your TLS connection is
already a `net.Conn`. But you need to implement `ConnectionState`. And
you also need to implement `HandshakeContext`.

The following code shows, for reference, how we initially implemented
this functionality in [ooni/probe-cli](https://github.com/ooni/probe-cli):

```Go
// uconn is an adapter from utls.UConn to TLSConn.
type uconn struct {
	*utls.UConn
}

// ConnectionState implements TLSConn's ConnectionState.
func (c *uconn) ConnectionState() tls.ConnectionState {
	ustate := c.UConn.ConnectionState()
	return tls.ConnectionState{
		Version:                     ustate.Version,
		HandshakeComplete:           ustate.HandshakeComplete,
		DidResume:                   ustate.DidResume,
		CipherSuite:                 ustate.CipherSuite,
		NegotiatedProtocol:          ustate.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  ustate.NegotiatedProtocolIsMutual,
		ServerName:                  ustate.ServerName,
		PeerCertificates:            ustate.PeerCertificates,
		VerifiedChains:              ustate.VerifiedChains,
		SignedCertificateTimestamps: ustate.SignedCertificateTimestamps,
		OCSPResponse:                ustate.OCSPResponse,
		TLSUnique:                   ustate.TLSUnique,
	}
}

// HandshakeContext implements TLSConn's HandshakeContext.
func (c *uconn) HandshakeContext(ctx context.Context) error {
	errch := make(chan error, 1)
	go func() {
		errch <- c.UConn.Handshake()
	}()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

See [example/example-utls/tls.go](example/example-utls/tls.go) for a real-world
example of writing a `TLSConn` compatible adapter.

## Issue tracker

Please, report issues in the [ooni/probe](https://github.com/ooni/probe)
repository. Make sure you mention `oohttp` in the issue title.

## Patches

We started from the `src/net/http` subtree at `go1.16` and we
applied patches to fork the codebase ([#1](https://github.com/ooni/oohttp/pull/1),
[#2](https://github.com/ooni/oohttp/pull/2) and [#3](
https://github.com/ooni/oohttp/pull/3)). Then, we introduced
the `http.TLSConn` abstraction that allows using different TLS
libraries ([#4](https://github.com/ooni/oohttp/pull/4)).

Every major change is documented by a pull request. We may push
minor changes (e.g., updating docs) directly on the `main` branch.

## Update procedure

(Adapted from refraction-networking/utls instructions.)

1. run the following commands:

```
git remote add golang git@github.com:golang/go.git || git fetch golang
git branch -D golang-upstream golang-http-upstream
git checkout -b golang-upstream go1.16.7
git subtree split -P src/net/http/ -b golang-http-upstream
git checkout merged-main
git merge golang-http-upstream
git push merged-main
```

2. make sure the codebase does not assume `*net.Conn` anywhere (every
merge may introduce new changes);

3. open a pull request on github and merge it with merge commit.
