// © 2020-2021 Truestamp Inc. All rights reserved.

import { createHash } from "https://deno.land/std@0.102.0/hash/mod.ts";
import * as path from "https://deno.land/std@0.102.0/path/mod.ts";

import { Buffer } from "https://deno.land/x/node_buffer@1.1.0/mod.ts";

// FIXME : using a tag from our own fork of this lib while waiting for upstream PR to merge
import { sleep } from "https://raw.githubusercontent.com/truestamp/sleep/v1.3.0/mod.ts";

import { decode, validate } from "https://deno.land/x/djwt@v2.2/mod.ts";

import { colors } from "https://deno.land/x/cliffy@v0.19.3/ansi/colors.ts";
import {
  Command,
  EnumType,
  ValidationError,
} from "https://deno.land/x/cliffy@v0.19.3/command/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.19.3/command/help/mod.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.19.3/command/completions/mod.ts";

import Conf from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.2-beta/mod.ts";

// ULID : https://github.com/ulid/javascript
//      : https://www.skypack.dev/view/ulid
import { ulid } from "https://cdn.skypack.dev/ulid?dts";

import { S3 } from "https://deno.land/x/aws_sdk@v3.22.0-1/client-s3/mod.ts";

import {
  getConfigForEnv,
  getConfigKeyForEnv,
  setConfigKeyForEnv,
} from "./config.ts";

import {
  deleteTokensInConfig,
  getAccessTokenWithPrompts,
  getConfigAccessToken,
  getConfigIdTokenPayload,
  getConfigRefreshToken,
} from "./auth.ts";

// See : https://www.skypack.dev/view/@truestamp/truestamp-js
// See SkyPack : https://docs.skypack.dev/skypack-cdn/api-reference/lookup-urls
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js@~v0.1.0?dts";
import { createTruestampClient } from "./truestamp.ts";

export {
  Buffer,
  colors,
  Command,
  CompletionsCommand,
  Conf,
  createHash,
  createTruestampClient,
  decode,
  deleteTokensInConfig,
  EnumType,
  getAccessTokenWithPrompts,
  getConfigAccessToken,
  getConfigForEnv,
  getConfigIdTokenPayload,
  getConfigKeyForEnv,
  getConfigRefreshToken,
  HelpCommand,
  path,
  S3,
  setConfigKeyForEnv,
  sleep,
  Truestamp,
  ulid,
  validate,
  ValidationError,
};

export type { ITypeInfo } from "https://deno.land/x/cliffy@v0.19.3/flags/mod.ts";
export type { Payload } from "https://deno.land/x/djwt@v2.2/mod.ts";

export type {
  ItemType,
  StoreType,
} from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.2-beta/mod.ts";

export type {
  S3ClientConfig,
} from "https://deno.land/x/aws_sdk@v3.22.0-1/client-s3/mod.ts";
