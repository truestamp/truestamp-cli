#!/bin/bash
# Copyright (c) 2019-2026 Truestamp, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# The `go test -exec` program nonet.sh installs: it loads the recorder
# into each test binary, and through the inherited environment into every
# process the test starts that is not SIP-protected. This script's own
# /bin/bash, like /bin/sh, drops the variable, so it is set again here; a
# child started through a SIP-protected binary is refused but not
# recorded.
export DYLD_INSERT_LIBRARIES="${NETREC_DYLIB:?NETREC_DYLIB is not set; run through tools/nonet/nonet.sh}"
exec "$@"
