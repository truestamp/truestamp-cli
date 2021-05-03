import configDir from "https://deno.land/x/config_dir/mod.ts"
import { sleep } from "https://deno.land/x/sleep/mod.ts"
import loadJsonFile from "https://deno.land/x/load_json_file@v1.0.0/mod.ts"
import { decode, validate } from "https://deno.land/x/djwt@v2.2/mod.ts"

import { colors } from "https://deno.land/x/cliffy@v0.18.2/ansi/colors.ts"
import { Command } from "https://deno.land/x/cliffy@v0.18.2/command/mod.ts"
import { HelpCommand } from "https://deno.land/x/cliffy@v0.18.2/command/help/mod.ts"
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.18.2/command/completions/mod.ts"

import {
  deleteSavedTokens,
  getSavedAccessToken,
  getSavedRefreshToken,
  getAccessTokenWithPrompts,
  getSavedIdTokenPayload,
} from "./auth.ts"

// Get SkyPack pinned URL using `curl -i "https://cdn.skypack.dev/@truestamp/truestamp-js?dts"`
import Truestamp from "https://cdn.skypack.dev/pin/@truestamp/truestamp-js@v0.0.39-s9peUMKFsrLpG0Cr1TTw/mode=imports/optimized/@truestamp/truestamp-js.js"
import { createTruestampClient } from "./truestamp.ts"

export {
  colors,
  Command,
  CompletionsCommand,
  configDir,
  decode,
  deleteSavedTokens,
  getSavedAccessToken,
  getSavedRefreshToken,
  getAccessTokenWithPrompts,
  HelpCommand,
  loadJsonFile,
  sleep,
  Truestamp,
  validate,
  getSavedIdTokenPayload,
  createTruestampClient,
}

export type { ITypeInfo } from "https://deno.land/x/cliffy/flags/mod.ts"
export type { Payload } from "https://deno.land/x/djwt@v2.2/mod.ts"
