import { DuplicateOptionName, UnknownType, ValidationError, } from "../flags/_errors.ts";
import { parseFlags } from "../flags/flags.ts";
import { getPermissions, hasPermission, isUnstable, parseArgumentsDefinition, splitArguments, } from "./_utils.ts";
import { bold, existsSync, red } from "./deps.ts";
import { CommandExecutableNotFound, CommandNotFound, DefaultCommandNotFound, DuplicateCommandAlias, DuplicateCommandName, DuplicateCompletion, DuplicateEnvironmentVariable, DuplicateExample, DuplicateType, EnvironmentVariableOptionalValue, EnvironmentVariableSingleValue, EnvironmentVariableVariadicValue, MissingArgument, MissingArguments, MissingCommandName, NoArgumentsAllowed, TooManyArguments, UnknownCommand, } from "./_errors.ts";
import { BooleanType } from "./types/boolean.ts";
import { NumberType } from "./types/number.ts";
import { StringType } from "./types/string.ts";
import { Type } from "./type.ts";
import { HelpGenerator } from "./help/_help_generator.ts";
export class Command {
    types = new Map();
    rawArgs = [];
    literalArgs = [];
    _name = "COMMAND";
    _parent;
    _globalParent;
    ver;
    desc = "";
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
    _allowEmpty = true;
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
    help(help) {
        if (typeof help === "string") {
            this.cmd._help = () => help;
        }
        else if (typeof help === "function") {
            this.cmd._help = help;
        }
        else {
            this.cmd._help = (cmd) => HelpGenerator.generate(cmd, help);
        }
        return this;
    }
    description(description) {
        this.cmd.desc = description;
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
    globalType(name, type, options) {
        return this.type(name, type, { ...options, global: true });
    }
    type(name, handler, options) {
        if (this.cmd.types.get(name) && !options?.override) {
            throw new DuplicateType(name);
        }
        this.cmd.types.set(name, { ...options, name, handler });
        if (handler instanceof Type && typeof handler.complete !== "undefined") {
            this.complete(name, (cmd, parent) => handler.complete?.(cmd, parent) || [], options);
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
    shouldThrowErrors() {
        return this.cmd.throwOnError || !!this.cmd._parent?.shouldThrowErrors();
    }
    globalOption(flags, desc, opts) {
        if (typeof opts === "function") {
            return this.option(flags, desc, { value: opts, global: true });
        }
        return this.option(flags, desc, { ...opts, global: true });
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
            typeDefinition: result.typeDefinition,
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
    async parse(args = Deno.args, dry) {
        try {
            this.reset();
            this.registerDefaults();
            this.rawArgs = args;
            const subCommand = args.length > 0 &&
                this.getCommand(args[0], true);
            if (subCommand) {
                subCommand._globalParent = this;
                return await subCommand.parse(this.rawArgs.slice(1), dry);
            }
            const result = {
                options: {},
                args: this.rawArgs,
                cmd: this,
                literal: this.literalArgs,
            };
            if (this.isExecutable) {
                if (!dry) {
                    await this.executeExecutable(this.rawArgs);
                }
                return result;
            }
            else if (this._useRawArgs) {
                if (dry) {
                    return result;
                }
                return await this.execute({}, ...this.rawArgs);
            }
            else {
                const { action, flags, unknown, literal } = this.parseFlags(this.rawArgs);
                this.literalArgs = literal;
                const params = this.parseArguments(unknown, flags);
                await this.validateEnvVars();
                if (dry || action) {
                    if (action) {
                        await action.call(this, flags, ...params);
                    }
                    return {
                        options: flags,
                        args: params,
                        cmd: this,
                        literal: this.literalArgs,
                    };
                }
                return await this.execute(flags, ...params);
            }
        }
        catch (error) {
            throw this.error(error);
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
        !this.types.has("boolean") &&
            this.type("boolean", new BooleanType(), { global: true });
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
                action: function () {
                    this.showVersion();
                    Deno.exit(0);
                },
                ...(this._versionOption?.opts ?? {}),
            });
        }
        if (this._helpOption !== false) {
            this.option(this._helpOption?.flags || "-h, --help", this._helpOption?.desc || "Show this help.", {
                standalone: true,
                global: true,
                prepend: true,
                action: function () {
                    this.showHelp();
                    Deno.exit(0);
                },
                ...(this._helpOption?.opts ?? {}),
            });
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
        return { options, args, cmd: this, literal: this.literalArgs };
    }
    async executeExecutable(args) {
        const permissions = await getPermissions();
        if (!permissions.read) {
            await Deno.permissions?.request({ name: "read" });
        }
        if (!permissions.run) {
            await Deno.permissions?.request({ name: "run" });
        }
        const [main, ...names] = this.getPath().split(" ");
        names.unshift(main.replace(/\.ts$/, ""));
        const executableName = names.join("-");
        const files = [];
        const parts = Deno.mainModule.replace(/^file:\/\//g, "")
            .split("/");
        parts.pop();
        const path = parts.join("/");
        files.push(path + "/" + executableName, path + "/" + executableName + ".ts");
        files.push(executableName, executableName + ".ts");
        const denoOpts = [];
        if (isUnstable()) {
            denoOpts.push("--unstable");
        }
        denoOpts.push("--allow-read", "--allow-run");
        Object.keys(permissions)
            .forEach((name) => {
            if (name === "read" || name === "run") {
                return;
            }
            if (permissions[name]) {
                denoOpts.push(`--allow-${name}`);
            }
        });
        for (const file of files) {
            if (!existsSync(file)) {
                continue;
            }
            const cmd = ["deno", "run", ...denoOpts, file, ...args];
            const process = Deno.run({
                cmd: cmd,
            });
            const status = await process.status();
            if (!status.success) {
                Deno.exit(status.code);
            }
            return;
        }
        throw new CommandExecutableNotFound(executableName, files);
    }
    parseFlags(args) {
        let action;
        const result = parseFlags(args, {
            stopEarly: this._stopEarly,
            allowEmpty: this._allowEmpty,
            flags: this.getOptions(true),
            parse: (type) => this.parseType(type),
            option: (option) => {
                if (!action && option.action) {
                    action = option.action;
                }
            },
        });
        return { ...result, action };
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
    async validateEnvVars() {
        if (!await hasPermission("env")) {
            return;
        }
        const envVars = this.getEnvVars(true);
        if (!envVars.length) {
            return;
        }
        envVars.forEach((env) => {
            const name = env.names.find((name) => !!Deno.env.get(name));
            if (name) {
                this.parseType({
                    label: "Environment variable",
                    type: env.type,
                    name,
                    value: Deno.env.get(name) ?? "",
                });
            }
        });
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
                    if (arg) {
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
        Deno.stderr.writeSync(new TextEncoder().encode(red(`  ${bold("error")}: ${error.message}\n`) + "\n"));
        Deno.exit(1);
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
    getShortDescription() {
        return this.getDescription()
            .trim()
            .split("\n")
            .shift();
    }
    getRawArgs() {
        return this.rawArgs;
    }
    getLiteralArgs() {
        return this.literalArgs;
    }
    showVersion() {
        Deno.stdout.writeSync(new TextEncoder().encode(this.getVersion()));
    }
    showHelp() {
        Deno.stdout.writeSync(new TextEncoder().encode(this.getHelp()));
    }
    getHelp() {
        this.registerDefaults();
        return this.getHelpHandler().call(this, this);
    }
    getHelpHandler() {
        return this._help ?? this._parent?.getHelpHandler();
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29tbWFuZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImNvbW1hbmQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUNMLG1CQUFtQixFQUNuQixXQUFXLEVBQ1gsZUFBZSxHQUNoQixNQUFNLHFCQUFxQixDQUFDO0FBQzdCLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUUvQyxPQUFPLEVBQ0wsY0FBYyxFQUNkLGFBQWEsRUFDYixVQUFVLEVBQ1Ysd0JBQXdCLEVBQ3hCLGNBQWMsR0FDZixNQUFNLGFBQWEsQ0FBQztBQUVyQixPQUFPLEVBQUUsSUFBSSxFQUFFLFVBQVUsRUFBRSxHQUFHLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDbEQsT0FBTyxFQUNMLHlCQUF5QixFQUN6QixlQUFlLEVBQ2Ysc0JBQXNCLEVBQ3RCLHFCQUFxQixFQUNyQixvQkFBb0IsRUFDcEIsbUJBQW1CLEVBQ25CLDRCQUE0QixFQUM1QixnQkFBZ0IsRUFDaEIsYUFBYSxFQUNiLGdDQUFnQyxFQUNoQyw4QkFBOEIsRUFDOUIsZ0NBQWdDLEVBQ2hDLGVBQWUsRUFDZixnQkFBZ0IsRUFDaEIsa0JBQWtCLEVBQ2xCLGtCQUFrQixFQUNsQixnQkFBZ0IsRUFDaEIsY0FBYyxHQUNmLE1BQU0sY0FBYyxDQUFDO0FBQ3RCLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUNqRCxPQUFPLEVBQUUsVUFBVSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDL0MsT0FBTyxFQUFFLFVBQVUsRUFBRSxNQUFNLG1CQUFtQixDQUFDO0FBQy9DLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDakMsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLDJCQUEyQixDQUFDO0FBNEMxRCxNQUFNLE9BQU8sT0FBTztJQVlWLEtBQUssR0FBdUIsSUFBSSxHQUFHLEVBQUUsQ0FBQztJQUN0QyxPQUFPLEdBQWEsRUFBRSxDQUFD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