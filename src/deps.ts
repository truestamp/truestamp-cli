// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { copy, readAllSync } from "https://deno.land/std@0.133.0/streams/conversion.ts"
import { crypto } from "https://deno.land/std@0.133.0/crypto/mod.ts";
import { parse } from "https://deno.land/std@0.133.0/path/mod.ts";

import { sleep } from "https://deno.land/x/sleep@v1.2.1/mod.ts";

import { decode, validate } from "https://deno.land/x/djwt@v2.4/mod.ts";
export type { Payload } from "https://deno.land/x/djwt@v2.4/mod.ts";

import { DB } from "https://deno.land/x/sqlite@v3.3.0/mod.ts";
export type {
  Row,
} from "https://deno.land/x/sqlite@v3.3.0/mod.ts";


import { colors } from "https://deno.land/x/cliffy@v0.22.2/ansi/colors.ts";
import {
  Command,
  EnumType,
  ValidationError
} from "https://deno.land/x/cliffy@v0.22.2/command/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.22.2/command/help/mod.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.22.2/command/completions/mod.ts";
export type { ITypeInfo } from "https://deno.land/x/cliffy@v0.22.2/flags/mod.ts";

import Conf from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.5-beta/mod.ts";
export type {
  Json,
  StoreType,
} from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.5-beta/mod.ts";

import appPaths from "https://raw.githubusercontent.com/truestamp/deno-app-paths/v1.0.1/mod.ts";

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
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js@~v0.9.4?dts";
import { createTruestampClient } from "./truestamp.ts";
import { decodeUnsafely } from "https://cdn.skypack.dev/@truestamp/truestamp-id@~v1.1.2?dts";

export {
  colors,
  Command,
  CompletionsCommand,
  Conf,
  copy,
  createTruestampClient,
  crypto,
  DB,
  decode,
  deleteTokensInConfig,
  appPaths,
  EnumType,
  getAccessTokenWithPrompts,
  getConfigAccessToken,
  getConfigForEnv,
  getConfigIdTokenPayload,
  getConfigKeyForEnv,
  getConfigRefreshToken,
  HelpCommand,
  parse,
  readAllSync,
  setConfigKeyForEnv,
  sleep,
  Truestamp,
  decodeUnsafely,
  validate,
  ValidationError,
};
