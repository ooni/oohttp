# github.com/ooni/oohttp

This repository contains a fork of Go's standard library `net/http`
package including patches to allow using this HTTP code with
[github.com/refraction-networking/utls](
https://github.com/refraction-networking/utls).

## Motivation and maintenance

We created this package because it simplifies testing URLs using
specific TLS Client Hello messages. We will continue to keep it up
to date as long as it serves our goals.

## Usage and limitations

To use this fork, replace

```Go
import "net/http"
```

with

```Go
import "github.com/ooni/oohttp"
```

Please, keep in mind the following limitations:

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

```
git remote add golang git@github.com:golang/go.git || git fetch golang
git branch -D golang-upstream golang-http-upstream
git checkout -b golang-upstream go1.16.7
git subtree split -P src/net/http/ -b golang-http-upstream
git checkout merged-main
git merge golang-http-upstream
git push merged-main
# then review and merge the PR creating a merge commit
```
