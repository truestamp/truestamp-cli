import process from "https://deno.land/std@0.101.0/node/process.ts";
import { CredentialsProviderError } from "../property-provider/mod.ts";
import { Provider } from "../types/mod.ts";

export type GetterFromEnv<T> = (env: {[key: string]: string}) => T | undefined;

/**
 * Get config value given the environment variable name or getter from
 * environment variable.
 */
export const fromEnv =
  <T = string>(envVarSelector: GetterFromEnv<T>): Provider<T> =>
  async () => {
    try {
      const config = envVarSelector(process.env);
      if (config === undefined) {
        throw new Error();
      }
      return config as T;
    } catch (e) {
      throw new CredentialsProviderError(
        e.message || `Cannot load config from environment variables with getter: ${envVarSelector}`
      );
    }
  };
