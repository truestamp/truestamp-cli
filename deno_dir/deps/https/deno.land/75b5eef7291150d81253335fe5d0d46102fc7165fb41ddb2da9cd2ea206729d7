import process from "https://deno.land/std@0.101.0/node/process.ts";
import {
  loadSharedConfigFiles,
  ParsedIniData,
  SharedConfigFiles,
  SharedConfigInit,
} from "../shared-ini-file-loader/mod.ts";

export const ENV_PROFILE = "AWS_PROFILE";
export const DEFAULT_PROFILE = "default";

export interface SourceProfileInit extends SharedConfigInit {
  /**
   * The configuration profile to use.
   */
  profile?: string;

  /**
   * A promise that will be resolved with loaded and parsed credentials files.
   * Used to avoid loading shared config files multiple times.
   *
   * @internal
   */
  loadedConfig?: Promise<SharedConfigFiles>;
}

/**
 * Load profiles from credentials and config INI files and normalize them into a
 * single profile list.
 *
 * @internal
 */
export const parseKnownFiles = async (init: SourceProfileInit): Promise<ParsedIniData> => {
  const { loadedConfig = loadSharedConfigFiles(init) } = init;

  const parsedFiles = await loadedConfig;
  return {
    ...parsedFiles.configFile,
    ...parsedFiles.credentialsFile,
  };
};

/**
 * @internal
 */
export const getMasterProfileName = (init: { profile?: string }): string =>
  init.profile || process.env[ENV_PROFILE] || DEFAULT_PROFILE;
