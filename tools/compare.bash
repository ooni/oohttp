#!/bin/bash
set -euo pipefail
upstreamrepo=./upstreamrepo
TAG=$(cat UPSTREAM)
test -d $upstreamrepo || git clone git@github.com:golang/go.git $upstreamrepo
(
	cd $upstreamrepo
	git checkout master
	git pull
	git checkout $TAG
)
for file in $(cd $upstreamrepo/src/net/http && find . -type f -name \*.go); do
	git diff --no-index $upstreamrepo/src/net/http/$file $file || true
done
