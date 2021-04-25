import configDir from "https://deno.land/x/config_dir/mod.ts";
import { sleep } from "https://deno.land/x/sleep/mod.ts";
import { colors } from "https://deno.land/x/cliffy@v0.18.2/ansi/colors.ts";
import { Command } from "https://deno.land/x/cliffy@v0.18.2/command/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.18.2/command/help/mod.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.18.2/command/completions/mod.ts";

import { getAccessTokenWithPrompts } from "./deviceflow.ts";

import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js?dts";

export {
  colors,
  Command,
  CompletionsCommand,
  configDir,
  getAccessTokenWithPrompts,
  HelpCommand,
  sleep,
  Truestamp,
};

export type { ITypeInfo } from "https://deno.land/x/cliffy/flags/mod.ts";
