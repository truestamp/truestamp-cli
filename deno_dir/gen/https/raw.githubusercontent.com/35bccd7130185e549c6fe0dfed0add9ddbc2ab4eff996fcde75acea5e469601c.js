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
            return {
                ...this.defaultValues,
                ...JSON.parse(Deno.readTextFileSync(this.path)),
            };
        }
        catch (error) {
            switch (error.name) {
                case "SyntaxError":
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