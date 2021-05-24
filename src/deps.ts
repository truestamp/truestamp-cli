// © 2020-2021 Truestamp Inc. All rights reserved.

import { createHash } from "https://deno.land/std@0.97.0/hash/mod.ts"
import * as path from "https://deno.land/std@0.97.0/path/mod.ts"

import { Buffer } from "http://deno.land/x/node_buffer@1.1.0/mod.ts"
import configDir from "https://deno.land/x/config_dir/mod.ts"
import { sleep } from "https://deno.land/x/sleep/mod.ts"
import loadJsonFile from "https://deno.land/x/load_json_file@v1.0.0/mod.ts"
import { decode, validate } from "https://deno.land/x/djwt@v2.2/mod.ts"

import { colors } from "https://deno.land/x/cliffy@v0.18.2/ansi/colors.ts"
import { Command } from "https://deno.land/x/cliffy@v0.18.2/command/mod.ts"
import { HelpCommand } from "https://deno.land/x/cliffy@v0.18.2/command/help/mod.ts"
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.18.2/command/completions/mod.ts"

// ULID : https://github.com/ulid/javascript
//      : https://www.skypack.dev/view/ulid
import { ulid } from "https://cdn.skypack.dev/ulid?dts"

import { S3 } from "https://deno.land/x/aws_sdk@v3.15.0.2/client-s3/mod.ts"

import {
  deleteSavedTokens,
  getAccessTokenWithPrompts,
  getSavedAccessToken,
  getSavedIdTokenPayload,
  getSavedRefreshToken,
} from "./auth.ts"

// See : https://www.skypack.dev/view/@truestamp/truestamp-js
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js?dts"
import { createTruestampClient } from "./truestamp.ts"

export {
  Buffer,
  colors,
  Command,
  CompletionsCommand,
  configDir,
  createHash,
  createTruestampClient,
  decode,
  deleteSavedTokens,
  getAccessTokenWithPrompts,
  getSavedAccessToken,
  getSavedIdTokenPayload,
  getSavedRefreshToken,
  HelpCommand,
  loadJsonFile,
  path,
  S3,
  sleep,
  Truestamp,
  ulid,
  validate,
}

export type { ITypeInfo } from "https://deno.land/x/cliffy/flags/mod.ts"
export type { Payload } from "https://deno.land/x/djwt@v2.2/mod.ts"
