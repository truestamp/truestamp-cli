import {
  dirname as pathDirname,
  resolve as pathResolve,
} from "https://deno.land/std@0.97.0/path/mod.ts";

import { existsSync } from "https://deno.land/std@0.97.0/fs/exists.ts";

import envPaths from "https://raw.githubusercontent.com/truestamp/deno-app-paths/main/mod.ts";

const plainObject = () => Object.create(null);

// We don't know what key names the user will want,
// but we do know what types are safe for conversion to JSON.
// The hardest part is defining the nested objects where
// the value can be of many types, including another
// nested object. This workaround permits several levels
// of object nesting. This is a hack that can probably be fixed
// with some better Typescript type definition knowlege.

type DeepNestedObjectType = Record<
  string,
  | (boolean | null | number | ObjectType | string)[]
  | boolean
  | null
  | number
  | string
>;

type NestedObjectType = Record<
  string,
  | (boolean | null | number | ObjectType | string)[]
  | boolean
  | DeepNestedObjectType
  | null
  | number
  | string
>;

type ObjectType = Record<
  string,
  | (boolean | null | number | ObjectType | string)[]
  | boolean
  | NestedObjectType
  | null
  | number
  | string
>;

export type ItemType =
  | (boolean | null | number | ObjectType | string)[]
  | boolean
  | null
  | number
  | ObjectType
  | string;

export type StoreType = Record<string, ItemType>;

export interface ConfigParameters {
  projectName: string;
  configName?: string;
  resetInvalidConfig?: boolean;
  defaults?: StoreType | null;
}

export default class Config {
  private _options: ConfigParameters = {
    projectName: "",
    configName: "config",
    resetInvalidConfig: false,
    defaults: null,
  };

  defaultValues: StoreType = plainObject();

  path: string;

  constructor(options: ConfigParameters) {
    this._options = {
      ...this._options,
      ...options,
    };

    // Were `defaults` provided?
    this.defaultValues = this._options.defaults
      ? this._options.defaults
      : plainObject();

    if (!this._options.projectName || this._options.projectName.trim() === "") {
      throw new Error("the projectName option must be provided and non-empty");
    }

    this._options.projectName = this._options.projectName.trim();

    this.path = pathResolve(
      envPaths(this._options.projectName).config,
      `${this._options.configName}.json`,
    );
  }

  // accessor properties (getter/setter)

  /**
   * Get the number of config items stored.
   * @returns {number} The count of config items
   */
  get size(): number {
    return Object.keys(this.store).length;
  }

  /**
   * Get the path of the config directory.
   * @returns {string} The directory portion of the config path
   */
  get dir(): string {
    return pathDirname(this.path);
  }

  /**
   * Get the contents of the config store, including defaults if present.
   * @returns {StoreType} The config store
   */
  get store(): StoreType {
    try {
      return {
        ...this.defaultValues,
        ...JSON.parse(Deno.readTextFileSync(this.path)),
      };
    } catch (error) {
      switch (error.name) {
        case "SyntaxError":
          // Unable to read the JSON file. Reset it to defaults if that is the
          // desired behavior.
          if (this._options.resetInvalidConfig) {
            this.reset();
            return { ...this.defaultValues };
          }
          break;
        case "NotFound":
          return { ...this.defaultValues, ...plainObject() };
      }

      throw error;
    }
  }

  /**
   * Set the contents of the config store to an Object.
   *
   * @param {StoreType} data
   * @returns {void}
   */
  set store(data: StoreType) {
    if (!existsSync(pathDirname(this.path))) {
      Deno.mkdirSync(pathDirname(this.path), { recursive: true });
    }

    Deno.writeTextFileSync(this.path, JSON.stringify(data, null, 2));
  }

  /**
   * Get the config store parameters.
   *
   * @returns {ConfigParameters}
   */
  get options(): ConfigParameters {
    return this._options;
  }

  /**
   * Returns boolean whether `key` is present in the config store.
   *
   * @param {string} key The key to search for.
   * @returns {boolean} Key exists in config store?
   */
  has(key: string): boolean {
    return key in this.store;
  }

  //

  /**
   * Destructively removes any existing config file and resets all
   * keys to defaults if present, writing them to a new config.
   *
   * If no defaults are present no new config file will be created.
   *
   * @returns {void}
   */
  reset(): void {
    if (this.path && existsSync(this.path)) {
      Deno.removeSync(this.path, { recursive: true });
    }

    // There are no default values. Just exit.
    if (Object.keys(this.defaultValues).length === 0) {
      return;
    }

    // There are default values, iterate and save each.
    Object.entries(this.defaultValues).forEach(([key, value]) => {
      // console.log(`setting ${key}:${value}`);
      this.set(key, value);
    });

    return;
  }

  /**
   * Destructively reset one or more keys to defaults if they exist.
   *
   * If no defaults are present then this will be a no-op for all
   * provided keys.
   *
   * If defaults are present then each key that matches one in defaults
   * will be overwritten with the default value.
   *
   * @param {string[]} keys An Array of string keys to reset to defaults.
   * @returns {void}
   */
  resetKeys(keys: string[]): void {
    if (Object.keys(this.defaultValues).length === 0) {
      return;
    }

    for (const key of keys) {
      if (this.defaultValues && key in this.defaultValues) {
        this.set(key, this.defaultValues[key]);
      }
    }
  }

  /**
   * Destructively remove a single item from the config store.
   *
   * @param {string} key The key to delete from the config store.
   * @returns {void}
   */
  delete(key: string): void {
    const { store } = this;
    if (store && key in store) {
      delete store[key];
      this.store = store;
    }
  }

  /**
   * Get a single item from the config store.
   *
   * @param {string} key The key to get from the config store.
   * @returns {ItemType} A single ItemType item.
   */
  get(key: string): ItemType {
    if (this.store && key in this.store) {
      return this.store[key];
    } else if (
      this.defaultValues && key in this.defaultValues
    ) {
      return this.defaultValues[key];
    } else {
      return null;
    }
  }

  /**
   * Set a single item into the config store.
   *
   * @param {string} key The key to write to the config store.
   * @param {ItemType} value The value to write to the config store.
   * @returns {void} void.
   */
  set(
    key: string,
    value: ItemType,
  ): void {
    const { store } = this;

    const innerSet = (
      key: string,
      value: ItemType,
    ) => {
      store[key] = value;
    };

    innerSet(key, value);
    this.store = store;
  }

  /**
   * Set multiple items into the config store.
   *
   * @param {StoreType} data The Object to write to the config store.
   * @returns {void} void.
   */
  setObject(
    data: StoreType,
  ): void {
    for (const [key, value] of Object.entries(data)) {
      this.set(key, value);
    }
  }

  // Allow Conf instance to be iterable
  *[Symbol.iterator]() {
    for (const [key, value] of Object.entries(this.store)) {
      yield [key, value];
    }
  }
}
