#!/bin/sh
# Truestamp CLI installer.
#
# Source:  https://github.com/truestamp/truestamp-cli/blob/main/docs/install.sh
# Served:  https://get.truestamp.com/install.sh
#
# Usage:
#   curl -fsSL https://get.truestamp.com/install.sh | sh
#
# Environment variables (all optional):
#   TRUESTAMP_VERSION         Tag to install, e.g. v0.3.0. Defaults to latest.
#   TRUESTAMP_INSTALL_DIR     Install target dir. Defaults to /usr/local/bin
#                             or ~/.local/bin.
#   TRUESTAMP_SKIP_CHECKSUM   Set to 1 to skip SHA-256 verification. Debug only.
#   TRUESTAMP_REQUIRE_COSIGN  Set to 1 to REQUIRE cosign signature verification
#                             of the checksums file. Refuses to install if cosign
#                             is not on PATH or the signature does not verify.
#                             Without this, cosign verification is best-effort:
#                             run when cosign is present, skipped silently if not.
#   TRUESTAMP_ALLOW_SUDO      Set to 1 to allow running as root / via sudo.
#   GITHUB_TOKEN              Bearer token for the GitHub releases API (raises
#                             rate limits; optional).
#
# Flags:
#   -h, --help                Print this usage summary and exit.

set -eu

REPO="truestamp/truestamp-cli"
PROJECT="truestamp-cli"
BINARY="truestamp"

TMPDIR_INSTALL=""

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

log()  { printf '%s\n' "$*" >&2; }
info() { log "  $*"; }
warn() { log "warning: $*"; }
die()  { log "error: $*"; exit 1; }

usage() {
    # Kept in sync manually with the header block above. This function is
    # reachable via `curl ... | sh -s -- --help`, where $0 is `sh` rather
    # than the script path, so we can't re-read the header at runtime.
    cat <<'EOF'
Truestamp CLI installer.

Usage:
    curl -fsSL https://get.truestamp.com/install.sh | sh
    curl -fsSL https://get.truestamp.com/install.sh | sh -s -- --help

Environment variables (all optional):
    TRUESTAMP_VERSION         Tag to install, e.g. v0.3.0. Defaults to latest.
    TRUESTAMP_INSTALL_DIR     Install target dir. Defaults to /usr/local/bin
                              or ~/.local/bin.
    TRUESTAMP_SKIP_CHECKSUM   Set to 1 to skip SHA-256 verification. Debug only.
    TRUESTAMP_REQUIRE_COSIGN  Set to 1 to REQUIRE cosign signature verification
                              of the checksums file.
    TRUESTAMP_ALLOW_SUDO      Set to 1 to allow running as root / via sudo.
    GITHUB_TOKEN              Bearer token for the GitHub releases API.
EOF
    exit 0
}

cleanup() {
    if [ -n "${TMPDIR_INSTALL}" ] && [ -d "${TMPDIR_INSTALL}" ]; then
        rm -rf "${TMPDIR_INSTALL}"
    fi
}

trap cleanup EXIT INT TERM

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

# Prefer curl; fall back to wget if absent.
fetch() {
    # args: <url> <output-path>
    if command -v curl >/dev/null 2>&1; then
        curl --fail --silent --show-error --location \
             --proto '=https' --tlsv1.2 \
             --output "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget --quiet --https-only --output-document="$2" "$1"
    else
        die "need curl or wget to download files"
    fi
}

fetch_stdout() {
    # args: <url>
    if command -v curl >/dev/null 2>&1; then
        if [ -n "${GITHUB_TOKEN:-}" ]; then
            curl --fail --silent --show-error --location \
                 --proto '=https' --tlsv1.2 \
                 -H "Authorization: Bearer ${GITHUB_TOKEN}" \
                 "$1"
        else
            curl --fail --silent --show-error --location \
                 --proto '=https' --tlsv1.2 \
                 "$1"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [ -n "${GITHUB_TOKEN:-}" ]; then
            wget --quiet --https-only \
                 --header="Authorization: Bearer ${GITHUB_TOKEN}" \
                 -O - "$1"
        else
            wget --quiet --https-only -O - "$1"
        fi
    else
        die "need curl or wget to download files"
    fi
}

sha256_of() {
    # args: <file>
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    elif command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        die "need shasum or sha256sum to verify downloads"
    fi
}

# -----------------------------------------------------------------------------
# Pre-flight
# -----------------------------------------------------------------------------

check_not_root() {
    if [ "$(id -u 2>/dev/null || echo 0)" = "0" ] && [ -z "${TRUESTAMP_ALLOW_SUDO:-}" ]; then
        die "refusing to run as root. re-run without sudo, or set TRUESTAMP_ALLOW_SUDO=1 to override."
    fi
}

detect_os_arch() {
    uname_s="$(uname -s)"
    uname_m="$(uname -m)"

    case "${uname_s}" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux"  ;;
        *)      die "unsupported OS: ${uname_s} (this installer supports macOS and Linux; Windows users: use 'go install' or download the zip from GitHub Releases)" ;;
    esac

    case "${uname_m}" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) die "unsupported architecture: ${uname_m} (supported: amd64, arm64)" ;;
    esac
}

# -----------------------------------------------------------------------------
# Version resolution
# -----------------------------------------------------------------------------

resolve_version() {
    if [ -n "${TRUESTAMP_VERSION:-}" ]; then
        VERSION="${TRUESTAMP_VERSION}"
        return
    fi

    info "fetching latest release tag from GitHub..."
    _api="https://api.github.com/repos/${REPO}/releases/latest"
    _json="$(fetch_stdout "${_api}")" || die "could not reach GitHub releases API"

    # Parse .tag_name without jq: grep the first tag_name field.
    VERSION="$(printf '%s\n' "${_json}" \
        | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' \
        | head -n 1 \
        | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"

    [ -n "${VERSION}" ] || die "could not determine latest release tag"
}

# -----------------------------------------------------------------------------
# Download + verify
# -----------------------------------------------------------------------------

download_and_verify() {
    TMPDIR_INSTALL="$(mktemp -d 2>/dev/null || mktemp -d -t truestamp-install)"

    # Strip leading v, if present, to match GoReleaser's archive naming.
    _ver_no_v="${VERSION#v}"
    ARCHIVE_NAME="${PROJECT}_${_ver_no_v}_${OS}_${ARCH}.tar.gz"
    _base_url="https://github.com/${REPO}/releases/download/${VERSION}"
    ARCHIVE_URL="${_base_url}/${ARCHIVE_NAME}"
    CHECKSUM_URL="${_base_url}/checksums.txt"
    CHECKSUM_BUNDLE_URL="${_base_url}/checksums.txt.sigstore"

    info "downloading ${ARCHIVE_NAME}..."
    fetch "${ARCHIVE_URL}" "${TMPDIR_INSTALL}/${ARCHIVE_NAME}" \
        || die "could not download ${ARCHIVE_URL}"

    if [ "${TRUESTAMP_SKIP_CHECKSUM:-}" = "1" ]; then
        warn "TRUESTAMP_SKIP_CHECKSUM=1 — skipping SHA-256 verification"
        return
    fi

    info "verifying SHA-256..."
    fetch "${CHECKSUM_URL}" "${TMPDIR_INSTALL}/checksums.txt" \
        || die "could not download ${CHECKSUM_URL}"

    verify_cosign_signature "${TMPDIR_INSTALL}/checksums.txt"

    _expected="$(grep "  ${ARCHIVE_NAME}\$" "${TMPDIR_INSTALL}/checksums.txt" \
        | awk '{print $1}')"
    [ -n "${_expected}" ] \
        || die "checksum for ${ARCHIVE_NAME} not found in checksums.txt"

    _actual="$(sha256_of "${TMPDIR_INSTALL}/${ARCHIVE_NAME}")"
    if [ "${_expected}" != "${_actual}" ]; then
        die "checksum mismatch!
  expected: ${_expected}
  actual:   ${_actual}
  (re-run the installer, or report this at https://github.com/${REPO}/issues)"
    fi
    info "sha256: ${_actual}"
}

# Verify the cosign keyless signature over checksums.txt. The signing identity
# is the release workflow in this repository; the OIDC issuer is GitHub
# Actions' token endpoint. If cosign is not available and
# TRUESTAMP_REQUIRE_COSIGN is unset, we skip silently — the SHA-256 check
# still guarantees integrity of the archive against whatever checksums.txt we
# got, just not that checksums.txt itself is authentic.
verify_cosign_signature() {
    _checksums="$1"

    if ! command -v cosign >/dev/null 2>&1; then
        if [ "${TRUESTAMP_REQUIRE_COSIGN:-}" = "1" ]; then
            die "TRUESTAMP_REQUIRE_COSIGN=1 but 'cosign' is not on PATH. Install cosign from https://github.com/sigstore/cosign"
        fi
        return
    fi

    # If the signature bundle isn't published for this release (e.g. a
    # legacy release from before cosign signing landed), soft-fail: in
    # best-effort mode we skip with a warning, in require mode we refuse.
    if ! fetch "${CHECKSUM_BUNDLE_URL}" "${TMPDIR_INSTALL}/checksums.txt.sigstore" 2>/dev/null; then
        if [ "${TRUESTAMP_REQUIRE_COSIGN:-}" = "1" ]; then
            die "TRUESTAMP_REQUIRE_COSIGN=1 but cosign signature bundle is not published for ${VERSION}"
        fi
        warn "cosign signature bundle not found for ${VERSION}; skipping signature verification"
        return
    fi

    info "verifying cosign signature..."

    # Identity: the release workflow file in this repo. OIDC issuer: GitHub Actions.
    cosign verify-blob \
        --bundle "${TMPDIR_INSTALL}/checksums.txt.sigstore" \
        --certificate-identity-regexp "^https://github\\.com/${REPO}/\\.github/workflows/release\\.yml@" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${_checksums}" >/dev/null 2>&1 \
        || die "cosign signature did not verify for checksums.txt"
}

# -----------------------------------------------------------------------------
# Extract + install
# -----------------------------------------------------------------------------

extract_and_install() {
    info "extracting archive..."
    (cd "${TMPDIR_INSTALL}" && tar -xzf "${ARCHIVE_NAME}") \
        || die "could not extract ${ARCHIVE_NAME}"

    _src="${TMPDIR_INSTALL}/${BINARY}"
    [ -f "${_src}" ] \
        || die "expected binary ${BINARY} inside archive, not found"
    chmod 0755 "${_src}"

    choose_install_dir
    _dest="${INSTALL_DIR}/${BINARY}"

    info "installing to ${_dest}..."
    mkdir -p "${INSTALL_DIR}"

    # Atomic replace within the same filesystem; fall back to cp + mv for
    # cross-filesystem (e.g. /tmp -> $HOME on some Linux distros).
    if ! mv -f "${_src}" "${_dest}" 2>/dev/null; then
        cp "${_src}" "${_dest}.new" || die "could not write to ${INSTALL_DIR}"
        chmod 0755 "${_dest}.new"
        mv -f "${_dest}.new" "${_dest}" || die "could not replace ${_dest}"
    fi

    # On macOS, any file that arrived via curl (or was downloaded via
    # the install.sh we fetched via curl) may carry the quarantine xattr
    # that Gatekeeper uses to block first-run. Clear it so the binary
    # runs without a "truestamp Not Opened" dialog.
    if [ "${OS}" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
        xattr -d com.apple.quarantine "${_dest}" >/dev/null 2>&1 || true
    fi
}

choose_install_dir() {
    if [ -n "${TRUESTAMP_INSTALL_DIR:-}" ]; then
        INSTALL_DIR="${TRUESTAMP_INSTALL_DIR}"
        return
    fi

    # /usr/local/bin is the traditional local-admin prefix, writable by
    # members of the admin group on macOS and by root on Linux.
    if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
        return
    fi

    # Fall back to the XDG-adjacent user-local bin dir.
    INSTALL_DIR="${HOME}/.local/bin"
}

verify_installed() {
    _dest="${INSTALL_DIR}/${BINARY}"
    if ! "${_dest}" --version >/dev/null 2>&1; then
        die "installed binary at ${_dest} failed to run"
    fi

    log ""
    log "✓ $("${_dest}" --version)"
    log "  installed at ${_dest}"

    # PATH hint if the install dir isn't on $PATH.
    case ":${PATH}:" in
        *:"${INSTALL_DIR}":*) : ;;
        *)
            log ""
            log "note: ${INSTALL_DIR} is not on your \$PATH. Add it to your shell profile:"
            log "    export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
    esac
}

# -----------------------------------------------------------------------------
# main
# -----------------------------------------------------------------------------

main() {
    # Parse flags. The only supported flag is --help / -h; anything else is
    # rejected so a typo like `sh -s -- --ifnstall-dir` doesn't silently
    # fall through to the default install path.
    while [ $# -gt 0 ]; do
        case "$1" in
            -h|--help) usage ;;
            *) die "unknown argument: $1 (run with --help for usage)" ;;
        esac
    done

    log ""
    log "Truestamp CLI installer"
    log "  https://github.com/${REPO}"
    log ""

    check_not_root
    need_cmd tar
    need_cmd uname

    detect_os_arch
    resolve_version

    info "target: ${PROJECT} ${VERSION} (${OS}/${ARCH})"

    download_and_verify
    extract_and_install
    verify_installed
}

main "$@"
