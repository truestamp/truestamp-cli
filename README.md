# Truestamp CLI

A Truestamp Command Line Interface (CLI), written in Typescript, that utilizes
the [@truestamp/client](https://socket.dev/npm/package/@truestamp/client) library to communicate with the [Truestamp API](https://docs.truestamp.com/api/intro).

This CLI is built using [Deno](https://deno.land/). Platform support is determined by the set that Deno [currently supports](<https://deno.land/manual@v1.29.1/tools/compiler> for cross-compilation.

Deno creates a single-file binary that makes download and installation easy. The
binaries have no pre-requisites that need to be installed ahead of time.

## Install

For manual installation please be sure to specify the current stable release
version in the release URL for pre-compiled binaries. Version `v0.0.0` is used
in these examples and must be replaced. The latest stable release versions can
be found on the [releases](https://github.com/truestamp/truestamp-cli/releases)
page.

### macOS

If you are using macOS it is recommended you install the client using
[Homebrew](https://brew.sh/).

#### Homebrew install

This is the recommended installation method for macOS. Please see the
instructions for installation using the [truestamp/homebrew-tap](https://github.com/truestamp/homebrew-tap/).

#### Manual install - Apple macOS Intel x86 Macs

Download and install the `darwin` platform `x86` arch `.tar.gz` file for your
chosen [release](https://github.com/truestamp/truestamp-cli/releases) version.

Example:

```sh
wget -q https://github.com/truestamp/truestamp-cli/releases/download/v0.0.0/truestamp-darwin-x86_64.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-darwin-x86_64.tar.gz

# assuming /usr/local/bin exists and is on your $PATH
mv ./truestamp /usr/local/bin

# Add this single file to the macOS Gatekeeper allow list
spctl --add  /usr/local/bin/truestamp
```

#### Manual install - Apple Silicon Macs

Download and install the `darwin` platform `aarch64` arch `.tar.gz` file for
your chosen [release](https://github.com/truestamp/truestamp-cli/releases)
version.

Example:

```sh
wget -q https://github.com/truestamp/truestamp-cli/releases/download/v0.0.0/truestamp-darwin-aarch64.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-darwin-aarch64.tar.gz

# assuming /usr/local/bin exists and is on your $PATH
mv ./truestamp /usr/local/bin

# Add this single file to the macOS Gatekeeper allow list
spctl --add  /usr/local/bin/truestamp
```

#### Manual install - Linux x86 64 bit [EXPERIMENTAL]

Download and install the `linux` platform `x86_64` arch `.tar.gz` file for your
chosen [release](https://github.com/truestamp/truestamp-cli/releases) version.

```sh
wget -q https://github.com/truestamp/truestamp-cli/releases/download/v0.0.0/truestamp-linux-x86_64.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-linux-x86_64.tar.gz

# assuming /usr/local/bin exists and is on your $PATH
mv ./truestamp /usr/local/bin
```

#### Manual install - Windows x86 64 bit [EXPERIMENTAL]

Download and install the `windows` platform `.zip` file for your chosen
[release](https://github.com/truestamp/truestamp-cli/releases) version.

```sh
# PowerShell (Replace v0.0.0 in path with appropriate version)

Invoke-WebRequest -OutFile truestamp-windows-x86_64.zip https://github.com/truestamp/truestamp-cli/releases/download/v0.0.0/truestamp-windows-x86_64.zip

# Expand-Archive -LiteralPath <PathToZipFile> -DestinationPath <PathToDestination>
Expand-Archive -LiteralPath truestamp-windows-x86_64.zip

# You should now have a `truestamp.exe` file that can be run in a command shell
truestamp -h
```

## Usage

The CLI has its own help system and every command and sub-command can be invoked
with `-h`, `--help`, or simply `help` to learn more. The help for each command also provides usage examples.

```txt
truestamp --help
truestamp items --help
truestamp items create --help
```

## Build and Release

For developers who want to build their own copy of the CLI.

Additional task are available in the `deno.json`.

### Local

`deno task build-local`

### Shared

All new commits to `main` will trigger a pre-release build of all binary assets and store them in a new Github [release](https://github.com/truestamp/truestamp-cli/releases) tagged with `latest`.

Steps for a public release:

- Ensure the `version` in [src/cli.ts](src/cli.ts) is updated to the desired version.
- Create a new `tag` (not a new Release!) where the tag name follows the form `vx.x.x` where the `x` represents a semantic version number. Example: `git tag -a v0.0.8 -m "v0.0.8"` followed by `git push origin v0.0.8`.
- Once the `tagged-release.yml` workflow succeeds, a new release, with an automatic changelog will have been created.
- One of the build artifacts is the `CHECKSUMS-SHA2-256.txt` file, which contains the `SHA2-256` checksum of each of the build files. These checksum values should be transferred to the [truestamp-cli](https://github.com/truestamp/homebrew-tap/blob/main/Formula/truestamp-cli.rb) Homebrew Formula and pushed and tested.

## Legal

Copyright © 2020-2022 Truestamp Inc. All rights reserved.
