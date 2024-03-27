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

# Classification of ./internal packages
#
# 1. packages that map directly and are captured by the following diff command
#
# - src/net/http/internal/ascii => ./internal/ascii
#
# - src/net/http/internal/testcert => ./internal/testcert

diff -ur $upstreamrepo/src/net/http . || true

# 2. packages that we need to diff for explicitly
#
# - src/internal/safefilepath => ./internal/safefilepath

diff -ur $upstreamrepo/src/internal/safefilepath ./internal/safefilepath || true

# 3. replacement packages
#
# - ./internal/fakerace fakes out src/internal/race
# - ./internal/testenv fakes out src/internal/testenv
