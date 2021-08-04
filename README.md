# Truestamp CLI

A Truestamp Command Line Interface (CLI) that utilizes the
[truestamp-js](https://github.com/truestamp/truestamp-js) library, written in
Typescript.

This CLI is compiled from Typescript using [Deno](https://deno.land/). The
platforms we support are determined by the set that Deno currently supports.

Deno provides the benefits of a single-file binary download, with no
pre-requisites or dependencies. Additionally Deno is security limited to allow
contact with only the limited set of internet addresses, environment variables,
and files it requires and has been configured to access at compile time.

## Install

If you are using macOS it is recommended you install this client using
[Homebrew](https://brew.sh/).

For manual installation please be sure to specify the current stable release
version in the release URL for pre-compiled binaries. Version `v0.0.0` is used
in these examples. The latest stable release versions can be found on the
[releases](https://github.com/truestamp/truestamp-cli/releases) page.

You will see a `lite` version of each released build. This should be
functionally identical but may have a somewhat smaller download footprint.

### macOS

#### Homebrew install

This is the recommended installation method for macOS. Please see the
instructions for installation using our
[truestamp/homebrew-tap](https://github.com/truestamp/homebrew-tap/).

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

#### Manual install - Apple Silicon (M1) Macs

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

#### Manual install - Linux x86 64 bit

Download and install the `linux` platform `x86_64` arch `.tar.gz` file for your
chosen [release](https://github.com/truestamp/truestamp-cli/releases) version.

```sh
wget -q https://github.com/truestamp/truestamp-cli/releases/download/v0.0.0/truestamp-linux-x86_64.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-linux-x86_64.tar.gz

# assuming /usr/local/bin exists and is on your $PATH
mv ./truestamp /usr/local/bin
```

#### Manual install - Windows x86 64 bit

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
with `-h`, `--help`, or simply `help` to learn more.

```txt
❯ truestamp -h

  Usage:   truestamp
  Version: v0.0.0

  Description:

    Truestamp CLI

  Options:

    -h, --help     - Show this help.
    -V, --version  - Show the version number for this program.

  Commands:

    auth                    - Login, logout, and show status of your authentication.
    completions             - Generate shell completions.
    documents               - Create, read, update, or destroy documents.
    heartbeat               - Display results of API server heartbeat call.
    help         [command]  - Show this help or the help of a sub-command.
```

## Releasing

All new commits to `main` will trigger a pre-release build of all binary assets
and store them in a new Github
[release](https://github.com/truestamp/truestamp-cli/releases) tagged with
`latest`.

Steps for a public release:

- Ensure the `version` in [src/cli.ts](src/cli.ts) is updated to the desired
  version.
- Create a new `tag` (not a new Release!) where the tag name follows the form
  `vx.x.x` where the `x` represents a semantic version number. Example:
  `git tag -a v0.0.8 -m "v0.0.8"` followed by `git push origin v0.0.8`.
- Once the `tagged-release.yml` workflow succeeds, a new release, with an
  automatic changelog will have been created.
- One of the build artifacts is the `CHECKSUMS-SHA2-256.txt` file, which
  contains the `SHA2-256` checksum of each of the build files. These checksum
  values should be transferred to the
  [truestamp-cli](https://github.com/truestamp/homebrew-tap/blob/main/Formula/truestamp-cli.rb)
  Homebrew Formula and pushed and tested.

## Legal

© 2020-2021 Truestamp Inc. All rights reserved.
