# github.com/ooni/oohttp

This repository contains a fork of Go's standard library `net/http`
package including patches to allow using this HTTP code with
[github.com/refraction-networking/utls](
https://github.com/refraction-networking/utls).

## Update procedure

(Adapted from refraction-networking/utls instructions.)

```
git remote add golang git@github.com:golang/go.git || git fetch golang
git branch -D golang-upstream golang-http-upstream
git checkout -b golang-upstream go1.16.7
git subtree split -P src/crypto/http/ -b golang-http-upstream
git checkout main
git merge golang-http-upstream
```
