import process from "https://deno.land/std@0.93.0/node/process.ts";
import { ProviderError } from "../property-provider/mod.ts";
import {
  loadSharedConfigFiles,
  Profile,
  SharedConfigFiles,
  SharedConfigInit as BaseSharedConfigInit,
} from "../shared-ini-file-loader/mod.ts";
import { Provider } from "../types/mod.ts";

const DEFAULT_PROFILE = "default";
export const ENV_PROFILE = "AWS_PROFILE";

export interface SharedConfigInit extends BaseSharedConfigInit {
  /**
   * The configuration profile to use.
   */
  profile?: string;

  /**
   * The preferred shared ini file to load the config. "config" option refers to
   * the shared config file(defaults to `~/.aws/config`). "credentials" option
   * refers to the shared credentials file(defaults to `~/.aws/credentials`)
   */
  preferredFile?: "config" | "credentials";

  /**
   * A promise that will be resolved with loaded and parsed credentials files.
   * Used to avoid loading shared config files multiple times.
   *
   * @internal
   */
  loadedConfig?: Promise<SharedConfigFiles>;
}

export type GetterFromConfig<T> = (profile: Profile) => T | undefined;

/**
 * Get config value from the shared config files with inferred profile name.
 */
export const fromSharedConfigFiles = <T = string>(
  configSelector: GetterFromConfig<T>,
  { preferredFile = "config", ...init }: SharedConfigInit = {}
): Provider<T> => async () => {
  const { loadedConfig = loadSharedConfigFiles(init), profile = process.env[ENV_PROFILE] || DEFAULT_PROFILE } = init;

  const { configFile, credentialsFile } = await loadedConfig;

  const profileFromCredentials = credentialsFile[profile] || {};
  const profileFromConfig = configFile[profile] || {};
  const mergedProfile =
    preferredFile === "config"
      ? { ...profileFromCredentials, ...profileFromConfig }
      : { ...profileFromConfig, ...profileFromCredentials };

  try {
    const configValue = configSelector(mergedProfile);
    if (configValue === undefined) {
      throw new Error();
    }
    return configValue;
  } catch (e) {
    throw new ProviderError(
      e.message || `Cannot load config for profile ${profile} in SDK configuration files with getter: ${configSelector}`
    );
  }
};
