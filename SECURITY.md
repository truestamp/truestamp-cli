# Security Policy

We take the security of `truestamp-cli` seriously because this CLI is what users rely on to verify Truestamp proofs end to end — any compromise of the binary or its verification logic directly undermines the cryptographic guarantees we offer.

## Reporting a vulnerability

**Do not open a public GitHub issue for security reports.**

Use one of these private channels instead:

1. **GitHub private vulnerability report** (preferred) — open one at <https://github.com/truestamp/truestamp-cli/security/advisories/new>. This keeps the conversation threaded with the repository and lets us assign a CVE if the issue warrants one.
2. **Email** — <security@truestamp.com>. Please include "truestamp-cli" in the subject line.

Include, where possible:

- The version (`truestamp version` output) and installation method (`install.sh`, Homebrew, `go install`, tarball).
- A minimal proof of concept or reproduction steps.
- The impact you believe a successful exploit would have.

## What to expect

- **Acknowledgement** within 3 business days.
- **Triage + preliminary assessment** within 10 business days.
- **Coordinated disclosure window** of up to 90 days from the first acknowledgement. We may ship a fix sooner and request an earlier public disclosure; we will not extend beyond 90 days without your agreement.
- **Credit** in the release notes and (if you want) in a published advisory. Let us know how you'd like to be named, or if you prefer to remain anonymous.

## Scope

In scope:

- The CLI binary published at <https://github.com/truestamp/truestamp-cli/releases>.
- The installer at <https://get.truestamp.com/install.sh>.
- The Homebrew cask published to `truestamp/homebrew-tap`.
- The `go install` path at `github.com/truestamp/truestamp-cli/cmd/truestamp`.

Out of scope (please report to the appropriate project):

- The Truestamp backend service (report via the channels above, but note that the fix ships through a different pipeline).
- Third-party dependencies — please report upstream first, and let us know so we can track the remediation here.
