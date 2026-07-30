# External API Calls

Enabled by default, skipped with `--skip-external`:

This table is the answer to "what does this binary talk to?"; keep it exhaustive. Every entry except the keyring is a package-level `var` in `internal/external` (the keyring URL is derived from `base_url` in `config.Load` and passed to `external.VerifyKeyring` as a parameter), so a new egress destination means a new row here.

| Service          | When                       | URL                                                                       |
| ---------------- | -------------------------- | ------------------------------------------------------------------------- |
| Truestamp Keyring | Key Binding step (every bundle) | `{base-url}/.well-known/keyring.json` (derived from `base_url`)     |
| Stellar Horizon  | Stellar commitment in `cx`, **or** an `entropy_stellar` (`t=31`) subject | `https://horizon-testnet.stellar.org` or `https://horizon.stellar.org` (`external.HorizonTestnetURL` / `HorizonPublicURL`) |
| Blockstream      | Two distinct callers with **opposite** network semantics: (a) a Bitcoin commitment in `cx` — network taken from the commitment (mainnet/testnet; regtest has no public API); (b) an `entropy_bitcoin` (`t=32`) subject — **always mainnet**, regardless of what the proof commits to | `https://blockstream.info/api` or `https://blockstream.info/testnet/api` (`external.BlockstreamMainnetURL` / `BlockstreamTestnetURL`) |
| NIST Randomness Beacon | `entropy_nist` (`t=30`) subject only: the Entropy Source step | `https://beacon.nist.gov/beacon/2.0/chain/{chainIndex}/pulse/{pulseIndex}` (`external.NISTBeaconURL`) |

Bitcoin regtest has no public API -- local crypto verification only.

Only the keyring base is reachable from the CLI surface (via `--base-url`). Horizon, Blockstream and the NIST beacon are package vars in `internal/external` with **deliberately no flag and no env override**: E.18/E.19/E.21 exist to consult a third party, and a supported override would let whoever supplies a proof also supply the "independent" chain that confirms it. `cmd/external_e2e_test.go` still proves those three failure modes end-to-end through a real `verify` process: it builds its own binary with `-ldflags -X internal/external.<Var>=<httptest URL>`, so the redirection exists only at link time in a test and never in a shipped artifact. `TestCLI_ExternalEndpoints_NoAmbientOverride` pins the no-override property. Do not add a flag or env var for these; extend the link-time harness instead.

