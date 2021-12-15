// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { sleep } from "https://deno.land/x/sleep@v1.2.1/mod.ts";

import { decode, validate } from "https://deno.land/x/djwt@v2.4/mod.ts";

import { colors } from "https://deno.land/x/cliffy@v0.20.1/ansi/colors.ts";
import {
  Command,
  EnumType,
  ValidationError,
} from "https://deno.land/x/cliffy@v0.20.1/command/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.20.1/command/help/mod.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.20.1/command/completions/mod.ts";

import Conf from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.2-beta/mod.ts";

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
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js@~v0.5.1?dts";
import { createTruestampClient } from "./truestamp.ts";

export {
  colors,
  Command,
  CompletionsCommand,
  Conf,
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
  setConfigKeyForEnv,
  sleep,
  Truestamp,
  validate,
  ValidationError,
};

export type { ITypeInfo } from "https://deno.land/x/cliffy@v0.20.1/flags/mod.ts";
export type { Payload } from "https://deno.land/x/djwt@v2.4/mod.ts";

export type {
  ItemType,
  StoreType,
} from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.2-beta/mod.ts";
