#!/bin/sh
# End-to-end smoke test for docs/install.sh.
#
# Runs the installer against a real, pinned GitHub release into a disposable
# directory, then asserts the resulting binary reports the expected version.
# This is deliberately not a mock test — the point is to exercise the actual
# GitHub Releases download path and SHA-256 verification.
#
# Usage:
#   sh docs/install.test.sh            # uses the default pinned version
#   TRUESTAMP_TEST_VERSION=vX.Y.Z sh docs/install.test.sh
#
# Exit code 0 on success, 1 on any failure.

set -eu

# Pin a specific release so the test is deterministic and doesn't start
# failing when a new release ships. Bump this when a new release becomes
# the oldest one we want to keep passing.
TEST_VERSION="${TRUESTAMP_TEST_VERSION:-v0.2.0}"

here="$(cd "$(dirname "$0")" && pwd)"
install_script="${here}/install.sh"

[ -f "${install_script}" ] \
    || { echo "install.sh not found at ${install_script}" >&2; exit 1; }

tmpdir="$(mktemp -d 2>/dev/null || mktemp -d -t truestamp-install-test)"
trap 'rm -rf "${tmpdir}"' EXIT INT TERM

echo "smoke test: installing ${TEST_VERSION} into ${tmpdir}"

# Run the installer with all network-affecting env vars pinned so the test
# outcome is reproducible.
TRUESTAMP_VERSION="${TEST_VERSION}" \
TRUESTAMP_INSTALL_DIR="${tmpdir}" \
    sh "${install_script}" >/dev/null

binary="${tmpdir}/truestamp"
[ -x "${binary}" ] \
    || { echo "FAIL: expected executable at ${binary}" >&2; exit 1; }

# The binary must run and report *something* version-like. We don't pin
# the exact output so the test keeps passing as the version string format
# evolves.
version_output="$("${binary}" --version 2>&1)" \
    || { echo "FAIL: '${binary} --version' exited non-zero" >&2; exit 1; }

case "${version_output}" in
    *"${TEST_VERSION#v}"*) : ;;
    *)
        echo "FAIL: --version output does not mention ${TEST_VERSION#v}" >&2
        echo "  got: ${version_output}" >&2
        exit 1
        ;;
esac

echo "PASS: ${version_output}"
