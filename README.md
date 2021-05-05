# Truestamp CLI

A Truestamp Command Line Interface (CLI) that utilizes the [truestamp-js](https://github.com/truestamp/truestamp-js) library, written in Typescript.

## Install

## macOS with Homebrew install

Please see the instructions for installation using our [truestamp/homebrew-tap](https://github.com/truestamp/homebrew-tap/).

## macOS manual install

Please be sure to specify the current stable release version in the release URL. Version `v0.0.1` is used in this example. The latest stable release versions can be found on the [releases](https://github.com/truestamp/truestamp-cli/releases) page.

### Download and unpack for Intel x86 Macs

Download the `darwin` platform `x86` arch `.tar.gz` file for your chosen release version.

Example:

```sh
wget https://github.com/truestamp/truestamp-cli/releases/download/v0.0.1/truestamp-darwin-x86_64-lite.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-darwin-x86_64-lite.tar.gz
```

### Download and unpack for Apple Silicon (M1) Macs

Download the `darwin` platform `aarch64` arch `.tar.gz` file for your chosen release version.

Example:

```sh
wget https://github.com/truestamp/truestamp-cli/releases/download/v0.0.1/truestamp-darwin-aarch64-lite.tar.gz

# unpack the `truestamp` binary for your system arch
tar -zxvf truestamp-darwin-aarch64-lite.tar.gz
```

### All Macs

```sh
# assuming /usr/local/bin exists and is on your $PATH
mv ./truestamp /usr/local/bin

# Add this single file to the macOS Gatekeeper allow list
spctl --add  /usr/local/bin/truestamp
```

### Other

Pre-compiled binaries for Linux, and Windows are also available as tagged [releases](https://github.com/truestamp/truestamp-cli/releases).

## Legal

© 2020-2021 Truestamp Inc. All rights reserved.
