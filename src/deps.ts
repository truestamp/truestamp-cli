// Copyright © 2020-2023 Truestamp Inc. All rights reserved.

import { copy } from "https://deno.land/std@0.173.0/streams/copy.ts";

import { readAllSync } from "https://deno.land/std@0.173.0/streams/read_all.ts";

import { crypto } from "https://deno.land/std@0.173.0/crypto/mod.ts";
import { parse } from "https://deno.land/std@0.173.0/path/mod.ts";

import { sleep } from "https://deno.land/x/sleep@v1.2.1/mod.ts";

import { decode, validate } from "https://deno.land/x/djwt@v2.7/mod.ts";

import { DB } from "https://deno.land/x/sqlite@v3.7.0/mod.ts";

import { colors } from "https://deno.land/x/cliffy@v0.25.7/ansi/colors.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.25.7/command/completions/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.25.7/command/help/mod.ts";
import {
  Command,
  EnumType,
  ValidationError,
} from "https://deno.land/x/cliffy@v0.25.7/command/mod.ts";

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

import { createTruestampClient } from "./truestamp.ts";

import { Table } from "https://deno.land/x/cliffy@v0.25.7/table/mod.ts";

export {
  decode as decodeBase64,
  encode as encodeBase64,
} from "https://deno.land/std@0.173.0/encoding/base64.ts";
export { Status } from "https://deno.land/std@0.173.0/http/http_status.ts";
export type { ITypeInfo } from "https://deno.land/x/cliffy@v0.25.7/flags/mod.ts";
export type { Payload } from "https://deno.land/x/djwt@v2.7/mod.ts";
export type { Row } from "https://deno.land/x/sqlite@v3.7.0/mod.ts";
export type {
  Json,
  StoreType,
} from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.6/mod.ts";
export { ZodError } from "zod";
export { fromZodError } from "zod-validation-error";
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
  validate,
  ValidationError,
};
