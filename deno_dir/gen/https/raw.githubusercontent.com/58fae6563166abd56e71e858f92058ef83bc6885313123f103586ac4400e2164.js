import { dirname, resolve } from "https://deno.land/std@0.144.0/path/mod.ts";
import { appPaths } from "https://raw.githubusercontent.com/truestamp/deno-app-paths/v1.1.0/mod.ts";
const plainObject = ()=>Object.create(null);
export default class Config {
    _options = {
        projectName: "",
        configName: "config",
        resetInvalidConfig: false,
        defaults: null
    };
    defaultValues = plainObject();
    path;
    constructor(options){
        this._options = {
            ...this._options,
            ...options
        };
        // Were `defaults` provided?
        this.defaultValues = this._options.defaults ? this._options.defaults : plainObject();
        if (!this._options.projectName || this._options.projectName.trim() === "") {
            throw new Error("the projectName option must be provided and non-empty");
        }
        this._options.projectName = this._options.projectName.trim();
        this.path = resolve(appPaths(this._options.projectName).config, `${this._options.configName}.json`);
    }
    // accessor properties (getter/setter)
    /**
   * Get the number of config items stored.
   * @returns {number} The count of config items
   */ get size() {
        return Object.keys(this.store).length;
    }
    /**
   * Get the path of the config directory.
   * @returns {string} The directory portion of the config path
   */ get dir() {
        return dirname(this.path);
    }
    /**
   * Get the contents of the config store, including defaults if present.
   * @returns {StoreType} The config store
   */ get store() {
        try {
            return {
                ...this.defaultValues,
                ...JSON.parse(Deno.readTextFileSync(this.path))
            };
        } catch (error) {
            switch(error.name){
                case "SyntaxError":
                    // Unable to read the JSON file. Reset it to defaults if that is the
                    // desired behavior.
                    if (this._options.resetInvalidConfig) {
                        this.reset();
                        return {
                            ...this.defaultValues
                        };
                    }
                    break;
                case "NotFound":
                    return {
                        ...this.defaultValues,
                        ...plainObject()
                    };
            }
            throw error;
        }
    }
    /**
   * Set the contents of the config store to an Object.
   *
   * @param {StoreType} data
   * @returns {void}
   */ set store(data) {
        Deno.mkdirSync(dirname(this.path), {
            recursive: true
        });
        Deno.writeTextFileSync(this.path, JSON.stringify(data, null, 2));
    }
    /**
   * Get the config store parameters.
   *
   * @returns {ConfigParameters}
   */ get options() {
        return this._options;
    }
    /**
   * Returns boolean whether `key` is present in the config store.
   *
   * @param {string} key The key to search for.
   * @returns {boolean} Key exists in config store?
   */ has(key) {
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
   */ reset() {
        Deno.removeSync(this.path, {
            recursive: true
        });
        // There are no default values. Just exit.
        if (Object.keys(this.defaultValues).length === 0) {
            return;
        }
        // There are default values, iterate and save each.
        Object.entries(this.defaultValues).forEach(([key, value])=>{
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
   */ resetKeys(keys) {
        if (Object.keys(this.defaultValues).length === 0) {
            return;
        }
        for (const key of keys){
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
   */ delete(key) {
        const { store  } = this;
        if (store && key in store) {
            delete store[key];
            this.store = store;
        }
    }
    /**
   * Get a single item from the config store.
   *
   * @param {string} key The key to get from the config store.
   * @returns {Json} Json.
   */ get(key) {
        if (this.store && key in this.store) {
            return this.store[key];
        } else if (this.defaultValues && key in this.defaultValues) {
            return this.defaultValues[key];
        } else {
            return null;
        }
    }
    /**
   * Set a single item into the config store.
   *
   * @param {string} key The key to write to the config store.
   * @param {Json} value The value to write to the config store.
   * @returns {void} void.
   */ set(key, value) {
        const { store  } = this;
        const innerSet = (key, value)=>{
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
   */ setObject(data) {
        for (const [key, value] of Object.entries(data)){
            this.set(key, value);
        }
    }
    // Allow Conf instance to be iterable
    *[Symbol.iterator]() {
        for (const [key, value] of Object.entries(this.store)){
            yield [
                key,
                value
            ];
        }
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vcmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbS90cnVlc3RhbXAvZGVuby1jb25mL3YxLjAuNi9tb2QudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHtcbiAgZGlybmFtZSxcbiAgcmVzb2x2ZSxcbn0gZnJvbSBcImh0dHBzOi8vZGVuby5sYW5kL3N0ZEAwLjE0NC4wL3BhdGgvbW9kLnRzXCI7XG5cbmltcG9ydCB7IGFwcFBhdGhzIH0gZnJvbSBcImh0dHBzOi8vcmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbS90cnVlc3RhbXAvZGVuby1hcHAtcGF0aHMvdjEuMS4wL21vZC50c1wiO1xuXG5jb25zdCBwbGFpbk9iamVjdCA9ICgpID0+IE9iamVjdC5jcmVhdGUobnVsbCk7XG5cbi8vIFJlY3Vyc2l2ZSBKU09OIHR5cGU6IGh0dHBzOi8vZGV2YmxvZ3MubWljcm9zb2Z0LmNvbS90eXBlc2NyaXB0L2Fubm91bmNpbmctdHlwZXNjcmlwdC0zLTcvI21vcmUtcmVjdXJzaXZlLXR5cGUtYWxpYXNlc1xuZXhwb3J0IHR5cGUgSnNvbiA9XG4gIHwgc3RyaW5nXG4gIHwgbnVtYmVyXG4gIHwgYm9vbGVhblxuICB8IG51bGxcbiAgfCBKc29uW11cbiAgfCB7IFtrZXk6IHN0cmluZ106IEpzb24gfVxuXG5leHBvcnQgdHlwZSBTdG9yZVR5cGUgPSBSZWNvcmQ8c3RyaW5nLCBKc29uPjtcblxuZXhwb3J0IGludGVyZmFjZSBDb25maWdQYXJhbWV0ZXJzIHtcbiAgcHJvamVjdE5hbWU6IHN0cmluZztcbiAgY29uZmlnTmFtZT86IHN0cmluZztcbiAgcmVzZXRJbnZhbGlkQ29uZmlnPzogYm9vbGVhbjtcbiAgZGVmYXVsdHM/OiBTdG9yZVR5cGUgfCBudWxsO1xufVxuXG5leHBvcnQgZGVmYXVsdCBjbGFzcyBDb25maWcge1xuICBwcml2YXRlIF9vcHRpb25zOiBDb25maWdQYXJhbWV0ZXJzID0ge1xuICAgIHByb2plY3ROYW1lOiBcIlwiLFxuICAgIGNvbmZpZ05hbWU6IFwiY29uZmlnXCIsXG4gICAgcmVzZXRJbnZhbGlkQ29uZmlnOiBmYWxzZSxcbiAgICBkZWZhdWx0czogbnVsbCxcbiAgfTtcblxuICBkZWZhdWx0VmFsdWVzOiBTdG9yZVR5cGUgPSBwbGFpbk9iamVjdCgpO1xuXG4gIHBhdGg6IHN0cmluZztcblxuICBjb25zdHJ1Y3RvciAob3B0aW9uczogQ29uZmlnUGFyYW1ldGVycykge1xuICAgIHRoaXMuX29wdGlvbnMgPSB7XG4gICAgICAuLi50aGlzLl9vcHRpb25zLFxuICAgICAgLi4ub3B0aW9ucyxcbiAgICB9O1xuXG4gICAgLy8gV2VyZSBgZGVmYXVsdHNgIHByb3ZpZGVkP1xuICAgIHRoaXMuZGVmYXVsdFZhbHVlcyA9IHRoaXMuX29wdGlvbnMuZGVmYXVsdHNcbiAgICAgID8gdGhpcy5fb3B0aW9ucy5kZWZhdWx0c1xuICAgICAgOiBwbGFpbk9iamVjdCgpO1xuXG4gICAgaWYgKCF0aGlzLl9vcHRpb25zLnByb2plY3ROYW1lIHx8IHRoaXMuX29wdGlvbnMucHJvamVjdE5hbWUudHJpbSgpID09PSBcIlwiKSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoXCJ0aGUgcHJvamVjdE5hbWUgb3B0aW9uIG11c3QgYmUgcHJvdmlkZWQgYW5kIG5vbi1lbXB0eVwiKTtcbiAgICB9XG5cbiAgICB0aGlzLl9vcHRpb25zLnByb2plY3ROYW1lID0gdGhpcy5fb3B0aW9ucy5wcm9qZWN0TmFtZS50cmltKCk7XG5cbiAgICB0aGlzLnBhdGggPSByZXNvbHZlKFxuICAgICAgYXBwUGF0aHModGhpcy5fb3B0aW9ucy5wcm9qZWN0TmFtZSkuY29uZmlnLFxuICAgICAgYCR7dGhpcy5fb3B0aW9ucy5jb25maWdOYW1lfS5qc29uYCxcbiAgICApO1xuICB9XG5cbiAgLy8gYWNjZXNzb3IgcHJvcGVydGllcyAoZ2V0dGVyL3NldHRlcilcblxuICAvKipcbiAgICogR2V0IHRoZSBudW1iZXIgb2YgY29uZmlnIGl0ZW1zIHN0b3JlZC5cbiAgICogQHJldHVybnMge251bWJlcn0gVGhlIGNvdW50IG9mIGNvbmZpZyBpdGVtc1xuICAgKi9cbiAgZ2V0IHNpemUoKTogbnVtYmVyIHtcbiAgICByZXR1cm4gT2JqZWN0LmtleXModGhpcy5zdG9yZSkubGVuZ3RoO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCB0aGUgcGF0aCBvZiB0aGUgY29uZmlnIGRpcmVjdG9yeS5cbiAgICogQHJldHVybnMge3N0cmluZ30gVGhlIGRpcmVjdG9yeSBwb3J0aW9uIG9mIHRoZSBjb25maWcgcGF0aFxuICAgKi9cbiAgZ2V0IGRpcigpOiBzdHJpbmcge1xuICAgIHJldHVybiBkaXJuYW1lKHRoaXMucGF0aCk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IHRoZSBjb250ZW50cyBvZiB0aGUgY29uZmlnIHN0b3JlLCBpbmNsdWRpbmcgZGVmYXVsdHMgaWYgcHJlc2VudC5cbiAgICogQHJldHVybnMge1N0b3JlVHlwZX0gVGhlIGNvbmZpZyBzdG9yZVxuICAgKi9cbiAgZ2V0IHN0b3JlKCk6IFN0b3JlVHlwZSB7XG4gICAgdHJ5IHtcbiAgICAgIHJldHVybiB7XG4gICAgICAgIC4uLnRoaXMuZGVmYXVsdFZhbHVlcyxcbiAgICAgICAgLi4uSlNPTi5wYXJzZShEZW5vLnJlYWRUZXh0RmlsZVN5bmModGhpcy5wYXRoKSksXG4gICAgICB9O1xuICAgIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgICBzd2l0Y2ggKGVycm9yLm5hbWUpIHtcbiAgICAgICAgY2FzZSBcIlN5bnRheEVycm9yXCI6XG4gICAgICAgICAgLy8gVW5hYmxlIHRvIHJlYWQgdGhlIEpTT04gZmlsZS4gUmVzZXQgaXQgdG8gZGVmYXVsdHMgaWYgdGhhdCBpcyB0aGVcbiAgICAgICAgICAvLyBkZXNpcmVkIGJlaGF2aW9yLlxuICAgICAgICAgIGlmICh0aGlzLl9vcHRpb25zLnJlc2V0SW52YWxpZENvbmZpZykge1xuICAgICAgICAgICAgdGhpcy5yZXNldCgpO1xuICAgICAgICAgICAgcmV0dXJuIHsgLi4udGhpcy5kZWZhdWx0VmFsdWVzIH07XG4gICAgICAgICAgfVxuICAgICAgICAgIGJyZWFrO1xuICAgICAgICBjYXNlIFwiTm90Rm91bmRcIjpcbiAgICAgICAgICByZXR1cm4geyAuLi50aGlzLmRlZmF1bHRWYWx1ZXMsIC4uLnBsYWluT2JqZWN0KCkgfTtcbiAgICAgIH1cblxuICAgICAgdGhyb3cgZXJyb3I7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFNldCB0aGUgY29udGVudHMgb2YgdGhlIGNvbmZpZyBzdG9yZSB0byBhbiBPYmplY3QuXG4gICAqXG4gICAqIEBwYXJhbSB7U3RvcmVUeXBlfSBkYXRhXG4gICAqIEByZXR1cm5zIHt2b2lkfVxuICAgKi9cbiAgc2V0IHN0b3JlKGRhdGE6IFN0b3JlVHlwZSkge1xuICAgIERlbm8ubWtkaXJTeW5jKGRpcm5hbWUodGhpcy5wYXRoKSwgeyByZWN1cnNpdmU6IHRydWUgfSk7XG4gICAgRGVuby53cml0ZVRleHRGaWxlU3luYyh0aGlzLnBhdGgsIEpTT04uc3RyaW5naWZ5KGRhdGEsIG51bGwsIDIpKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgdGhlIGNvbmZpZyBzdG9yZSBwYXJhbWV0ZXJzLlxuICAgKlxuICAgKiBAcmV0dXJucyB7Q29uZmlnUGFyYW1ldGVyc31cbiAgICovXG4gIGdldCBvcHRpb25zKCk6IENvbmZpZ1BhcmFtZXRlcnMge1xuICAgIHJldHVybiB0aGlzLl9vcHRpb25zO1xuICB9XG5cbiAgLyoqXG4gICAqIFJldHVybnMgYm9vbGVhbiB3aGV0aGVyIGBrZXlgIGlzIHByZXNlbnQgaW4gdGhlIGNvbmZpZyBzdG9yZS5cbiAgICpcbiAgICogQHBhcmFtIHtzdHJpbmd9IGtleSBUaGUga2V5IHRvIHNlYXJjaCBmb3IuXG4gICAqIEByZXR1cm5zIHtib29sZWFufSBLZXkgZXhpc3RzIGluIGNvbmZpZyBzdG9yZT9cbiAgICovXG4gIGhhcyhrZXk6IHN0cmluZyk6IGJvb2xlYW4ge1xuICAgIHJldHVybiBrZXkgaW4gdGhpcy5zdG9yZTtcbiAgfVxuXG4gIC8vXG5cbiAgLyoqXG4gICAqIERlc3RydWN0aXZlbHkgcmVtb3ZlcyBhbnkgZXhpc3RpbmcgY29uZmlnIGZpbGUgYW5kIHJlc2V0cyBhbGxcbiAgICoga2V5cyB0byBkZWZhdWx0cyBpZiBwcmVzZW50LCB3cml0aW5nIHRoZW0gdG8gYSBuZXcgY29uZmlnLlxuICAgKlxuICAgKiBJZiBubyBkZWZhdWx0cyBhcmUgcHJlc2VudCBubyBuZXcgY29uZmlnIGZpbGUgd2lsbCBiZSBjcmVhdGVkLlxuICAgKlxuICAgKiBAcmV0dXJucyB7dm9pZH1cbiAgICovXG4gIHJlc2V0KCk6IHZvaWQge1xuICAgIERlbm8ucmVtb3ZlU3luYyh0aGlzLnBhdGgsIHsgcmVjdXJzaXZlOiB0cnVlIH0pO1xuXG4gICAgLy8gVGhlcmUgYXJlIG5vIGRlZmF1bHQgdmFsdWVzLiBKdXN0IGV4aXQuXG4gICAgaWYgKE9iamVjdC5rZXlzKHRoaXMuZGVmYXVsdFZhbHVlcykubGVuZ3RoID09PSAwKSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgLy8gVGhlcmUgYXJlIGRlZmF1bHQgdmFsdWVzLCBpdGVyYXRlIGFuZCBzYXZlIGVhY2guXG4gICAgT2JqZWN0LmVudHJpZXModGhpcy5kZWZhdWx0VmFsdWVzKS5mb3JFYWNoKChba2V5LCB2YWx1ZV0pID0+IHtcbiAgICAgIC8vIGNvbnNvbGUubG9nKGBzZXR0aW5nICR7a2V5fToke3ZhbHVlfWApO1xuICAgICAgdGhpcy5zZXQoa2V5LCB2YWx1ZSk7XG4gICAgfSk7XG5cbiAgICByZXR1cm47XG4gIH1cblxuICAvKipcbiAgICogRGVzdHJ1Y3RpdmVseSByZXNldCBvbmUgb3IgbW9yZSBrZXlzIHRvIGRlZmF1bHRzIGlmIHRoZXkgZXhpc3QuXG4gICAqXG4gICAqIElmIG5vIGRlZmF1bHRzIGFyZSBwcmVzZW50IHRoZW4gdGhpcyB3aWxsIGJlIGEgbm8tb3AgZm9yIGFsbFxuICAgKiBwcm92aWRlZCBrZXlzLlxuICAgKlxuICAgKiBJZiBkZWZhdWx0cyBhcmUgcHJlc2VudCB0aGVuIGVhY2gga2V5IHRoYXQgbWF0Y2hlcyBvbmUgaW4gZGVmYXVsdHNcbiAgICogd2lsbCBiZSBvdmVyd3JpdHRlbiB3aXRoIHRoZSBkZWZhdWx0IHZhbHVlLlxuICAgKlxuICAgKiBAcGFyYW0ge3N0cmluZ1tdfSBrZXlzIEFuIEFycmF5IG9mIHN0cmluZyBrZXlzIHRvIHJlc2V0IHRvIGRlZmF1bHRzLlxuICAgKiBAcmV0dXJucyB7dm9pZH1cbiAgICovXG4gIHJlc2V0S2V5cyhrZXlzOiBzdHJpbmdbXSk6IHZvaWQge1xuICAgIGlmIChPYmplY3Qua2V5cyh0aGlzLmRlZmF1bHRWYWx1ZXMpLmxlbmd0aCA9PT0gMCkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIGZvciAoY29uc3Qga2V5IG9mIGtleXMpIHtcbiAgICAgIGlmICh0aGlzLmRlZmF1bHRWYWx1ZXMgJiYga2V5IGluIHRoaXMuZGVmYXVsdFZhbHVlcykge1xuICAgICAgICB0aGlzLnNldChrZXksIHRoaXMuZGVmYXVsdFZhbHVlc1trZXldKTtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogRGVzdHJ1Y3RpdmVseSByZW1vdmUgYSBzaW5nbGUgaXRlbSBmcm9tIHRoZSBjb25maWcgc3RvcmUuXG4gICAqXG4gICAqIEBwYXJhbSB7c3RyaW5nfSBrZXkgVGhlIGtleSB0byBkZWxldGUgZnJvbSB0aGUgY29uZmlnIHN0b3JlLlxuICAgKiBAcmV0dXJucyB7dm9pZH1cbiAgICovXG4gIGRlbGV0ZShrZXk6IHN0cmluZyk6IHZvaWQge1xuICAgIGNvbnN0IHsgc3RvcmUgfSA9IHRoaXM7XG4gICAgaWYgKHN0b3JlICYmIGtleSBpbiBzdG9yZSkge1xuICAgICAgZGVsZXRlIHN0b3JlW2tleV07XG4gICAgICB0aGlzLnN0b3JlID0gc3RvcmU7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIEdldCBhIHNpbmdsZSBpdGVtIGZyb20gdGhlIGNvbmZpZyBzdG9yZS5cbiAgICpcbiAgICogQHBhcmFtIHtzdHJpbmd9IGtleSBUaGUga2V5IHRvIGdldCBmcm9tIHRoZSBjb25maWcgc3RvcmUuXG4gICAqIEByZXR1cm5zIHtKc29ufSBKc29uLlxuICAgKi9cbiAgZ2V0KGtleTogc3RyaW5nKTogSnNvbiB7XG4gICAgaWYgKHRoaXMuc3RvcmUgJiYga2V5IGluIHRoaXMuc3RvcmUpIHtcbiAgICAgIHJldHVybiB0aGlzLnN0b3JlW2tleV07XG4gICAgfSBlbHNlIGlmIChcbiAgICAgIHRoaXMuZGVmYXVsdFZhbHVlcyAmJiBrZXkgaW4gdGhpcy5kZWZhdWx0VmFsdWVzXG4gICAgKSB7XG4gICAgICByZXR1cm4gdGhpcy5kZWZhdWx0VmFsdWVzW2tleV07XG4gICAgfSBlbHNlIHtcbiAgICAgIHJldHVybiBudWxsO1xuICAgIH1cbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgYSBzaW5nbGUgaXRlbSBpbnRvIHRoZSBjb25maWcgc3RvcmUuXG4gICAqXG4gICAqIEBwYXJhbSB7c3RyaW5nfSBrZXkgVGhlIGtleSB0byB3cml0ZSB0byB0aGUgY29uZmlnIHN0b3JlLlxuICAgKiBAcGFyYW0ge0pzb259IHZhbHVlIFRoZSB2YWx1ZSB0byB3cml0ZSB0byB0aGUgY29uZmlnIHN0b3JlLlxuICAgKiBAcmV0dXJucyB7dm9pZH0gdm9pZC5cbiAgICovXG4gIHNldChcbiAgICBrZXk6IHN0cmluZyxcbiAgICB2YWx1ZTogSnNvbixcbiAgKTogdm9pZCB7XG4gICAgY29uc3QgeyBzdG9yZSB9ID0gdGhpcztcblxuICAgIGNvbnN0IGlubmVyU2V0ID0gKFxuICAgICAga2V5OiBzdHJpbmcsXG4gICAgICB2YWx1ZTogSnNvbixcbiAgICApID0+IHtcbiAgICAgIHN0b3JlW2tleV0gPSB2YWx1ZTtcbiAgICB9O1xuXG4gICAgaW5uZXJTZXQoa2V5LCB2YWx1ZSk7XG4gICAgdGhpcy5zdG9yZSA9IHN0b3JlO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBtdWx0aXBsZSBpdGVtcyBpbnRvIHRoZSBjb25maWcgc3RvcmUuXG4gICAqXG4gICAqIEBwYXJhbSB7U3RvcmVUeXBlfSBkYXRhIFRoZSBPYmplY3QgdG8gd3JpdGUgdG8gdGhlIGNvbmZpZyBzdG9yZS5cbiAgICogQHJldHVybnMge3ZvaWR9IHZvaWQuXG4gICAqL1xuICBzZXRPYmplY3QoXG4gICAgZGF0YTogU3RvcmVUeXBlLFxuICApOiB2b2lkIHtcbiAgICBmb3IgKGNvbnN0IFtrZXksIHZhbHVlXSBvZiBPYmplY3QuZW50cmllcyhkYXRhKSkge1xuICAgICAgdGhpcy5zZXQoa2V5LCB2YWx1ZSk7XG4gICAgfVxuICB9XG5cbiAgLy8gQWxsb3cgQ29uZiBpbnN0YW5jZSB0byBiZSBpdGVyYWJsZVxuICAqW1N5bWJvbC5pdGVyYXRvcl0oKSB7XG4gICAgZm9yIChjb25zdCBba2V5LCB2YWx1ZV0gb2YgT2JqZWN0LmVudHJpZXModGhpcy5zdG9yZSkpIHtcbiAgICAgIHlpZWxkIFtrZXksIHZhbHVlXTtcbiAgICB9XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUNFLE9BQU8sRUFDUCxPQUFPLFFBQ0YsMkNBQTJDLENBQUM7QUFFbkQsU0FBUyxRQUFRLFFBQVEsMEVBQTBFLENBQUM7QUFFcEcsTUFBTSxXQUFXLEdBQUcsSUFBTSxNQUFNLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxBQUFDO0FBb0I5QyxlQUFlLE1BQU0sTUFBTTtJQUN6QixBQUFRLFFBQVEsR0FBcUI7UUFDbkMsV0FBVyxFQUFFLEVBQUU7UUFDZixVQUFVLEVBQUUsUUFBUTtRQUNwQixrQkFBa0IsRUFBRSxLQUFLO1FBQ3pCLFFBQVEsRUFBRSxJQUFJO0tBQ2YsQ0FBQztJQUVGLGFBQWEsR0FBYyxXQUFXLEVBQUUsQ0FBQztJQUV6QyxJQUFJLENBQVM7SUFFYixZQUFhLE9BQXlCLENBQUU7UUFDdEMsSUFBSSxDQUFDLFFBQVEsR0FBRztZQUNkLEdBQUcsSUFBSSxDQUFDLFFBQVE7WUFDaEIsR0FBRyxPQUFPO1NBQ1gsQ0FBQztRQUVGLDRCQUE0QjtRQUM1QixJQUFJLENB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