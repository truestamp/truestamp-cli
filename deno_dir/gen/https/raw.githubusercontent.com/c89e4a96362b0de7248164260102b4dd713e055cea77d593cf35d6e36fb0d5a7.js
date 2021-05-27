import { dirname as pathDirname, resolve as pathResolve, } from "https://deno.land/std@0.97.0/path/mod.ts";
import { existsSync } from "https://deno.land/std@0.97.0/fs/exists.ts";
import envPaths from "https://raw.githubusercontent.com/truestamp/deno-app-paths/main/mod.ts";
const plainObject = () => Object.create(null);
export default class Config {
    _options = {
        projectName: "",
        configName: "config",
        resetInvalidConfig: false,
        defaults: null,
    };
    defaultValues = plainObject();
    path;
    constructor(options) {
        this._options = {
            ...this._options,
            ...options,
        };
        this.defaultValues = this._options.defaults
            ? this._options.defaults
            : plainObject();
        if (!this._options.projectName || this._options.projectName.trim() === "") {
            throw new Error("the projectName option must be provided and non-empty");
        }
        this._options.projectName = this._options.projectName.trim();
        this.path = pathResolve(envPaths(this._options.projectName).config, `${this._options.configName}.json`);
    }
    get size() {
        return Object.keys(this.store).length;
    }
    get dir() {
        return pathDirname(this.path);
    }
    get store() {
        try {
            return JSON.parse(Deno.readTextFileSync(this.path));
        }
        catch (error) {
            switch (error.name) {
                case "SyntaxError":
                    if (this._options.resetInvalidConfig) {
                        this.reset();
                        return this.defaultValues;
                    }
                    break;
                case "NotFound":
                    return plainObject();
            }
            throw error;
        }
    }
    set store(data) {
        if (!existsSync(pathDirname(this.path))) {
            Deno.mkdirSync(pathDirname(this.path), { recursive: true });
        }
        Deno.writeTextFileSync(this.path, JSON.stringify(data, null, 2));
    }
    get options() {
        return this._options;
    }
    has(key) {
        return key in this.store;
    }
    reset() {
        if (this.path && existsSync(this.path)) {
            Deno.removeSync(this.path, { recursive: true });
        }
        if (Object.keys(this.defaultValues).length === 0) {
            return;
        }
        Object.entries(this.defaultValues).forEach(([key, value]) => {
            this.set(key, value);
        });
        return;
    }
    resetKeys(keys) {
        if (Object.keys(this.defaultValues).length === 0) {
            return;
        }
        for (const key of keys) {
            if (this.defaultValues && key in this.defaultValues) {
                this.set(key, this.defaultValues[key]);
            }
        }
    }
    delete(key) {
        const { store } = this;
        if (store && key in store) {
            delete store[key];
            this.store = store;
        }
    }
    get(key) {
        if (this.store && key in this.store) {
            return this.store[key];
        }
        else if (this.defaultValues && key in this.defaultValues) {
            return this.defaultValues[key];
        }
        else {
            return null;
        }
    }
    set(key, value) {
        const { store } = this;
        const innerSet = (key, value) => {
            store[key] = value;
        };
        innerSet(key, value);
        this.store = store;
    }
    setObject(data) {
        for (const [key, value] of Object.entries(data)) {
            this.set(key, value);
        }
    }
    *[Symbol.iterator]() {
        for (const [key, value] of Object.entries(this.store)) {
            yield [key, value];
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFDTCxPQUFPLElBQUksV0FBVyxFQUN0QixPQUFPLElBQUksV0FBVyxHQUN2QixNQUFNLDBDQUEwQyxDQUFDO0FBRWxELE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSwyQ0FBMkMsQ0FBQztBQUV2RSxPQUFPLFFBQVEsTUFBTSx3RUFBd0UsQ0FBQztBQUU5RixNQUFNLFdBQVcsR0FBRyxHQUFHLEVBQUUsQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxDQUFDO0FBd0Q5QyxNQUFNLENBQUMsT0FBTyxPQUFPLE1BQU07SUFDakIsUUFBUSxHQUFxQjtRQUNuQyxXQUFXLEVBQUUsRUFBRTtRQUNmLFVBQVUsRUFBRSxRQUFRO1FBQ3BCLGtCQUFrQixFQUFFLEtBQUs7UUFDekIsUUFBUSxFQUFFLElBQUk7S0FDZixDQUFDO0lBRUYsYUFBYSxHQUFjLFdBQVcsRUFBRSxDQUFDO0lBRXpDLElBQUksQ0FBUztJQUViLFlBQVksT0FBeUI7UUFDbkMsSUFBSSxDQUFDLFFBQVEsR0FBRztZQUNkLEdBQUcsSUFBSSxDQUFDLFFBQVE7WUFDaEIsR0FBRyxPQUFPO1NBQ1gsQ0FBQztRQUdGLElBQUksQ0FBQyxhQUFhLEdBQUcsSUFBSSxDQUFDLFFBQVEsQ0FBQyxRQUFRO1lBQ3pDLENBQUMsQ0FBQyxJQUFJLENBQUMsUUFBUSxDQUFDLFFBQVE7WUFDeEIsQ0FBQyxDQUFDLFdBQVcsRUFBRSxDQUFDO1FBRWxCLElBQUksQ0FBQyxJQUFJLENBQUMsUUFBUSxDQUFDLFdBQVcsSUFBSSxJQUFJLENBQUMsUUFBUSxDQUFDLFdBQVcsQ0FBQyxJQUFJLEVBQUUsS0FBSyxFQUFFLEVBQUU7WUFDekUsTUFBTSxJQUFJLEtBQUssQ0FBQyx1REFBdUQsQ0FBQyxDQUFDO1NBQzFFO1FBRUQsSUFBSSxDQUFDLFFBQVEsQ0FBQyxXQUFXLEdBQUcs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