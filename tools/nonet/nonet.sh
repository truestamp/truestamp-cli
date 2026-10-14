#!/bin/bash
# Copyright (c) 2019-2026 Truestamp, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Usage: NONET_WORK=DIR tools/nonet/nonet.sh CMD...
#
# Runs CMD at the root of the truestamp-cli tree this script sits in, under
# the sandbox profile beside it (no-network.sb). It refuses exactly: every
# connection off loopback; loopback ports 4000 to 4003, 5432 and 4369; DNS
# (the mDNSResponder socket and its mach service); the Docker socket (any
# path ending in /.docker/run/docker.sock, which /var/run/docker.sock
# resolves to) and the Postgres socket (any path ending in
# /.s.PGSQL.5432, such as /tmp/.s.PGSQL.5432); and running the keychain
# tool, /usr/bin/security. Every other loopback port and Unix socket stays
# reachable. A control module (control/) runs first and must see each
# refusal. The recorder, built by `task test-nonet` into DIR, is loaded
# into CMD and every process it starts that is not SIP-protected, and logs
# each attempt with its argv; a child started through a SIP-protected
# binary such as /bin/sh is refused but not recorded.
#
# Writes nothing into the tree: HOME, TMPDIR, GOCACHE, STATICCHECK_CACHE
# and the recorder's log live under DIR, which must be outside the tree.
# The module cache is used read-only (GOPROXY=off, -mod=readonly), and
# -buildvcs=false keeps the go command from running git in the tree.
#
# Exit: 2 bad setup, 3 the control failed, 4 CMD passed but a network
# attempt was recorded, otherwise CMD's own status. Run through Task, any
# failure exits 201; `task -x test-nonet` exits with this code instead, and
# Task's "exit status N" line names it either way.
set -euo pipefail

G="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
T="$(cd "$G/../.." && pwd)"
die() { echo "nonet: $*" >&2; exit 2; }

[ $# -gt 0 ] || die "usage: NONET_WORK=DIR $0 CMD..."
[ -f "$T/go.mod" ] && grep -q '^module github.com/truestamp/truestamp-cli$' "$T/go.mod" || die "$T is not a truestamp-cli tree"
W="${NONET_WORK:-}"
[ -n "$W" ] || die "NONET_WORK is not set; run through task test-nonet"
mkdir -p "$W"
W="$(cd "$W" && pwd)"
case "$W/" in "$T"/*) die "NONET_WORK ($W) must be outside the tree" ;; esac
case "$G$W" in *" "*) die "a path with a space cannot go into GOFLAGS: $G or $W" ;; esac
[ -f "$W/netrec.dylib" ] || die "no recorder at $W/netrec.dylib; run through task test-nonet"
mkdir -p "$W/gocache" "$W/tmp" "$W/home" "$W/staticcheck"

MODCACHE="$(go env GOMODCACHE)"
MISE_DATA="${MISE_DATA_DIR:-$HOME/.local/share/mise}"
MISE_CONFIG="${MISE_CONFIG_DIR:-$HOME/.config/mise}"
LOG="$W/netrec.log"

run() {
  sandbox-exec -f "$G/no-network.sb" /usr/bin/env -i \
    PATH="$PATH" HOME="$W/home" TMPDIR="$W/tmp/" GOTMPDIR="$W/tmp" \
    GOCACHE="$W/gocache" GOMODCACHE="$MODCACHE" GOPATH="$(dirname "$(dirname "$MODCACHE")")" \
    GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local GOFLAGS="-mod=readonly -buildvcs=false -exec=$G/exec.sh" \
    STATICCHECK_CACHE="$W/staticcheck" NETREC_LOG="$LOG" NETREC_DYLIB="$W/netrec.dylib" \
    DYLD_INSERT_LIBRARIES="$W/netrec.dylib" \
    MISE_DATA_DIR="$MISE_DATA" MISE_CONFIG_DIR="$MISE_CONFIG" MISE_OFFLINE=1 \
    "$@"
}

# Lines for the fixed dead loopback ports 1 and 9, which tests dial to get
# a refused connection, are not attempts.
attempts() { grep -vE 'loopback-fixed-port [^ ]+:(1|9) ' "$LOG" 2>/dev/null || true; }

rm -f "$LOG"
( cd "$G/control" && run go test -count=1 . ) || { echo "nonet: CONTROL FAILED: the guard did not refuse as required" >&2; exit 3; }
grep -q 'netguardcontrol.test connect 1.1.1.1:443' "$LOG" 2>/dev/null || { echo "nonet: CONTROL FAILED: the recorder saw no attempt" >&2; exit 3; }
echo "nonet: control fired ($(wc -l < "$LOG" | tr -d ' ') refused attempts recorded)" >&2

rm -f "$LOG"
set +e; ( cd "$T" && run "$@" ); rc=$?; set -e
n="$(attempts | grep -c . || true)"
if [ "$n" -gt 0 ]; then
  echo "nonet: NETWORK ATTEMPTS recorded (each refused, but for a loopback-fixed-port line on a port the profile allows):" >&2
  attempts | sed -E 's#^[0-9]+ [^ ]*/##' | sort | uniq -c >&2
  [ $rc -eq 0 ] && rc=4
else
  echo "nonet: 0 recorded network attempts" >&2
fi
exit $rc
