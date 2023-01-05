#!/bin/bash
set -euxo pipefail
git checkout main
git remote add golang git@github.com:golang/go.git || git fetch golang
git branch -D golang-upstream golang-http-upstream merged-main || true
git fetch golang
git checkout -b golang-upstream $(cat UPSTREAM)
git subtree split -P src/net/http/ -b golang-http-upstream
git checkout main
git checkout -b merged-main
git merge golang-http-upstream
