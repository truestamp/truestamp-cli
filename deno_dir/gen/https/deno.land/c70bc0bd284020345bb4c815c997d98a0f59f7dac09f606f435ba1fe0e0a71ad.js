import { UnknownType, ValidationError as FlagsValidationError, } from "../flags/_errors.ts";
import { MissingRequiredEnvVar } from "./_errors.ts";
import { parseFlags } from "../flags/flags.ts";
import { getDescription, parseArgumentsDefinition, splitArguments, } from "./_utils.ts";
import { blue, bold, red, yellow } from "./deps.ts";
import { CommandExecutableNotFound, CommandNotFound, DefaultCommandNotFound, DuplicateCommandAlias, DuplicateCommandName, DuplicateCompletion, DuplicateEnvironmentVariable, DuplicateExample, DuplicateOptionName, DuplicateType, EnvironmentVariableOptionalValue, EnvironmentVariableSingleValue, EnvironmentVariableVariadicValue, MissingArgument, MissingArguments, MissingCommandName, NoArgumentsAllowed, TooManyArguments, UnknownCommand, ValidationError, } from "./_errors.ts";
import { BooleanType } from "./types/boolean.ts";
import { FileType } from "./types/file.ts";
import { NumberType } from "./types/number.ts";
import { StringType } from "./types/string.ts";
import { Type } from "./type.ts";
import { HelpGenerator } from "./help/_help_generator.ts";
import { IntegerType } from "./types/integer.ts";
import { underscoreToCamelCase } from "../flags/_utils.ts";
export class Command {
    types = new Map();
    rawArgs = [];
    literalArgs = [];
    _name = "COMMAND";
    _parent;
    _globalParent;
    ver;
    desc = "";
    _usage;
    fn;
    options = [];
    commands = new Map();
    examples = [];
    envVars = [];
    aliases = [];
    completions = new Map();
    cmd = this;
    argsDefinition;
    isExecutable = false;
    throwOnError = false;
    _allowEmpty = false;
    _stopEarly = false;
    defaultCommand;
    _useRawArgs = false;
    args = [];
    isHidden = false;
    isGlobal = false;
    hasDefaults = false;
    _versionOption;
    _helpOption;
    _help;
    _shouldExit;
    _meta = {};
    _groupName;
    versionOption(flags, desc, opts) {
        this._versionOption = flags === false ? flags : {
            flags,
            desc,
            opts: typeof opts === "function" ? { action: opts } : opts,
        };
        return this;
    }
    helpOption(flags, desc, opts) {
        this._helpOption = flags === false ? flags : {
            flags,
            desc,
            opts: typeof opts === "function" ? { action: opts } : opts,
        };
        return this;
    }
    command(nameAndArguments, cmdOrDescription, override) {
        this.reset();
        const result = splitArguments(nameAndArguments);
        const name = result.flags.shift();
        const aliases = result.flags;
        if (!name) {
            throw new MissingCommandName();
        }
        if (this.getBaseCommand(name, true)) {
            if (!override) {
                throw new DuplicateCommandName(name);
            }
            this.removeCommand(name);
        }
        let description;
        let cmd;
        if (typeof cmdOrDescription === "string") {
            description = cmdOrDescription;
        }
        if (cmdOrDescription instanceof Command) {
            cmd = cmdOrDescription.reset();
        }
        else {
            cmd = new Command();
        }
        cmd._name = name;
        cmd._parent = this;
        if (description) {
            cmd.description(description);
        }
        if (result.typeDefinition) {
            cmd.arguments(result.typeDefinition);
        }
        aliases.forEach((alias) => cmd.alias(alias));
        this.commands.set(name, cmd);
        this.select(name);
        return this;
    }
    alias(alias) {
        if (this.cmd._name === alias || this.cmd.aliases.includes(alias)) {
            throw new DuplicateCommandAlias(alias);
        }
        this.cmd.aliases.push(alias);
        return this;
    }
    reset() {
        this._groupName = undefined;
        this.cmd = this;
        return this;
    }
    select(name) {
        const cmd = this.getBaseCommand(name, true);
        if (!cmd) {
            throw new CommandNotFound(name, this.getBaseCommands(true));
        }
        this.cmd = cmd;
        return this;
    }
    name(name) {
        this.cmd._name = name;
        return this;
    }
    version(version) {
        if (typeof version === "string") {
            this.cmd.ver = () => version;
        }
        else if (typeof version === "function") {
            this.cmd.ver = version;
        }
        return this;
    }
    meta(name, value) {
        this.cmd._meta[name] = value;
        return this;
    }
    getMeta(name) {
        return typeof name === "undefined" ? this._meta : this._meta[name];
    }
    help(help) {
        if (typeof help === "string") {
            this.cmd._help = () => help;
        }
        else if (typeof help === "function") {
            this.cmd._help = help;
        }
        else {
            this.cmd._help = (cmd, options) => HelpGenerator.generate(cmd, { ...help, ...options });
        }
        return this;
    }
    description(description) {
        this.cmd.desc = description;
        return this;
    }
    usage(usage) {
        this.cmd._usage = usage;
        return this;
    }
    hidden() {
        this.cmd.isHidden = true;
        return this;
    }
    global() {
        this.cmd.isGlobal = true;
        return this;
    }
    executable() {
        this.cmd.isExecutable = true;
        return this;
    }
    arguments(args) {
        this.cmd.argsDefinition = args;
        return this;
    }
    action(fn) {
        this.cmd.fn = fn;
        return this;
    }
    allowEmpty(allowEmpty = true) {
        this.cmd._allowEmpty = allowEmpty;
        return this;
    }
    stopEarly(stopEarly = true) {
        this.cmd._stopEarly = stopEarly;
        return this;
    }
    useRawArgs(useRawArgs = true) {
        this.cmd._useRawArgs = useRawArgs;
        return this;
    }
    default(name) {
        this.cmd.defaultCommand = name;
        return this;
    }
    globalType(name, handler, options) {
        return this.type(name, handler, { ...options, global: true });
    }
    type(name, handler, options) {
        if (this.cmd.types.get(name) && !options?.override) {
            throw new DuplicateType(name);
        }
        this.cmd.types.set(name, { ...options, name, handler });
        if (handler instanceof Type &&
            (typeof handler.complete !== "undefined" ||
                typeof handler.values !== "undefined")) {
            const completeHandler = (cmd, parent) => handler.complete?.(cmd, parent) || [];
            this.complete(name, completeHandler, options);
        }
        return this;
    }
    globalComplete(name, complete, options) {
        return this.complete(name, complete, { ...options, global: true });
    }
    complete(name, complete, options) {
        if (this.cmd.completions.has(name) && !options?.override) {
            throw new DuplicateCompletion(name);
        }
        this.cmd.completions.set(name, {
            name,
            complete,
            ...options,
        });
        return this;
    }
    throwErrors() {
        this.cmd.throwOnError = true;
        return this;
    }
    noExit() {
        this.cmd._shouldExit = false;
        this.throwErrors();
        return this;
    }
    shouldThrowErrors() {
        return this.cmd.throwOnError || !!this.cmd._parent?.shouldThrowErrors();
    }
    shouldExit() {
        return this.cmd._shouldExit ?? this.cmd._parent?.shouldExit() ?? true;
    }
    globalOption(flags, desc, opts) {
        if (typeof opts === "function") {
            return this.option(flags, desc, { value: opts, global: true });
        }
        return this.option(flags, desc, { ...opts, global: true });
    }
    group(name) {
        this.cmd._groupName = name;
        return this;
    }
    option(flags, desc, opts) {
        if (typeof opts === "function") {
            return this.option(flags, desc, { value: opts });
        }
        const result = splitArguments(flags);
        const args = result.typeDefinition
            ? parseArgumentsDefinition(result.typeDefinition)
            : [];
        const option = {
            ...opts,
            name: "",
            description: desc,
            args,
            flags: result.flags,
            equalsSign: result.equalsSign,
            typeDefinition: result.typeDefinition,
            groupName: this._groupName,
        };
        if (option.separator) {
            for (const arg of args) {
                if (arg.list) {
                    arg.separator = option.separator;
                }
            }
        }
        for (const part of option.flags) {
            const arg = part.trim();
            const isLong = /^--/.test(arg);
            const name = isLong ? arg.slice(2) : arg.slice(1);
            if (this.cmd.getBaseOption(name, true)) {
                if (opts?.override) {
                    this.removeOption(name);
                }
                else {
                    throw new DuplicateOptionName(name);
                }
            }
            if (!option.name && isLong) {
                option.name = name;
            }
            else if (!option.aliases) {
                option.aliases = [name];
            }
            else {
                option.aliases.push(name);
            }
        }
        if (option.prepend) {
            this.cmd.options.unshift(option);
        }
        else {
            this.cmd.options.push(option);
        }
        return this;
    }
    example(name, description) {
        if (this.cmd.hasExample(name)) {
            throw new DuplicateExample(name);
        }
        this.cmd.examples.push({ name, description });
        return this;
    }
    globalEnv(name, description, options) {
        return this.env(name, description, { ...options, global: true });
    }
    env(name, description, options) {
        const result = splitArguments(name);
        if (!result.typeDefinition) {
            result.typeDefinition = "<value:boolean>";
        }
        if (result.flags.some((envName) => this.cmd.getBaseEnvVar(envName, true))) {
            throw new DuplicateEnvironmentVariable(name);
        }
        const details = parseArgumentsDefinition(result.typeDefinition);
        if (details.length > 1) {
            throw new EnvironmentVariableSingleValue(name);
        }
        else if (details.length && details[0].optionalValue) {
            throw new EnvironmentVariableOptionalValue(name);
        }
        else if (details.length && details[0].variadic) {
            throw new EnvironmentVariableVariadicValue(name);
        }
        this.cmd.envVars.push({
            name: result.flags[0],
            names: result.flags,
            description,
            type: details[0].type,
            details: details.shift(),
            ...options,
        });
        return this;
    }
    async parse(args = Deno.args) {
        try {
            this.reset();
            this.registerDefaults();
            this.rawArgs = args;
            if (args.length > 0) {
                const subCommand = this.getCommand(args[0], true);
                if (subCommand) {
                    subCommand._globalParent = this;
                    return subCommand.parse(this.rawArgs.slice(1));
                }
            }
            if (this.isExecutable) {
                await this.executeExecutable(this.rawArgs);
                return {
                    options: {},
                    args: [],
                    cmd: this,
                    literal: [],
                };
            }
            else if (this._useRawArgs) {
                const env = await this.parseEnvVars();
                return this.execute(env, ...this.rawArgs);
            }
            else {
                const env = await this.parseEnvVars();
                const { actionOption, flags, unknown, literal } = this
                    .parseFlags(this.rawArgs, env);
                this.literalArgs = literal;
                const options = { ...env, ...flags };
                const params = this.parseArguments(unknown, options);
                if (actionOption) {
                    await actionOption.action.call(this, options, ...params);
                    if (actionOption.standalone) {
                        return {
                            options,
                            args: params,
                            cmd: this,
                            literal: this.literalArgs,
                        };
                    }
                }
                return this.execute(options, ...params);
            }
        }
        catch (error) {
            if (error instanceof Error) {
                throw this.error(error);
            }
            else {
                throw this.error(new Error(`[non-error-thrown] ${error}`));
            }
        }
    }
    registerDefaults() {
        if (this.hasDefaults || this.getParent()) {
            return this;
        }
        this.hasDefaults = true;
        this.reset();
        !this.types.has("string") &&
            this.type("string", new StringType(), { global: true });
        !this.types.has("number") &&
            this.type("number", new NumberType(), { global: true });
        !this.types.has("integer") &&
            this.type("integer", new IntegerType(), { global: true });
        !this.types.has("boolean") &&
            this.type("boolean", new BooleanType(), { global: true });
        !this.types.has("file") &&
            this.type("file", new FileType(), { global: true });
        if (!this._help) {
            this.help({
                hints: true,
                types: false,
            });
        }
        if (this._versionOption !== false && (this._versionOption || this.ver)) {
            this.option(this._versionOption?.flags || "-V, --version", this._versionOption?.desc ||
                "Show the version number for this program.", {
                standalone: true,
                prepend: true,
                action: async function () {
                    const long = this.getRawArgs().includes(`--${versionOption.name}`);
                    if (long) {
                        await this.checkVersion();
                        this.showLongVersion();
                    }
                    else {
                        this.showVersion();
                    }
                    this.exit();
                },
                ...(this._versionOption?.opts ?? {}),
            });
            const versionOption = this.options[0];
        }
        if (this._helpOption !== false) {
            this.option(this._helpOption?.flags || "-h, --help", this._helpOption?.desc || "Show this help.", {
                standalone: true,
                global: true,
                prepend: true,
                action: async function () {
                    const long = this.getRawArgs().includes(`--${helpOption.name}`);
                    await this.checkVersion();
                    this.showHelp({ long });
                    this.exit();
                },
                ...(this._helpOption?.opts ?? {}),
            });
            const helpOption = this.options[0];
        }
        return this;
    }
    async execute(options, ...args) {
        if (this.fn) {
            await this.fn(options, ...args);
        }
        else if (this.defaultCommand) {
            const cmd = this.getCommand(this.defaultCommand, true);
            if (!cmd) {
                throw new DefaultCommandNotFound(this.defaultCommand, this.getCommands());
            }
            cmd._globalParent = this;
            await cmd.execute(options, ...args);
        }
        return {
            options,
            args,
            cmd: this,
            literal: this.literalArgs,
        };
    }
    async executeExecutable(args) {
        const command = this.getPath().replace(/\s+/g, "-");
        await Deno.permissions.request({ name: "run", command });
        try {
            const process = Deno.run({
                cmd: [command, ...args],
            });
            const status = await process.status();
            if (!status.success) {
                Deno.exit(status.code);
            }
        }
        catch (error) {
            if (error instanceof Deno.errors.NotFound) {
                throw new CommandExecutableNotFound(command);
            }
            throw error;
        }
    }
    parseFlags(args, env) {
        try {
            let actionOption;
            const result = parseFlags(args, {
                stopEarly: this._stopEarly,
                allowEmpty: this._allowEmpty,
                flags: this.getOptions(true),
                ignoreDefaults: env,
                parse: (type) => this.parseType(type),
                option: (option) => {
                    if (!actionOption && option.action) {
                        actionOption = option;
                    }
                },
            });
            return { ...result, actionOption };
        }
        catch (error) {
            if (error instanceof FlagsValidationError) {
                throw new ValidationError(error.message);
            }
            throw error;
        }
    }
    parseType(type) {
        const typeSettings = this.getType(type.type);
        if (!typeSettings) {
            throw new UnknownType(type.type, this.getTypes().map((type) => type.name));
        }
        return typeSettings.handler instanceof Type
            ? typeSettings.handler.parse(type)
            : typeSettings.handler(type);
    }
    async parseEnvVars() {
        const envVars = this.getEnvVars(true);
        const result = {};
        if (!envVars.length) {
            return result;
        }
        const hasEnvPermissions = (await Deno.permissions.query({
            name: "env",
        })).state === "granted";
        for (const env of envVars) {
            const name = hasEnvPermissions && env.names.find((name) => !!Deno.env.get(name));
            if (name) {
                const propertyName = underscoreToCamelCase(env.prefix
                    ? env.names[0].replace(new RegExp(`^${env.prefix}`), "")
                    : env.names[0]);
                if (env.details.list) {
                    const values = Deno.env.get(name)
                        ?.split(env.details.separator ?? ",") ?? [""];
                    result[propertyName] = values.map((value) => this.parseType({
                        label: "Environment variable",
                        type: env.type,
                        name,
                        value,
                    }));
                }
                else {
                    result[propertyName] = this.parseType({
                        label: "Environment variable",
                        type: env.type,
                        name,
                        value: Deno.env.get(name) ?? "",
                    });
                }
                if (env.value && typeof result[propertyName] !== "undefined") {
                    result[propertyName] = env.value(result[propertyName]);
                }
            }
            else if (env.required) {
                throw new MissingRequiredEnvVar(env);
            }
        }
        return result;
    }
    parseArguments(args, flags) {
        const params = [];
        args = args.slice(0);
        if (!this.hasArguments()) {
            if (args.length) {
                if (this.hasCommands(true)) {
                    throw new UnknownCommand(args[0], this.getCommands());
                }
                else {
                    throw new NoArgumentsAllowed(this.getPath());
                }
            }
        }
        else {
            if (!args.length) {
                const required = this.getArguments()
                    .filter((expectedArg) => !expectedArg.optionalValue)
                    .map((expectedArg) => expectedArg.name);
                if (required.length) {
                    const flagNames = Object.keys(flags);
                    const hasStandaloneOption = !!flagNames.find((name) => this.getOption(name, true)?.standalone);
                    if (!hasStandaloneOption) {
                        throw new MissingArguments(required);
                    }
                }
            }
            else {
                for (const expectedArg of this.getArguments()) {
                    if (!args.length) {
                        if (expectedArg.optionalValue) {
                            break;
                        }
                        throw new MissingArgument(`Missing argument: ${expectedArg.name}`);
                    }
                    let arg;
                    if (expectedArg.variadic) {
                        arg = args.splice(0, args.length)
                            .map((value) => this.parseType({
                            label: "Argument",
                            type: expectedArg.type,
                            name: expectedArg.name,
                            value,
                        }));
                    }
                    else {
                        arg = this.parseType({
                            label: "Argument",
                            type: expectedArg.type,
                            name: expectedArg.name,
                            value: args.shift(),
                        });
                    }
                    if (typeof arg !== "undefined") {
                        params.push(arg);
                    }
                }
                if (args.length) {
                    throw new TooManyArguments(args);
                }
            }
        }
        return params;
    }
    error(error) {
        if (this.shouldThrowErrors() || !(error instanceof ValidationError)) {
            return error;
        }
        this.showHelp();
        console.error(red(`  ${bold("error")}: ${error.message}\n`));
        Deno.exit(error instanceof ValidationError ? error.exitCode : 1);
    }
    getName() {
        return this._name;
    }
    getParent() {
        return this._parent;
    }
    getGlobalParent() {
        return this._globalParent;
    }
    getMainCommand() {
        return this._parent?.getMainCommand() ?? this;
    }
    getAliases() {
        return this.aliases;
    }
    getPath() {
        return this._parent
            ? this._parent.getPath() + " " + this._name
            : this._name;
    }
    getArgsDefinition() {
        return this.argsDefinition;
    }
    getArgument(name) {
        return this.getArguments().find((arg) => arg.name === name);
    }
    getArguments() {
        if (!this.args.length && this.argsDefinition) {
            this.args = parseArgumentsDefinition(this.argsDefinition);
        }
        return this.args;
    }
    hasArguments() {
        return !!this.argsDefinition;
    }
    getVersion() {
        return this.getVersionHandler()?.call(this, this);
    }
    getVersionHandler() {
        return this.ver ?? this._parent?.getVersionHandler();
    }
    getDescription() {
        return typeof this.desc === "function"
            ? this.desc = this.desc()
            : this.desc;
    }
    getUsage() {
        return this._usage ?? this.getArgsDefinition();
    }
    getShortDescription() {
        return getDescription(this.getDescription(), true);
    }
    getRawArgs() {
        return this.rawArgs;
    }
    getLiteralArgs() {
        return this.literalArgs;
    }
    showVersion() {
        console.log(this.getVersion());
    }
    getLongVersion() {
        return `${bold(this.getMainCommand().getName())} ${blue(this.getVersion() ?? "")}` +
            Object.entries(this.getMeta()).map(([k, v]) => `\n${bold(k)} ${blue(v)}`).join("");
    }
    showLongVersion() {
        console.log(this.getLongVersion());
    }
    showHelp(options) {
        console.log(this.getHelp(options));
    }
    getHelp(options) {
        this.registerDefaults();
        return this.getHelpHandler().call(this, this, options ?? {});
    }
    getHelpHandler() {
        return this._help ?? this._parent?.getHelpHandler();
    }
    exit(code = 0) {
        if (this.shouldExit()) {
            Deno.exit(code);
        }
    }
    async checkVersion() {
        const mainCommand = this.getMainCommand();
        const upgradeCommand = mainCommand.getCommand("upgrade");
        if (isUpgradeCommand(upgradeCommand)) {
            const latestVersion = await upgradeCommand.getLatestVersion();
            const currentVersion = mainCommand.getVersion();
            if (currentVersion !== latestVersion) {
                mainCommand.version(`${currentVersion}  ${bold(yellow(`(New version available: ${latestVersion}. Run '${mainCommand.getName()} upgrade' to upgrade to the latest version!)`))}`);
            }
        }
    }
    hasOptions(hidden) {
        return this.getOptions(hidden).length > 0;
    }
    getOptions(hidden) {
        return this.getGlobalOptions(hidden).concat(this.getBaseOptions(hidden));
    }
    getBaseOptions(hidden) {
        if (!this.options.length) {
            return [];
        }
        return hidden
            ? this.options.slice(0)
            : this.options.filter((opt) => !opt.hidden);
    }
    getGlobalOptions(hidden) {
        const getOptions = (cmd, options = [], names = []) => {
            if (cmd) {
                if (cmd.options.length) {
                    cmd.options.forEach((option) => {
                        if (option.global &&
                            !this.options.find((opt) => opt.name === option.name) &&
                            names.indexOf(option.name) === -1 &&
                            (hidden || !option.hidden)) {
                            names.push(option.name);
                            options.push(option);
                        }
                    });
                }
                return getOptions(cmd._parent, options, names);
            }
            return options;
        };
        return getOptions(this._parent);
    }
    hasOption(name, hidden) {
        return !!this.getOption(name, hidden);
    }
    getOption(name, hidden) {
        return this.getBaseOption(name, hidden) ??
            this.getGlobalOption(name, hidden);
    }
    getBaseOption(name, hidden) {
        const option = this.options.find((option) => option.name === name);
        return option && (hidden || !option.hidden) ? option : undefined;
    }
    getGlobalOption(name, hidden) {
        if (!this._parent) {
            return;
        }
        const option = this._parent.getBaseOption(name, hidden);
        if (!option || !option.global) {
            return this._parent.getGlobalOption(name, hidden);
        }
        return option;
    }
    removeOption(name) {
        const index = this.options.findIndex((option) => option.name === name);
        if (index === -1) {
            return;
        }
        return this.options.splice(index, 1)[0];
    }
    hasCommands(hidden) {
        return this.getCommands(hidden).length > 0;
    }
    getCommands(hidden) {
        return this.getGlobalCommands(hidden).concat(this.getBaseCommands(hidden));
    }
    getBaseCommands(hidden) {
        const commands = Array.from(this.commands.values());
        return hidden ? commands : commands.filter((cmd) => !cmd.isHidden);
    }
    getGlobalCommands(hidden) {
        const getCommands = (cmd, commands = [], names = []) => {
            if (cmd) {
                if (cmd.commands.size) {
                    cmd.commands.forEach((cmd) => {
                        if (cmd.isGlobal &&
                            this !== cmd &&
                            !this.commands.has(cmd._name) &&
                            names.indexOf(cmd._name) === -1 &&
                            (hidden || !cmd.isHidden)) {
                            names.push(cmd._name);
                            commands.push(cmd);
                        }
                    });
                }
                return getCommands(cmd._parent, commands, names);
            }
            return commands;
        };
        return getCommands(this._parent);
    }
    hasCommand(name, hidden) {
        return !!this.getCommand(name, hidden);
    }
    getCommand(name, hidden) {
        return this.getBaseCommand(name, hidden) ??
            this.getGlobalCommand(name, hidden);
    }
    getBaseCommand(name, hidden) {
        for (const cmd of this.commands.values()) {
            if (cmd._name === name || cmd.aliases.includes(name)) {
                return (cmd && (hidden || !cmd.isHidden) ? cmd : undefined);
            }
        }
    }
    getGlobalCommand(name, hidden) {
        if (!this._parent) {
            return;
        }
        const cmd = this._parent.getBaseCommand(name, hidden);
        if (!cmd?.isGlobal) {
            return this._parent.getGlobalCommand(name, hidden);
        }
        return cmd;
    }
    removeCommand(name) {
        const command = this.getBaseCommand(name, true);
        if (command) {
            this.commands.delete(command._name);
        }
        return command;
    }
    getTypes() {
        return this.getGlobalTypes().concat(this.getBaseTypes());
    }
    getBaseTypes() {
        return Array.from(this.types.values());
    }
    getGlobalTypes() {
        const getTypes = (cmd, types = [], names = []) => {
            if (cmd) {
                if (cmd.types.size) {
                    cmd.types.forEach((type) => {
                        if (type.global &&
                            !this.types.has(type.name) &&
                            names.indexOf(type.name) === -1) {
                            names.push(type.name);
                            types.push(type);
                        }
                    });
                }
                return getTypes(cmd._parent, types, names);
            }
            return types;
        };
        return getTypes(this._parent);
    }
    getType(name) {
        return this.getBaseType(name) ?? this.getGlobalType(name);
    }
    getBaseType(name) {
        return this.types.get(name);
    }
    getGlobalType(name) {
        if (!this._parent) {
            return;
        }
        const cmd = this._parent.getBaseType(name);
        if (!cmd?.global) {
            return this._parent.getGlobalType(name);
        }
        return cmd;
    }
    getCompletions() {
        return this.getGlobalCompletions().concat(this.getBaseCompletions());
    }
    getBaseCompletions() {
        return Array.from(this.completions.values());
    }
    getGlobalCompletions() {
        const getCompletions = (cmd, completions = [], names = []) => {
            if (cmd) {
                if (cmd.completions.size) {
                    cmd.completions.forEach((completion) => {
                        if (completion.global &&
                            !this.completions.has(completion.name) &&
                            names.indexOf(completion.name) === -1) {
                            names.push(completion.name);
                            completions.push(completion);
                        }
                    });
                }
                return getCompletions(cmd._parent, completions, names);
            }
            return completions;
        };
        return getCompletions(this._parent);
    }
    getCompletion(name) {
        return this.getBaseCompletion(name) ?? this.getGlobalCompletion(name);
    }
    getBaseCompletion(name) {
        return this.completions.get(name);
    }
    getGlobalCompletion(name) {
        if (!this._parent) {
            return;
        }
        const completion = this._parent.getBaseCompletion(name);
        if (!completion?.global) {
            return this._parent.getGlobalCompletion(name);
        }
        return completion;
    }
    hasEnvVars(hidden) {
        return this.getEnvVars(hidden).length > 0;
    }
    getEnvVars(hidden) {
        return this.getGlobalEnvVars(hidden).concat(this.getBaseEnvVars(hidden));
    }
    getBaseEnvVars(hidden) {
        if (!this.envVars.length) {
            return [];
        }
        return hidden
            ? this.envVars.slice(0)
            : this.envVars.filter((env) => !env.hidden);
    }
    getGlobalEnvVars(hidden) {
        const getEnvVars = (cmd, envVars = [], names = []) => {
            if (cmd) {
                if (cmd.envVars.length) {
                    cmd.envVars.forEach((envVar) => {
                        if (envVar.global &&
                            !this.envVars.find((env) => env.names[0] === envVar.names[0]) &&
                            names.indexOf(envVar.names[0]) === -1 &&
                            (hidden || !envVar.hidden)) {
                            names.push(envVar.names[0]);
                            envVars.push(envVar);
                        }
                    });
                }
                return getEnvVars(cmd._parent, envVars, names);
            }
            return envVars;
        };
        return getEnvVars(this._parent);
    }
    hasEnvVar(name, hidden) {
        return !!this.getEnvVar(name, hidden);
    }
    getEnvVar(name, hidden) {
        return this.getBaseEnvVar(name, hidden) ??
            this.getGlobalEnvVar(name, hidden);
    }
    getBaseEnvVar(name, hidden) {
        const envVar = this.envVars.find((env) => env.names.indexOf(name) !== -1);
        return envVar && (hidden || !envVar.hidden) ? envVar : undefined;
    }
    getGlobalEnvVar(name, hidden) {
        if (!this._parent) {
            return;
        }
        const envVar = this._parent.getBaseEnvVar(name, hidden);
        if (!envVar?.global) {
            return this._parent.getGlobalEnvVar(name, hidden);
        }
        return envVar;
    }
    hasExamples() {
        return this.examples.length > 0;
    }
    getExamples() {
        return this.examples;
    }
    hasExample(name) {
        return !!this.getExample(name);
    }
    getExample(name) {
        return this.examples.find((example) => example.name === name);
    }
}
function isUpgradeCommand(command) {
    return command instanceof Command && "getLatestVersion" in command;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29tbWFuZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImNvbW1hbmQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUNMLFdBQVcsRUFDWCxlQUFlLElBQUksb0JBQW9CLEdBQ3hDLE1BQU0scUJBQXFCLENBQUM7QUFDN0IsT0FBTyxFQUFFLHFCQUFxQixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBQ3JELE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUUvQyxPQUFPLEVBQ0wsY0FBYyxFQUNkLHdCQUF3QixFQUN4QixjQUFjLEdBQ2YsTUFBTSxhQUFhLENBQUM7QUFDckIsT0FBTyxFQUFFLElBQUksRUFBRSxJQUFJLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUNwRCxPQUFPLEVBQ0wseUJBQXlCLEVBQ3pCLGVBQWUsRUFDZixzQkFBc0IsRUFDdEIscUJBQXFCLEVBQ3JCLG9CQUFvQixFQUNwQixtQkFBbUIsRUFDbkIsNEJBQTRCLEVBQzVCLGdCQUFnQixFQUNoQixtQkFBbUIsRUFDbkIsYUFBYSxFQUNiLGdDQUFnQyxFQUNoQyw4QkFBOEIsRUFDOUIsZ0NBQWdDLEVBQ2hDLGVBQWUsRUFDZixnQkFBZ0IsRUFDaEIsa0JBQWtCLEVBQ2xCLGtCQUFrQixFQUNsQixnQkFBZ0IsRUFDaEIsY0FBYyxFQUNkLGVBQWUsR0FDaEIsTUFBTSxjQUFjLENBQUM7QUFDdEIsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ2pELE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxpQkFBaUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsVUFBVSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDL0MsT0FBTyxFQUFFLFVBQVUsRUFBRSxNQUFNLG1CQUFtQixDQUFDO0FBQy9DLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDakMsT0FBTyxFQUFFLGFBQWEsRUFBRSxN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bnQtaWdub3JlLWZpbGUgbm8tZXhwbGljaXQtYW55XG5pbXBvcnQge1xuICBVbmtub3duVHlwZSxcbiAgVmFsaWRhdGlvbkVycm9yIGFzIEZsYWdzVmFsaWRhdGlvbkVycm9yLFxufSBmcm9tIFwiLi4vZmxhZ3MvX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgTWlzc2luZ1JlcXVpcmVkRW52VmFyIH0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgcGFyc2VGbGFncyB9IGZyb20gXCIuLi9mbGFncy9mbGFncy50c1wiO1xuaW1wb3J0IHR5cGUgeyBJRGVmYXVsdFZhbHVlLCBJRmxhZ3NSZXN1bHQgfSBmcm9tIFwiLi4vZmxhZ3MvdHlwZXMudHNcIjtcbmltcG9ydCB7XG4gIGdldERlc2NyaXB0aW9uLFxuICBwYXJzZUFyZ3VtZW50c0RlZmluaXRpb24sXG4gIHNwbGl0QXJndW1lbnRzLFxufSBmcm9tIFwiLi9fdXRpbHMudHNcIjtcbmltcG9ydCB7IGJsdWUsIGJvbGQsIHJlZCwgeWVsbG93IH0gZnJvbSBcIi4vZGVwcy50c1wiO1xuaW1wb3J0IHtcbiAgQ29tbWFuZEV4ZWN1dGFibGVOb3RGb3VuZCxcbiAgQ29tbWFuZE5vdEZvdW5kLFxuICBEZWZhdWx0Q29tbWFuZE5vdEZvdW5kLFxuICBEdXBsaWNhdGVDb21tYW5kQWxpYXMsXG4gIER1cGxpY2F0ZUNvbW1hbmROYW1lLFxuICBEdXBsaWNhdGVDb21wbGV0aW9uLFxuICBEdXBsaWNhdGVFbnZpcm9ubWVudFZhcmlhYmxlLFxuICBEdXBsaWNhdGVFeGFtcGxlLFxuICBEdXBsaWNhdGVPcHRpb25OYW1lLFxuICBEdXBsaWNhdGVUeXBlLFxuICBFbnZpcm9ubWVudFZhcmlhYmxlT3B0aW9uYWxWYWx1ZSxcbiAgRW52aXJvbm1lbnRWYXJpYWJsZVNpbmdsZVZhbHVlLFxuICBFbnZpcm9ubWVudFZhcmlhYmxlVmFyaWFkaWNWYWx1ZSxcbiAgTWlzc2luZ0FyZ3VtZW50LFxuICBNaXNzaW5nQXJndW1lbnRzLFxuICBNaXNzaW5nQ29tbWFuZE5hbWUsXG4gIE5vQXJndW1lbnRzQWxsb3dlZCxcbiAgVG9vTWFueUFyZ3VtZW50cyxcbiAgVW5rbm93bkNvbW1hbmQsXG4gIFZhbGlkYXRpb25FcnJvcixcbn0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgQm9vbGVhblR5cGUgfSBmcm9tIFwiLi90eXBlcy9ib29sZWFuLnRzXCI7XG5pbXBvcnQgeyBGaWxlVHlwZSB9IGZyb20gXCIuL3R5cGVzL2ZpbGUudHNcIjtcbmltcG9ydCB7IE51bWJlclR5cGUgfSBmcm9tIFwiLi90eXBlcy9udW1iZXIudHNcIjtcbmltcG9ydCB7IFN0cmluZ1R5cGUgfSBmcm9tIFwiLi90eXBlcy9zdHJpbmcudHNcIjtcbmltcG9ydCB7IFR5cGUgfSBmcm9tIFwiLi90eXBlLnRzXCI7XG5pbXBvcnQgeyBIZWxwR2VuZXJhdG9yIH0gZnJvbSBcIi4vaGVscC9faGVscF9nZW5lcmF0b3IudHNcIjtcbmltcG9ydCB0eXBlIHsgSGVscE9wdGlvbnMgfSBmcm9tIFwiLi9oZWxwL19oZWxwX2dlbmVyYXRvci50c1wiO1xuaW1wb3J0IHR5cGUge1xuICBJQWN0aW9uLFxuICBJQXJndW1lbnQsXG4gIElDb21tYW5kR2xvYmFsT3B0aW9uLFxuICBJQ29tbWFuZE9wdGlvbixcbiAgSUNvbXBsZXRlSGFuZGxlcixcbiAgSUNvbXBsZXRlT3B0aW9ucyxcbiAgSUNvbXBsZXRpb24sXG4gIElEZXNjcmlwdGlvbixcbiAgSUVudlZhcixcbiAgSUVudlZhck9wdGlvbnMsXG4gIElFbnZWYXJWYWx1ZUhhbmRsZXIsXG4gIElFeGFtcGxlLFxuICBJRmxhZ1ZhbHVlSGFuZGxlcixcbiAgSUdsb2JhbEVudlZhck9wdGlvbnMsXG4gIElIZWxwSGFuZGxlcixcbiAgSU9wdGlvbixcbiAgSVBhcnNlUmVzdWx0LFxuICBJVHlwZSxcbiAgSVR5cGVJbmZvLFxuICBJVHlwZU9wdGlvbnMsXG4gIElWZXJzaW9uSGFuZGxlcixcbiAgTWFwVHlwZXMsXG4gIFR5cGVPclR5cGVIYW5kbGVyLFxufSBmcm9tIFwiLi90eXBlcy50c1wiO1xuaW1wb3J0IHsgSW50ZWdlclR5cGUgfSBmcm9tIFwiLi90eXBlcy9pbnRlZ2VyLnRzXCI7XG5pbXBvcnQgeyB1bmRlcnNjb3JlVG9DYW1lbENhc2UgfSBmcm9tIFwiLi4vZmxhZ3MvX3V0aWxzLnRzXCI7XG5cbmV4cG9ydCBjbGFzcyBDb21tYW5kPFxuICBDUEcgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCA9IHZvaWQsXG4gIENQVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdm9pZCxcbiAgQ08gZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHZvaWQsXG4gIENBIGV4dGVuZHMgQXJyYXk8dW5rbm93bj4gPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiBbXSxcbiAgQ0cgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHZvaWQsXG4gIENUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQgPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiB7XG4gICAgbnVtYmVyOiBudW1iZXI7XG4gICAgaW50ZWdlcjogbnVtYmVyO1xuICAgIHN0cmluZzogc3RyaW5nO1xuICAgIGJvb2xlYW46IGJvb2xlYW47XG4gICAgZmlsZTogc3RyaW5nO1xuICB9LFxuICBDR1QgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHZvaWQsXG4gIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdW5kZWZpbmVkLFxuPiB7XG4gIHByaXZhdGUgdHlwZXM6IE1hcDxzdHJpbmcsIElUeXBlPiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSByYXdBcmdzOiBBcnJheTxzdHJpbmc+ID0gW107XG4gIHByaXZhdGUgbGl0ZXJhbEFyZ3M6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgLy8gQFRPRE86IGdldCBzY3JpcHQgbmFtZTogaHR0cHM6Ly9naXRodWIuY29tL2Rlbm9sYW5kL2Rlbm8vcHVsbC81MDM0XG4gIC8vIHByaXZhdGUgbmFtZTogc3RyaW5nID0gbG9jYXRpb24ucGF0aG5hbWUuc3BsaXQoICcvJyApLnBvcCgpIGFzIHN0cmluZztcbiAgcHJpdmF0ZSBfbmFtZSA9IFwiQ09NTUFORFwiO1xuICBwcml2YXRlIF9wYXJlbnQ/OiBDUDtcbiAgcHJpdmF0ZSBfZ2xvYmFsUGFyZW50PzogQ29tbWFuZDxhbnk+O1xuICBwcml2YXRlIHZlcj86IElWZXJzaW9uSGFuZGxlcjtcbiAgcHJpdmF0ZSBkZXNjOiBJRGVzY3JpcHRpb24gPSBcIlwiO1xuICBwcml2YXRlIF91c2FnZT86IHN0cmluZztcbiAgcHJpdmF0ZSBmbj86IElBY3Rpb247XG4gIHByaXZhdGUgb3B0aW9uczogQXJyYXk8SU9wdGlvbj4gPSBbXTtcbiAgcHJpdmF0ZSBjb21tYW5kczogTWFwPHN0cmluZywgQ29tbWFuZDxhbnk+PiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSBleGFtcGxlczogQXJyYXk8SUV4YW1wbGU+ID0gW107XG4gIHByaXZhdGUgZW52VmFyczogQXJyYXk8SUVudlZhcj4gPSBbXTtcbiAgcHJpdmF0ZSBhbGlhc2VzOiBBcnJheTxzdHJpbmc+ID0gW107XG4gIHByaXZhdGUgY29tcGxldGlvbnM6IE1hcDxzdHJpbmcsIElDb21wbGV0aW9uPiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSBjbWQ6IENvbW1hbmQ8YW55PiA9IHRoaXM7XG4gIHByaXZhdGUgYXJnc0RlZmluaXRpb24/OiBzdHJpbmc7XG4gIHByaXZhdGUgaXNFeGVjdXRhYmxlID0gZmFsc2U7XG4gIHByaXZhdGUgdGhyb3dPbkVycm9yID0gZmFsc2U7XG4gIHByaXZhdGUgX2FsbG93RW1wdHkgPSBmYWxzZTtcbiAgcHJpdmF0ZSBfc3RvcEVhcmx5ID0gZmFsc2U7XG4gIHByaXZhdGUgZGVmYXVsdENvbW1hbmQ/OiBzdHJpbmc7XG4gIHByaXZhdGUgX3VzZVJhd0FyZ3MgPSBmYWxzZTtcbiAgcHJpdmF0ZSBhcmdzOiBBcnJheTxJQXJndW1lbnQ+ID0gW107XG4gIHByaXZhdGUgaXNIaWRkZW4gPSBmYWxzZTtcbiAgcHJpdmF0ZSBpc0dsb2JhbCA9IGZhbHNlO1xuICBwcml2YXRlIGhhc0RlZmF1bHRzID0gZmFsc2U7XG4gIHByaXZhdGUgX3ZlcnNpb25PcHRpb24/OiBJRGVmYXVsdE9wdGlvbiB8IGZhbHNlO1xuICBwcml2YXRlIF9oZWxwT3B0aW9uPzogSURlZmF1bHRPcHRpb24gfCBmYWxzZTtcbiAgcHJpdmF0ZSBfaGVscD86IElIZWxwSGFuZGxlcjtcbiAgcHJpdmF0ZSBfc2hvdWxkRXhpdD86IGJvb2xlYW47XG4gIHByaXZhdGUgX21ldGE6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4gPSB7fTtcbiAgcHJpdmF0ZSBfZ3JvdXBOYW1lPzogc3RyaW5nO1xuXG4gIC8qKiBEaXNhYmxlIHZlcnNpb24gb3B0aW9uLiAqL1xuICBwdWJsaWMgdmVyc2lvbk9wdGlvbihlbmFibGU6IGZhbHNlKTogdGhpcztcbiAgLyoqXG4gICAqIFNldCBnbG9iYWwgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBUaGUgZmxhZ3Mgb2YgdGhlIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gZGVzYyAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgIFZlcnNpb24gb3B0aW9uIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgdmVyc2lvbk9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86IElDb21tYW5kT3B0aW9uPFBhcnRpYWw8Q08+LCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4gJiB7XG4gICAgICBnbG9iYWw6IHRydWU7XG4gICAgfSxcbiAgKTogdGhpcztcbiAgLyoqXG4gICAqIFNldCB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGZsYWdzIFRoZSBmbGFncyBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyAgVmVyc2lvbiBvcHRpb24gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUNvbW1hbmRPcHRpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgKTogdGhpcztcbiAgLyoqXG4gICAqIFNldCB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGZsYWdzIFRoZSBmbGFncyBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyAgVGhlIGFjdGlvbiBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqL1xuICBwdWJsaWMgdmVyc2lvbk9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86IElBY3Rpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgKTogdGhpcztcbiAgcHVibGljIHZlcnNpb25PcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyB8IGZhbHNlLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86XG4gICAgICB8IElBY3Rpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPlxuICAgICAgfCBJQ29tbWFuZE9wdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+XG4gICAgICB8IElDb21tYW5kT3B0aW9uPFBhcnRpYWw8Q08+LCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4gJiB7XG4gICAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICAgIH0sXG4gICk6IHRoaXMge1xuICAgIHRoaXMuX3ZlcnNpb25PcHRpb24gPSBmbGFncyA9PT0gZmFsc2UgPyBmbGFncyA6IHtcbiAgICAgIGZsYWdzLFxuICAgICAgZGVzYyxcbiAgICAgIG9wdHM6IHR5cGVvZiBvcHRzID09PSBcImZ1bmN0aW9uXCIgPyB7IGFjdGlvbjogb3B0cyB9IDogb3B0cyxcbiAgICB9O1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqIERpc2FibGUgaGVscCBvcHRpb24uICovXG4gIHB1YmxpYyBoZWxwT3B0aW9uKGVuYWJsZTogZmFsc2UpOiB0aGlzO1xuICAvKipcbiAgICogU2V0IGdsb2JhbCBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIGZsYWdzIFRoZSBmbGFncyBvZiB0aGUgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyAgSGVscCBvcHRpb24gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBoZWxwT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUNvbW1hbmRPcHRpb248UGFydGlhbDxDTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPiAmIHtcbiAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICB9LFxuICApOiB0aGlzO1xuICAvKipcbiAgICogU2V0IGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBIZWxwIG9wdGlvbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuICAvKipcbiAgICogU2V0IGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBUaGUgYWN0aW9uIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICovXG4gIHB1YmxpYyBoZWxwT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuICBwdWJsaWMgaGVscE9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nIHwgZmFsc2UsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzpcbiAgICAgIHwgSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+XG4gICAgICB8IElDb21tYW5kT3B0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD5cbiAgICAgIHwgSUNvbW1hbmRPcHRpb248UGFydGlhbDxDTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPiAmIHtcbiAgICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgICAgfSxcbiAgKTogdGhpcyB7XG4gICAgdGhpcy5faGVscE9wdGlvbiA9IGZsYWdzID09PSBmYWxzZSA/IGZsYWdzIDoge1xuICAgICAgZmxhZ3MsXG4gICAgICBkZXNjLFxuICAgICAgb3B0czogdHlwZW9mIG9wdHMgPT09IFwiZnVuY3Rpb25cIiA/IHsgYWN0aW9uOiBvcHRzIH0gOiBvcHRzLFxuICAgIH07XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogQWRkIG5ldyBzdWItY29tbWFuZC5cbiAgICogQHBhcmFtIG5hbWUgICAgICBDb21tYW5kIGRlZmluaXRpb24uIEUuZzogYG15LWNvbW1hbmQgPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPmBcbiAgICogQHBhcmFtIGNtZCAgICAgICBUaGUgbmV3IGNoaWxkIGNvbW1hbmQgdG8gcmVnaXN0ZXIuXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgY2hpbGQgY29tbWFuZC5cbiAgICovXG4gIHB1YmxpYyBjb21tYW5kPFxuICAgIEMgZXh0ZW5kcyBDb21tYW5kPFxuICAgICAgRyB8IHZvaWQgfCB1bmRlZmluZWQsXG4gICAgICBUIHwgdm9pZCB8IHVuZGVmaW5lZCxcbiAgICAgIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkLFxuICAgICAgQXJyYXk8dW5rbm93bj4sXG4gICAgICBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCxcbiAgICAgIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQsXG4gICAgICBPbmVPZjxDUCwgdGhpcz4gfCB1bmRlZmluZWRcbiAgICA+LFxuICAgIEcgZXh0ZW5kcyAoQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUEcgOiBNZXJnZTxDUEcsIENHPiksXG4gICAgVCBleHRlbmRzIChDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQVCA6IE1lcmdlPENQVCwgQ1Q+KSxcbiAgPihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY21kOiBDLFxuICAgIG92ZXJyaWRlPzogYm9vbGVhbixcbiAgKTogQyBleHRlbmRzIENvbW1hbmQ8XG4gICAgYW55LFxuICAgIGFueSxcbiAgICBpbmZlciBPcHRpb25zLFxuICAgIGluZmVyIEFyZ3VtZW50cyxcbiAgICBpbmZlciBHbG9iYWxPcHRpb25zLFxuICAgIGluZmVyIFR5cGVzLFxuICAgIGluZmVyIEdsb2JhbFR5cGVzLFxuICAgIGFueVxuICA+ID8gQ29tbWFuZDxcbiAgICBHLFxuICAgIFQsXG4gICAgT3B0aW9ucyxcbiAgICBBcmd1bWVudHMsXG4gICAgR2xvYmFsT3B0aW9ucyxcbiAgICBUeXBlcyxcbiAgICBHbG9iYWxUeXBlcyxcbiAgICBPbmVPZjxDUCwgdGhpcz5cbiAgPlxuICAgIDogbmV2ZXI7XG5cbiAgLyoqXG4gICAqIEFkZCBuZXcgc3ViLWNvbW1hbmQuXG4gICAqIEBwYXJhbSBuYW1lICAgICAgQ29tbWFuZCBkZWZpbml0aW9uLiBFLmc6IGBteS1jb21tYW5kIDxpbnB1dC1maWxlOnN0cmluZz4gPG91dHB1dC1maWxlOnN0cmluZz5gXG4gICAqIEBwYXJhbSBkZXNjICAgICAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBuZXcgY2hpbGQgY29tbWFuZC5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyBjaGlsZCBjb21tYW5kLlxuICAgKi9cbiAgcHVibGljIGNvbW1hbmQ8XG4gICAgTiBleHRlbmRzIHN0cmluZyxcbiAgICBBIGV4dGVuZHMgVHlwZWRDb21tYW5kQXJndW1lbnRzPFxuICAgICAgTixcbiAgICAgIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gQ1BUIDogTWVyZ2U8Q1BULCBDR1Q+XG4gICAgPixcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3ZlcnJpZGU/OiBib29sZWFuLFxuICApOiBDUEcgZXh0ZW5kcyBudW1iZXIgPyBDb21tYW5kPGFueT4gOiBDb21tYW5kPFxuICAgIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gQ1BHIDogTWVyZ2U8Q1BHLCBDRz4sXG4gICAgQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUFQgOiBNZXJnZTxDUFQsIENHVD4sXG4gICAgdm9pZCxcbiAgICBBLFxuICAgIHZvaWQsXG4gICAgdm9pZCxcbiAgICB2b2lkLFxuICAgIE9uZU9mPENQLCB0aGlzPlxuICA+O1xuXG4gIC8qKlxuICAgKiBBZGQgbmV3IHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gbmFtZUFuZEFyZ3VtZW50cyAgQ29tbWFuZCBkZWZpbml0aW9uLiBFLmc6IGBteS1jb21tYW5kIDxpbnB1dC1maWxlOnN0cmluZz4gPG91dHB1dC1maWxlOnN0cmluZz5gXG4gICAqIEBwYXJhbSBjbWRPckRlc2NyaXB0aW9uICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIG5ldyBjaGlsZCBjb21tYW5kLlxuICAgKiBAcGFyYW0gb3ZlcnJpZGUgICAgICAgICAgT3ZlcnJpZGUgZXhpc3RpbmcgY2hpbGQgY29tbWFuZC5cbiAgICovXG4gIGNvbW1hbmQoXG4gICAgbmFtZUFuZEFyZ3VtZW50czogc3RyaW5nLFxuICAgIGNtZE9yRGVzY3JpcHRpb24/OiBDb21tYW5kPGFueT4gfCBzdHJpbmcsXG4gICAgb3ZlcnJpZGU/OiBib29sZWFuLFxuICApOiBDb21tYW5kPGFueT4ge1xuICAgIHRoaXMucmVzZXQoKTtcblxuICAgIGNvbnN0IHJlc3VsdCA9IHNwbGl0QXJndW1lbnRzKG5hbWVBbmRBcmd1bWVudHMpO1xuXG4gICAgY29uc3QgbmFtZTogc3RyaW5nIHwgdW5kZWZpbmVkID0gcmVzdWx0LmZsYWdzLnNoaWZ0KCk7XG4gICAgY29uc3QgYWxpYXNlczogc3RyaW5nW10gPSByZXN1bHQuZmxhZ3M7XG5cbiAgICBpZiAoIW5hbWUpIHtcbiAgICAgIHRocm93IG5ldyBNaXNzaW5nQ29tbWFuZE5hbWUoKTtcbiAgICB9XG5cbiAgICBpZiAodGhpcy5nZXRCYXNlQ29tbWFuZChuYW1lLCB0cnVlKSkge1xuICAgICAgaWYgKCFvdmVycmlkZSkge1xuICAgICAgICB0aHJvdyBuZXcgRHVwbGljYXRlQ29tbWFuZE5hbWUobmFtZSk7XG4gICAgICB9XG4gICAgICB0aGlzLnJlbW92ZUNvbW1hbmQobmFtZSk7XG4gICAgfVxuXG4gICAgbGV0IGRlc2NyaXB0aW9uOiBzdHJpbmcgfCB1bmRlZmluZWQ7XG4gICAgbGV0IGNtZDogQ29tbWFuZDxhbnk+O1xuXG4gICAgaWYgKHR5cGVvZiBjbWRPckRlc2NyaXB0aW9uID09PSBcInN0cmluZ1wiKSB7XG4gICAgICBkZXNjcmlwdGlvbiA9IGNtZE9yRGVzY3JpcHRpb247XG4gICAgfVxuXG4gICAgaWYgKGNtZE9yRGVzY3JpcHRpb24gaW5zdGFuY2VvZiBDb21tYW5kKSB7XG4gICAgICBjbWQgPSBjbWRPckRlc2NyaXB0aW9uLnJlc2V0KCk7XG4gICAgfSBlbHNlIHtcbiAgICAgIGNtZCA9IG5ldyBDb21tYW5kKCk7XG4gICAgfVxuXG4gICAgY21kLl9uYW1lID0gbmFtZTtcbiAgICBjbWQuX3BhcmVudCA9IHRoaXM7XG5cbiAgICBpZiAoZGVzY3JpcHRpb24pIHtcbiAgICAgIGNtZC5kZXNjcmlwdGlvbihkZXNjcmlwdGlvbik7XG4gICAgfVxuXG4gICAgaWYgKHJlc3VsdC50eXBlRGVmaW5pdGlvbikge1xuICAgICAgY21kLmFyZ3VtZW50cyhyZXN1bHQudHlwZURlZmluaXRpb24pO1xuICAgIH1cblxuICAgIGFsaWFzZXMuZm9yRWFjaCgoYWxpYXM6IHN0cmluZykgPT4gY21kLmFsaWFzKGFsaWFzKSk7XG5cbiAgICB0aGlzLmNvbW1hbmRzLnNldChuYW1lLCBjbWQpO1xuXG4gICAgdGhpcy5zZWxlY3QobmFtZSk7XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgbmV3IGNvbW1hbmQgYWxpYXMuXG4gICAqIEBwYXJhbSBhbGlhcyBUaGEgbmFtZSBvZiB0aGUgYWxpYXMuXG4gICAqL1xuICBwdWJsaWMgYWxpYXMoYWxpYXM6IHN0cmluZyk6IHRoaXMge1xuICAgIGlmICh0aGlzLmNtZC5fbmFtZSA9PT0gYWxpYXMgfHwgdGhpcy5jbWQuYWxpYXNlcy5pbmNsdWRlcyhhbGlhcykpIHtcbiAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVDb21tYW5kQWxpYXMoYWxpYXMpO1xuICAgIH1cblxuICAgIHRoaXMuY21kLmFsaWFzZXMucHVzaChhbGlhcyk7XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBSZXNldCBpbnRlcm5hbCBjb21tYW5kIHJlZmVyZW5jZSB0byBtYWluIGNvbW1hbmQuICovXG4gIHB1YmxpYyByZXNldCgpOiBPbmVPZjxDUCwgdGhpcz4ge1xuICAgIHRoaXMuX2dyb3VwTmFtZSA9IHVuZGVmaW5lZDtcbiAgICB0aGlzLmNtZCA9IHRoaXM7XG4gICAgcmV0dXJuIHRoaXMgYXMgT25lT2Y8Q1AsIHRoaXM+O1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBpbnRlcm5hbCBjb21tYW5kIHBvaW50ZXIgdG8gY2hpbGQgY29tbWFuZCB3aXRoIGdpdmVuIG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIFRoZSBuYW1lIG9mIHRoZSBjb21tYW5kIHRvIHNlbGVjdC5cbiAgICovXG4gIHB1YmxpYyBzZWxlY3Q8XG4gICAgTyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCA9IGFueSxcbiAgICBBIGV4dGVuZHMgQXJyYXk8dW5rbm93bj4gPSBhbnksXG4gICAgRyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCA9IGFueSxcbiAgPihuYW1lOiBzdHJpbmcpOiBDb21tYW5kPENQRywgQ1BULCBPLCBBLCBHLCBDVCwgQ0dULCBDUD4ge1xuICAgIGNvbnN0IGNtZCA9IHRoaXMuZ2V0QmFzZUNvbW1hbmQobmFtZSwgdHJ1ZSk7XG5cbiAgICBpZiAoIWNtZCkge1xuICAgICAgdGhyb3cgbmV3IENvbW1hbmROb3RGb3VuZChuYW1lLCB0aGlzLmdldEJhc2VDb21tYW5kcyh0cnVlKSk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQgPSBjbWQ7XG5cbiAgICByZXR1cm4gdGhpcyBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKiogU1VCIEhBTkRMRVIgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKi9cblxuICAvKiogU2V0IGNvbW1hbmQgbmFtZS4gKi9cbiAgcHVibGljIG5hbWUobmFtZTogc3RyaW5nKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX25hbWUgPSBuYW1lO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjb21tYW5kIHZlcnNpb24uXG4gICAqIEBwYXJhbSB2ZXJzaW9uIFNlbWFudGljIHZlcnNpb24gc3RyaW5nIHN0cmluZyBvciBtZXRob2QgdGhhdCByZXR1cm5zIHRoZSB2ZXJzaW9uIHN0cmluZy5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uKFxuICAgIHZlcnNpb246XG4gICAgICB8IHN0cmluZ1xuICAgICAgfCBJVmVyc2lvbkhhbmRsZXI8UGFydGlhbDxDTz4sIFBhcnRpYWw8Q0E+LCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgKTogdGhpcyB7XG4gICAgaWYgKHR5cGVvZiB2ZXJzaW9uID09PSBcInN0cmluZ1wiKSB7XG4gICAgICB0aGlzLmNtZC52ZXIgPSAoKSA9PiB2ZXJzaW9uO1xuICAgIH0gZWxzZSBpZiAodHlwZW9mIHZlcnNpb24gPT09IFwiZnVuY3Rpb25cIikge1xuICAgICAgdGhpcy5jbWQudmVyID0gdmVyc2lvbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICBwdWJsaWMgbWV0YShuYW1lOiBzdHJpbmcsIHZhbHVlOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fbWV0YVtuYW1lXSA9IHZhbHVlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgcHVibGljIGdldE1ldGEoKTogUmVjb3JkPHN0cmluZywgc3RyaW5nPjtcbiAgcHVibGljIGdldE1ldGEobmFtZTogc3RyaW5nKTogc3RyaW5nO1xuICBwdWJsaWMgZ2V0TWV0YShuYW1lPzogc3RyaW5nKTogUmVjb3JkPHN0cmluZywgc3RyaW5nPiB8IHN0cmluZyB7XG4gICAgcmV0dXJuIHR5cGVvZiBuYW1lID09PSBcInVuZGVmaW5lZFwiID8gdGhpcy5fbWV0YSA6IHRoaXMuX21ldGFbbmFtZV07XG4gIH1cblxuICAvKipcbiAgICogU2V0IGNvbW1hbmQgaGVscC5cbiAgICogQHBhcmFtIGhlbHAgSGVscCBzdHJpbmcsIG1ldGhvZCwgb3IgY29uZmlnIGZvciBnZW5lcmF0b3IgdGhhdCByZXR1cm5zIHRoZSBoZWxwIHN0cmluZy5cbiAgICovXG4gIHB1YmxpYyBoZWxwKFxuICAgIGhlbHA6XG4gICAgICB8IHN0cmluZ1xuICAgICAgfCBJSGVscEhhbmRsZXI8UGFydGlhbDxDTz4sIFBhcnRpYWw8Q0E+LCBDRywgQ1BHPlxuICAgICAgfCBIZWxwT3B0aW9ucyxcbiAgKTogdGhpcyB7XG4gICAgaWYgKHR5cGVvZiBoZWxwID09PSBcInN0cmluZ1wiKSB7XG4gICAgICB0aGlzLmNtZC5faGVscCA9ICgpID0+IGhlbHA7XG4gICAgfSBlbHNlIGlmICh0eXBlb2YgaGVscCA9PT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICB0aGlzLmNtZC5faGVscCA9IGhlbHA7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRoaXMuY21kLl9oZWxwID0gKGNtZDogQ29tbWFuZCwgb3B0aW9uczogSGVscE9wdGlvbnMpOiBzdHJpbmcgPT5cbiAgICAgICAgSGVscEdlbmVyYXRvci5nZW5lcmF0ZShjbWQsIHsgLi4uaGVscCwgLi4ub3B0aW9ucyB9KTtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IHRoZSBsb25nIGNvbW1hbmQgZGVzY3JpcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjcmlwdGlvbiBUaGUgY29tbWFuZCBkZXNjcmlwdGlvbi5cbiAgICovXG4gIHB1YmxpYyBkZXNjcmlwdGlvbihcbiAgICBkZXNjcmlwdGlvbjogSURlc2NyaXB0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICk6IHRoaXMge1xuICAgIHRoaXMuY21kLmRlc2MgPSBkZXNjcmlwdGlvbjtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgdGhlIGNvbW1hbmQgdXNhZ2UuIERlZmF1bHRzIHRvIGFyZ3VtZW50cy5cbiAgICogQHBhcmFtIHVzYWdlIFRoZSBjb21tYW5kIHVzYWdlLlxuICAgKi9cbiAgcHVibGljIHVzYWdlKHVzYWdlOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fdXNhZ2UgPSB1c2FnZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBIaWRlIGNvbW1hbmQgZnJvbSBoZWxwLCBjb21wbGV0aW9ucywgZXRjLlxuICAgKi9cbiAgcHVibGljIGhpZGRlbigpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5pc0hpZGRlbiA9IHRydWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogTWFrZSBjb21tYW5kIGdsb2JhbGx5IGF2YWlsYWJsZS4gKi9cbiAgcHVibGljIGdsb2JhbCgpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5pc0dsb2JhbCA9IHRydWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogTWFrZSBjb21tYW5kIGV4ZWN1dGFibGUuICovXG4gIHB1YmxpYyBleGVjdXRhYmxlKCk6IHRoaXMge1xuICAgIHRoaXMuY21kLmlzRXhlY3V0YWJsZSA9IHRydWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IGNvbW1hbmQgYXJndW1lbnRzOlxuICAgKlxuICAgKiAgIDxyZXF1aXJlZEFyZzpzdHJpbmc+IFtvcHRpb25hbEFyZzogbnVtYmVyXSBbLi4ucmVzdEFyZ3M6c3RyaW5nXVxuICAgKi9cbiAgcHVibGljIGFyZ3VtZW50czxcbiAgICBBIGV4dGVuZHMgVHlwZWRBcmd1bWVudHM8TiwgTWVyZ2U8Q1BULCBNZXJnZTxDR1QsIENUPj4+LFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcgPSBzdHJpbmcsXG4gID4oXG4gICAgYXJnczogTixcbiAgKTogQ29tbWFuZDxDUEcsIENQVCwgQ08sIEEsIENHLCBDVCwgQ0dULCBDUD4ge1xuICAgIHRoaXMuY21kLmFyZ3NEZWZpbml0aW9uID0gYXJncztcbiAgICByZXR1cm4gdGhpcyBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKipcbiAgICogU2V0IGNvbW1hbmQgY2FsbGJhY2sgbWV0aG9kLlxuICAgKiBAcGFyYW0gZm4gQ29tbWFuZCBhY3Rpb24gaGFuZGxlci5cbiAgICovXG4gIHB1YmxpYyBhY3Rpb24oZm46IElBY3Rpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPik6IHRoaXMge1xuICAgIHRoaXMuY21kLmZuID0gZm47XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogRG9uJ3QgdGhyb3cgYW4gZXJyb3IgaWYgdGhlIGNvbW1hbmQgd2FzIGNhbGxlZCB3aXRob3V0IGFyZ3VtZW50cy5cbiAgICogQHBhcmFtIGFsbG93RW1wdHkgRW5hYmxlL2Rpc2FibGUgYWxsb3cgZW1wdHkuXG4gICAqL1xuICBwdWJsaWMgYWxsb3dFbXB0eShhbGxvd0VtcHR5ID0gdHJ1ZSk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9hbGxvd0VtcHR5ID0gYWxsb3dFbXB0eTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBFbmFibGUgc3RvcCBlYXJseS4gSWYgZW5hYmxlZCwgYWxsIGFyZ3VtZW50cyBzdGFydGluZyBmcm9tIHRoZSBmaXJzdCBub25cbiAgICogb3B0aW9uIGFyZ3VtZW50IHdpbGwgYmUgcGFzc2VkIGFzIGFyZ3VtZW50cyB3aXRoIHR5cGUgc3RyaW5nIHRvIHRoZSBjb21tYW5kXG4gICAqIGFjdGlvbiBoYW5kbGVyLlxuICAgKlxuICAgKiBGb3IgZXhhbXBsZTpcbiAgICogICAgIGBjb21tYW5kIC0tZGVidWctbGV2ZWwgd2FybmluZyBzZXJ2ZXIgLS1wb3J0IDgwYFxuICAgKlxuICAgKiBXaWxsIHJlc3VsdCBpbjpcbiAgICogICAgIC0gb3B0aW9uczogYHtkZWJ1Z0xldmVsOiAnd2FybmluZyd9YFxuICAgKiAgICAgLSBhcmdzOiBgWydzZXJ2ZXInLCAnLS1wb3J0JywgJzgwJ11gXG4gICAqXG4gICAqIEBwYXJhbSBzdG9wRWFybHkgRW5hYmxlL2Rpc2FibGUgc3RvcCBlYXJseS5cbiAgICovXG4gIHB1YmxpYyBzdG9wRWFybHkoc3RvcEVhcmx5ID0gdHJ1ZSk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9zdG9wRWFybHkgPSBzdG9wRWFybHk7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogRGlzYWJsZSBwYXJzaW5nIGFyZ3VtZW50cy4gSWYgZW5hYmxlZCB0aGUgcmF3IGFyZ3VtZW50cyB3aWxsIGJlIHBhc3NlZCB0b1xuICAgKiB0aGUgYWN0aW9uIGhhbmRsZXIuIFRoaXMgaGFzIG5vIGVmZmVjdCBmb3IgcGFyZW50IG9yIGNoaWxkIGNvbW1hbmRzLiBPbmx5XG4gICAqIGZvciB0aGUgY29tbWFuZCBvbiB3aGljaCB0aGlzIG1ldGhvZCB3YXMgY2FsbGVkLlxuICAgKiBAcGFyYW0gdXNlUmF3QXJncyBFbmFibGUvZGlzYWJsZSByYXcgYXJndW1lbnRzLlxuICAgKi9cbiAgcHVibGljIHVzZVJhd0FyZ3MoXG4gICAgdXNlUmF3QXJncyA9IHRydWUsXG4gICk6IENvbW1hbmQ8dm9pZCwgdm9pZCwgdm9pZCwgQXJyYXk8c3RyaW5nPiwgdm9pZCwgdm9pZCwgdm9pZCwgQ1A+IHtcbiAgICB0aGlzLmNtZC5fdXNlUmF3QXJncyA9IHVzZVJhd0FyZ3M7XG4gICAgcmV0dXJuIHRoaXMgYXMgQ29tbWFuZDxhbnk+O1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBkZWZhdWx0IGNvbW1hbmQuIFRoZSBkZWZhdWx0IGNvbW1hbmQgaXMgZXhlY3V0ZWQgd2hlbiB0aGUgcHJvZ3JhbVxuICAgKiB3YXMgY2FsbGVkIHdpdGhvdXQgYW55IGFyZ3VtZW50IGFuZCBpZiBubyBhY3Rpb24gaGFuZGxlciBpcyByZWdpc3RlcmVkLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBkZWZhdWx0IGNvbW1hbmQuXG4gICAqL1xuICBwdWJsaWMgZGVmYXVsdChuYW1lOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5kZWZhdWx0Q29tbWFuZCA9IG5hbWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICBwdWJsaWMgZ2xvYmFsVHlwZTxcbiAgICBIIGV4dGVuZHMgVHlwZU9yVHlwZUhhbmRsZXI8dW5rbm93bj4sXG4gICAgTiBleHRlbmRzIHN0cmluZyA9IHN0cmluZyxcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGhhbmRsZXI6IEgsXG4gICAgb3B0aW9ucz86IE9taXQ8SVR5cGVPcHRpb25zLCBcImdsb2JhbFwiPixcbiAgKTogQ29tbWFuZDxcbiAgICBDUEcsXG4gICAgQ1BULFxuICAgIENPLFxuICAgIENBLFxuICAgIENHLFxuICAgIENULFxuICAgIE1lcmdlPENHVCwgVHlwZWRUeXBlPE4sIEg+PixcbiAgICBDUFxuICA+IHtcbiAgICByZXR1cm4gdGhpcy50eXBlKG5hbWUsIGhhbmRsZXIsIHsgLi4ub3B0aW9ucywgZ2xvYmFsOiB0cnVlIH0pO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlZ2lzdGVyIGN1c3RvbSB0eXBlLlxuICAgKiBAcGFyYW0gbmFtZSAgICBUaGUgbmFtZSBvZiB0aGUgdHlwZS5cbiAgICogQHBhcmFtIGhhbmRsZXIgVGhlIGNhbGxiYWNrIG1ldGhvZCB0byBwYXJzZSB0aGUgdHlwZS5cbiAgICogQHBhcmFtIG9wdGlvbnMgVHlwZSBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIHR5cGU8XG4gICAgSCBleHRlbmRzIFR5cGVPclR5cGVIYW5kbGVyPHVua25vd24+LFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcgPSBzdHJpbmcsXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBoYW5kbGVyOiBILFxuICAgIG9wdGlvbnM/OiBJVHlwZU9wdGlvbnMsXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBDTyxcbiAgICBDQSxcbiAgICBDRyxcbiAgICBNZXJnZTxDVCwgVHlwZWRUeXBlPE4sIEg+PixcbiAgICBDR1QsXG4gICAgQ1BcbiAgPiB7XG4gICAgaWYgKHRoaXMuY21kLnR5cGVzLmdldChuYW1lKSAmJiAhb3B0aW9ucz8ub3ZlcnJpZGUpIHtcbiAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVUeXBlKG5hbWUpO1xuICAgIH1cblxuICAgIHRoaXMuY21kLnR5cGVzLnNldChuYW1lLCB7IC4uLm9wdGlvbnMsIG5hbWUsIGhhbmRsZXIgfSk7XG5cbiAgICBpZiAoXG4gICAgICBoYW5kbGVyIGluc3RhbmNlb2YgVHlwZSAmJlxuICAgICAgKHR5cGVvZiBoYW5kbGVyLmNvbXBsZXRlICE9PSBcInVuZGVmaW5lZFwiIHx8XG4gICAgICAgIHR5cGVvZiBoYW5kbGVyLnZhbHVlcyAhPT0gXCJ1bmRlZmluZWRcIilcbiAgICApIHtcbiAgICAgIGNvbnN0IGNvbXBsZXRlSGFuZGxlcjogSUNvbXBsZXRlSGFuZGxlciA9IChcbiAgICAgICAgY21kOiBDb21tYW5kLFxuICAgICAgICBwYXJlbnQ/OiBDb21tYW5kLFxuICAgICAgKSA9PiBoYW5kbGVyLmNvbXBsZXRlPy4oY21kLCBwYXJlbnQpIHx8IFtdO1xuICAgICAgdGhpcy5jb21wbGV0ZShuYW1lLCBjb21wbGV0ZUhhbmRsZXIsIG9wdGlvbnMpO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIHB1YmxpYyBnbG9iYWxDb21wbGV0ZShcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY29tcGxldGU6IElDb21wbGV0ZUhhbmRsZXIsXG4gICAgb3B0aW9ucz86IE9taXQ8SUNvbXBsZXRlT3B0aW9ucywgXCJnbG9iYWxcIj4sXG4gICk6IHRoaXMge1xuICAgIHJldHVybiB0aGlzLmNvbXBsZXRlKG5hbWUsIGNvbXBsZXRlLCB7IC4uLm9wdGlvbnMsIGdsb2JhbDogdHJ1ZSB9KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZWdpc3RlciBjb21tYW5kIHNwZWNpZmljIGN1c3RvbSB0eXBlLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgIFRoZSBuYW1lIG9mIHRoZSBjb21wbGV0aW9uLlxuICAgKiBAcGFyYW0gY29tcGxldGUgIFRoZSBjYWxsYmFjayBtZXRob2QgdG8gY29tcGxldGUgdGhlIHR5cGUuXG4gICAqIEBwYXJhbSBvcHRpb25zICAgQ29tcGxldGUgb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBjb21wbGV0ZShcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY29tcGxldGU6IElDb21wbGV0ZUhhbmRsZXI8XG4gICAgICBQYXJ0aWFsPENPPixcbiAgICAgIFBhcnRpYWw8Q0E+LFxuICAgICAgQ0csXG4gICAgICBDUEcsXG4gICAgICBDVCxcbiAgICAgIENHVCxcbiAgICAgIENQVCxcbiAgICAgIGFueVxuICAgID4sXG4gICAgb3B0aW9uczogSUNvbXBsZXRlT3B0aW9ucyAmIHsgZ2xvYmFsOiBib29sZWFuIH0sXG4gICk6IHRoaXM7XG4gIHB1YmxpYyBjb21wbGV0ZShcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY29tcGxldGU6IElDb21wbGV0ZUhhbmRsZXI8Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgICBvcHRpb25zPzogSUNvbXBsZXRlT3B0aW9ucyxcbiAgKTogdGhpcztcbiAgY29tcGxldGUoXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGNvbXBsZXRlOlxuICAgICAgfCBJQ29tcGxldGVIYW5kbGVyPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD5cbiAgICAgIHwgSUNvbXBsZXRlSGFuZGxlcjxcbiAgICAgICAgUGFydGlhbDxDTz4sXG4gICAgICAgIFBhcnRpYWw8Q0E+LFxuICAgICAgICBDRyxcbiAgICAgICAgQ1BHLFxuICAgICAgICBDVCxcbiAgICAgICAgQ0dULFxuICAgICAgICBDUFQsXG4gICAgICAgIGFueVxuICAgICAgPixcbiAgICBvcHRpb25zPzogSUNvbXBsZXRlT3B0aW9ucyxcbiAgKTogdGhpcyB7XG4gICAgaWYgKHRoaXMuY21kLmNvbXBsZXRpb25zLmhhcyhuYW1lKSAmJiAhb3B0aW9ucz8ub3ZlcnJpZGUpIHtcbiAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVDb21wbGV0aW9uKG5hbWUpO1xuICAgIH1cblxuICAgIHRoaXMuY21kLmNvbXBsZXRpb25zLnNldChuYW1lLCB7XG4gICAgICBuYW1lLFxuICAgICAgY29tcGxldGUsXG4gICAgICAuLi5vcHRpb25zLFxuICAgIH0pO1xuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogVGhyb3cgdmFsaWRhdGlvbiBlcnJvcnMgaW5zdGVhZCBvZiBjYWxsaW5nIGBEZW5vLmV4aXQoKWAgdG8gaGFuZGxlXG4gICAqIHZhbGlkYXRpb24gZXJyb3JzIG1hbnVhbGx5LlxuICAgKlxuICAgKiBBIHZhbGlkYXRpb24gZXJyb3IgaXMgdGhyb3duIHdoZW4gdGhlIGNvbW1hbmQgaXMgd3JvbmdseSB1c2VkIGJ5IHRoZSB1c2VyLlxuICAgKiBGb3IgZXhhbXBsZTogSWYgdGhlIHVzZXIgcGFzc2VzIHNvbWUgaW52YWxpZCBvcHRpb25zIG9yIGFyZ3VtZW50cyB0byB0aGVcbiAgICogY29tbWFuZC5cbiAgICpcbiAgICogVGhpcyBoYXMgbm8gZWZmZWN0IGZvciBwYXJlbnQgY29tbWFuZHMuIE9ubHkgZm9yIHRoZSBjb21tYW5kIG9uIHdoaWNoIHRoaXNcbiAgICogbWV0aG9kIHdhcyBjYWxsZWQgYW5kIGFsbCBjaGlsZCBjb21tYW5kcy5cbiAgICpcbiAgICogKipFeGFtcGxlOioqXG4gICAqXG4gICAqIGBgYFxuICAgKiB0cnkge1xuICAgKiAgIGNtZC5wYXJzZSgpO1xuICAgKiB9IGNhdGNoKGVycm9yKSB7XG4gICAqICAgaWYgKGVycm9yIGluc3RhbmNlb2YgVmFsaWRhdGlvbkVycm9yKSB7XG4gICAqICAgICBjbWQuc2hvd0hlbHAoKTtcbiAgICogICAgIERlbm8uZXhpdCgxKTtcbiAgICogICB9XG4gICAqICAgdGhyb3cgZXJyb3I7XG4gICAqIH1cbiAgICogYGBgXG4gICAqXG4gICAqIEBzZWUgVmFsaWRhdGlvbkVycm9yXG4gICAqL1xuICBwdWJsaWMgdGhyb3dFcnJvcnMoKTogdGhpcyB7XG4gICAgdGhpcy5jbWQudGhyb3dPbkVycm9yID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTYW1lIGFzIGAudGhyb3dFcnJvcnMoKWAgYnV0IGFsc28gcHJldmVudHMgY2FsbGluZyBgRGVuby5leGl0YCBhZnRlclxuICAgKiBwcmludGluZyBoZWxwIG9yIHZlcnNpb24gd2l0aCB0aGUgLS1oZWxwIGFuZCAtLXZlcnNpb24gb3B0aW9uLlxuICAgKi9cbiAgcHVibGljIG5vRXhpdCgpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fc2hvdWxkRXhpdCA9IGZhbHNlO1xuICAgIHRoaXMudGhyb3dFcnJvcnMoKTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBDaGVjayB3aGV0aGVyIHRoZSBjb21tYW5kIHNob3VsZCB0aHJvdyBlcnJvcnMgb3IgZXhpdC4gKi9cbiAgcHJvdGVjdGVkIHNob3VsZFRocm93RXJyb3JzKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmNtZC50aHJvd09uRXJyb3IgfHwgISF0aGlzLmNtZC5fcGFyZW50Py5zaG91bGRUaHJvd0Vycm9ycygpO1xuICB9XG5cbiAgLyoqIENoZWNrIHdoZXRoZXIgdGhlIGNvbW1hbmQgc2hvdWxkIGV4aXQgYWZ0ZXIgcHJpbnRpbmcgaGVscCBvciB2ZXJzaW9uLiAqL1xuICBwcm90ZWN0ZWQgc2hvdWxkRXhpdCgpOiBib29sZWFuIHtcbiAgICByZXR1cm4gdGhpcy5jbWQuX3Nob3VsZEV4aXQgPz8gdGhpcy5jbWQuX3BhcmVudD8uc2hvdWxkRXhpdCgpID8/IHRydWU7XG4gIH1cblxuICBwdWJsaWMgZ2xvYmFsT3B0aW9uPFxuICAgIEYgZXh0ZW5kcyBzdHJpbmcsXG4gICAgRyBleHRlbmRzIFR5cGVkT3B0aW9uPEYsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUiwgRD4sXG4gICAgTUcgZXh0ZW5kcyBNYXBWYWx1ZTxHLCBWLCBDPixcbiAgICBSIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBDIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJjb2xsZWN0XCJdID0gdW5kZWZpbmVkLFxuICAgIEQgPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBmbGFnczogRixcbiAgICBkZXNjOiBzdHJpbmcsXG4gICAgb3B0cz86XG4gICAgICB8IE9taXQ8XG4gICAgICAgIElDb21tYW5kR2xvYmFsT3B0aW9uPFxuICAgICAgICAgIFBhcnRpYWw8Q08+LFxuICAgICAgICAgIENBLFxuICAgICAgICAgIE1lcmdlT3B0aW9uczxGLCBDRywgRz4sXG4gICAgICAgICAgQ1BHLFxuICAgICAgICAgIENULFxuICAgICAgICAgIENHVCxcbiAgICAgICAgICBDUFQsXG4gICAgICAgICAgQ1BcbiAgICAgICAgPixcbiAgICAgICAgXCJ2YWx1ZVwiXG4gICAgICA+XG4gICAgICAgICYge1xuICAgICAgICAgIGRlZmF1bHQ/OiBJRGVmYXVsdFZhbHVlPEQ+O1xuICAgICAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgICAgICBjb2xsZWN0PzogQztcbiAgICAgICAgICB2YWx1ZT86IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPjtcbiAgICAgICAgfVxuICAgICAgfCBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj4sXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBDTyxcbiAgICBDQSxcbiAgICBNZXJnZU9wdGlvbnM8RiwgQ0csIE1HPixcbiAgICBDVCxcbiAgICBDR1QsXG4gICAgQ1BcbiAgPiB7XG4gICAgaWYgKHR5cGVvZiBvcHRzID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHJldHVybiB0aGlzLm9wdGlvbihcbiAgICAgICAgZmxhZ3MsXG4gICAgICAgIGRlc2MsXG4gICAgICAgIHsgdmFsdWU6IG9wdHMsIGdsb2JhbDogdHJ1ZSB9IGFzIElDb21tYW5kT3B0aW9uLFxuICAgICAgKSBhcyBDb21tYW5kPGFueT47XG4gICAgfVxuICAgIHJldHVybiB0aGlzLm9wdGlvbihcbiAgICAgIGZsYWdzLFxuICAgICAgZGVzYyxcbiAgICAgIHsgLi4ub3B0cywgZ2xvYmFsOiB0cnVlIH0gYXMgSUNvbW1hbmRPcHRpb24sXG4gICAgKSBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKipcbiAgICogRW5hYmxlIGdyb3VwaW5nIG9mIG9wdGlvbnMgYW5kIHNldCB0aGUgbmFtZSBvZiB0aGUgZ3JvdXAuXG4gICAqIEFsbCBvcHRpb24gd2hpY2ggYXJlIGFkZGVkIGFmdGVyIGNhbGxpbmcgdGhlIGAuZ3JvdXAoKWAgbWV0aG9kIHdpbGwgYmVcbiAgICogZ3JvdXBlZCBpbiB0aGUgaGVscCBvdXRwdXQuIElmIHRoZSBgLmdyb3VwKClgIG1ldGhvZCBjYW4gYmUgdXNlIG11bHRpcGxlXG4gICAqIHRpbWVzIHRvIGNyZWF0ZSBtb3JlIGdyb3Vwcy5cbiAgICpcbiAgICogQHBhcmFtIG5hbWUgVGhlIG5hbWUgb2YgdGhlIG9wdGlvbiBncm91cC5cbiAgICovXG4gIHB1YmxpYyBncm91cChuYW1lOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fZ3JvdXBOYW1lID0gbmFtZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgYSBuZXcgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgRmxhZ3Mgc3RyaW5nIGUuZzogLWgsIC0taGVscCwgLS1tYW51YWwgPHJlcXVpcmVkQXJnOnN0cmluZz4gW29wdGlvbmFsQXJnOm51bWJlcl0gWy4uLnJlc3RBcmdzOnN0cmluZ11cbiAgICogQHBhcmFtIGRlc2MgRmxhZyBkZXNjcmlwdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgRmxhZyBvcHRpb25zIG9yIGN1c3RvbSBoYW5kbGVyIGZvciBwcm9jZXNzaW5nIGZsYWcgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgb3B0aW9uPFxuICAgIEYgZXh0ZW5kcyBzdHJpbmcsXG4gICAgRyBleHRlbmRzIFR5cGVkT3B0aW9uPEYsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUiwgRD4sXG4gICAgTUcgZXh0ZW5kcyBNYXBWYWx1ZTxHLCBWLCBDPixcbiAgICBSIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBDIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJjb2xsZWN0XCJdID0gdW5kZWZpbmVkLFxuICAgIEQgPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBmbGFnczogRixcbiAgICBkZXNjOiBzdHJpbmcsXG4gICAgb3B0czpcbiAgICAgIHwgT21pdDxcbiAgICAgICAgSUNvbW1hbmRPcHRpb248XG4gICAgICAgICAgUGFydGlhbDxDTz4sXG4gICAgICAgICAgQ0EsXG4gICAgICAgICAgTWVyZ2VPcHRpb25zPEYsIENHLCBHPixcbiAgICAgICAgICBDUEcsXG4gICAgICAgICAgQ1QsXG4gICAgICAgICAgQ0dULFxuICAgICAgICAgIENQVCxcbiAgICAgICAgICBDUFxuICAgICAgICA+LFxuICAgICAgICBcInZhbHVlXCJcbiAgICAgID5cbiAgICAgICAgJiB7XG4gICAgICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgICAgICAgIGRlZmF1bHQ/OiBJRGVmYXVsdFZhbHVlPEQ+O1xuICAgICAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgICAgICBjb2xsZWN0PzogQztcbiAgICAgICAgICB2YWx1ZT86IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPjtcbiAgICAgICAgfVxuICAgICAgfCBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj4sXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBDTyxcbiAgICBDQSxcbiAgICBNZXJnZU9wdGlvbnM8RiwgQ0csIE1HPixcbiAgICBDVCxcbiAgICBDR1QsXG4gICAgQ1BcbiAgPjtcblxuICBwdWJsaWMgb3B0aW9uPFxuICAgIEYgZXh0ZW5kcyBzdHJpbmcsXG4gICAgTyBleHRlbmRzIFR5cGVkT3B0aW9uPEYsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUiwgRD4sXG4gICAgTU8gZXh0ZW5kcyBNYXBWYWx1ZTxPLCBWLCBDPixcbiAgICBSIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBDIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJjb2xsZWN0XCJdID0gdW5kZWZpbmVkLFxuICAgIEQgPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBmbGFnczogRixcbiAgICBkZXNjOiBzdHJpbmcsXG4gICAgb3B0cz86XG4gICAgICB8IE9taXQ8XG4gICAgICAgIElDb21tYW5kT3B0aW9uPE1lcmdlT3B0aW9uczxGLCBDTywgTU8+LCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICAgICAgIFwidmFsdWVcIlxuICAgICAgPlxuICAgICAgICAmIHtcbiAgICAgICAgICBkZWZhdWx0PzogSURlZmF1bHRWYWx1ZTxEPjtcbiAgICAgICAgICByZXF1aXJlZD86IFI7XG4gICAgICAgICAgY29sbGVjdD86IEM7XG4gICAgICAgICAgdmFsdWU/OiBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPE8+PiwgVj47XG4gICAgICAgIH1cbiAgICAgIHwgSUZsYWdWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxPPj4sIFY+LFxuICApOiBDb21tYW5kPFxuICAgIENQRyxcbiAgICBDUFQsXG4gICAgTWVyZ2VPcHRpb25zPEYsIENPLCBNTz4sXG4gICAgQ0EsXG4gICAgQ0csXG4gICAgQ1QsXG4gICAgQ0dULFxuICAgIENQXG4gID47XG5cbiAgcHVibGljIG9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nLFxuICAgIGRlc2M6IHN0cmluZyxcbiAgICBvcHRzPzogSUNvbW1hbmRPcHRpb24gfCBJRmxhZ1ZhbHVlSGFuZGxlcixcbiAgKTogQ29tbWFuZDxhbnk+IHtcbiAgICBpZiAodHlwZW9mIG9wdHMgPT09IFwiZnVuY3Rpb25cIikge1xuICAgICAgcmV0dXJuIHRoaXMub3B0aW9uKGZsYWdzLCBkZXNjLCB7IHZhbHVlOiBvcHRzIH0pO1xuICAgIH1cblxuICAgIGNvbnN0IHJlc3VsdCA9IHNwbGl0QXJndW1lbnRzKGZsYWdzKTtcblxuICAgIGNvbnN0IGFyZ3M6IElBcmd1bWVudFtdID0gcmVzdWx0LnR5cGVEZWZpbml0aW9uXG4gICAgICA/IHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbihyZXN1bHQudHlwZURlZmluaXRpb24pXG4gICAgICA6IFtdO1xuXG4gICAgY29uc3Qgb3B0aW9uOiBJT3B0aW9uID0ge1xuICAgICAgLi4ub3B0cyxcbiAgICAgIG5hbWU6IFwiXCIsXG4gICAgICBkZXNjcmlwdGlvbjogZGVzYyxcbiAgICAgIGFyZ3MsXG4gICAgICBmbGFnczogcmVzdWx0LmZsYWdzLFxuICAgICAgZXF1YWxzU2lnbjogcmVzdWx0LmVxdWFsc1NpZ24sXG4gICAgICB0eXBlRGVmaW5pdGlvbjogcmVzdWx0LnR5cGVEZWZpbml0aW9uLFxuICAgICAgZ3JvdXBOYW1lOiB0aGlzLl9ncm91cE5hbWUsXG4gICAgfTtcblxuICAgIGlmIChvcHRpb24uc2VwYXJhdG9yKSB7XG4gICAgICBmb3IgKGNvbnN0IGFyZyBvZiBhcmdzKSB7XG4gICAgICAgIGlmIChhcmcubGlzdCkge1xuICAgICAgICAgIGFyZy5zZXBhcmF0b3IgPSBvcHRpb24uc2VwYXJhdG9yO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgZm9yIChjb25zdCBwYXJ0IG9mIG9wdGlvbi5mbGFncykge1xuICAgICAgY29uc3QgYXJnID0gcGFydC50cmltKCk7XG4gICAgICBjb25zdCBpc0xvbmcgPSAvXi0tLy50ZXN0KGFyZyk7XG4gICAgICBjb25zdCBuYW1lID0gaXNMb25nID8gYXJnLnNsaWNlKDIpIDogYXJnLnNsaWNlKDEpO1xuXG4gICAgICBpZiAodGhpcy5jbWQuZ2V0QmFzZU9wdGlvbihuYW1lLCB0cnVlKSkge1xuICAgICAgICBpZiAob3B0cz8ub3ZlcnJpZGUpIHtcbiAgICAgICAgICB0aGlzLnJlbW92ZU9wdGlvbihuYW1lKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICB0aHJvdyBuZXcgRHVwbGljYXRlT3B0aW9uTmFtZShuYW1lKTtcbiAgICAgICAgfVxuICAgICAgfVxuXG4gICAgICBpZiAoIW9wdGlvbi5uYW1lICYmIGlzTG9uZykge1xuICAgICAgICBvcHRpb24ubmFtZSA9IG5hbWU7XG4gICAgICB9IGVsc2UgaWYgKCFvcHRpb24uYWxpYXNlcykge1xuICAgICAgICBvcHRpb24uYWxpYXNlcyA9IFtuYW1lXTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIG9wdGlvbi5hbGlhc2VzLnB1c2gobmFtZSk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKG9wdGlvbi5wcmVwZW5kKSB7XG4gICAgICB0aGlzLmNtZC5vcHRpb25zLnVuc2hpZnQob3B0aW9uKTtcbiAgICB9IGVsc2Uge1xuICAgICAgdGhpcy5jbWQub3B0aW9ucy5wdXNoKG9wdGlvbik7XG4gICAgfVxuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogQWRkIG5ldyBjb21tYW5kIGV4YW1wbGUuXG4gICAqIEBwYXJhbSBuYW1lICAgICAgICAgIE5hbWUgb2YgdGhlIGV4YW1wbGUuXG4gICAqIEBwYXJhbSBkZXNjcmlwdGlvbiAgIFRoZSBjb250ZW50IG9mIHRoZSBleGFtcGxlLlxuICAgKi9cbiAgcHVibGljIGV4YW1wbGUobmFtZTogc3RyaW5nLCBkZXNjcmlwdGlvbjogc3RyaW5nKTogdGhpcyB7XG4gICAgaWYgKHRoaXMuY21kLmhhc0V4YW1wbGUobmFtZSkpIHtcbiAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVFeGFtcGxlKG5hbWUpO1xuICAgIH1cblxuICAgIHRoaXMuY21kLmV4YW1wbGVzLnB1c2goeyBuYW1lLCBkZXNjcmlwdGlvbiB9KTtcblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgcHVibGljIGdsb2JhbEVudjxcbiAgICBOIGV4dGVuZHMgc3RyaW5nLFxuICAgIEcgZXh0ZW5kcyBUeXBlZEVudjxOLCBQLCBDTywgTWVyZ2U8Q1BULCBNZXJnZTxDR1QsIENUPj4sIFI+LFxuICAgIE1HIGV4dGVuZHMgTWFwVmFsdWU8RywgVj4sXG4gICAgUiBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicmVxdWlyZWRcIl0gPSB1bmRlZmluZWQsXG4gICAgUCBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicHJlZml4XCJdID0gdW5kZWZpbmVkLFxuICAgIFYgPSB1bmRlZmluZWQsXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBkZXNjcmlwdGlvbjogc3RyaW5nLFxuICAgIG9wdGlvbnM/OiBPbWl0PElHbG9iYWxFbnZWYXJPcHRpb25zLCBcInZhbHVlXCI+ICYge1xuICAgICAgcmVxdWlyZWQ/OiBSO1xuICAgICAgcHJlZml4PzogUDtcbiAgICAgIHZhbHVlPzogSUVudlZhclZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj47XG4gICAgfSxcbiAgKTogQ29tbWFuZDxDUEcsIENQVCwgQ08sIENBLCBNZXJnZTxDRywgTUc+LCBDVCwgQ0dULCBDUD4ge1xuICAgIHJldHVybiB0aGlzLmVudihcbiAgICAgIG5hbWUsXG4gICAgICBkZXNjcmlwdGlvbixcbiAgICAgIHsgLi4ub3B0aW9ucywgZ2xvYmFsOiB0cnVlIH0gYXMgSUVudlZhck9wdGlvbnMsXG4gICAgKSBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKipcbiAgICogQWRkIG5ldyBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIG5hbWUgICAgICAgICAgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBkZXNjcmlwdGlvbiAgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBvcHRpb25zICAgICAgIEVudmlyb25tZW50IHZhcmlhYmxlIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZW52PFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcsXG4gICAgRyBleHRlbmRzIFR5cGVkRW52PE4sIFAsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUj4sXG4gICAgTUcgZXh0ZW5kcyBNYXBWYWx1ZTxHLCBWPixcbiAgICBSIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBQIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJwcmVmaXhcIl0gPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGRlc2NyaXB0aW9uOiBzdHJpbmcsXG4gICAgb3B0aW9uczogT21pdDxJRW52VmFyT3B0aW9ucywgXCJ2YWx1ZVwiPiAmIHtcbiAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgIHByZWZpeD86IFA7XG4gICAgICB2YWx1ZT86IElFbnZWYXJWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxHPj4sIFY+O1xuICAgIH0sXG4gICk6IENvbW1hbmQ8Q1BHLCBDUFQsIENPLCBDQSwgTWVyZ2U8Q0csIE1HPiwgQ1QsIENHVCwgQ1A+O1xuXG4gIHB1YmxpYyBlbnY8XG4gICAgTiBleHRlbmRzIHN0cmluZyxcbiAgICBPIGV4dGVuZHMgVHlwZWRFbnY8TiwgUCwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSPixcbiAgICBNTyBleHRlbmRzIE1hcFZhbHVlPE8sIFY+LFxuICAgIFIgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIFAgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInByZWZpeFwiXSA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgZGVzY3JpcHRpb246IHN0cmluZyxcbiAgICBvcHRpb25zPzogT21pdDxJRW52VmFyT3B0aW9ucywgXCJ2YWx1ZVwiPiAmIHtcbiAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgIHByZWZpeD86IFA7XG4gICAgICB2YWx1ZT86IElFbnZWYXJWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxPPj4sIFY+O1xuICAgIH0sXG4gICk6IENvbW1hbmQ8Q1BHLCBDUFQsIE1lcmdlPENPLCBNTz4sIENBLCBDRywgQ1QsIENHVCwgQ1A+O1xuXG4gIHB1YmxpYyBlbnYoXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGRlc2NyaXB0aW9uOiBzdHJpbmcsXG4gICAgb3B0aW9ucz86IElFbnZWYXJPcHRpb25zLFxuICApOiBDb21tYW5kPGFueT4ge1xuICAgIGNvbnN0IHJlc3VsdCA9IHNwbGl0QXJndW1lbnRzKG5hbWUpO1xuXG4gICAgaWYgKCFyZXN1bHQudHlwZURlZmluaXRpb24pIHtcbiAgICAgIHJlc3VsdC50eXBlRGVmaW5pdGlvbiA9IFwiPHZhbHVlOmJvb2xlYW4+XCI7XG4gICAgfVxuXG4gICAgaWYgKHJlc3VsdC5mbGFncy5zb21lKChlbnZOYW1lKSA9PiB0aGlzLmNtZC5nZXRCYXNlRW52VmFyKGVudk5hbWUsIHRydWUpKSkge1xuICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZUVudmlyb25tZW50VmFyaWFibGUobmFtZSk7XG4gICAgfVxuXG4gICAgY29uc3QgZGV0YWlsczogSUFyZ3VtZW50W10gPSBwYXJzZUFyZ3VtZW50c0RlZmluaXRpb24oXG4gICAgICByZXN1bHQudHlwZURlZmluaXRpb24sXG4gICAgKTtcblxuICAgIGlmIChkZXRhaWxzLmxlbmd0aCA+IDEpIHtcbiAgICAgIHRocm93IG5ldyBFbnZpcm9ubWVudFZhcmlhYmxlU2luZ2xlVmFsdWUobmFtZSk7XG4gICAgfSBlbHNlIGlmIChkZXRhaWxzLmxlbmd0aCAmJiBkZXRhaWxzWzBdLm9wdGlvbmFsVmFsdWUpIHtcbiAgICAgIHRocm93IG5ldyBFbnZpcm9ubWVudFZhcmlhYmxlT3B0aW9uYWxWYWx1ZShuYW1lKTtcbiAgICB9IGVsc2UgaWYgKGRldGFpbHMubGVuZ3RoICYmIGRldGFpbHNbMF0udmFyaWFkaWMpIHtcbiAgICAgIHRocm93IG5ldyBFbnZpcm9ubWVudFZhcmlhYmxlVmFyaWFkaWNWYWx1ZShuYW1lKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZC5lbnZWYXJzLnB1c2goe1xuICAgICAgbmFtZTogcmVzdWx0LmZsYWdzWzBdLFxuICAgICAgbmFtZXM6IHJlc3VsdC5mbGFncyxcbiAgICAgIGRlc2NyaXB0aW9uLFxuICAgICAgdHlwZTogZGV0YWlsc1swXS50eXBlLFxuICAgICAgZGV0YWlsczogZGV0YWlscy5zaGlmdCgpIGFzIElBcmd1bWVudCxcbiAgICAgIC4uLm9wdGlvbnMsXG4gICAgfSk7XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKiBNQUlOIEhBTkRMRVIgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqL1xuXG4gIC8qKlxuICAgKiBQYXJzZSBjb21tYW5kIGxpbmUgYXJndW1lbnRzIGFuZCBleGVjdXRlIG1hdGNoZWQgY29tbWFuZC5cbiAgICogQHBhcmFtIGFyZ3MgQ29tbWFuZCBsaW5lIGFyZ3MgdG8gcGFyc2UuIEV4OiBgY21kLnBhcnNlKCBEZW5vLmFyZ3MgKWBcbiAgICovXG4gIHB1YmxpYyBhc3luYyBwYXJzZShcbiAgICBhcmdzOiBzdHJpbmdbXSA9IERlbm8uYXJncyxcbiAgKTogUHJvbWlzZTxcbiAgICBDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IElQYXJzZVJlc3VsdDxcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgQXJyYXk8dW5rbm93bj4sXG4gICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgdW5kZWZpbmVkXG4gICAgPlxuICAgICAgOiBJUGFyc2VSZXN1bHQ8XG4gICAgICAgIE1hcFR5cGVzPENPPixcbiAgICAgICAgTWFwVHlwZXM8Q0E+LFxuICAgICAgICBNYXBUeXBlczxDRz4sXG4gICAgICAgIE1hcFR5cGVzPENQRz4sXG4gICAgICAgIENULFxuICAgICAgICBDR1QsXG4gICAgICAgIENQVCxcbiAgICAgICAgQ1BcbiAgICAgID5cbiAgPiB7XG4gICAgdHJ5IHtcbiAgICAgIHRoaXMucmVzZXQoKTtcbiAgICAgIHRoaXMucmVnaXN0ZXJEZWZhdWx0cygpO1xuICAgICAgdGhpcy5yYXdBcmdzID0gYXJncztcblxuICAgICAgaWYgKGFyZ3MubGVuZ3RoID4gMCkge1xuICAgICAgICBjb25zdCBzdWJDb21tYW5kID0gdGhpcy5nZXRDb21tYW5kKGFyZ3NbMF0sIHRydWUpO1xuICAgICAgICBpZiAoc3ViQ29tbWFuZCkge1xuICAgICAgICAgIHN1YkNvbW1hbmQuX2dsb2JhbFBhcmVudCA9IHRoaXM7XG4gICAgICAgICAgcmV0dXJuIHN1YkNvbW1hbmQucGFyc2UoXG4gICAgICAgICAgICB0aGlzLnJhd0FyZ3Muc2xpY2UoMSksXG4gICAgICAgICAgKSBhcyBhbnk7XG4gICAgICAgIH1cbiAgICAgIH1cblxuICAgICAgaWYgKHRoaXMuaXNFeGVjdXRhYmxlKSB7XG4gICAgICAgIGF3YWl0IHRoaXMuZXhlY3V0ZUV4ZWN1dGFibGUodGhpcy5yYXdBcmdzKTtcbiAgICAgICAgcmV0dXJuIHtcbiAgICAgICAgICBvcHRpb25zOiB7fSxcbiAgICAgICAgICBhcmdzOiBbXSxcbiAgICAgICAgICBjbWQ6IHRoaXMsXG4gICAgICAgICAgbGl0ZXJhbDogW10sXG4gICAgICAgIH0gYXMgYW55O1xuICAgICAgfSBlbHNlIGlmICh0aGlzLl91c2VSYXdBcmdzKSB7XG4gICAgICAgIGNvbnN0IGVudjogUmVjb3JkPHN0cmluZywgdW5rbm93bj4gPSBhd2FpdCB0aGlzLnBhcnNlRW52VmFycygpO1xuICAgICAgICByZXR1cm4gdGhpcy5leGVjdXRlKGVudiwgLi4udGhpcy5yYXdBcmdzKSBhcyBhbnk7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBjb25zdCBlbnY6IFJlY29yZDxzdHJpbmcsIHVua25vd24+ID0gYXdhaXQgdGhpcy5wYXJzZUVudlZhcnMoKTtcbiAgICAgICAgY29uc3QgeyBhY3Rpb25PcHRpb24sIGZsYWdzLCB1bmtub3duLCBsaXRlcmFsIH0gPSB0aGlzXG4gICAgICAgICAgLnBhcnNlRmxhZ3MoXG4gICAgICAgICAgICB0aGlzLnJhd0FyZ3MsXG4gICAgICAgICAgICBlbnYsXG4gICAgICAgICAgKTtcblxuICAgICAgICB0aGlzLmxpdGVyYWxBcmdzID0gbGl0ZXJhbDtcblxuICAgICAgICBjb25zdCBvcHRpb25zOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiA9IHsgLi4uZW52LCAuLi5mbGFncyB9O1xuICAgICAgICBjb25zdCBwYXJhbXMgPSB0aGlzLnBhcnNlQXJndW1lbnRzKHVua25vd24sIG9wdGlvbnMpO1xuXG4gICAgICAgIGlmIChhY3Rpb25PcHRpb24pIHtcbiAgICAgICAgICBhd2FpdCBhY3Rpb25PcHRpb24uYWN0aW9uLmNhbGwodGhpcywgb3B0aW9ucywgLi4ucGFyYW1zKTtcbiAgICAgICAgICBpZiAoYWN0aW9uT3B0aW9uLnN0YW5kYWxvbmUpIHtcbiAgICAgICAgICAgIHJldHVybiB7XG4gICAgICAgICAgICAgIG9wdGlvbnMsXG4gICAgICAgICAgICAgIGFyZ3M6IHBhcmFtcyxcbiAgICAgICAgICAgICAgY21kOiB0aGlzLFxuICAgICAgICAgICAgICBsaXRlcmFsOiB0aGlzLmxpdGVyYWxBcmdzLFxuICAgICAgICAgICAgfSBhcyBhbnk7XG4gICAgICAgICAgfVxuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIHRoaXMuZXhlY3V0ZShvcHRpb25zLCAuLi5wYXJhbXMpIGFzIGFueTtcbiAgICAgIH1cbiAgICB9IGNhdGNoIChlcnJvcjogdW5rbm93bikge1xuICAgICAgaWYgKGVycm9yIGluc3RhbmNlb2YgRXJyb3IpIHtcbiAgICAgICAgdGhyb3cgdGhpcy5lcnJvcihlcnJvcik7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICB0aHJvdyB0aGlzLmVycm9yKG5ldyBFcnJvcihgW25vbi1lcnJvci10aHJvd25dICR7ZXJyb3J9YCkpO1xuICAgICAgfVxuICAgIH1cbiAgfVxuXG4gIC8qKiBSZWdpc3RlciBkZWZhdWx0IG9wdGlvbnMgbGlrZSBgLS12ZXJzaW9uYCBhbmQgYC0taGVscGAuICovXG4gIHByaXZhdGUgcmVnaXN0ZXJEZWZhdWx0cygpOiB0aGlzIHtcbiAgICBpZiAodGhpcy5oYXNEZWZhdWx0cyB8fCB0aGlzLmdldFBhcmVudCgpKSB7XG4gICAgICByZXR1cm4gdGhpcztcbiAgICB9XG4gICAgdGhpcy5oYXNEZWZhdWx0cyA9IHRydWU7XG5cbiAgICB0aGlzLnJlc2V0KCk7XG5cbiAgICAhdGhpcy50eXBlcy5oYXMoXCJzdHJpbmdcIikgJiZcbiAgICAgIHRoaXMudHlwZShcInN0cmluZ1wiLCBuZXcgU3RyaW5nVHlwZSgpLCB7IGdsb2JhbDogdHJ1ZSB9KTtcbiAgICAhdGhpcy50eXBlcy5oYXMoXCJudW1iZXJcIikgJiZcbiAgICAgIHRoaXMudHlwZShcIm51bWJlclwiLCBuZXcgTnVtYmVyVHlwZSgpLCB7IGdsb2JhbDogdHJ1ZSB9KTtcbiAgICAhdGhpcy50eXBlcy5oYXMoXCJpbnRlZ2VyXCIpICYmXG4gICAgICB0aGlzLnR5cGUoXCJpbnRlZ2VyXCIsIG5ldyBJbnRlZ2VyVHlwZSgpLCB7IGdsb2JhbDogdHJ1ZSB9KTtcbiAgICAhdGhpcy50eXBlcy5oYXMoXCJib29sZWFuXCIpICYmXG4gICAgICB0aGlzLnR5cGUoXCJib29sZWFuXCIsIG5ldyBCb29sZWFuVHlwZSgpLCB7IGdsb2JhbDogdHJ1ZSB9KTtcbiAgICAhdGhpcy50eXBlcy5oYXMoXCJmaWxlXCIpICYmXG4gICAgICB0aGlzLnR5cGUoXCJmaWxlXCIsIG5ldyBGaWxlVHlwZSgpLCB7IGdsb2JhbDogdHJ1ZSB9KTtcblxuICAgIGlmICghdGhpcy5faGVscCkge1xuICAgICAgdGhpcy5oZWxwKHtcbiAgICAgICAgaGludHM6IHRydWUsXG4gICAgICAgIHR5cGVzOiBmYWxzZSxcbiAgICAgIH0pO1xuICAgIH1cblxuICAgIGlmICh0aGlzLl92ZXJzaW9uT3B0aW9uICE9PSBmYWxzZSAmJiAodGhpcy5fdmVyc2lvbk9wdGlvbiB8fCB0aGlzLnZlcikpIHtcbiAgICAgIHRoaXMub3B0aW9uKFxuICAgICAgICB0aGlzLl92ZXJzaW9uT3B0aW9uPy5mbGFncyB8fCBcIi1WLCAtLXZlcnNpb25cIixcbiAgICAgICAgdGhpcy5fdmVyc2lvbk9wdGlvbj8uZGVzYyB8fFxuICAgICAgICAgIFwiU2hvdyB0aGUgdmVyc2lvbiBudW1iZXIgZm9yIHRoaXMgcHJvZ3JhbS5cIixcbiAgICAgICAge1xuICAgICAgICAgIHN0YW5kYWxvbmU6IHRydWUsXG4gICAgICAgICAgcHJlcGVuZDogdHJ1ZSxcbiAgICAgICAgICBhY3Rpb246IGFzeW5jIGZ1bmN0aW9uICgpIHtcbiAgICAgICAgICAgIGNvbnN0IGxvbmcgPSB0aGlzLmdldFJhd0FyZ3MoKS5pbmNsdWRlcyhgLS0ke3ZlcnNpb25PcHRpb24ubmFtZX1gKTtcbiAgICAgICAgICAgIGlmIChsb25nKSB7XG4gICAgICAgICAgICAgIGF3YWl0IHRoaXMuY2hlY2tWZXJzaW9uKCk7XG4gICAgICAgICAgICAgIHRoaXMuc2hvd0xvbmdWZXJzaW9uKCk7XG4gICAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgICB0aGlzLnNob3dWZXJzaW9uKCk7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgICB0aGlzLmV4aXQoKTtcbiAgICAgICAgICB9LFxuICAgICAgICAgIC4uLih0aGlzLl92ZXJzaW9uT3B0aW9uPy5vcHRzID8/IHt9KSxcbiAgICAgICAgfSxcbiAgICAgICk7XG4gICAgICBjb25zdCB2ZXJzaW9uT3B0aW9uID0gdGhpcy5vcHRpb25zWzBdO1xuICAgIH1cblxuICAgIGlmICh0aGlzLl9oZWxwT3B0aW9uICE9PSBmYWxzZSkge1xuICAgICAgdGhpcy5vcHRpb24oXG4gICAgICAgIHRoaXMuX2hlbHBPcHRpb24/LmZsYWdzIHx8IFwiLWgsIC0taGVscFwiLFxuICAgICAgICB0aGlzLl9oZWxwT3B0aW9uPy5kZXNjIHx8IFwiU2hvdyB0aGlzIGhlbHAuXCIsXG4gICAgICAgIHtcbiAgICAgICAgICBzdGFuZGFsb25lOiB0cnVlLFxuICAgICAgICAgIGdsb2JhbDogdHJ1ZSxcbiAgICAgICAgICBwcmVwZW5kOiB0cnVlLFxuICAgICAgICAgIGFjdGlvbjogYXN5bmMgZnVuY3Rpb24gKCkge1xuICAgICAgICAgICAgY29uc3QgbG9uZyA9IHRoaXMuZ2V0UmF3QXJncygpLmluY2x1ZGVzKGAtLSR7aGVscE9wdGlvbi5uYW1lfWApO1xuICAgICAgICAgICAgYXdhaXQgdGhpcy5jaGVja1ZlcnNpb24oKTtcbiAgICAgICAgICAgIHRoaXMuc2hvd0hlbHAoeyBsb25nIH0pO1xuICAgICAgICAgICAgdGhpcy5leGl0KCk7XG4gICAgICAgICAgfSxcbiAgICAgICAgICAuLi4odGhpcy5faGVscE9wdGlvbj8ub3B0cyA/PyB7fSksXG4gICAgICAgIH0sXG4gICAgICApO1xuICAgICAgY29uc3QgaGVscE9wdGlvbiA9IHRoaXMub3B0aW9uc1swXTtcbiAgICB9XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBFeGVjdXRlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBvcHRpb25zIEEgbWFwIG9mIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBhcmdzIENvbW1hbmQgYXJndW1lbnRzLlxuICAgKi9cbiAgcHJvdGVjdGVkIGFzeW5jIGV4ZWN1dGUoXG4gICAgb3B0aW9uczogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgLi4uYXJnczogQXJyYXk8dW5rbm93bj5cbiAgKTogUHJvbWlzZTxJUGFyc2VSZXN1bHQ+IHtcbiAgICBpZiAodGhpcy5mbikge1xuICAgICAgYXdhaXQgdGhpcy5mbihvcHRpb25zLCAuLi5hcmdzKTtcbiAgICB9IGVsc2UgaWYgKHRoaXMuZGVmYXVsdENvbW1hbmQpIHtcbiAgICAgIGNvbnN0IGNtZCA9IHRoaXMuZ2V0Q29tbWFuZCh0aGlzLmRlZmF1bHRDb21tYW5kLCB0cnVlKTtcblxuICAgICAgaWYgKCFjbWQpIHtcbiAgICAgICAgdGhyb3cgbmV3IERlZmF1bHRDb21tYW5kTm90Rm91bmQoXG4gICAgICAgICAgdGhpcy5kZWZhdWx0Q29tbWFuZCxcbiAgICAgICAgICB0aGlzLmdldENvbW1hbmRzKCksXG4gICAgICAgICk7XG4gICAgICB9XG5cbiAgICAgIGNtZC5fZ2xvYmFsUGFyZW50ID0gdGhpcztcbiAgICAgIGF3YWl0IGNtZC5leGVjdXRlKG9wdGlvbnMsIC4uLmFyZ3MpO1xuICAgIH1cblxuICAgIHJldHVybiB7XG4gICAgICBvcHRpb25zLFxuICAgICAgYXJncyxcbiAgICAgIGNtZDogdGhpcyxcbiAgICAgIGxpdGVyYWw6IHRoaXMubGl0ZXJhbEFyZ3MsXG4gICAgfTtcbiAgfVxuXG4gIC8qKlxuICAgKiBFeGVjdXRlIGV4dGVybmFsIHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gYXJncyBSYXcgY29tbWFuZCBsaW5lIGFyZ3VtZW50cy5cbiAgICovXG4gIHByb3RlY3RlZCBhc3luYyBleGVjdXRlRXhlY3V0YWJsZShhcmdzOiBzdHJpbmdbXSkge1xuICAgIGNvbnN0IGNvbW1hbmQgPSB0aGlzLmdldFBhdGgoKS5yZXBsYWNlKC9cXHMrL2csIFwiLVwiKTtcblxuICAgIGF3YWl0IERlbm8ucGVybWlzc2lvbnMucmVxdWVzdCh7IG5hbWU6IFwicnVuXCIsIGNvbW1hbmQgfSk7XG5cbiAgICB0cnkge1xuICAgICAgY29uc3QgcHJvY2VzczogRGVuby5Qcm9jZXNzID0gRGVuby5ydW4oe1xuICAgICAgICBjbWQ6IFtjb21tYW5kLCAuLi5hcmdzXSxcbiAgICAgIH0pO1xuICAgICAgY29uc3Qgc3RhdHVzOiBEZW5vLlByb2Nlc3NTdGF0dXMgPSBhd2FpdCBwcm9jZXNzLnN0YXR1cygpO1xuICAgICAgaWYgKCFzdGF0dXMuc3VjY2Vzcykge1xuICAgICAgICBEZW5vLmV4aXQoc3RhdHVzLmNvZGUpO1xuICAgICAgfVxuICAgIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBEZW5vLmVycm9ycy5Ob3RGb3VuZCkge1xuICAgICAgICB0aHJvdyBuZXcgQ29tbWFuZEV4ZWN1dGFibGVOb3RGb3VuZChjb21tYW5kKTtcbiAgICAgIH1cbiAgICAgIHRocm93IGVycm9yO1xuICAgIH1cbiAgfVxuXG4gIC8qKlxuICAgKiBQYXJzZSByYXcgY29tbWFuZCBsaW5lIGFyZ3VtZW50cy5cbiAgICogQHBhcmFtIGFyZ3MgUmF3IGNvbW1hbmQgbGluZSBhcmd1bWVudHMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcGFyc2VGbGFncyhcbiAgICBhcmdzOiBzdHJpbmdbXSxcbiAgICBlbnY6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICApOiBJRmxhZ3NSZXN1bHQgJiB7IGFjdGlvbk9wdGlvbj86IElPcHRpb24gJiB7IGFjdGlvbjogSUFjdGlvbiB9IH0ge1xuICAgIHRyeSB7XG4gICAgICBsZXQgYWN0aW9uT3B0aW9uOiBJT3B0aW9uICYgeyBhY3Rpb246IElBY3Rpb24gfSB8IHVuZGVmaW5lZDtcbiAgICAgIGNvbnN0IHJlc3VsdCA9IHBhcnNlRmxhZ3MoYXJncywge1xuICAgICAgICBzdG9wRWFybHk6IHRoaXMuX3N0b3BFYXJseSxcbiAgICAgICAgYWxsb3dFbXB0eTogdGhpcy5fYWxsb3dFbXB0eSxcbiAgICAgICAgZmxhZ3M6IHRoaXMuZ2V0T3B0aW9ucyh0cnVlKSxcbiAgICAgICAgaWdub3JlRGVmYXVsdHM6IGVudixcbiAgICAgICAgcGFyc2U6ICh0eXBlOiBJVHlwZUluZm8pID0+IHRoaXMucGFyc2VUeXBlKHR5cGUpLFxuICAgICAgICBvcHRpb246IChvcHRpb246IElPcHRpb24pID0+IHtcbiAgICAgICAgICBpZiAoIWFjdGlvbk9wdGlvbiAmJiBvcHRpb24uYWN0aW9uKSB7XG4gICAgICAgICAgICBhY3Rpb25PcHRpb24gPSBvcHRpb24gYXMgSU9wdGlvbiAmIHsgYWN0aW9uOiBJQWN0aW9uIH07XG4gICAgICAgICAgfVxuICAgICAgICB9LFxuICAgICAgfSk7XG4gICAgICByZXR1cm4geyAuLi5yZXN1bHQsIGFjdGlvbk9wdGlvbiB9O1xuICAgIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBGbGFnc1ZhbGlkYXRpb25FcnJvcikge1xuICAgICAgICB0aHJvdyBuZXcgVmFsaWRhdGlvbkVycm9yKGVycm9yLm1lc3NhZ2UpO1xuICAgICAgfVxuICAgICAgdGhyb3cgZXJyb3I7XG4gICAgfVxuICB9XG5cbiAgLyoqIFBhcnNlIGFyZ3VtZW50IHR5cGUuICovXG4gIHByb3RlY3RlZCBwYXJzZVR5cGUodHlwZTogSVR5cGVJbmZvKTogdW5rbm93biB7XG4gICAgY29uc3QgdHlwZVNldHRpbmdzOiBJVHlwZSB8IHVuZGVmaW5lZCA9IHRoaXMuZ2V0VHlwZSh0eXBlLnR5cGUpO1xuXG4gICAgaWYgKCF0eXBlU2V0dGluZ3MpIHtcbiAgICAgIHRocm93IG5ldyBVbmtub3duVHlwZShcbiAgICAgICAgdHlwZS50eXBlLFxuICAgICAgICB0aGlzLmdldFR5cGVzKCkubWFwKCh0eXBlKSA9PiB0eXBlLm5hbWUpLFxuICAgICAgKTtcbiAgICB9XG5cbiAgICByZXR1cm4gdHlwZVNldHRpbmdzLmhhbmRsZXIgaW5zdGFuY2VvZiBUeXBlXG4gICAgICA/IHR5cGVTZXR0aW5ncy5oYW5kbGVyLnBhcnNlKHR5cGUpXG4gICAgICA6IHR5cGVTZXR0aW5ncy5oYW5kbGVyKHR5cGUpO1xuICB9XG5cbiAgLyoqIFZhbGlkYXRlIGVudmlyb25tZW50IHZhcmlhYmxlcy4gKi9cbiAgcHJvdGVjdGVkIGFzeW5jIHBhcnNlRW52VmFycygpOiBQcm9taXNlPFJlY29yZDxzdHJpbmcsIHVua25vd24+PiB7XG4gICAgY29uc3QgZW52VmFycyA9IHRoaXMuZ2V0RW52VmFycyh0cnVlKTtcbiAgICBjb25zdCByZXN1bHQ6IFJlY29yZDxzdHJpbmcsIHVua25vd24+ID0ge307XG5cbiAgICBpZiAoIWVudlZhcnMubGVuZ3RoKSB7XG4gICAgICByZXR1cm4gcmVzdWx0O1xuICAgIH1cblxuICAgIGNvbnN0IGhhc0VudlBlcm1pc3Npb25zID0gKGF3YWl0IERlbm8ucGVybWlzc2lvbnMucXVlcnkoe1xuICAgICAgbmFtZTogXCJlbnZcIixcbiAgICB9KSkuc3RhdGUgPT09IFwiZ3JhbnRlZFwiO1xuXG4gICAgZm9yIChjb25zdCBlbnYgb2YgZW52VmFycykge1xuICAgICAgY29uc3QgbmFtZSA9IGhhc0VudlBlcm1pc3Npb25zICYmIGVudi5uYW1lcy5maW5kKFxuICAgICAgICAobmFtZTogc3RyaW5nKSA9PiAhIURlbm8uZW52LmdldChuYW1lKSxcbiAgICAgICk7XG5cbiAgICAgIGlmIChuYW1lKSB7XG4gICAgICAgIGNvbnN0IHByb3BlcnR5TmFtZSA9IHVuZGVyc2NvcmVUb0NhbWVsQ2FzZShcbiAgICAgICAgICBlbnYucHJlZml4XG4gICAgICAgICAgICA/IGVudi5uYW1lc1swXS5yZXBsYWNlKG5ldyBSZWdFeHAoYF4ke2Vudi5wcmVmaXh9YCksIFwiXCIpXG4gICAgICAgICAgICA6IGVudi5uYW1lc1swXSxcbiAgICAgICAgKTtcblxuICAgICAgICBpZiAoZW52LmRldGFpbHMubGlzdCkge1xuICAgICAgICAgIGNvbnN0IHZhbHVlcyA9IERlbm8uZW52LmdldChuYW1lKVxuICAgICAgICAgICAgPy5zcGxpdChlbnYuZGV0YWlscy5zZXBhcmF0b3IgPz8gXCIsXCIpID8/IFtcIlwiXTtcblxuICAgICAgICAgIHJlc3VsdFtwcm9wZXJ0eU5hbWVdID0gdmFsdWVzLm1hcCgodmFsdWUpID0+XG4gICAgICAgICAgICB0aGlzLnBhcnNlVHlwZSh7XG4gICAgICAgICAgICAgIGxhYmVsOiBcIkVudmlyb25tZW50IHZhcmlhYmxlXCIsXG4gICAgICAgICAgICAgIHR5cGU6IGVudi50eXBlLFxuICAgICAgICAgICAgICBuYW1lLFxuICAgICAgICAgICAgICB2YWx1ZSxcbiAgICAgICAgICAgIH0pXG4gICAgICAgICAgKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHRbcHJvcGVydHlOYW1lXSA9IHRoaXMucGFyc2VUeXBlKHtcbiAgICAgICAgICAgIGxhYmVsOiBcIkVudmlyb25tZW50IHZhcmlhYmxlXCIsXG4gICAgICAgICAgICB0eXBlOiBlbnYudHlwZSxcbiAgICAgICAgICAgIG5hbWUsXG4gICAgICAgICAgICB2YWx1ZTogRGVuby5lbnYuZ2V0KG5hbWUpID8/IFwiXCIsXG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoZW52LnZhbHVlICYmIHR5cGVvZiByZXN1bHRbcHJvcGVydHlOYW1lXSAhPT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICAgIHJlc3VsdFtwcm9wZXJ0eU5hbWVdID0gZW52LnZhbHVlKHJlc3VsdFtwcm9wZXJ0eU5hbWVdKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChlbnYucmVxdWlyZWQpIHtcbiAgICAgICAgdGhyb3cgbmV3IE1pc3NpbmdSZXF1aXJlZEVudlZhcihlbnYpO1xuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cblxuICAvKipcbiAgICogUGFyc2UgY29tbWFuZC1saW5lIGFyZ3VtZW50cy5cbiAgICogQHBhcmFtIGFyZ3MgIFJhdyBjb21tYW5kIGxpbmUgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gZmxhZ3MgUGFyc2VkIGNvbW1hbmQgbGluZSBvcHRpb25zLlxuICAgKi9cbiAgcHJvdGVjdGVkIHBhcnNlQXJndW1lbnRzKGFyZ3M6IHN0cmluZ1tdLCBmbGFnczogUmVjb3JkPHN0cmluZywgdW5rbm93bj4pOiBDQSB7XG4gICAgY29uc3QgcGFyYW1zOiBBcnJheTx1bmtub3duPiA9IFtdO1xuXG4gICAgLy8gcmVtb3ZlIGFycmF5IHJlZmVyZW5jZVxuICAgIGFyZ3MgPSBhcmdzLnNsaWNlKDApO1xuXG4gICAgaWYgKCF0aGlzLmhhc0FyZ3VtZW50cygpKSB7XG4gICAgICBpZiAoYXJncy5sZW5ndGgpIHtcbiAgICAgICAgaWYgKHRoaXMuaGFzQ29tbWFuZHModHJ1ZSkpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgVW5rbm93bkNvbW1hbmQoYXJnc1swXSwgdGhpcy5nZXRDb21tYW5kcygpKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICB0aHJvdyBuZXcgTm9Bcmd1bWVudHNBbGxvd2VkKHRoaXMuZ2V0UGF0aCgpKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH0gZWxzZSB7XG4gICAgICBpZiAoIWFyZ3MubGVuZ3RoKSB7XG4gICAgICAgIGNvbnN0IHJlcXVpcmVkID0gdGhpcy5nZXRBcmd1bWVudHMoKVxuICAgICAgICAgIC5maWx0ZXIoKGV4cGVjdGVkQXJnKSA9PiAhZXhwZWN0ZWRBcmcub3B0aW9uYWxWYWx1ZSlcbiAgICAgICAgICAubWFwKChleHBlY3RlZEFyZykgPT4gZXhwZWN0ZWRBcmcubmFtZSk7XG5cbiAgICAgICAgaWYgKHJlcXVpcmVkLmxlbmd0aCkge1xuICAgICAgICAgIGNvbnN0IGZsYWdOYW1lczogc3RyaW5nW10gPSBPYmplY3Qua2V5cyhmbGFncyk7XG4gICAgICAgICAgY29uc3QgaGFzU3RhbmRhbG9uZU9wdGlvbiA9ICEhZmxhZ05hbWVzLmZpbmQoKG5hbWUpID0+XG4gICAgICAgICAgICB0aGlzLmdldE9wdGlvbihuYW1lLCB0cnVlKT8uc3RhbmRhbG9uZVxuICAgICAgICAgICk7XG5cbiAgICAgICAgICBpZiAoIWhhc1N0YW5kYWxvbmVPcHRpb24pIHtcbiAgICAgICAgICAgIHRocm93IG5ldyBNaXNzaW5nQXJndW1lbnRzKHJlcXVpcmVkKTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIGZvciAoY29uc3QgZXhwZWN0ZWRBcmcgb2YgdGhpcy5nZXRBcmd1bWVudHMoKSkge1xuICAgICAgICAgIGlmICghYXJncy5sZW5ndGgpIHtcbiAgICAgICAgICAgIGlmIChleHBlY3RlZEFyZy5vcHRpb25hbFZhbHVlKSB7XG4gICAgICAgICAgICAgIGJyZWFrO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgdGhyb3cgbmV3IE1pc3NpbmdBcmd1bWVudChgTWlzc2luZyBhcmd1bWVudDogJHtleHBlY3RlZEFyZy5uYW1lfWApO1xuICAgICAgICAgIH1cblxuICAgICAgICAgIGxldCBhcmc6IHVua25vd247XG5cbiAgICAgICAgICBpZiAoZXhwZWN0ZWRBcmcudmFyaWFkaWMpIHtcbiAgICAgICAgICAgIGFyZyA9IGFyZ3Muc3BsaWNlKDAsIGFyZ3MubGVuZ3RoKVxuICAgICAgICAgICAgICAubWFwKCh2YWx1ZSkgPT5cbiAgICAgICAgICAgICAgICB0aGlzLnBhcnNlVHlwZSh7XG4gICAgICAgICAgICAgICAgICBsYWJlbDogXCJBcmd1bWVudFwiLFxuICAgICAgICAgICAgICAgICAgdHlwZTogZXhwZWN0ZWRBcmcudHlwZSxcbiAgICAgICAgICAgICAgICAgIG5hbWU6IGV4cGVjdGVkQXJnLm5hbWUsXG4gICAgICAgICAgICAgICAgICB2YWx1ZSxcbiAgICAgICAgICAgICAgICB9KVxuICAgICAgICAgICAgICApO1xuICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICBhcmcgPSB0aGlzLnBhcnNlVHlwZSh7XG4gICAgICAgICAgICAgIGxhYmVsOiBcIkFyZ3VtZW50XCIsXG4gICAgICAgICAgICAgIHR5cGU6IGV4cGVjdGVkQXJnLnR5cGUsXG4gICAgICAgICAgICAgIG5hbWU6IGV4cGVjdGVkQXJnLm5hbWUsXG4gICAgICAgICAgICAgIHZhbHVlOiBhcmdzLnNoaWZ0KCkgYXMgc3RyaW5nLFxuICAgICAgICAgICAgfSk7XG4gICAgICAgICAgfVxuXG4gICAgICAgICAgaWYgKHR5cGVvZiBhcmcgIT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICAgIHBhcmFtcy5wdXNoKGFyZyk7XG4gICAgICAgICAgfVxuICAgICAgICB9XG5cbiAgICAgICAgaWYgKGFyZ3MubGVuZ3RoKSB7XG4gICAgICAgICAgdGhyb3cgbmV3IFRvb01hbnlBcmd1bWVudHMoYXJncyk7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcGFyYW1zIGFzIENBO1xuICB9XG5cbiAgLyoqXG4gICAqIEhhbmRsZSBlcnJvci4gSWYgYHRocm93RXJyb3JzYCBpcyBlbmFibGVkIHRoZSBlcnJvciB3aWxsIGJlIHJldHVybmVkLFxuICAgKiBvdGhlcndpc2UgYSBmb3JtYXR0ZWQgZXJyb3IgbWVzc2FnZSB3aWxsIGJlIHByaW50ZWQgYW5kIGBEZW5vLmV4aXQoMSlgXG4gICAqIHdpbGwgYmUgY2FsbGVkLlxuICAgKiBAcGFyYW0gZXJyb3IgRXJyb3IgdG8gaGFuZGxlLlxuICAgKi9cbiAgcHJvdGVjdGVkIGVycm9yKGVycm9yOiBFcnJvcik6IEVycm9yIHtcbiAgICBpZiAodGhpcy5zaG91bGRUaHJvd0Vycm9ycygpIHx8ICEoZXJyb3IgaW5zdGFuY2VvZiBWYWxpZGF0aW9uRXJyb3IpKSB7XG4gICAgICByZXR1cm4gZXJyb3I7XG4gICAgfVxuICAgIHRoaXMuc2hvd0hlbHAoKTtcbiAgICBjb25zb2xlLmVycm9yKHJlZChgICAke2JvbGQoXCJlcnJvclwiKX06ICR7ZXJyb3IubWVzc2FnZX1cXG5gKSk7XG4gICAgRGVuby5leGl0KGVycm9yIGluc3RhbmNlb2YgVmFsaWRhdGlvbkVycm9yID8gZXJyb3IuZXhpdENvZGUgOiAxKTtcbiAgfVxuXG4gIC8qKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKiBHRVRURVIgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqL1xuXG4gIC8qKiBHZXQgY29tbWFuZCBuYW1lLiAqL1xuICBwdWJsaWMgZ2V0TmFtZSgpOiBzdHJpbmcge1xuICAgIHJldHVybiB0aGlzLl9uYW1lO1xuICB9XG5cbiAgLyoqIEdldCBwYXJlbnQgY29tbWFuZC4gKi9cbiAgcHVibGljIGdldFBhcmVudCgpOiBDUCB7XG4gICAgcmV0dXJuIHRoaXMuX3BhcmVudCBhcyBDUDtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgcGFyZW50IGNvbW1hbmQgZnJvbSBnbG9iYWwgZXhlY3V0ZWQgY29tbWFuZC5cbiAgICogQmUgc3VyZSwgdG8gY2FsbCB0aGlzIG1ldGhvZCBvbmx5IGluc2lkZSBhbiBhY3Rpb24gaGFuZGxlci4gVW5sZXNzIHRoaXMgb3IgYW55IGNoaWxkIGNvbW1hbmQgd2FzIGV4ZWN1dGVkLFxuICAgKiB0aGlzIG1ldGhvZCByZXR1cm5zIGFsd2F5cyB1bmRlZmluZWQuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsUGFyZW50KCk6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuX2dsb2JhbFBhcmVudDtcbiAgfVxuXG4gIC8qKiBHZXQgbWFpbiBjb21tYW5kLiAqL1xuICBwdWJsaWMgZ2V0TWFpbkNvbW1hbmQoKTogQ29tbWFuZDxhbnk+IHtcbiAgICByZXR1cm4gdGhpcy5fcGFyZW50Py5nZXRNYWluQ29tbWFuZCgpID8/IHRoaXM7XG4gIH1cblxuICAvKiogR2V0IGNvbW1hbmQgbmFtZSBhbGlhc2VzLiAqL1xuICBwdWJsaWMgZ2V0QWxpYXNlcygpOiBzdHJpbmdbXSB7XG4gICAgcmV0dXJuIHRoaXMuYWxpYXNlcztcbiAgfVxuXG4gIC8qKiBHZXQgZnVsbCBjb21tYW5kIHBhdGguICovXG4gIHB1YmxpYyBnZXRQYXRoKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIHRoaXMuX3BhcmVudFxuICAgICAgPyB0aGlzLl9wYXJlbnQuZ2V0UGF0aCgpICsgXCIgXCIgKyB0aGlzLl9uYW1lXG4gICAgICA6IHRoaXMuX25hbWU7XG4gIH1cblxuICAvKiogR2V0IGFyZ3VtZW50cyBkZWZpbml0aW9uLiBFLmc6IDxpbnB1dC1maWxlOnN0cmluZz4gPG91dHB1dC1maWxlOnN0cmluZz4gKi9cbiAgcHVibGljIGdldEFyZ3NEZWZpbml0aW9uKCk6IHN0cmluZyB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuYXJnc0RlZmluaXRpb247XG4gIH1cblxuICAvKipcbiAgICogR2V0IGFyZ3VtZW50IGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGFyZ3VtZW50LlxuICAgKi9cbiAgcHVibGljIGdldEFyZ3VtZW50KG5hbWU6IHN0cmluZyk6IElBcmd1bWVudCB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QXJndW1lbnRzKCkuZmluZCgoYXJnKSA9PiBhcmcubmFtZSA9PT0gbmFtZSk7XG4gIH1cblxuICAvKiogR2V0IGFyZ3VtZW50cy4gKi9cbiAgcHVibGljIGdldEFyZ3VtZW50cygpOiBJQXJndW1lbnRbXSB7XG4gICAgaWYgKCF0aGlzLmFyZ3MubGVuZ3RoICYmIHRoaXMuYXJnc0RlZmluaXRpb24pIHtcbiAgICAgIHRoaXMuYXJncyA9IHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbih0aGlzLmFyZ3NEZWZpbml0aW9uKTtcbiAgICB9XG5cbiAgICByZXR1cm4gdGhpcy5hcmdzO1xuICB9XG5cbiAgLyoqIENoZWNrIGlmIGNvbW1hbmQgaGFzIGFyZ3VtZW50cy4gKi9cbiAgcHVibGljIGhhc0FyZ3VtZW50cygpIHtcbiAgICByZXR1cm4gISF0aGlzLmFyZ3NEZWZpbml0aW9uO1xuICB9XG5cbiAgLyoqIEdldCBjb21tYW5kIHZlcnNpb24uICovXG4gIHB1YmxpYyBnZXRWZXJzaW9uKCk6IHN0cmluZyB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0VmVyc2lvbkhhbmRsZXIoKT8uY2FsbCh0aGlzLCB0aGlzKTtcbiAgfVxuXG4gIC8qKiBHZXQgaGVscCBoYW5kbGVyIG1ldGhvZC4gKi9cbiAgcHJpdmF0ZSBnZXRWZXJzaW9uSGFuZGxlcigpOiBJVmVyc2lvbkhhbmRsZXIgfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLnZlciA/PyB0aGlzLl9wYXJlbnQ/LmdldFZlcnNpb25IYW5kbGVyKCk7XG4gIH1cblxuICAvKiogR2V0IGNvbW1hbmQgZGVzY3JpcHRpb24uICovXG4gIHB1YmxpYyBnZXREZXNjcmlwdGlvbigpOiBzdHJpbmcge1xuICAgIC8vIGNhbGwgZGVzY3JpcHRpb24gbWV0aG9kIG9ubHkgb25jZVxuICAgIHJldHVybiB0eXBlb2YgdGhpcy5kZXNjID09PSBcImZ1bmN0aW9uXCJcbiAgICAgID8gdGhpcy5kZXNjID0gdGhpcy5kZXNjKClcbiAgICAgIDogdGhpcy5kZXNjO1xuICB9XG5cbiAgcHVibGljIGdldFVzYWdlKCkge1xuICAgIHJldHVybiB0aGlzLl91c2FnZSA/PyB0aGlzLmdldEFyZ3NEZWZpbml0aW9uKCk7XG4gIH1cblxuICAvKiogR2V0IHNob3J0IGNvbW1hbmQgZGVzY3JpcHRpb24uIFRoaXMgaXMgdGhlIGZpcnN0IGxpbmUgb2YgdGhlIGRlc2NyaXB0aW9uLiAqL1xuICBwdWJsaWMgZ2V0U2hvcnREZXNjcmlwdGlvbigpOiBzdHJpbmcge1xuICAgIHJldHVybiBnZXREZXNjcmlwdGlvbih0aGlzLmdldERlc2NyaXB0aW9uKCksIHRydWUpO1xuICB9XG5cbiAgLyoqIEdldCBvcmlnaW5hbCBjb21tYW5kLWxpbmUgYXJndW1lbnRzLiAqL1xuICBwdWJsaWMgZ2V0UmF3QXJncygpOiBzdHJpbmdbXSB7XG4gICAgcmV0dXJuIHRoaXMucmF3QXJncztcbiAgfVxuXG4gIC8qKiBHZXQgYWxsIGFyZ3VtZW50cyBkZWZpbmVkIGFmdGVyIHRoZSBkb3VibGUgZGFzaC4gKi9cbiAgcHVibGljIGdldExpdGVyYWxBcmdzKCk6IHN0cmluZ1tdIHtcbiAgICByZXR1cm4gdGhpcy5saXRlcmFsQXJncztcbiAgfVxuXG4gIC8qKiBPdXRwdXQgZ2VuZXJhdGVkIGhlbHAgd2l0aG91dCBleGl0aW5nLiAqL1xuICBwdWJsaWMgc2hvd1ZlcnNpb24oKTogdm9pZCB7XG4gICAgY29uc29sZS5sb2codGhpcy5nZXRWZXJzaW9uKCkpO1xuICB9XG5cbiAgLyoqIFJldHVybnMgY29tbWFuZCBuYW1lLCB2ZXJzaW9uIGFuZCBtZXRhIGRhdGEuICovXG4gIHB1YmxpYyBnZXRMb25nVmVyc2lvbigpOiBzdHJpbmcge1xuICAgIHJldHVybiBgJHtib2xkKHRoaXMuZ2V0TWFpbkNvbW1hbmQoKS5nZXROYW1lKCkpfSAke1xuICAgICAgYmx1ZSh0aGlzLmdldFZlcnNpb24oKSA/PyBcIlwiKVxuICAgIH1gICtcbiAgICAgIE9iamVjdC5lbnRyaWVzKHRoaXMuZ2V0TWV0YSgpKS5tYXAoXG4gICAgICAgIChbaywgdl0pID0+IGBcXG4ke2JvbGQoayl9ICR7Ymx1ZSh2KX1gLFxuICAgICAgKS5qb2luKFwiXCIpO1xuICB9XG5cbiAgLyoqIE91dHB1dHMgY29tbWFuZCBuYW1lLCB2ZXJzaW9uIGFuZCBtZXRhIGRhdGEuICovXG4gIHB1YmxpYyBzaG93TG9uZ1ZlcnNpb24oKTogdm9pZCB7XG4gICAgY29uc29sZS5sb2codGhpcy5nZXRMb25nVmVyc2lvbigpKTtcbiAgfVxuXG4gIC8qKiBPdXRwdXQgZ2VuZXJhdGVkIGhlbHAgd2l0aG91dCBleGl0aW5nLiAqL1xuICBwdWJsaWMgc2hvd0hlbHAob3B0aW9ucz86IEhlbHBPcHRpb25zKTogdm9pZCB7XG4gICAgY29uc29sZS5sb2codGhpcy5nZXRIZWxwKG9wdGlvbnMpKTtcbiAgfVxuXG4gIC8qKiBHZXQgZ2VuZXJhdGVkIGhlbHAuICovXG4gIHB1YmxpYyBnZXRIZWxwKG9wdGlvbnM/OiBIZWxwT3B0aW9ucyk6IHN0cmluZyB7XG4gICAgdGhpcy5yZWdpc3RlckRlZmF1bHRzKCk7XG4gICAgcmV0dXJuIHRoaXMuZ2V0SGVscEhhbmRsZXIoKS5jYWxsKHRoaXMsIHRoaXMsIG9wdGlvbnMgPz8ge30pO1xuICB9XG5cbiAgLyoqIEdldCBoZWxwIGhhbmRsZXIgbWV0aG9kLiAqL1xuICBwcml2YXRlIGdldEhlbHBIYW5kbGVyKCk6IElIZWxwSGFuZGxlciB7XG4gICAgcmV0dXJuIHRoaXMuX2hlbHAgPz8gdGhpcy5fcGFyZW50Py5nZXRIZWxwSGFuZGxlcigpIGFzIElIZWxwSGFuZGxlcjtcbiAgfVxuXG4gIHByaXZhdGUgZXhpdChjb2RlID0gMCkge1xuICAgIGlmICh0aGlzLnNob3VsZEV4aXQoKSkge1xuICAgICAgRGVuby5leGl0KGNvZGUpO1xuICAgIH1cbiAgfVxuXG4gIC8qKiBDaGVjayBpZiBuZXcgdmVyc2lvbiBpcyBhdmFpbGFibGUgYW5kIGFkZCBoaW50IHRvIHZlcnNpb24uICovXG4gIHB1YmxpYyBhc3luYyBjaGVja1ZlcnNpb24oKTogUHJvbWlzZTx2b2lkPiB7XG4gICAgY29uc3QgbWFpbkNvbW1hbmQgPSB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgY29uc3QgdXBncmFkZUNvbW1hbmQgPSBtYWluQ29tbWFuZC5nZXRDb21tYW5kKFwidXBncmFkZVwiKTtcbiAgICBpZiAoaXNVcGdyYWRlQ29tbWFuZCh1cGdyYWRlQ29tbWFuZCkpIHtcbiAgICAgIGNvbnN0IGxhdGVzdFZlcnNpb24gPSBhd2FpdCB1cGdyYWRlQ29tbWFuZC5nZXRMYXRlc3RWZXJzaW9uKCk7XG4gICAgICBjb25zdCBjdXJyZW50VmVyc2lvbiA9IG1haW5Db21tYW5kLmdldFZlcnNpb24oKTtcbiAgICAgIGlmIChjdXJyZW50VmVyc2lvbiAhPT0gbGF0ZXN0VmVyc2lvbikge1xuICAgICAgICBtYWluQ29tbWFuZC52ZXJzaW9uKFxuICAgICAgICAgIGAke2N1cnJlbnRWZXJzaW9ufSAgJHtcbiAgICAgICAgICAgIGJvbGQoXG4gICAgICAgICAgICAgIHllbGxvdyhcbiAgICAgICAgICAgICAgICBgKE5ldyB2ZXJzaW9uIGF2YWlsYWJsZTogJHtsYXRlc3RWZXJzaW9ufS4gUnVuICcke21haW5Db21tYW5kLmdldE5hbWUoKX0gdXBncmFkZScgdG8gdXBncmFkZSB0byB0aGUgbGF0ZXN0IHZlcnNpb24hKWAsXG4gICAgICAgICAgICAgICksXG4gICAgICAgICAgICApXG4gICAgICAgICAgfWAsXG4gICAgICAgICk7XG4gICAgICB9XG4gICAgfVxuICB9XG5cbiAgLyoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqIE9wdGlvbnMgR0VUVEVSICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKiovXG5cbiAgLyoqXG4gICAqIENoZWNrcyB3aGV0aGVyIHRoZSBjb21tYW5kIGhhcyBvcHRpb25zIG9yIG5vdC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhhc09wdGlvbnMoaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmdldE9wdGlvbnMoaGlkZGVuKS5sZW5ndGggPiAwO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBvcHRpb25zLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0T3B0aW9ucyhoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbltdIHtcbiAgICByZXR1cm4gdGhpcy5nZXRHbG9iYWxPcHRpb25zKGhpZGRlbikuY29uY2F0KHRoaXMuZ2V0QmFzZU9wdGlvbnMoaGlkZGVuKSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2Ugb3B0aW9ucy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VPcHRpb25zKGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uW10ge1xuICAgIGlmICghdGhpcy5vcHRpb25zLmxlbmd0aCkge1xuICAgICAgcmV0dXJuIFtdO1xuICAgIH1cblxuICAgIHJldHVybiBoaWRkZW5cbiAgICAgID8gdGhpcy5vcHRpb25zLnNsaWNlKDApXG4gICAgICA6IHRoaXMub3B0aW9ucy5maWx0ZXIoKG9wdCkgPT4gIW9wdC5oaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgb3B0aW9ucy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbE9wdGlvbnMoaGlkZGVuPzogYm9vbGVhbik6IElPcHRpb25bXSB7XG4gICAgY29uc3QgZ2V0T3B0aW9ucyA9IChcbiAgICAgIGNtZDogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkLFxuICAgICAgb3B0aW9uczogSU9wdGlvbltdID0gW10sXG4gICAgICBuYW1lczogc3RyaW5nW10gPSBbXSxcbiAgICApOiBJT3B0aW9uW10gPT4ge1xuICAgICAgaWYgKGNtZCkge1xuICAgICAgICBpZiAoY21kLm9wdGlvbnMubGVuZ3RoKSB7XG4gICAgICAgICAgY21kLm9wdGlvbnMuZm9yRWFjaCgob3B0aW9uOiBJT3B0aW9uKSA9PiB7XG4gICAgICAgICAgICBpZiAoXG4gICAgICAgICAgICAgIG9wdGlvbi5nbG9iYWwgJiZcbiAgICAgICAgICAgICAgIXRoaXMub3B0aW9ucy5maW5kKChvcHQpID0+IG9wdC5uYW1lID09PSBvcHRpb24ubmFtZSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZihvcHRpb24ubmFtZSkgPT09IC0xICYmXG4gICAgICAgICAgICAgIChoaWRkZW4gfHwgIW9wdGlvbi5oaWRkZW4pXG4gICAgICAgICAgICApIHtcbiAgICAgICAgICAgICAgbmFtZXMucHVzaChvcHRpb24ubmFtZSk7XG4gICAgICAgICAgICAgIG9wdGlvbnMucHVzaChvcHRpb24pO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIGdldE9wdGlvbnMoY21kLl9wYXJlbnQsIG9wdGlvbnMsIG5hbWVzKTtcbiAgICAgIH1cblxuICAgICAgcmV0dXJuIG9wdGlvbnM7XG4gICAgfTtcblxuICAgIHJldHVybiBnZXRPcHRpb25zKHRoaXMuX3BhcmVudCk7XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGFuIG9wdGlvbiB3aXRoIGdpdmVuIG5hbWUgb3Igbm90LlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhhc09wdGlvbihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gISF0aGlzLmdldE9wdGlvbihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBvcHRpb24gYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgb3B0aW9uLiBNdXN0IGJlIGluIHBhcmFtLWNhc2UuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBnZXRPcHRpb24obmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZU9wdGlvbihuYW1lLCBoaWRkZW4pID8/XG4gICAgICB0aGlzLmdldEdsb2JhbE9wdGlvbihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIG9wdGlvbiBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VPcHRpb24obmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgY29uc3Qgb3B0aW9uID0gdGhpcy5vcHRpb25zLmZpbmQoKG9wdGlvbikgPT4gb3B0aW9uLm5hbWUgPT09IG5hbWUpO1xuXG4gICAgcmV0dXJuIG9wdGlvbiAmJiAoaGlkZGVuIHx8ICFvcHRpb24uaGlkZGVuKSA/IG9wdGlvbiA6IHVuZGVmaW5lZDtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZ2xvYmFsIG9wdGlvbiBmcm9tIHBhcmVudCBjb21tYW5kcyBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbE9wdGlvbihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICBpZiAoIXRoaXMuX3BhcmVudCkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIGNvbnN0IG9wdGlvbjogSU9wdGlvbiB8IHVuZGVmaW5lZCA9IHRoaXMuX3BhcmVudC5nZXRCYXNlT3B0aW9uKFxuICAgICAgbmFtZSxcbiAgICAgIGhpZGRlbixcbiAgICApO1xuXG4gICAgaWYgKCFvcHRpb24gfHwgIW9wdGlvbi5nbG9iYWwpIHtcbiAgICAgIHJldHVybiB0aGlzLl9wYXJlbnQuZ2V0R2xvYmFsT3B0aW9uKG5hbWUsIGhpZGRlbik7XG4gICAgfVxuXG4gICAgcmV0dXJuIG9wdGlvbjtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW1vdmUgb3B0aW9uIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKi9cbiAgcHVibGljIHJlbW92ZU9wdGlvbihuYW1lOiBzdHJpbmcpOiBJT3B0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBpbmRleCA9IHRoaXMub3B0aW9ucy5maW5kSW5kZXgoKG9wdGlvbikgPT4gb3B0aW9uLm5hbWUgPT09IG5hbWUpO1xuXG4gICAgaWYgKGluZGV4ID09PSAtMSkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzLm9wdGlvbnMuc3BsaWNlKGluZGV4LCAxKVswXTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgc3ViLWNvbW1hbmRzIG9yIG5vdC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBoYXNDb21tYW5kcyhoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0Q29tbWFuZHMoaGlkZGVuKS5sZW5ndGggPiAwO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBjb21tYW5kcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRDb21tYW5kcyhoaWRkZW4/OiBib29sZWFuKTogQXJyYXk8Q29tbWFuZDxhbnk+PiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsQ29tbWFuZHMoaGlkZGVuKS5jb25jYXQodGhpcy5nZXRCYXNlQ29tbWFuZHMoaGlkZGVuKSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgY29tbWFuZHMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbW1hbmRzKGhpZGRlbj86IGJvb2xlYW4pOiBBcnJheTxDb21tYW5kPGFueT4+IHtcbiAgICBjb25zdCBjb21tYW5kcyA9IEFycmF5LmZyb20odGhpcy5jb21tYW5kcy52YWx1ZXMoKSk7XG4gICAgcmV0dXJuIGhpZGRlbiA/IGNvbW1hbmRzIDogY29tbWFuZHMuZmlsdGVyKChjbWQpID0+ICFjbWQuaXNIaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgY29tbWFuZHMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tbWFuZHMoaGlkZGVuPzogYm9vbGVhbik6IEFycmF5PENvbW1hbmQ8YW55Pj4ge1xuICAgIGNvbnN0IGdldENvbW1hbmRzID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQsXG4gICAgICBjb21tYW5kczogQXJyYXk8Q29tbWFuZDxhbnk+PiA9IFtdLFxuICAgICAgbmFtZXM6IHN0cmluZ1tdID0gW10sXG4gICAgKTogQXJyYXk8Q29tbWFuZDxhbnk+PiA9PiB7XG4gICAgICBpZiAoY21kKSB7XG4gICAgICAgIGlmIChjbWQuY29tbWFuZHMuc2l6ZSkge1xuICAgICAgICAgIGNtZC5jb21tYW5kcy5mb3JFYWNoKChjbWQ6IENvbW1hbmQ8YW55PikgPT4ge1xuICAgICAgICAgICAgaWYgKFxuICAgICAgICAgICAgICBjbWQuaXNHbG9iYWwgJiZcbiAgICAgICAgICAgICAgdGhpcyAhPT0gY21kICYmXG4gICAgICAgICAgICAgICF0aGlzLmNvbW1hbmRzLmhhcyhjbWQuX25hbWUpICYmXG4gICAgICAgICAgICAgIG5hbWVzLmluZGV4T2YoY21kLl9uYW1lKSA9PT0gLTEgJiZcbiAgICAgICAgICAgICAgKGhpZGRlbiB8fCAhY21kLmlzSGlkZGVuKVxuICAgICAgICAgICAgKSB7XG4gICAgICAgICAgICAgIG5hbWVzLnB1c2goY21kLl9uYW1lKTtcbiAgICAgICAgICAgICAgY29tbWFuZHMucHVzaChjbWQpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIGdldENvbW1hbmRzKGNtZC5fcGFyZW50LCBjb21tYW5kcywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gY29tbWFuZHM7XG4gICAgfTtcblxuICAgIHJldHVybiBnZXRDb21tYW5kcyh0aGlzLl9wYXJlbnQpO1xuICB9XG5cbiAgLyoqXG4gICAqIENoZWNrcyB3aGV0aGVyIGEgY2hpbGQgY29tbWFuZCBleGlzdHMgYnkgZ2l2ZW4gbmFtZSBvciBhbGlhcy5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvciBhbGlhcyBvZiB0aGUgY29tbWFuZC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBoYXNDb21tYW5kKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0Q29tbWFuZChuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBjb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0Q29tbWFuZDxDIGV4dGVuZHMgQ29tbWFuZDxhbnk+PihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgaGlkZGVuPzogYm9vbGVhbixcbiAgKTogQyB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZUNvbW1hbmQobmFtZSwgaGlkZGVuKSA/P1xuICAgICAgdGhpcy5nZXRHbG9iYWxDb21tYW5kKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgY29tbWFuZCBieSBuYW1lIG9yIGFsaWFzLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9yIGFsaWFzIG9mIHRoZSBjb21tYW5kLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VDb21tYW5kPEMgZXh0ZW5kcyBDb21tYW5kPGFueT4+KFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBoaWRkZW4/OiBib29sZWFuLFxuICApOiBDIHwgdW5kZWZpbmVkIHtcbiAgICBmb3IgKGNvbnN0IGNtZCBvZiB0aGlzLmNvbW1hbmRzLnZhbHVlcygpKSB7XG4gICAgICBpZiAoY21kLl9uYW1lID09PSBuYW1lIHx8IGNtZC5hbGlhc2VzLmluY2x1ZGVzKG5hbWUpKSB7XG4gICAgICAgIHJldHVybiAoY21kICYmIChoaWRkZW4gfHwgIWNtZC5pc0hpZGRlbikgPyBjbWQgOiB1bmRlZmluZWQpIGFzXG4gICAgICAgICAgfCBDXG4gICAgICAgICAgfCB1bmRlZmluZWQ7XG4gICAgICB9XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgY29tbWFuZCBieSBuYW1lIG9yIGFsaWFzLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9yIGFsaWFzIG9mIHRoZSBjb21tYW5kLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbENvbW1hbmQ8QyBleHRlbmRzIENvbW1hbmQ8YW55Pj4oXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGhpZGRlbj86IGJvb2xlYW4sXG4gICk6IEMgfCB1bmRlZmluZWQge1xuICAgIGlmICghdGhpcy5fcGFyZW50KSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3QgY21kID0gdGhpcy5fcGFyZW50LmdldEJhc2VDb21tYW5kKG5hbWUsIGhpZGRlbik7XG5cbiAgICBpZiAoIWNtZD8uaXNHbG9iYWwpIHtcbiAgICAgIHJldHVybiB0aGlzLl9wYXJlbnQuZ2V0R2xvYmFsQ29tbWFuZChuYW1lLCBoaWRkZW4pO1xuICAgIH1cblxuICAgIHJldHVybiBjbWQgYXMgQztcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW1vdmUgc3ViLWNvbW1hbmQgYnkgbmFtZSBvciBhbGlhcy5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvciBhbGlhcyBvZiB0aGUgY29tbWFuZC5cbiAgICovXG4gIHB1YmxpYyByZW1vdmVDb21tYW5kKG5hbWU6IHN0cmluZyk6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCB7XG4gICAgY29uc3QgY29tbWFuZCA9IHRoaXMuZ2V0QmFzZUNvbW1hbmQobmFtZSwgdHJ1ZSk7XG5cbiAgICBpZiAoY29tbWFuZCkge1xuICAgICAgdGhpcy5jb21tYW5kcy5kZWxldGUoY29tbWFuZC5fbmFtZSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIGNvbW1hbmQ7XG4gIH1cblxuICAvKiogR2V0IHR5cGVzLiAqL1xuICBwdWJsaWMgZ2V0VHlwZXMoKTogSVR5cGVbXSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsVHlwZXMoKS5jb25jYXQodGhpcy5nZXRCYXNlVHlwZXMoKSk7XG4gIH1cblxuICAvKiogR2V0IGJhc2UgdHlwZXMuICovXG4gIHB1YmxpYyBnZXRCYXNlVHlwZXMoKTogSVR5cGVbXSB7XG4gICAgcmV0dXJuIEFycmF5LmZyb20odGhpcy50eXBlcy52YWx1ZXMoKSk7XG4gIH1cblxuICAvKiogR2V0IGdsb2JhbCB0eXBlcy4gKi9cbiAgcHVibGljIGdldEdsb2JhbFR5cGVzKCk6IElUeXBlW10ge1xuICAgIGNvbnN0IGdldFR5cGVzID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQsXG4gICAgICB0eXBlczogSVR5cGVbXSA9IFtdLFxuICAgICAgbmFtZXM6IHN0cmluZ1tdID0gW10sXG4gICAgKTogSVR5cGVbXSA9PiB7XG4gICAgICBpZiAoY21kKSB7XG4gICAgICAgIGlmIChjbWQudHlwZXMuc2l6ZSkge1xuICAgICAgICAgIGNtZC50eXBlcy5mb3JFYWNoKCh0eXBlOiBJVHlwZSkgPT4ge1xuICAgICAgICAgICAgaWYgKFxuICAgICAgICAgICAgICB0eXBlLmdsb2JhbCAmJlxuICAgICAgICAgICAgICAhdGhpcy50eXBlcy5oYXModHlwZS5uYW1lKSAmJlxuICAgICAgICAgICAgICBuYW1lcy5pbmRleE9mKHR5cGUubmFtZSkgPT09IC0xXG4gICAgICAgICAgICApIHtcbiAgICAgICAgICAgICAgbmFtZXMucHVzaCh0eXBlLm5hbWUpO1xuICAgICAgICAgICAgICB0eXBlcy5wdXNoKHR5cGUpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIGdldFR5cGVzKGNtZC5fcGFyZW50LCB0eXBlcywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gdHlwZXM7XG4gICAgfTtcblxuICAgIHJldHVybiBnZXRUeXBlcyh0aGlzLl9wYXJlbnQpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCB0eXBlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIHR5cGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0VHlwZShuYW1lOiBzdHJpbmcpOiBJVHlwZSB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZVR5cGUobmFtZSkgPz8gdGhpcy5nZXRHbG9iYWxUeXBlKG5hbWUpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIHR5cGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgdHlwZS5cbiAgICovXG4gIHB1YmxpYyBnZXRCYXNlVHlwZShuYW1lOiBzdHJpbmcpOiBJVHlwZSB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMudHlwZXMuZ2V0KG5hbWUpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgdHlwZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSB0eXBlLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbFR5cGUobmFtZTogc3RyaW5nKTogSVR5cGUgfCB1bmRlZmluZWQge1xuICAgIGlmICghdGhpcy5fcGFyZW50KSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3QgY21kOiBJVHlwZSB8IHVuZGVmaW5lZCA9IHRoaXMuX3BhcmVudC5nZXRCYXNlVHlwZShuYW1lKTtcblxuICAgIGlmICghY21kPy5nbG9iYWwpIHtcbiAgICAgIHJldHVybiB0aGlzLl9wYXJlbnQuZ2V0R2xvYmFsVHlwZShuYW1lKTtcbiAgICB9XG5cbiAgICByZXR1cm4gY21kO1xuICB9XG5cbiAgLyoqIEdldCBjb21wbGV0aW9ucy4gKi9cbiAgcHVibGljIGdldENvbXBsZXRpb25zKCkge1xuICAgIHJldHVybiB0aGlzLmdldEdsb2JhbENvbXBsZXRpb25zKCkuY29uY2F0KHRoaXMuZ2V0QmFzZUNvbXBsZXRpb25zKCkpO1xuICB9XG5cbiAgLyoqIEdldCBiYXNlIGNvbXBsZXRpb25zLiAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbXBsZXRpb25zKCk6IElDb21wbGV0aW9uW10ge1xuICAgIHJldHVybiBBcnJheS5mcm9tKHRoaXMuY29tcGxldGlvbnMudmFsdWVzKCkpO1xuICB9XG5cbiAgLyoqIEdldCBnbG9iYWwgY29tcGxldGlvbnMuICovXG4gIHB1YmxpYyBnZXRHbG9iYWxDb21wbGV0aW9ucygpOiBJQ29tcGxldGlvbltdIHtcbiAgICBjb25zdCBnZXRDb21wbGV0aW9ucyA9IChcbiAgICAgIGNtZDogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkLFxuICAgICAgY29tcGxldGlvbnM6IElDb21wbGV0aW9uW10gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IElDb21wbGV0aW9uW10gPT4ge1xuICAgICAgaWYgKGNtZCkge1xuICAgICAgICBpZiAoY21kLmNvbXBsZXRpb25zLnNpemUpIHtcbiAgICAgICAgICBjbWQuY29tcGxldGlvbnMuZm9yRWFjaCgoY29tcGxldGlvbjogSUNvbXBsZXRpb24pID0+IHtcbiAgICAgICAgICAgIGlmIChcbiAgICAgICAgICAgICAgY29tcGxldGlvbi5nbG9iYWwgJiZcbiAgICAgICAgICAgICAgIXRoaXMuY29tcGxldGlvbnMuaGFzKGNvbXBsZXRpb24ubmFtZSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZihjb21wbGV0aW9uLm5hbWUpID09PSAtMVxuICAgICAgICAgICAgKSB7XG4gICAgICAgICAgICAgIG5hbWVzLnB1c2goY29tcGxldGlvbi5uYW1lKTtcbiAgICAgICAgICAgICAgY29tcGxldGlvbnMucHVzaChjb21wbGV0aW9uKTtcbiAgICAgICAgICAgIH1cbiAgICAgICAgICB9KTtcbiAgICAgICAgfVxuXG4gICAgICAgIHJldHVybiBnZXRDb21wbGV0aW9ucyhjbWQuX3BhcmVudCwgY29tcGxldGlvbnMsIG5hbWVzKTtcbiAgICAgIH1cblxuICAgICAgcmV0dXJuIGNvbXBsZXRpb25zO1xuICAgIH07XG5cbiAgICByZXR1cm4gZ2V0Q29tcGxldGlvbnModGhpcy5fcGFyZW50KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgY29tcGxldGlvbiBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBjb21wbGV0aW9uLlxuICAgKi9cbiAgcHVibGljIGdldENvbXBsZXRpb24obmFtZTogc3RyaW5nKTogSUNvbXBsZXRpb24gfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLmdldEJhc2VDb21wbGV0aW9uKG5hbWUpID8/IHRoaXMuZ2V0R2xvYmFsQ29tcGxldGlvbihuYW1lKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBjb21wbGV0aW9uIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGNvbXBsZXRpb24uXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbXBsZXRpb24obmFtZTogc3RyaW5nKTogSUNvbXBsZXRpb24gfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLmNvbXBsZXRpb25zLmdldChuYW1lKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZ2xvYmFsIGNvbXBsZXRpb25zIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGNvbXBsZXRpb24uXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tcGxldGlvbihuYW1lOiBzdHJpbmcpOiBJQ29tcGxldGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgaWYgKCF0aGlzLl9wYXJlbnQpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG5cbiAgICBjb25zdCBjb21wbGV0aW9uOiBJQ29tcGxldGlvbiB8IHVuZGVmaW5lZCA9IHRoaXMuX3BhcmVudC5nZXRCYXNlQ29tcGxldGlvbihcbiAgICAgIG5hbWUsXG4gICAgKTtcblxuICAgIGlmICghY29tcGxldGlvbj8uZ2xvYmFsKSB7XG4gICAgICByZXR1cm4gdGhpcy5fcGFyZW50LmdldEdsb2JhbENvbXBsZXRpb24obmFtZSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIGNvbXBsZXRpb247XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGVudmlyb25tZW50IHZhcmlhYmxlcyBvciBub3QuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgaGFzRW52VmFycyhoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0RW52VmFycyhoaWRkZW4pLmxlbmd0aCA+IDA7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGVudmlyb25tZW50IHZhcmlhYmxlcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRFbnZWYXJzKGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyW10ge1xuICAgIHJldHVybiB0aGlzLmdldEdsb2JhbEVudlZhcnMoaGlkZGVuKS5jb25jYXQodGhpcy5nZXRCYXNlRW52VmFycyhoaWRkZW4pKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBlbnZpcm9ubWVudCB2YXJpYWJsZXMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUVudlZhcnMoaGlkZGVuPzogYm9vbGVhbik6IElFbnZWYXJbXSB7XG4gICAgaWYgKCF0aGlzLmVudlZhcnMubGVuZ3RoKSB7XG4gICAgICByZXR1cm4gW107XG4gICAgfVxuXG4gICAgcmV0dXJuIGhpZGRlblxuICAgICAgPyB0aGlzLmVudlZhcnMuc2xpY2UoMClcbiAgICAgIDogdGhpcy5lbnZWYXJzLmZpbHRlcigoZW52KSA9PiAhZW52LmhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBlbnZpcm9ubWVudCB2YXJpYWJsZXMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsRW52VmFycyhoaWRkZW4/OiBib29sZWFuKTogSUVudlZhcltdIHtcbiAgICBjb25zdCBnZXRFbnZWYXJzID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQsXG4gICAgICBlbnZWYXJzOiBJRW52VmFyW10gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IElFbnZWYXJbXSA9PiB7XG4gICAgICBpZiAoY21kKSB7XG4gICAgICAgIGlmIChjbWQuZW52VmFycy5sZW5ndGgpIHtcbiAgICAgICAgICBjbWQuZW52VmFycy5mb3JFYWNoKChlbnZWYXI6IElFbnZWYXIpID0+IHtcbiAgICAgICAgICAgIGlmIChcbiAgICAgICAgICAgICAgZW52VmFyLmdsb2JhbCAmJlxuICAgICAgICAgICAgICAhdGhpcy5lbnZWYXJzLmZpbmQoKGVudikgPT4gZW52Lm5hbWVzWzBdID09PSBlbnZWYXIubmFtZXNbMF0pICYmXG4gICAgICAgICAgICAgIG5hbWVzLmluZGV4T2YoZW52VmFyLm5hbWVzWzBdKSA9PT0gLTEgJiZcbiAgICAgICAgICAgICAgKGhpZGRlbiB8fCAhZW52VmFyLmhpZGRlbilcbiAgICAgICAgICAgICkge1xuICAgICAgICAgICAgICBuYW1lcy5wdXNoKGVudlZhci5uYW1lc1swXSk7XG4gICAgICAgICAgICAgIGVudlZhcnMucHVzaChlbnZWYXIpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIGdldEVudlZhcnMoY21kLl9wYXJlbnQsIGVudlZhcnMsIG5hbWVzKTtcbiAgICAgIH1cblxuICAgICAgcmV0dXJuIGVudlZhcnM7XG4gICAgfTtcblxuICAgIHJldHVybiBnZXRFbnZWYXJzKHRoaXMuX3BhcmVudCk7XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGFuIGVudmlyb25tZW50IHZhcmlhYmxlIHdpdGggZ2l2ZW4gbmFtZSBvciBub3QuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGhhc0VudlZhcihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gISF0aGlzLmdldEVudlZhcihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBlbnZpcm9ubWVudCB2YXJpYWJsZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRFbnZWYXIobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSUVudlZhciB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZUVudlZhcihuYW1lLCBoaWRkZW4pID8/XG4gICAgICB0aGlzLmdldEdsb2JhbEVudlZhcihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIGVudmlyb25tZW50IHZhcmlhYmxlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VFbnZWYXIobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSUVudlZhciB8IHVuZGVmaW5lZCB7XG4gICAgY29uc3QgZW52VmFyOiBJRW52VmFyIHwgdW5kZWZpbmVkID0gdGhpcy5lbnZWYXJzLmZpbmQoKGVudikgPT5cbiAgICAgIGVudi5uYW1lcy5pbmRleE9mKG5hbWUpICE9PSAtMVxuICAgICk7XG5cbiAgICByZXR1cm4gZW52VmFyICYmIChoaWRkZW4gfHwgIWVudlZhci5oaWRkZW4pID8gZW52VmFyIDogdW5kZWZpbmVkO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgZW52aXJvbm1lbnQgdmFyaWFibGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsRW52VmFyKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IElFbnZWYXIgfCB1bmRlZmluZWQge1xuICAgIGlmICghdGhpcy5fcGFyZW50KSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3QgZW52VmFyOiBJRW52VmFyIHwgdW5kZWZpbmVkID0gdGhpcy5fcGFyZW50LmdldEJhc2VFbnZWYXIoXG4gICAgICBuYW1lLFxuICAgICAgaGlkZGVuLFxuICAgICk7XG5cbiAgICBpZiAoIWVudlZhcj8uZ2xvYmFsKSB7XG4gICAgICByZXR1cm4gdGhpcy5fcGFyZW50LmdldEdsb2JhbEVudlZhcihuYW1lLCBoaWRkZW4pO1xuICAgIH1cblxuICAgIHJldHVybiBlbnZWYXI7XG4gIH1cblxuICAvKiogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGV4YW1wbGVzIG9yIG5vdC4gKi9cbiAgcHVibGljIGhhc0V4YW1wbGVzKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmV4YW1wbGVzLmxlbmd0aCA+IDA7XG4gIH1cblxuICAvKiogR2V0IGFsbCBleGFtcGxlcy4gKi9cbiAgcHVibGljIGdldEV4YW1wbGVzKCk6IElFeGFtcGxlW10ge1xuICAgIHJldHVybiB0aGlzLmV4YW1wbGVzO1xuICB9XG5cbiAgLyoqIENoZWNrcyB3aGV0aGVyIHRoZSBjb21tYW5kIGhhcyBhbiBleGFtcGxlIHdpdGggZ2l2ZW4gbmFtZSBvciBub3QuICovXG4gIHB1YmxpYyBoYXNFeGFtcGxlKG5hbWU6IHN0cmluZyk6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0RXhhbXBsZShuYW1lKTtcbiAgfVxuXG4gIC8qKiBHZXQgZXhhbXBsZSB3aXRoIGdpdmVuIG5hbWUuICovXG4gIHB1YmxpYyBnZXRFeGFtcGxlKG5hbWU6IHN0cmluZyk6IElFeGFtcGxlIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5leGFtcGxlcy5maW5kKChleGFtcGxlKSA9PiBleGFtcGxlLm5hbWUgPT09IG5hbWUpO1xuICB9XG59XG5cbmZ1bmN0aW9uIGlzVXBncmFkZUNvbW1hbmQoY29tbWFuZDogdW5rbm93bik6IGNvbW1hbmQgaXMgVXBncmFkZUNvbW1hbmRJbXBsIHtcbiAgcmV0dXJuIGNvbW1hbmQgaW5zdGFuY2VvZiBDb21tYW5kICYmIFwiZ2V0TGF0ZXN0VmVyc2lvblwiIGluIGNvbW1hbmQ7XG59XG5cbmludGVyZmFjZSBVcGdyYWRlQ29tbWFuZEltcGwge1xuICBnZXRMYXRlc3RWZXJzaW9uKCk6IFByb21pc2U8c3RyaW5nPjtcbn1cblxuaW50ZXJmYWNlIElEZWZhdWx0T3B0aW9uIHtcbiAgZmxhZ3M6IHN0cmluZztcbiAgZGVzYz86IHN0cmluZztcbiAgb3B0cz86IElDb21tYW5kT3B0aW9uO1xufVxuXG50eXBlIFRyaW1MZWZ0PFQgZXh0ZW5kcyBzdHJpbmcsIFYgZXh0ZW5kcyBzdHJpbmcgfCB1bmRlZmluZWQ+ID0gVCBleHRlbmRzXG4gIGAke1Z9JHtpbmZlciBVfWAgPyBVXG4gIDogVDtcblxudHlwZSBUcmltUmlnaHQ8VCBleHRlbmRzIHN0cmluZywgViBleHRlbmRzIHN0cmluZz4gPSBUIGV4dGVuZHMgYCR7aW5mZXIgVX0ke1Z9YFxuICA/IFVcbiAgOiBUO1xuXG50eXBlIExvd2VyPFYgZXh0ZW5kcyBzdHJpbmc+ID0gViBleHRlbmRzIFVwcGVyY2FzZTxWPiA/IExvd2VyY2FzZTxWPlxuICA6IFVuY2FwaXRhbGl6ZTxWPjtcblxudHlwZSBDYW1lbENhc2U8VCBleHRlbmRzIHN0cmluZz4gPSBUIGV4dGVuZHMgYCR7aW5mZXIgVn1fJHtpbmZlciBSZXN0fWBcbiAgPyBgJHtMb3dlcjxWPn0ke0NhcGl0YWxpemU8Q2FtZWxDYXNlPFJlc3Q+Pn1gXG4gIDogVCBleHRlbmRzIGAke2luZmVyIFZ9LSR7aW5mZXIgUmVzdH1gXG4gICAgPyBgJHtMb3dlcjxWPn0ke0NhcGl0YWxpemU8Q2FtZWxDYXNlPFJlc3Q+Pn1gXG4gIDogTG93ZXI8VD47XG5cbnR5cGUgT25lT2Y8VCwgVj4gPSBUIGV4dGVuZHMgdm9pZCA/IFYgOiBUO1xuXG50eXBlIE1lcmdlPEwsIFI+ID0gTCBleHRlbmRzIHZvaWQgPyBSXG4gIDogUiBleHRlbmRzIHZvaWQgPyBMXG4gIDogTCAmIFI7XG5cbi8vIHR5cGUgTWVyZ2U8TCwgUj4gPSBMIGV4dGVuZHMgdm9pZCA/IFJcbi8vICAgOiBSIGV4dGVuZHMgdm9pZCA/IExcbi8vICAgOiBPbWl0PEwsIGtleW9mIFI+ICYgUjtcblxudHlwZSBNZXJnZVJlY3Vyc2l2ZTxMLCBSPiA9IEwgZXh0ZW5kcyB2b2lkID8gUlxuICA6IFIgZXh0ZW5kcyB2b2lkID8gTFxuICA6IEwgJiBSO1xuXG50eXBlIE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFQgZXh0ZW5kcyBzdHJpbmc+ID0gYFske1R9XWAgfCBgPCR7VH0+YDtcbnR5cGUgUmVzdFZhbHVlID0gYC4uLiR7c3RyaW5nfWAgfCBgJHtzdHJpbmd9Li4uYDtcblxuLyoqXG4gKiBSZXN0IGFyZ3Mgd2l0aCBsaXN0IHR5cGUgYW5kIGNvbXBsZXRpb25zLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGVbXTpjb21wbGV0aW9uXWBcbiAqIC0gYDwuLi5uYW1lOnR5cGVbXTpjb21wbGV0aW9uPmBcbiAqIC0gYFtuYW1lLi4uOnR5cGVbXTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGVbXTpjb21wbGV0aW9uPmBcbiAqL1xudHlwZSBSZXN0QXJnc0xpc3RUeXBlQ29tcGxldGlvbjxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH1bXToke3N0cmluZ31gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIGxpc3QgdHlwZS5cbiAqXG4gKiAtIGBbLi4ubmFtZTp0eXBlW11dYFxuICogLSBgPC4uLm5hbWU6dHlwZVtdPmBcbiAqIC0gYFtuYW1lLi4uOnR5cGVbXV1gXG4gKiAtIGA8bmFtZS4uLjp0eXBlW10+YFxuICovXG50eXBlIFJlc3RBcmdzTGlzdFR5cGU8VCBleHRlbmRzIHN0cmluZz4gPSBPcHRpb25hbE9yUmVxdWlyZWRWYWx1ZTxcbiAgYCR7UmVzdFZhbHVlfToke1R9W11gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIHR5cGUgYW5kIGNvbXBsZXRpb25zLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGU6Y29tcGxldGlvbl1gXG4gKiAtIGA8Li4ubmFtZTp0eXBlOmNvbXBsZXRpb24+YFxuICogLSBgW25hbWUuLi46dHlwZTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGU6Y29tcGxldGlvbj5gXG4gKi9cbnR5cGUgUmVzdEFyZ3NUeXBlQ29tcGxldGlvbjxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH06JHtzdHJpbmd9YFxuPjtcblxuLyoqXG4gKiBSZXN0IGFyZ3Mgd2l0aCB0eXBlLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGVdYFxuICogLSBgPC4uLm5hbWU6dHlwZT5gXG4gKiAtIGBbbmFtZS4uLjp0eXBlXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGU+YFxuICovXG50eXBlIFJlc3RBcmdzVHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH1gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncy5cbiAqIC0gYFsuLi5uYW1lXWBcbiAqIC0gYDwuLi5uYW1lPmBcbiAqIC0gYFtuYW1lLi4uXWBcbiAqIC0gYDxuYW1lLi4uPmBcbiAqL1xudHlwZSBSZXN0QXJncyA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9YFxuPjtcblxuLyoqXG4gKiBTaW5nbGUgYXJnIHdpdGggbGlzdCB0eXBlIGFuZCBjb21wbGV0aW9ucy5cbiAqXG4gKiAtIGBbbmFtZTp0eXBlW106Y29tcGxldGlvbl1gXG4gKiAtIGA8bmFtZTp0eXBlW106Y29tcGxldGlvbj5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnTGlzdFR5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfVtdOiR7c3RyaW5nfWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZyB3aXRoIGxpc3QgdHlwZS5cbiAqXG4gKiAtIGBbbmFtZTp0eXBlW11dYFxuICogLSBgPG5hbWU6dHlwZVtdPmBcbiAqL1xudHlwZSBTaW5nbGVBcmdMaXN0VHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9OiR7VH1bXWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZyAgd2l0aCB0eXBlIGFuZCBjb21wbGV0aW9uLlxuICpcbiAqIC0gYFtuYW1lOnR5cGU6Y29tcGxldGlvbl1gXG4gKiAtIGA8bmFtZTp0eXBlOmNvbXBsZXRpb24+YFxuICovXG50eXBlIFNpbmdsZUFyZ1R5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfToke3N0cmluZ31gXG4+O1xuXG4vKipcbiAqIFNpbmdsZSBhcmcgd2l0aCB0eXBlLlxuICpcbiAqIC0gYFtuYW1lOnR5cGVdYFxuICogLSBgPG5hbWU6dHlwZT5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnVHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9OiR7VH1gXG4+O1xuXG4vKipcbiAqIFNpbmdsZSBhcmcuXG4gKlxuICogLSBgW25hbWVdYFxuICogLSBgPG5hbWU+YFxuICovXG50eXBlIFNpbmdsZUFyZyA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9YFxuPjtcblxudHlwZSBEZWZhdWx0VHlwZXMgPSB7XG4gIG51bWJlcjogTnVtYmVyVHlwZTtcbiAgaW50ZWdlcjogSW50ZWdlclR5cGU7XG4gIHN0cmluZzogU3RyaW5nVHlwZTtcbiAgYm9vbGVhbjogQm9vbGVhblR5cGU7XG4gIGZpbGU6IEZpbGVUeXBlO1xufTtcblxudHlwZSBBcmd1bWVudFR5cGU8QSBleHRlbmRzIHN0cmluZywgVSwgVCA9IE1lcmdlPERlZmF1bHRUeXBlcywgVT4+ID0gQSBleHRlbmRzXG4gIFJlc3RBcmdzTGlzdFR5cGVDb21wbGV0aW9uPGluZmVyIFR5cGU+XG4gID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PEFycmF5PFI+PiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3NMaXN0VHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PEFycmF5PFI+PiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3NUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBSZXN0QXJnc1R5cGU8aW5mZXIgVHlwZT5cbiAgICA/IFQgZXh0ZW5kcyBSZWNvcmQ8VHlwZSwgaW5mZXIgUj4gPyBBcnJheTxSPiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3MgPyBBcnJheTxzdHJpbmc+XG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ0xpc3RUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmdMaXN0VHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmdUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IFIgOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ1R5cGU8aW5mZXIgVHlwZT5cbiAgICA/IFQgZXh0ZW5kcyBSZWNvcmQ8VHlwZSwgaW5mZXIgUj4gPyBSIDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmcgPyBzdHJpbmdcbiAgOiB1bmtub3duO1xuXG50eXBlIEFyZ3VtZW50VHlwZXM8QSBleHRlbmRzIHN0cmluZywgVD4gPSBBIGV4dGVuZHMgYCR7c3RyaW5nfSAke3N0cmluZ31gXG4gID8gVHlwZWRBcmd1bWVudHM8QSwgVD5cbiAgOiBBcmd1bWVudFR5cGU8QSwgVD47XG5cbnR5cGUgR2V0QXJndW1lbnRzPEEgZXh0ZW5kcyBzdHJpbmc+ID0gQSBleHRlbmRzIGAtJHtzdHJpbmd9PSR7aW5mZXIgUmVzdH1gXG4gID8gR2V0QXJndW1lbnRzPFJlc3Q+XG4gIDogQSBleHRlbmRzIGAtJHtzdHJpbmd9ICR7aW5mZXIgUmVzdH1gID8gR2V0QXJndW1lbnRzPFJlc3Q+XG4gIDogQTtcblxudHlwZSBPcHRpb25OYW1lPE5hbWUgZXh0ZW5kcyBzdHJpbmc+ID0gTmFtZSBleHRlbmRzIFwiKlwiID8gc3RyaW5nXG4gIDogQ2FtZWxDYXNlPFRyaW1SaWdodDxOYW1lLCBcIixcIj4+O1xuXG50eXBlIElzUmVxdWlyZWQ8UiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQsIEQ+ID0gUiBleHRlbmRzIHRydWUgPyB0cnVlXG4gIDogRCBleHRlbmRzIHVuZGVmaW5lZCA/IGZhbHNlXG4gIDogdHJ1ZTtcblxudHlwZSBOZWdhdGFibGVPcHRpb248XG4gIEYgZXh0ZW5kcyBzdHJpbmcsXG4gIENPLFxuICBELFxuICBOIGV4dGVuZHMgc3RyaW5nID0gT3B0aW9uTmFtZTxGPixcbj4gPSBEIGV4dGVuZHMgdW5kZWZpbmVkXG4gID8gTiBleHRlbmRzIGtleW9mIENPID8geyBbSyBpbiBOXT86IGZhbHNlIH0gOiB7IFtLIGluIE5dOiBib29sZWFuIH1cbiAgOiB7IFtLIGluIE5dOiBOb25OdWxsYWJsZTxEPiB8IGZhbHNlIH07XG5cbnR5cGUgQm9vbGVhbk9wdGlvbjxcbiAgTiBleHRlbmRzIHN0cmluZyxcbiAgQ08sXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IE4gZXh0ZW5kcyBgbm8tJHtpbmZlciBOYW1lfWAgPyBOZWdhdGFibGVPcHRpb248TmFtZSwgQ08sIEQ+XG4gIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9LiR7aW5mZXIgUmVzdH1gXG4gICAgPyAoUiBleHRlbmRzIHRydWVcbiAgICAgID8geyBbSyBpbiBPcHRpb25OYW1lPE5hbWU+XTogQm9vbGVhbk9wdGlvbjxSZXN0LCBDTywgUiwgRD4gfVxuICAgICAgOiB7IFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dPzogQm9vbGVhbk9wdGlvbjxSZXN0LCBDTywgUiwgRD4gfSlcbiAgOiAoUiBleHRlbmRzIHRydWUgPyB7IFtLIGluIE9wdGlvbk5hbWU8Tj5dOiB0cnVlIHwgRCB9XG4gICAgOiB7IFtLIGluIE9wdGlvbk5hbWU8Tj5dPzogdHJ1ZSB8IEQgfSk7XG5cbnR5cGUgVmFsdWVPcHRpb248XG4gIE4gZXh0ZW5kcyBzdHJpbmcsXG4gIEYgZXh0ZW5kcyBzdHJpbmcsXG4gIFYsXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IE4gZXh0ZW5kcyBgJHtpbmZlciBOYW1lfS4ke2luZmVyIFJlc3ROYW1lfWAgPyAoUiBleHRlbmRzIHRydWUgPyB7XG4gIFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dOiBWYWx1ZU9wdGlvbjxSZXN0TmFtZSwgRiwgViwgUiwgRD47XG59XG4gIDoge1xuICAgIFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dPzogVmFsdWVPcHRpb248UmVzdE5hbWUsIEYsIFYsIFIsIEQ+O1xuICB9KVxuICA6IChSIGV4dGVuZHMgdHJ1ZSA/IHtcbiAgICBbSyBpbiBPcHRpb25OYW1lPE4+XTogR2V0QXJndW1lbnRzPEY+IGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgPyBOb25OdWxsYWJsZTxEPiB8IHRydWUgfCBBcmd1bWVudFR5cGU8R2V0QXJndW1lbnRzPEY+LCBWPlxuICAgICAgOiBOb25OdWxsYWJsZTxEPiB8IEFyZ3VtZW50VHlwZTxHZXRBcmd1bWVudHM8Rj4sIFY+O1xuICB9XG4gICAgOiB7XG4gICAgICBbSyBpbiBPcHRpb25OYW1lPE4+XT86IEdldEFyZ3VtZW50czxGPiBleHRlbmRzIGBbJHtzdHJpbmd9XWBcbiAgICAgICAgPyBOb25OdWxsYWJsZTxEPiB8IHRydWUgfCBBcmd1bWVudFR5cGU8R2V0QXJndW1lbnRzPEY+LCBWPlxuICAgICAgICA6IE5vbk51bGxhYmxlPEQ+IHwgQXJndW1lbnRUeXBlPEdldEFyZ3VtZW50czxGPiwgVj47XG4gICAgfSk7XG5cbnR5cGUgVmFsdWVzT3B0aW9uPFxuICBUIGV4dGVuZHMgc3RyaW5nLFxuICBSZXN0IGV4dGVuZHMgc3RyaW5nLFxuICBWLFxuICBSIGV4dGVuZHMgYm9vbGVhbiB8IHVuZGVmaW5lZCA9IHVuZGVmaW5lZCxcbiAgRCA9IHVuZGVmaW5lZCxcbj4gPSBUIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0uJHtpbmZlciBSZXN0TmFtZX1gID8gKFIgZXh0ZW5kcyB0cnVlID8ge1xuICBbTiBpbiBPcHRpb25OYW1lPE5hbWU+XTogVmFsdWVzT3B0aW9uPFJlc3ROYW1lLCBSZXN0LCBWLCBSLCBEPjtcbn1cbiAgOiB7XG4gICAgW04gaW4gT3B0aW9uTmFtZTxOYW1lPl0/OiBWYWx1ZXNPcHRpb248UmVzdE5hbWUsIFJlc3QsIFYsIFIsIEQ+O1xuICB9KVxuICA6IChSIGV4dGVuZHMgdHJ1ZSA/IHtcbiAgICBbTiBpbiBPcHRpb25OYW1lPFQ+XTogR2V0QXJndW1lbnRzPFJlc3Q+IGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgPyBOb25OdWxsYWJsZTxEPiB8IHRydWUgfCBBcmd1bWVudFR5cGVzPEdldEFyZ3VtZW50czxSZXN0PiwgVj5cbiAgICAgIDogTm9uTnVsbGFibGU8RD4gfCBBcmd1bWVudFR5cGVzPEdldEFyZ3VtZW50czxSZXN0PiwgVj47XG4gIH1cbiAgICA6IHtcbiAgICAgIFtOIGluIE9wdGlvbk5hbWU8VD5dPzogR2V0QXJndW1lbnRzPFJlc3Q+IGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgICA/IE5vbk51bGxhYmxlPEQ+IHwgdHJ1ZSB8IEFyZ3VtZW50VHlwZXM8R2V0QXJndW1lbnRzPFJlc3Q+LCBWPlxuICAgICAgICA6IE5vbk51bGxhYmxlPEQ+IHwgQXJndW1lbnRUeXBlczxHZXRBcmd1bWVudHM8UmVzdD4sIFY+O1xuICAgIH0pO1xuXG50eXBlIE1hcFZhbHVlPE8sIFYsIEMgPSB1bmRlZmluZWQ+ID0gViBleHRlbmRzIHVuZGVmaW5lZCA/IEMgZXh0ZW5kcyB0cnVlID8ge1xuICBbSyBpbiBrZXlvZiBPXTogT1tLXSBleHRlbmRzIChSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHVuZGVmaW5lZClcbiAgICA/IE1hcFZhbHVlPE9bS10sIFY+XG4gICAgOiBBcnJheTxOb25OdWxsYWJsZTxPW0tdPj47XG59XG46IE9cbiAgOiB7XG4gICAgW0sgaW4ga2V5b2YgT106IE9bS10gZXh0ZW5kcyAoUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB1bmRlZmluZWQpXG4gICAgICA/IE1hcFZhbHVlPE9bS10sIFY+XG4gICAgICA6IFY7XG4gIH07XG5cbnR5cGUgR2V0T3B0aW9uTmFtZTxUPiA9IFQgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9PSR7c3RyaW5nfWBcbiAgPyBUcmltUmlnaHQ8TmFtZSwgXCIsXCI+XG4gIDogVCBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX0gJHtzdHJpbmd9YCA/IFRyaW1SaWdodDxOYW1lLCBcIixcIj5cbiAgOiBUIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfWAgPyBOYW1lXG4gIDogVCBleHRlbmRzIGAtJHtpbmZlciBOYW1lfT0ke3N0cmluZ31gID8gVHJpbVJpZ2h0PE5hbWUsIFwiLFwiPlxuICA6IFQgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX0gJHtzdHJpbmd9YCA/IFRyaW1SaWdodDxOYW1lLCBcIixcIj5cbiAgOiBUIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9YCA/IE5hbWVcbiAgOiB1bmtub3duO1xuXG50eXBlIE1lcmdlT3B0aW9uczxULCBDTywgTywgTiA9IEdldE9wdGlvbk5hbWU8VD4+ID0gTiBleHRlbmRzIGBuby0ke3N0cmluZ31gXG4gID8gU3ByZWFkPENPLCBPPlxuICA6IE4gZXh0ZW5kcyBgJHtzdHJpbmd9LiR7c3RyaW5nfWAgPyBNZXJnZVJlY3Vyc2l2ZTxDTywgTz5cbiAgOiBNZXJnZTxDTywgTz47XG5cbi8vIHR5cGUgTWVyZ2VPcHRpb25zPFQsIENPLCBPLCBOID0gR2V0T3B0aW9uTmFtZTxUPj4gPSBOIGV4dGVuZHMgYG5vLSR7c3RyaW5nfWBcbi8vICAgPyBTcHJlYWQ8Q08sIE8+XG4vLyAgIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9LiR7aW5mZXIgQ2hpbGR9YFxuLy8gICAgID8gKE9wdGlvbk5hbWU8TmFtZT4gZXh0ZW5kcyBrZXlvZiBNZXJnZTxDTywgTz5cbi8vICAgICAgID8gT3B0aW9uTmFtZTxDaGlsZD4gZXh0ZW5kc1xuLy8gICAgICAgICBrZXlvZiBOb25OdWxsYWJsZTxNZXJnZTxDTywgTz5bT3B0aW9uTmFtZTxOYW1lPl0+ID8gU3ByZWFkVHdvPENPLCBPPlxuLy8gICAgICAgOiBNZXJnZVJlY3Vyc2l2ZTxDTywgTz5cbi8vICAgICAgIDogTWVyZ2VSZWN1cnNpdmU8Q08sIE8+KVxuLy8gICA6IE1lcmdlPENPLCBPPjtcblxudHlwZSBUeXBlZE9wdGlvbjxcbiAgRiBleHRlbmRzIHN0cmluZyxcbiAgQ08sXG4gIFQsXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IG51bWJlciBleHRlbmRzIFQgPyBhbnlcbiAgOiBGIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfT0ke2luZmVyIFJlc3R9YFxuICAgID8gVmFsdWVzT3B0aW9uPE5hbWUsIFJlc3QsIFQsIElzUmVxdWlyZWQ8UiwgRD4sIEQ+XG4gIDogRiBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX0gJHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlc09wdGlvbjxOYW1lLCBSZXN0LCBULCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9YFxuICAgID8gQm9vbGVhbk9wdGlvbjxOYW1lLCBDTywgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBGIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9PSR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZXNPcHRpb248TmFtZSwgUmVzdCwgVCwgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBGIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9ICR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZXNPcHRpb248TmFtZSwgUmVzdCwgVCwgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBGIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9YCA/IEJvb2xlYW5PcHRpb248TmFtZSwgQ08sIElzUmVxdWlyZWQ8UiwgRD4sIEQ+XG4gIDogUmVjb3JkPHN0cmluZywgdW5rbm93bj47XG5cbnR5cGUgVHlwZWRBcmd1bWVudHM8QSBleHRlbmRzIHN0cmluZywgVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkPiA9XG4gIG51bWJlciBleHRlbmRzIFQgPyBhbnlcbiAgICA6IEEgZXh0ZW5kcyBgJHtpbmZlciBBcmd9ICR7aW5mZXIgUmVzdH1gXG4gICAgICA/IEFyZyBleHRlbmRzIGBbJHtzdHJpbmd9XWBcbiAgICAgICAgPyBbQXJndW1lbnRUeXBlPEFyZywgVD4/LCAuLi5UeXBlZEFyZ3VtZW50czxSZXN0LCBUPl1cbiAgICAgIDogW0FyZ3VtZW50VHlwZTxBcmcsIFQ+LCAuLi5UeXBlZEFyZ3VtZW50czxSZXN0LCBUPl1cbiAgICA6IEEgZXh0ZW5kcyBgWyR7c3RyaW5nfV1gID8gW0FyZ3VtZW50VHlwZTxBLCBUPj9dXG4gICAgOiBbQXJndW1lbnRUeXBlPEEsIFQ+XTtcblxudHlwZSBUeXBlZENvbW1hbmRBcmd1bWVudHM8TiBleHRlbmRzIHN0cmluZywgVD4gPSBudW1iZXIgZXh0ZW5kcyBUID8gYW55XG4gIDogTiBleHRlbmRzIGAke3N0cmluZ30gJHtpbmZlciBBcmdzfWAgPyBUeXBlZEFyZ3VtZW50czxBcmdzLCBUPlxuICA6IFtdO1xuXG50eXBlIFR5cGVkRW52PFxuICBOIGV4dGVuZHMgc3RyaW5nLFxuICBQIGV4dGVuZHMgc3RyaW5nIHwgdW5kZWZpbmVkLFxuICBDTyxcbiAgVCxcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gbnVtYmVyIGV4dGVuZHMgVCA/IGFueVxuICA6IE4gZXh0ZW5kcyBgJHtpbmZlciBOYW1lfT0ke2luZmVyIFJlc3R9YFxuICAgID8gVmFsdWVPcHRpb248VHJpbUxlZnQ8TmFtZSwgUD4sIFJlc3QsIFQsIFIsIEQ+XG4gIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9ICR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZU9wdGlvbjxUcmltTGVmdDxOYW1lLCBQPiwgUmVzdCwgVCwgUiwgRD5cbiAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX1gID8gQm9vbGVhbk9wdGlvbjxUcmltTGVmdDxOYW1lLCBQPiwgQ08sIFIsIEQ+XG4gIDogUmVjb3JkPHN0cmluZywgdW5rbm93bj47XG5cbnR5cGUgVHlwZWRUeXBlPFxuICBOYW1lIGV4dGVuZHMgc3RyaW5nLFxuICBIYW5kbGVyIGV4dGVuZHMgVHlwZU9yVHlwZUhhbmRsZXI8dW5rbm93bj4sXG4+ID0geyBbTiBpbiBOYW1lXTogSGFuZGxlciB9O1xuXG50eXBlIFJlcXVpcmVkS2V5czxUPiA9IHtcbiAgLy8gZGVuby1saW50LWlnbm9yZSBiYW4tdHlwZXNcbiAgW0sgaW4ga2V5b2YgVF0tPzoge30gZXh0ZW5kcyBQaWNrPFQsIEs+ID8gbmV2ZXIgOiBLO1xufVtrZXlvZiBUXTtcblxudHlwZSBPcHRpb25hbEtleXM8VD4gPSB7XG4gIC8vIGRlbm8tbGludC1pZ25vcmUgYmFuLXR5cGVzXG4gIFtLIGluIGtleW9mIFRdLT86IHt9IGV4dGVuZHMgUGljazxULCBLPiA/IEsgOiBuZXZlcjtcbn1ba2V5b2YgVF07XG5cbnR5cGUgU3ByZWFkUmVxdWlyZWRQcm9wZXJ0aWVzPFxuICBMLFxuICBSLFxuICBLIGV4dGVuZHMga2V5b2YgTCAmIGtleW9mIFIsXG4+ID0ge1xuICBbUCBpbiBLXTogRXhjbHVkZTxMW1BdLCB1bmRlZmluZWQ+IHwgRXhjbHVkZTxSW1BdLCB1bmRlZmluZWQ+O1xufTtcblxudHlwZSBTcHJlYWRPcHRpb25hbFByb3BlcnRpZXM8XG4gIEwsXG4gIFIsXG4gIEsgZXh0ZW5kcyBrZXlvZiBMICYga2V5b2YgUixcbj4gPSB7XG4gIFtQIGluIEtdPzogTFtQXSB8IFJbUF07XG59O1xuXG4vKiogTWVyZ2UgdHlwZXMgb2YgdHdvIG9iamVjdHMuICovXG50eXBlIFNwcmVhZDxMLCBSPiA9IEwgZXh0ZW5kcyB2b2lkID8gUiA6IFIgZXh0ZW5kcyB2b2lkID8gTFxuOiAvLyBQcm9wZXJ0aWVzIGluIEwgdGhhdCBkb24ndCBleGlzdCBpbiBSLlxuJiBPbWl0PEwsIGtleW9mIFI+XG4vLyBQcm9wZXJ0aWVzIGluIFIgdGhhdCBkb24ndCBleGlzdCBpbiBMLlxuJiBPbWl0PFIsIGtleW9mIEw+XG4vLyBSZXF1aXJlZCBwcm9wZXJ0aWVzIGluIFIgdGhhdCBleGlzdCBpbiBMLlxuJiBTcHJlYWRSZXF1aXJlZFByb3BlcnRpZXM8TCwgUiwgUmVxdWlyZWRLZXlzPFI+ICYga2V5b2YgTD5cbi8vIFJlcXVpcmVkIHByb3BlcnRpZXMgaW4gTCB0aGF0IGV4aXN0IGluIFIuXG4mIFNwcmVhZFJlcXVpcmVkUHJvcGVydGllczxMLCBSLCBSZXF1aXJlZEtleXM8TD4gJiBrZXlvZiBSPlxuLy8gT3B0aW9uYWwgcHJvcGVydGllcyBpbiBMIGFuZCBSLlxuJiBTcHJlYWRPcHRpb25hbFByb3BlcnRpZXM8XG4gIEwsXG4gIFIsXG4gIE9wdGlvbmFsS2V5czxMPiAmIE9wdGlvbmFsS2V5czxSPlxuPjtcblxudHlwZSBWYWx1ZU9mPFQ+ID0gVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGluZmVyIFY+ID8gVmFsdWVPZjxWPiA6IFQ7XG4iXX0=