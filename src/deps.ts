// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { crypto } from "https://deno.land/std@0.155.0/crypto/mod.ts";
import { parse } from "https://deno.land/std@0.155.0/path/mod.ts";
import {
  copy,
  readAllSync,
} from "https://deno.land/std@0.155.0/streams/conversion.ts";

import { sleep } from "https://deno.land/x/sleep@v1.2.1/mod.ts";

import { decode, validate } from "https://deno.land/x/djwt@v2.7/mod.ts";
export type { ITypeInfo } from "https://deno.land/x/cliffy@v0.25.0/flags/mod.ts";
export type { Payload } from "https://deno.land/x/djwt@v2.7/mod.ts";
export type { Row } from "https://deno.land/x/sqlite@v3.4.1/mod.ts";
export type {
  Json,
  StoreType,
} from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.6/mod.ts";
export {
  appPaths,
  colors,
  Command,
  CompletionsCommand,
  Conf,
  copy,
  createTruestampClient,
  crypto,
  DB,
  decode,
  decodeUnsafely,
  deleteTokensInConfig,
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
  Table,
  Truestamp,
  validate,
  ValidationError,
  verify,
};

import { DB } from "https://deno.land/x/sqlite@v3.4.1/mod.ts";

import { colors } from "https://deno.land/x/cliffy@v0.25.0/ansi/colors.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.25.0/command/completions/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.25.0/command/help/mod.ts";
import {
  Command,
  EnumType,
  ValidationError,
} from "https://deno.land/x/cliffy@v0.25.0/command/mod.ts";

import Conf from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.6/mod.ts";

import { appPaths } from "https://raw.githubusercontent.com/truestamp/deno-app-paths/v1.1.0/mod.ts";

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
import { decodeUnsafely } from "https://cdn.skypack.dev/@truestamp/id@~v1.3.1?dts";
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js@~v0.13.1?dts";
import { verify } from "https://cdn.skypack.dev/@truestamp/verify@~v0.3.0?dts";
import { createTruestampClient } from "./truestamp.ts";

import { Table } from "https://deno.land/x/cliffy@v0.25.0/table/mod.ts";
