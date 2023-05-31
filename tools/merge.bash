#!/bin/bash
set -euxo pipefail
git checkout stable
git remote add golang git@github.com:golang/go.git || git fetch golang
git branch -D golang-upstream golang-http-upstream merged-stable || true
git fetch golang
git checkout -b golang-upstream $(cat UPSTREAM)
git subtree split -P src/net/http/ -b golang-http-upstream
git checkout stable
git checkout -b merged-stable
git merge golang-http-upstream
