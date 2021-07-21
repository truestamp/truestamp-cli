import { chain, memoize } from "../property-provider/mod.ts";
import { Provider } from "../types/mod.ts";

import { fromEnv, GetterFromEnv } from "./fromEnv.ts";
import { fromSharedConfigFiles, GetterFromConfig, SharedConfigInit } from "./fromSharedConfigFiles.ts";
import { fromStatic, FromStaticConfig } from "./fromStatic.ts";

export type LocalConfigOptions = SharedConfigInit;

export interface LoadedConfigSelectors<T> {
  /**
   * A getter function getting the config values from all the environment
   * variables.
   */
  environmentVariableSelector: GetterFromEnv<T>;
  /**
   * A getter function getting config values associated with the inferred
   * profile from shared INI files
   */
  configFileSelector: GetterFromConfig<T>;
  /**
   * Default value or getter
   */
  default: FromStaticConfig<T>;
}

export const loadConfig = <T = string>(
  { environmentVariableSelector, configFileSelector, default: defaultValue }: LoadedConfigSelectors<T>,
  configuration: LocalConfigOptions = {}
): Provider<T> =>
  memoize(
    chain(
      fromEnv(environmentVariableSelector),
      fromSharedConfigFiles(configFileSelector, configuration),
      fromStatic(defaultValue)
    )
  );
