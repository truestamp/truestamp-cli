import { fromStatic as convertToProvider } from "../property-provider/mod.ts";
import { Provider } from "../types/mod.ts";

export type FromStaticConfig<T> = T | (() => T);
type Getter<T> = () => T;
const isFunction = <T>(func: FromStaticConfig<T>): func is Getter<T> => typeof func === "function";

export const fromStatic = <T>(defaultValue: FromStaticConfig<T>): Provider<T> =>
  isFunction(defaultValue) ? async () => defaultValue() : convertToProvider(defaultValue);
