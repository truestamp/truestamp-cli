// deno-lint-ignore-file no-explicit-any
import { UnknownType, ValidationError as FlagsValidationError } from "../flags/_errors.ts";
import { MissingRequiredEnvVar } from "./_errors.ts";
import { parseFlags } from "../flags/flags.ts";
import { getDescription, parseArgumentsDefinition, splitArguments } from "./_utils.ts";
import { blue, bold, red, yellow } from "./deps.ts";
import { CommandExecutableNotFound, CommandNotFound, DefaultCommandNotFound, DuplicateCommandAlias, DuplicateCommandName, DuplicateCompletion, DuplicateEnvironmentVariable, DuplicateExample, DuplicateOptionName, DuplicateType, EnvironmentVariableOptionalValue, EnvironmentVariableSingleValue, EnvironmentVariableVariadicValue, MissingArgument, MissingArguments, MissingCommandName, NoArgumentsAllowed, TooManyArguments, UnknownCommand, ValidationError } from "./_errors.ts";
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
    // @TODO: get script name: https://github.com/denoland/deno/pull/5034
    // private name: string = location.pathname.split( '/' ).pop() as string;
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
    _versionOptions;
    _helpOptions;
    _versionOption;
    _helpOption;
    _help;
    _shouldExit;
    _meta = {};
    _groupName;
    _noGlobals = false;
    versionOption(flags, desc, opts) {
        this._versionOptions = flags === false ? flags : {
            flags,
            desc,
            opts: typeof opts === "function" ? {
                action: opts
            } : opts
        };
        return this;
    }
    helpOption(flags, desc, opts) {
        this._helpOptions = flags === false ? flags : {
            flags,
            desc,
            opts: typeof opts === "function" ? {
                action: opts
            } : opts
        };
        return this;
    }
    /**
   * Add new sub-command.
   * @param nameAndArguments  Command definition. E.g: `my-command <input-file:string> <output-file:string>`
   * @param cmdOrDescription  The description of the new child command.
   * @param override          Override existing child command.
   */ command(nameAndArguments, cmdOrDescription, override) {
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
        } else {
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
        aliases.forEach((alias)=>cmd.alias(alias));
        this.commands.set(name, cmd);
        this.select(name);
        return this;
    }
    /**
   * Add new command alias.
   * @param alias Tha name of the alias.
   */ alias(alias) {
        if (this.cmd._name === alias || this.cmd.aliases.includes(alias)) {
            throw new DuplicateCommandAlias(alias);
        }
        this.cmd.aliases.push(alias);
        return this;
    }
    /** Reset internal command reference to main command. */ reset() {
        this._groupName = undefined;
        this.cmd = this;
        return this;
    }
    /**
   * Set internal command pointer to child command with given name.
   * @param name The name of the command to select.
   */ select(name) {
        const cmd = this.getBaseCommand(name, true);
        if (!cmd) {
            throw new CommandNotFound(name, this.getBaseCommands(true));
        }
        this.cmd = cmd;
        return this;
    }
    /*****************************************************************************
   **** SUB HANDLER ************************************************************
   *****************************************************************************/ /** Set command name. */ name(name) {
        this.cmd._name = name;
        return this;
    }
    /**
   * Set command version.
   * @param version Semantic version string string or method that returns the version string.
   */ version(version) {
        if (typeof version === "string") {
            this.cmd.ver = ()=>version;
        } else if (typeof version === "function") {
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
    /**
   * Set command help.
   * @param help Help string, method, or config for generator that returns the help string.
   */ help(help) {
        if (typeof help === "string") {
            this.cmd._help = ()=>help;
        } else if (typeof help === "function") {
            this.cmd._help = help;
        } else {
            this.cmd._help = (cmd, options)=>HelpGenerator.generate(cmd, {
                    ...help,
                    ...options
                });
        }
        return this;
    }
    /**
   * Set the long command description.
   * @param description The command description.
   */ description(description) {
        this.cmd.desc = description;
        return this;
    }
    /**
   * Set the command usage. Defaults to arguments.
   * @param usage The command usage.
   */ usage(usage) {
        this.cmd._usage = usage;
        return this;
    }
    /**
   * Hide command from help, completions, etc.
   */ hidden() {
        this.cmd.isHidden = true;
        return this;
    }
    /** Make command globally available. */ global() {
        this.cmd.isGlobal = true;
        return this;
    }
    /** Make command executable. */ executable() {
        this.cmd.isExecutable = true;
        return this;
    }
    /**
   * Set command arguments:
   *
   *   <requiredArg:string> [optionalArg: number] [...restArgs:string]
   */ arguments(args) {
        this.cmd.argsDefinition = args;
        return this;
    }
    /**
   * Set command callback method.
   * @param fn Command action handler.
   */ action(fn) {
        this.cmd.fn = fn;
        return this;
    }
    /**
   * Don't throw an error if the command was called without arguments.
   * @param allowEmpty Enable/disable allow empty.
   */ allowEmpty(allowEmpty) {
        this.cmd._allowEmpty = allowEmpty !== false;
        return this;
    }
    /**
   * Enable stop early. If enabled, all arguments starting from the first non
   * option argument will be passed as arguments with type string to the command
   * action handler.
   *
   * For example:
   *     `command --debug-level warning server --port 80`
   *
   * Will result in:
   *     - options: `{debugLevel: 'warning'}`
   *     - args: `['server', '--port', '80']`
   *
   * @param stopEarly Enable/disable stop early.
   */ stopEarly(stopEarly = true) {
        this.cmd._stopEarly = stopEarly;
        return this;
    }
    /**
   * Disable parsing arguments. If enabled the raw arguments will be passed to
   * the action handler. This has no effect for parent or child commands. Only
   * for the command on which this method was called.
   * @param useRawArgs Enable/disable raw arguments.
   */ useRawArgs(useRawArgs = true) {
        this.cmd._useRawArgs = useRawArgs;
        return this;
    }
    /**
   * Set default command. The default command is executed when the program
   * was called without any argument and if no action handler is registered.
   * @param name Name of the default command.
   */ default(name) {
        this.cmd.defaultCommand = name;
        return this;
    }
    globalType(name, handler, options) {
        return this.type(name, handler, {
            ...options,
            global: true
        });
    }
    /**
   * Register custom type.
   * @param name    The name of the type.
   * @param handler The callback method to parse the type.
   * @param options Type options.
   */ type(name, handler, options) {
        if (this.cmd.types.get(name) && !options?.override) {
            throw new DuplicateType(name);
        }
        this.cmd.types.set(name, {
            ...options,
            name,
            handler
        });
        if (handler instanceof Type && (typeof handler.complete !== "undefined" || typeof handler.values !== "undefined")) {
            const completeHandler = (cmd, parent)=>handler.complete?.(cmd, parent) || [];
            this.complete(name, completeHandler, options);
        }
        return this;
    }
    globalComplete(name, complete, options) {
        return this.complete(name, complete, {
            ...options,
            global: true
        });
    }
    complete(name, complete, options) {
        if (this.cmd.completions.has(name) && !options?.override) {
            throw new DuplicateCompletion(name);
        }
        this.cmd.completions.set(name, {
            name,
            complete,
            ...options
        });
        return this;
    }
    /**
   * Throw validation errors instead of calling `Deno.exit()` to handle
   * validation errors manually.
   *
   * A validation error is thrown when the command is wrongly used by the user.
   * For example: If the user passes some invalid options or arguments to the
   * command.
   *
   * This has no effect for parent commands. Only for the command on which this
   * method was called and all child commands.
   *
   * **Example:**
   *
   * ```
   * try {
   *   cmd.parse();
   * } catch(error) {
   *   if (error instanceof ValidationError) {
   *     cmd.showHelp();
   *     Deno.exit(1);
   *   }
   *   throw error;
   * }
   * ```
   *
   * @see ValidationError
   */ throwErrors() {
        this.cmd.throwOnError = true;
        return this;
    }
    /**
   * Same as `.throwErrors()` but also prevents calling `Deno.exit` after
   * printing help or version with the --help and --version option.
   */ noExit() {
        this.cmd._shouldExit = false;
        this.throwErrors();
        return this;
    }
    /**
   * Disable inheriting global commands, options and environment variables from
   * parent commands.
   */ noGlobals() {
        this.cmd._noGlobals = true;
        return this;
    }
    /** Check whether the command should throw errors or exit. */ shouldThrowErrors() {
        return this.throwOnError || !!this._parent?.shouldThrowErrors();
    }
    /** Check whether the command should exit after printing help or version. */ shouldExit() {
        return this._shouldExit ?? this._parent?.shouldExit() ?? true;
    }
    globalOption(flags, desc, opts) {
        if (typeof opts === "function") {
            return this.option(flags, desc, {
                value: opts,
                global: true
            });
        }
        return this.option(flags, desc, {
            ...opts,
            global: true
        });
    }
    /**
   * Enable grouping of options and set the name of the group.
   * All option which are added after calling the `.group()` method will be
   * grouped in the help output. If the `.group()` method can be use multiple
   * times to create more groups.
   *
   * @param name The name of the option group.
   */ group(name) {
        this.cmd._groupName = name;
        return this;
    }
    option(flags, desc, opts) {
        if (typeof opts === "function") {
            return this.option(flags, desc, {
                value: opts
            });
        }
        const result = splitArguments(flags);
        const args = result.typeDefinition ? parseArgumentsDefinition(result.typeDefinition) : [];
        const option = {
            ...opts,
            name: "",
            description: desc,
            args,
            flags: result.flags,
            equalsSign: result.equalsSign,
            typeDefinition: result.typeDefinition,
            groupName: this._groupName
        };
        if (option.separator) {
            for (const arg of args){
                if (arg.list) {
                    arg.separator = option.separator;
                }
            }
        }
        for (const part of option.flags){
            const arg1 = part.trim();
            const isLong = /^--/.test(arg1);
            const name = isLong ? arg1.slice(2) : arg1.slice(1);
            if (this.cmd.getBaseOption(name, true)) {
                if (opts?.override) {
                    this.removeOption(name);
                } else {
                    throw new DuplicateOptionName(name);
                }
            }
            if (!option.name && isLong) {
                option.name = name;
            } else if (!option.aliases) {
                option.aliases = [
                    name
                ];
            } else {
                option.aliases.push(name);
            }
        }
        if (option.prepend) {
            this.cmd.options.unshift(option);
        } else {
            this.cmd.options.push(option);
        }
        return this;
    }
    /**
   * Add new command example.
   * @param name          Name of the example.
   * @param description   The content of the example.
   */ example(name, description) {
        if (this.cmd.hasExample(name)) {
            throw new DuplicateExample(name);
        }
        this.cmd.examples.push({
            name,
            description
        });
        return this;
    }
    globalEnv(name, description, options) {
        return this.env(name, description, {
            ...options,
            global: true
        });
    }
    env(name, description, options) {
        const result = splitArguments(name);
        if (!result.typeDefinition) {
            result.typeDefinition = "<value:boolean>";
        }
        if (result.flags.some((envName)=>this.cmd.getBaseEnvVar(envName, true))) {
            throw new DuplicateEnvironmentVariable(name);
        }
        const details = parseArgumentsDefinition(result.typeDefinition);
        if (details.length > 1) {
            throw new EnvironmentVariableSingleValue(name);
        } else if (details.length && details[0].optionalValue) {
            throw new EnvironmentVariableOptionalValue(name);
        } else if (details.length && details[0].variadic) {
            throw new EnvironmentVariableVariadicValue(name);
        }
        this.cmd.envVars.push({
            name: result.flags[0],
            names: result.flags,
            description,
            type: details[0].type,
            details: details.shift(),
            ...options
        });
        return this;
    }
    /*****************************************************************************
   **** MAIN HANDLER ***********************************************************
   *****************************************************************************/ /**
   * Parse command line arguments and execute matched command.
   * @param args Command line args to parse. Ex: `cmd.parse( Deno.args )`
   */ async parse(args = Deno.args) {
        try {
            return await this.parseCommand({
                args
            });
        } catch (error) {
            this.throw(error instanceof Error ? error : new Error(`[non-error-thrown] ${error}`));
        }
    }
    async parseCommand(ctx) {
        this.reset();
        this.registerDefaults();
        this.rawArgs = ctx.args;
        if (this.isExecutable) {
            await this.executeExecutable(ctx.args);
            return {
                options: {},
                args: [],
                cmd: this,
                literal: []
            };
        }
        if (this._useRawArgs) {
            const env = await this.parseEnvVars(this.envVars);
            return this.execute(env, ...ctx.args);
        }
        let preParseGlobals = false;
        let subCommand;
        // Pre parse globals to support: cmd --global-option sub-command --option
        if (ctx.args.length > 0) {
            // Detect sub command.
            subCommand = this.getCommand(ctx.args[0], true);
            if (subCommand) {
                ctx.args = ctx.args.slice(1);
            } else {
                // Only pre parse globals if first arg ist a global option.
                const optionName = ctx.args[0].replace(/^-+/, "");
                preParseGlobals = this.getOption(optionName, true)?.global === true;
                // Parse global options & env vars.
                if (preParseGlobals) {
                    ctx = await this.parseGlobalOptionsAndEnvVars(ctx);
                }
            }
        } else {
            preParseGlobals = false;
        }
        // Parse sub command.
        if (subCommand || ctx.args.length > 0) {
            if (!subCommand) {
                subCommand = this.getCommand(ctx.args[0], true);
                if (subCommand) {
                    ctx.args = ctx.args.slice(1);
                }
            }
            if (subCommand) {
                subCommand._globalParent = this;
                return subCommand.parseCommand(ctx);
            }
        }
        // Parse rest options & env vars.
        ctx = await this.parseOptionsAndEnvVars(ctx, preParseGlobals);
        this.literalArgs = ctx.literal ?? [];
        // Merge env and global options.
        const options = {
            ...ctx.env,
            ...ctx.options
        };
        // Parse arguments.
        const params = this.parseArguments(ctx.args, options);
        // Execute option action.
        if (ctx.action) {
            await ctx.action.action.call(this, options, ...params);
            if (ctx.action.standalone) {
                return {
                    options,
                    args: params,
                    cmd: this,
                    literal: this.literalArgs
                };
            }
        }
        return this.execute(options, ...params);
    }
    async parseGlobalOptionsAndEnvVars(ctx) {
        // Parse global env vars.
        const envVars = [
            ...this.envVars.filter((envVar)=>envVar.global),
            ...this.getGlobalEnvVars(true), 
        ];
        const isHelpOption = this.getHelpOption()?.flags.includes(ctx.args[0]);
        const env = await this.parseEnvVars(envVars, !isHelpOption);
        // Parse global options.
        const options = [
            ...this.options.filter((option)=>option.global),
            ...this.getGlobalOptions(true), 
        ];
        return this.parseOptions(ctx, options, env, true);
    }
    async parseOptionsAndEnvVars(ctx, preParseGlobals) {
        // Parse env vars.
        const envVars = preParseGlobals ? this.envVars.filter((envVar)=>!envVar.global) : this.getEnvVars(true);
        const helpOption = this.getHelpOption();
        const isVersionOption = this._versionOption?.flags.includes(ctx.args[0]);
        const isHelpOption = helpOption && ctx.options?.[helpOption.name] === true;
        const env = {
            ...ctx.env,
            ...await this.parseEnvVars(envVars, !isHelpOption && !isVersionOption)
        };
        // Parse options.
        const options = preParseGlobals ? this.options.filter((option)=>!option.global) : this.getOptions(true);
        return this.parseOptions(ctx, options, env);
    }
    /** Register default options like `--version` and `--help`. */ registerDefaults() {
        if (this.hasDefaults || this.getParent()) {
            return this;
        }
        this.hasDefaults = true;
        this.reset();
        !this.types.has("string") && this.type("string", new StringType(), {
            global: true
        });
        !this.types.has("number") && this.type("number", new NumberType(), {
            global: true
        });
        !this.types.has("integer") && this.type("integer", new IntegerType(), {
            global: true
        });
        !this.types.has("boolean") && this.type("boolean", new BooleanType(), {
            global: true
        });
        !this.types.has("file") && this.type("file", new FileType(), {
            global: true
        });
        if (!this._help) {
            this.help({
                hints: true,
                types: false
            });
        }
        if (this._versionOptions !== false && (this._versionOptions || this.ver)) {
            this.option(this._versionOptions?.flags || "-V, --version", this._versionOptions?.desc || "Show the version number for this program.", {
                standalone: true,
                prepend: true,
                action: async function() {
                    const long = this.getRawArgs().includes(`--${this._versionOption?.name}`);
                    if (long) {
                        await this.checkVersion();
                        this.showLongVersion();
                    } else {
                        this.showVersion();
                    }
                    this.exit();
                },
                ...this._versionOptions?.opts ?? {}
            });
            this._versionOption = this.options[0];
        }
        if (this._helpOptions !== false) {
            this.option(this._helpOptions?.flags || "-h, --help", this._helpOptions?.desc || "Show this help.", {
                standalone: true,
                global: true,
                prepend: true,
                action: async function() {
                    const long = this.getRawArgs().includes(`--${this.getHelpOption()?.name}`);
                    await this.checkVersion();
                    this.showHelp({
                        long
                    });
                    this.exit();
                },
                ...this._helpOptions?.opts ?? {}
            });
            this._helpOption = this.options[0];
        }
        return this;
    }
    /**
   * Execute command.
   * @param options A map of options.
   * @param args Command arguments.
   */ async execute(options, ...args) {
        if (this.fn) {
            await this.fn(options, ...args);
        } else if (this.defaultCommand) {
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
            literal: this.literalArgs
        };
    }
    /**
   * Execute external sub-command.
   * @param args Raw command line arguments.
   */ async executeExecutable(args) {
        const command = this.getPath().replace(/\s+/g, "-");
        await Deno.permissions.request({
            name: "run",
            command
        });
        try {
            const process = Deno.run({
                cmd: [
                    command,
                    ...args
                ]
            });
            const status = await process.status();
            if (!status.success) {
                Deno.exit(status.code);
            }
        } catch (error) {
            if (error instanceof Deno.errors.NotFound) {
                throw new CommandExecutableNotFound(command);
            }
            throw error;
        }
    }
    /**
   * Parse raw command line arguments.
   * @param args Raw command line arguments.
   */ parseOptions(ctx, options, env, stopEarly = this._stopEarly) {
        try {
            let action;
            const parseResult = parseFlags(ctx.args, {
                stopEarly,
                allowEmpty: this._allowEmpty,
                flags: options,
                ignoreDefaults: env,
                parse: (type)=>this.parseType(type),
                option: (option)=>{
                    if (!action && option.action) {
                        action = option;
                    }
                }
            });
            // Merge context.
            return {
                args: parseResult.unknown,
                options: {
                    ...ctx.options,
                    ...parseResult.flags
                },
                env: {
                    ...ctx.env,
                    ...env
                },
                action: ctx.action ?? action,
                literal: parseResult.literal
            };
        } catch (error) {
            if (error instanceof FlagsValidationError) {
                throw new ValidationError(error.message);
            }
            throw error;
        }
    }
    /** Parse argument type. */ parseType(type) {
        const typeSettings = this.getType(type.type);
        if (!typeSettings) {
            throw new UnknownType(type.type, this.getTypes().map((type)=>type.name));
        }
        try {
            return typeSettings.handler instanceof Type ? typeSettings.handler.parse(type) : typeSettings.handler(type);
        } catch (error) {
            if (error instanceof FlagsValidationError) {
                throw new ValidationError(error.message);
            }
            throw error;
        }
    }
    /**
   * Read and validate environment variables.
   * @param envVars env vars defined by the command
   * @param validate when true, throws an error if a required env var is missing
   */ async parseEnvVars(envVars, validate = true) {
        const result = {};
        for (const env of envVars){
            const found = await this.findEnvVar(env.names);
            if (found) {
                const { name , value  } = found;
                const propertyName = underscoreToCamelCase(env.prefix ? env.names[0].replace(new RegExp(`^${env.prefix}`), "") : env.names[0]);
                if (env.details.list) {
                    const values = value.split(env.details.separator ?? ",");
                    result[propertyName] = values.map((value)=>this.parseType({
                            label: "Environment variable",
                            type: env.type,
                            name,
                            value
                        }));
                } else {
                    result[propertyName] = this.parseType({
                        label: "Environment variable",
                        type: env.type,
                        name,
                        value
                    });
                }
                if (env.value && typeof result[propertyName] !== "undefined") {
                    result[propertyName] = env.value(result[propertyName]);
                }
            } else if (env.required && validate) {
                throw new MissingRequiredEnvVar(env);
            }
        }
        return result;
    }
    async findEnvVar(names) {
        for (const name of names){
            const status = await Deno.permissions.query({
                name: "env",
                variable: name
            });
            if (status.state === "granted") {
                const value = Deno.env.get(name);
                if (value) {
                    return {
                        name,
                        value
                    };
                }
            }
        }
        return undefined;
    }
    /**
   * Parse command-line arguments.
   * @param args  Raw command line arguments.
   * @param options Parsed command line options.
   */ parseArguments(args, options) {
        const params = [];
        // remove array reference
        args = args.slice(0);
        if (!this.hasArguments()) {
            if (args.length) {
                if (this.hasCommands(true)) {
                    throw new UnknownCommand(args[0], this.getCommands());
                } else {
                    throw new NoArgumentsAllowed(this.getPath());
                }
            }
        } else {
            if (!args.length) {
                const required = this.getArguments().filter((expectedArg)=>!expectedArg.optionalValue).map((expectedArg)=>expectedArg.name);
                if (required.length) {
                    const optionNames = Object.keys(options);
                    const hasStandaloneOption = !!optionNames.find((name)=>this.getOption(name, true)?.standalone);
                    if (!hasStandaloneOption) {
                        throw new MissingArguments(required);
                    }
                }
            } else {
                for (const expectedArg of this.getArguments()){
                    if (!args.length) {
                        if (expectedArg.optionalValue) {
                            break;
                        }
                        throw new MissingArgument(`Missing argument: ${expectedArg.name}`);
                    }
                    let arg;
                    const parseArgValue = (value)=>{
                        return expectedArg.list ? value.split(",").map((value)=>parseArgType(value)) : parseArgType(value);
                    };
                    const parseArgType = (value)=>{
                        return this.parseType({
                            label: "Argument",
                            type: expectedArg.type,
                            name: expectedArg.name,
                            value
                        });
                    };
                    if (expectedArg.variadic) {
                        arg = args.splice(0, args.length).map((value)=>parseArgValue(value));
                    } else {
                        arg = parseArgValue(args.shift());
                    }
                    if (expectedArg.variadic && Array.isArray(arg)) {
                        params.push(...arg);
                    } else if (typeof arg !== "undefined") {
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
    /**
   * Handle error. If `throwErrors` is enabled the error will be returned,
   * otherwise a formatted error message will be printed and `Deno.exit(1)`
   * will be called.
   * @param error Error to handle.
   */ throw(error) {
        if (this.shouldThrowErrors() || !(error instanceof ValidationError)) {
            throw error;
        }
        this.showHelp();
        console.error(red(`  ${bold("error")}: ${error.message}\n`));
        Deno.exit(error instanceof ValidationError ? error.exitCode : 1);
    }
    /*****************************************************************************
   **** GETTER *****************************************************************
   *****************************************************************************/ /** Get command name. */ getName() {
        return this._name;
    }
    /** Get parent command. */ getParent() {
        return this._parent;
    }
    /**
   * Get parent command from global executed command.
   * Be sure, to call this method only inside an action handler. Unless this or any child command was executed,
   * this method returns always undefined.
   */ getGlobalParent() {
        return this._globalParent;
    }
    /** Get main command. */ getMainCommand() {
        return this._parent?.getMainCommand() ?? this;
    }
    /** Get command name aliases. */ getAliases() {
        return this.aliases;
    }
    /** Get full command path. */ getPath() {
        return this._parent ? this._parent.getPath() + " " + this._name : this._name;
    }
    /** Get arguments definition. E.g: <input-file:string> <output-file:string> */ getArgsDefinition() {
        return this.argsDefinition;
    }
    /**
   * Get argument by name.
   * @param name Name of the argument.
   */ getArgument(name) {
        return this.getArguments().find((arg)=>arg.name === name);
    }
    /** Get arguments. */ getArguments() {
        if (!this.args.length && this.argsDefinition) {
            this.args = parseArgumentsDefinition(this.argsDefinition);
        }
        return this.args;
    }
    /** Check if command has arguments. */ hasArguments() {
        return !!this.argsDefinition;
    }
    /** Get command version. */ getVersion() {
        return this.getVersionHandler()?.call(this, this);
    }
    /** Get help handler method. */ getVersionHandler() {
        return this.ver ?? this._parent?.getVersionHandler();
    }
    /** Get command description. */ getDescription() {
        // call description method only once
        return typeof this.desc === "function" ? this.desc = this.desc() : this.desc;
    }
    getUsage() {
        return this._usage ?? this.getArgsDefinition();
    }
    /** Get short command description. This is the first line of the description. */ getShortDescription() {
        return getDescription(this.getDescription(), true);
    }
    /** Get original command-line arguments. */ getRawArgs() {
        return this.rawArgs;
    }
    /** Get all arguments defined after the double dash. */ getLiteralArgs() {
        return this.literalArgs;
    }
    /** Output generated help without exiting. */ showVersion() {
        console.log(this.getVersion());
    }
    /** Returns command name, version and meta data. */ getLongVersion() {
        return `${bold(this.getMainCommand().getName())} ${blue(this.getVersion() ?? "")}` + Object.entries(this.getMeta()).map(([k, v])=>`\n${bold(k)} ${blue(v)}`).join("");
    }
    /** Outputs command name, version and meta data. */ showLongVersion() {
        console.log(this.getLongVersion());
    }
    /** Output generated help without exiting. */ showHelp(options) {
        console.log(this.getHelp(options));
    }
    /** Get generated help. */ getHelp(options) {
        this.registerDefaults();
        return this.getHelpHandler().call(this, this, options ?? {});
    }
    /** Get help handler method. */ getHelpHandler() {
        return this._help ?? this._parent?.getHelpHandler();
    }
    exit(code = 0) {
        if (this.shouldExit()) {
            Deno.exit(code);
        }
    }
    /** Check if new version is available and add hint to version. */ async checkVersion() {
        const mainCommand = this.getMainCommand();
        const upgradeCommand = mainCommand.getCommand("upgrade");
        if (!isUpgradeCommand(upgradeCommand)) {
            return;
        }
        const latestVersion = await upgradeCommand.getLatestVersion();
        const currentVersion = mainCommand.getVersion();
        if (currentVersion === latestVersion) {
            return;
        }
        const versionHelpText = `(New version available: ${latestVersion}. Run '${mainCommand.getName()} upgrade' to upgrade to the latest version!)`;
        mainCommand.version(`${currentVersion}  ${bold(yellow(versionHelpText))}`);
    }
    /*****************************************************************************
   **** Options GETTER *********************************************************
   *****************************************************************************/ /**
   * Checks whether the command has options or not.
   * @param hidden Include hidden options.
   */ hasOptions(hidden) {
        return this.getOptions(hidden).length > 0;
    }
    /**
   * Get options.
   * @param hidden Include hidden options.
   */ getOptions(hidden) {
        return this.getGlobalOptions(hidden).concat(this.getBaseOptions(hidden));
    }
    /**
   * Get base options.
   * @param hidden Include hidden options.
   */ getBaseOptions(hidden) {
        if (!this.options.length) {
            return [];
        }
        return hidden ? this.options.slice(0) : this.options.filter((opt)=>!opt.hidden);
    }
    /**
   * Get global options.
   * @param hidden Include hidden options.
   */ getGlobalOptions(hidden) {
        const helpOption = this.getHelpOption();
        const getGlobals = (cmd, noGlobals, options = [], names = [])=>{
            if (cmd.options.length) {
                for (const option of cmd.options){
                    if (option.global && !this.options.find((opt)=>opt.name === option.name) && names.indexOf(option.name) === -1 && (hidden || !option.hidden)) {
                        if (noGlobals && option !== helpOption) {
                            continue;
                        }
                        names.push(option.name);
                        options.push(option);
                    }
                }
            }
            return cmd._parent ? getGlobals(cmd._parent, noGlobals || cmd._noGlobals, options, names) : options;
        };
        return this._parent ? getGlobals(this._parent, this._noGlobals) : [];
    }
    /**
   * Checks whether the command has an option with given name or not.
   * @param name Name of the option. Must be in param-case.
   * @param hidden Include hidden options.
   */ hasOption(name, hidden) {
        return !!this.getOption(name, hidden);
    }
    /**
   * Get option by name.
   * @param name Name of the option. Must be in param-case.
   * @param hidden Include hidden options.
   */ getOption(name, hidden) {
        return this.getBaseOption(name, hidden) ?? this.getGlobalOption(name, hidden);
    }
    /**
   * Get base option by name.
   * @param name Name of the option. Must be in param-case.
   * @param hidden Include hidden options.
   */ getBaseOption(name, hidden) {
        const option = this.options.find((option)=>option.name === name || option.aliases?.includes(name));
        return option && (hidden || !option.hidden) ? option : undefined;
    }
    /**
   * Get global option from parent commands by name.
   * @param name Name of the option. Must be in param-case.
   * @param hidden Include hidden options.
   */ getGlobalOption(name, hidden) {
        const helpOption = this.getHelpOption();
        const getGlobalOption = (parent, noGlobals)=>{
            const option = parent.getBaseOption(name, hidden);
            if (!option?.global) {
                return parent._parent && getGlobalOption(parent._parent, noGlobals || parent._noGlobals);
            }
            if (noGlobals && option !== helpOption) {
                return;
            }
            return option;
        };
        return this._parent && getGlobalOption(this._parent, this._noGlobals);
    }
    /**
   * Remove option by name.
   * @param name Name of the option. Must be in param-case.
   */ removeOption(name) {
        const index = this.options.findIndex((option)=>option.name === name);
        if (index === -1) {
            return;
        }
        return this.options.splice(index, 1)[0];
    }
    /**
   * Checks whether the command has sub-commands or not.
   * @param hidden Include hidden commands.
   */ hasCommands(hidden) {
        return this.getCommands(hidden).length > 0;
    }
    /**
   * Get commands.
   * @param hidden Include hidden commands.
   */ getCommands(hidden) {
        return this.getGlobalCommands(hidden).concat(this.getBaseCommands(hidden));
    }
    /**
   * Get base commands.
   * @param hidden Include hidden commands.
   */ getBaseCommands(hidden) {
        const commands = Array.from(this.commands.values());
        return hidden ? commands : commands.filter((cmd)=>!cmd.isHidden);
    }
    /**
   * Get global commands.
   * @param hidden Include hidden commands.
   */ getGlobalCommands(hidden) {
        const getCommands = (command, noGlobals, commands = [], names = [])=>{
            if (command.commands.size) {
                for (const [_, cmd] of command.commands){
                    if (cmd.isGlobal && this !== cmd && !this.commands.has(cmd._name) && names.indexOf(cmd._name) === -1 && (hidden || !cmd.isHidden)) {
                        if (noGlobals && cmd?.getName() !== "help") {
                            continue;
                        }
                        names.push(cmd._name);
                        commands.push(cmd);
                    }
                }
            }
            return command._parent ? getCommands(command._parent, noGlobals || command._noGlobals, commands, names) : commands;
        };
        return this._parent ? getCommands(this._parent, this._noGlobals) : [];
    }
    /**
   * Checks whether a child command exists by given name or alias.
   * @param name Name or alias of the command.
   * @param hidden Include hidden commands.
   */ hasCommand(name, hidden) {
        return !!this.getCommand(name, hidden);
    }
    /**
   * Get command by name or alias.
   * @param name Name or alias of the command.
   * @param hidden Include hidden commands.
   */ getCommand(name, hidden) {
        return this.getBaseCommand(name, hidden) ?? this.getGlobalCommand(name, hidden);
    }
    /**
   * Get base command by name or alias.
   * @param name Name or alias of the command.
   * @param hidden Include hidden commands.
   */ getBaseCommand(name, hidden) {
        for (const cmd of this.commands.values()){
            if (cmd._name === name || cmd.aliases.includes(name)) {
                return cmd && (hidden || !cmd.isHidden) ? cmd : undefined;
            }
        }
    }
    /**
   * Get global command by name or alias.
   * @param name Name or alias of the command.
   * @param hidden Include hidden commands.
   */ getGlobalCommand(name, hidden) {
        const getGlobalCommand = (parent, noGlobals)=>{
            const cmd = parent.getBaseCommand(name, hidden);
            if (!cmd?.isGlobal) {
                return parent._parent && getGlobalCommand(parent._parent, noGlobals || parent._noGlobals);
            }
            if (noGlobals && cmd.getName() !== "help") {
                return;
            }
            return cmd;
        };
        return this._parent && getGlobalCommand(this._parent, this._noGlobals);
    }
    /**
   * Remove sub-command by name or alias.
   * @param name Name or alias of the command.
   */ removeCommand(name) {
        const command = this.getBaseCommand(name, true);
        if (command) {
            this.commands.delete(command._name);
        }
        return command;
    }
    /** Get types. */ getTypes() {
        return this.getGlobalTypes().concat(this.getBaseTypes());
    }
    /** Get base types. */ getBaseTypes() {
        return Array.from(this.types.values());
    }
    /** Get global types. */ getGlobalTypes() {
        const getTypes = (cmd, types = [], names = [])=>{
            if (cmd) {
                if (cmd.types.size) {
                    cmd.types.forEach((type)=>{
                        if (type.global && !this.types.has(type.name) && names.indexOf(type.name) === -1) {
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
    /**
   * Get type by name.
   * @param name Name of the type.
   */ getType(name) {
        return this.getBaseType(name) ?? this.getGlobalType(name);
    }
    /**
   * Get base type by name.
   * @param name Name of the type.
   */ getBaseType(name) {
        return this.types.get(name);
    }
    /**
   * Get global type by name.
   * @param name Name of the type.
   */ getGlobalType(name) {
        if (!this._parent) {
            return;
        }
        const cmd = this._parent.getBaseType(name);
        if (!cmd?.global) {
            return this._parent.getGlobalType(name);
        }
        return cmd;
    }
    /** Get completions. */ getCompletions() {
        return this.getGlobalCompletions().concat(this.getBaseCompletions());
    }
    /** Get base completions. */ getBaseCompletions() {
        return Array.from(this.completions.values());
    }
    /** Get global completions. */ getGlobalCompletions() {
        const getCompletions = (cmd, completions = [], names = [])=>{
            if (cmd) {
                if (cmd.completions.size) {
                    cmd.completions.forEach((completion)=>{
                        if (completion.global && !this.completions.has(completion.name) && names.indexOf(completion.name) === -1) {
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
    /**
   * Get completion by name.
   * @param name Name of the completion.
   */ getCompletion(name) {
        return this.getBaseCompletion(name) ?? this.getGlobalCompletion(name);
    }
    /**
   * Get base completion by name.
   * @param name Name of the completion.
   */ getBaseCompletion(name) {
        return this.completions.get(name);
    }
    /**
   * Get global completions by name.
   * @param name Name of the completion.
   */ getGlobalCompletion(name) {
        if (!this._parent) {
            return;
        }
        const completion = this._parent.getBaseCompletion(name);
        if (!completion?.global) {
            return this._parent.getGlobalCompletion(name);
        }
        return completion;
    }
    /**
   * Checks whether the command has environment variables or not.
   * @param hidden Include hidden environment variable.
   */ hasEnvVars(hidden) {
        return this.getEnvVars(hidden).length > 0;
    }
    /**
   * Get environment variables.
   * @param hidden Include hidden environment variable.
   */ getEnvVars(hidden) {
        return this.getGlobalEnvVars(hidden).concat(this.getBaseEnvVars(hidden));
    }
    /**
   * Get base environment variables.
   * @param hidden Include hidden environment variable.
   */ getBaseEnvVars(hidden) {
        if (!this.envVars.length) {
            return [];
        }
        return hidden ? this.envVars.slice(0) : this.envVars.filter((env)=>!env.hidden);
    }
    /**
   * Get global environment variables.
   * @param hidden Include hidden environment variable.
   */ getGlobalEnvVars(hidden) {
        if (this._noGlobals) {
            return [];
        }
        const getEnvVars = (cmd, envVars = [], names = [])=>{
            if (cmd) {
                if (cmd.envVars.length) {
                    cmd.envVars.forEach((envVar)=>{
                        if (envVar.global && !this.envVars.find((env)=>env.names[0] === envVar.names[0]) && names.indexOf(envVar.names[0]) === -1 && (hidden || !envVar.hidden)) {
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
    /**
   * Checks whether the command has an environment variable with given name or not.
   * @param name Name of the environment variable.
   * @param hidden Include hidden environment variable.
   */ hasEnvVar(name, hidden) {
        return !!this.getEnvVar(name, hidden);
    }
    /**
   * Get environment variable by name.
   * @param name Name of the environment variable.
   * @param hidden Include hidden environment variable.
   */ getEnvVar(name, hidden) {
        return this.getBaseEnvVar(name, hidden) ?? this.getGlobalEnvVar(name, hidden);
    }
    /**
   * Get base environment variable by name.
   * @param name Name of the environment variable.
   * @param hidden Include hidden environment variable.
   */ getBaseEnvVar(name, hidden) {
        const envVar = this.envVars.find((env)=>env.names.indexOf(name) !== -1);
        return envVar && (hidden || !envVar.hidden) ? envVar : undefined;
    }
    /**
   * Get global environment variable by name.
   * @param name Name of the environment variable.
   * @param hidden Include hidden environment variable.
   */ getGlobalEnvVar(name, hidden) {
        if (!this._parent || this._noGlobals) {
            return;
        }
        const envVar = this._parent.getBaseEnvVar(name, hidden);
        if (!envVar?.global) {
            return this._parent.getGlobalEnvVar(name, hidden);
        }
        return envVar;
    }
    /** Checks whether the command has examples or not. */ hasExamples() {
        return this.examples.length > 0;
    }
    /** Get all examples. */ getExamples() {
        return this.examples;
    }
    /** Checks whether the command has an example with given name or not. */ hasExample(name) {
        return !!this.getExample(name);
    }
    /** Get example with given name. */ getExample(name) {
        return this.examples.find((example)=>example.name === name);
    }
    getHelpOption() {
        return this._helpOption ?? this._parent?.getHelpOption();
    }
}
function isUpgradeCommand(command) {
    return command instanceof Command && "getLatestVersion" in command;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC9jb21tYW5kLnRzIl0sInNvdXJjZXNDb250ZW50IjpbIi8vIGRlbm8tbGludC1pZ25vcmUtZmlsZSBuby1leHBsaWNpdC1hbnlcbmltcG9ydCB7XG4gIFVua25vd25UeXBlLFxuICBWYWxpZGF0aW9uRXJyb3IgYXMgRmxhZ3NWYWxpZGF0aW9uRXJyb3IsXG59IGZyb20gXCIuLi9mbGFncy9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBNaXNzaW5nUmVxdWlyZWRFbnZWYXIgfSBmcm9tIFwiLi9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBwYXJzZUZsYWdzIH0gZnJvbSBcIi4uL2ZsYWdzL2ZsYWdzLnRzXCI7XG5pbXBvcnQgdHlwZSB7IElEZWZhdWx0VmFsdWUgfSBmcm9tIFwiLi4vZmxhZ3MvdHlwZXMudHNcIjtcbmltcG9ydCB7XG4gIGdldERlc2NyaXB0aW9uLFxuICBwYXJzZUFyZ3VtZW50c0RlZmluaXRpb24sXG4gIHNwbGl0QXJndW1lbnRzLFxufSBmcm9tIFwiLi9fdXRpbHMudHNcIjtcbmltcG9ydCB7IGJsdWUsIGJvbGQsIHJlZCwgeWVsbG93IH0gZnJvbSBcIi4vZGVwcy50c1wiO1xuaW1wb3J0IHtcbiAgQ29tbWFuZEV4ZWN1dGFibGVOb3RGb3VuZCxcbiAgQ29tbWFuZE5vdEZvdW5kLFxuICBEZWZhdWx0Q29tbWFuZE5vdEZvdW5kLFxuICBEdXBsaWNhdGVDb21tYW5kQWxpYXMsXG4gIER1cGxpY2F0ZUNvbW1hbmROYW1lLFxuICBEdXBsaWNhdGVDb21wbGV0aW9uLFxuICBEdXBsaWNhdGVFbnZpcm9ubWVudFZhcmlhYmxlLFxuICBEdXBsaWNhdGVFeGFtcGxlLFxuICBEdXBsaWNhdGVPcHRpb25OYW1lLFxuICBEdXBsaWNhdGVUeXBlLFxuICBFbnZpcm9ubWVudFZhcmlhYmxlT3B0aW9uYWxWYWx1ZSxcbiAgRW52aXJvbm1lbnRWYXJpYWJsZVNpbmdsZVZhbHVlLFxuICBFbnZpcm9ubWVudFZhcmlhYmxlVmFyaWFkaWNWYWx1ZSxcbiAgTWlzc2luZ0FyZ3VtZW50LFxuICBNaXNzaW5nQXJndW1lbnRzLFxuICBNaXNzaW5nQ29tbWFuZE5hbWUsXG4gIE5vQXJndW1lbnRzQWxsb3dlZCxcbiAgVG9vTWFueUFyZ3VtZW50cyxcbiAgVW5rbm93bkNvbW1hbmQsXG4gIFZhbGlkYXRpb25FcnJvcixcbn0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgQm9vbGVhblR5cGUgfSBmcm9tIFwiLi90eXBlcy9ib29sZWFuLnRzXCI7XG5pbXBvcnQgeyBGaWxlVHlwZSB9IGZyb20gXCIuL3R5cGVzL2ZpbGUudHNcIjtcbmltcG9ydCB7IE51bWJlclR5cGUgfSBmcm9tIFwiLi90eXBlcy9udW1iZXIudHNcIjtcbmltcG9ydCB7IFN0cmluZ1R5cGUgfSBmcm9tIFwiLi90eXBlcy9zdHJpbmcudHNcIjtcbmltcG9ydCB7IFR5cGUgfSBmcm9tIFwiLi90eXBlLnRzXCI7XG5pbXBvcnQgeyBIZWxwR2VuZXJhdG9yIH0gZnJvbSBcIi4vaGVscC9faGVscF9nZW5lcmF0b3IudHNcIjtcbmltcG9ydCB0eXBlIHsgSGVscE9wdGlvbnMgfSBmcm9tIFwiLi9oZWxwL19oZWxwX2dlbmVyYXRvci50c1wiO1xuaW1wb3J0IHR5cGUge1xuICBJQWN0aW9uLFxuICBJQXJndW1lbnQsXG4gIElDb21tYW5kR2xvYmFsT3B0aW9uLFxuICBJQ29tbWFuZE9wdGlvbixcbiAgSUNvbXBsZXRlSGFuZGxlcixcbiAgSUNvbXBsZXRlT3B0aW9ucyxcbiAgSUNvbXBsZXRpb24sXG4gIElEZXNjcmlwdGlvbixcbiAgSUVudlZhcixcbiAgSUVudlZhck9wdGlvbnMsXG4gIElFbnZWYXJWYWx1ZUhhbmRsZXIsXG4gIElFeGFtcGxlLFxuICBJRmxhZ1ZhbHVlSGFuZGxlcixcbiAgSUdsb2JhbEVudlZhck9wdGlvbnMsXG4gIElIZWxwSGFuZGxlcixcbiAgSU9wdGlvbixcbiAgSVBhcnNlUmVzdWx0LFxuICBJVHlwZSxcbiAgSVR5cGVJbmZvLFxuICBJVHlwZU9wdGlvbnMsXG4gIElWZXJzaW9uSGFuZGxlcixcbiAgTWFwVHlwZXMsXG4gIFR5cGVPclR5cGVIYW5kbGVyLFxufSBmcm9tIFwiLi90eXBlcy50c1wiO1xuaW1wb3J0IHsgSW50ZWdlclR5cGUgfSBmcm9tIFwiLi90eXBlcy9pbnRlZ2VyLnRzXCI7XG5pbXBvcnQgeyB1bmRlcnNjb3JlVG9DYW1lbENhc2UgfSBmcm9tIFwiLi4vZmxhZ3MvX3V0aWxzLnRzXCI7XG5cbmV4cG9ydCBjbGFzcyBDb21tYW5kPFxuICBDUEcgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQgPSB2b2lkLFxuICBDUFQgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQgPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiB2b2lkLFxuICBDTyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHZvaWQsXG4gIENBIGV4dGVuZHMgQXJyYXk8dW5rbm93bj4gPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiBbXSxcbiAgQ0cgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQgPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiB2b2lkLFxuICBDVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHtcbiAgICBudW1iZXI6IG51bWJlcjtcbiAgICBpbnRlZ2VyOiBudW1iZXI7XG4gICAgc3RyaW5nOiBzdHJpbmc7XG4gICAgYm9vbGVhbjogYm9vbGVhbjtcbiAgICBmaWxlOiBzdHJpbmc7XG4gIH0sXG4gIENHVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHZvaWQsXG4gIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdW5kZWZpbmVkLFxuPiB7XG4gIHByaXZhdGUgdHlwZXM6IE1hcDxzdHJpbmcsIElUeXBlPiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSByYXdBcmdzOiBBcnJheTxzdHJpbmc+ID0gW107XG4gIHByaXZhdGUgbGl0ZXJhbEFyZ3M6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgLy8gQFRPRE86IGdldCBzY3JpcHQgbmFtZTogaHR0cHM6Ly9naXRodWIuY29tL2Rlbm9sYW5kL2Rlbm8vcHVsbC81MDM0XG4gIC8vIHByaXZhdGUgbmFtZTogc3RyaW5nID0gbG9jYXRpb24ucGF0aG5hbWUuc3BsaXQoICcvJyApLnBvcCgpIGFzIHN0cmluZztcbiAgcHJpdmF0ZSBfbmFtZSA9IFwiQ09NTUFORFwiO1xuICBwcml2YXRlIF9wYXJlbnQ/OiBDUDtcbiAgcHJpdmF0ZSBfZ2xvYmFsUGFyZW50PzogQ29tbWFuZDxhbnk+O1xuICBwcml2YXRlIHZlcj86IElWZXJzaW9uSGFuZGxlcjtcbiAgcHJpdmF0ZSBkZXNjOiBJRGVzY3JpcHRpb24gPSBcIlwiO1xuICBwcml2YXRlIF91c2FnZT86IHN0cmluZztcbiAgcHJpdmF0ZSBmbj86IElBY3Rpb247XG4gIHByaXZhdGUgb3B0aW9uczogQXJyYXk8SU9wdGlvbj4gPSBbXTtcbiAgcHJpdmF0ZSBjb21tYW5kczogTWFwPHN0cmluZywgQ29tbWFuZDxhbnk+PiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSBleGFtcGxlczogQXJyYXk8SUV4YW1wbGU+ID0gW107XG4gIHByaXZhdGUgZW52VmFyczogQXJyYXk8SUVudlZhcj4gPSBbXTtcbiAgcHJpdmF0ZSBhbGlhc2VzOiBBcnJheTxzdHJpbmc+ID0gW107XG4gIHByaXZhdGUgY29tcGxldGlvbnM6IE1hcDxzdHJpbmcsIElDb21wbGV0aW9uPiA9IG5ldyBNYXAoKTtcbiAgcHJpdmF0ZSBjbWQ6IENvbW1hbmQ8YW55PiA9IHRoaXM7XG4gIHByaXZhdGUgYXJnc0RlZmluaXRpb24/OiBzdHJpbmc7XG4gIHByaXZhdGUgaXNFeGVjdXRhYmxlID0gZmFsc2U7XG4gIHByaXZhdGUgdGhyb3dPbkVycm9yID0gZmFsc2U7XG4gIHByaXZhdGUgX2FsbG93RW1wdHkgPSBmYWxzZTtcbiAgcHJpdmF0ZSBfc3RvcEVhcmx5ID0gZmFsc2U7XG4gIHByaXZhdGUgZGVmYXVsdENvbW1hbmQ/OiBzdHJpbmc7XG4gIHByaXZhdGUgX3VzZVJhd0FyZ3MgPSBmYWxzZTtcbiAgcHJpdmF0ZSBhcmdzOiBBcnJheTxJQXJndW1lbnQ+ID0gW107XG4gIHByaXZhdGUgaXNIaWRkZW4gPSBmYWxzZTtcbiAgcHJpdmF0ZSBpc0dsb2JhbCA9IGZhbHNlO1xuICBwcml2YXRlIGhhc0RlZmF1bHRzID0gZmFsc2U7XG4gIHByaXZhdGUgX3ZlcnNpb25PcHRpb25zPzogSURlZmF1bHRPcHRpb24gfCBmYWxzZTtcbiAgcHJpdmF0ZSBfaGVscE9wdGlvbnM/OiBJRGVmYXVsdE9wdGlvbiB8IGZhbHNlO1xuICBwcml2YXRlIF92ZXJzaW9uT3B0aW9uPzogSU9wdGlvbjtcbiAgcHJpdmF0ZSBfaGVscE9wdGlvbj86IElPcHRpb247XG4gIHByaXZhdGUgX2hlbHA/OiBJSGVscEhhbmRsZXI7XG4gIHByaXZhdGUgX3Nob3VsZEV4aXQ/OiBib29sZWFuO1xuICBwcml2YXRlIF9tZXRhOiBSZWNvcmQ8c3RyaW5nLCBzdHJpbmc+ID0ge307XG4gIHByaXZhdGUgX2dyb3VwTmFtZT86IHN0cmluZztcbiAgcHJpdmF0ZSBfbm9HbG9iYWxzID0gZmFsc2U7XG5cbiAgLyoqIERpc2FibGUgdmVyc2lvbiBvcHRpb24uICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKGVuYWJsZTogZmFsc2UpOiB0aGlzO1xuXG4gIC8qKlxuICAgKiBTZXQgZ2xvYmFsIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBWZXJzaW9uIG9wdGlvbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIHZlcnNpb25PcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbjxQYXJ0aWFsPENPPiwgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+ICYge1xuICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgIH0sXG4gICk6IHRoaXM7XG5cbiAgLyoqXG4gICAqIFNldCB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGZsYWdzIFRoZSBmbGFncyBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyAgVmVyc2lvbiBvcHRpb24gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUNvbW1hbmRPcHRpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgKTogdGhpcztcblxuICAvKipcbiAgICogU2V0IHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBUaGUgYWN0aW9uIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcgfCBmYWxzZSxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OlxuICAgICAgfCBJQWN0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD5cbiAgICAgIHwgSUNvbW1hbmRPcHRpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPlxuICAgICAgfCBJQ29tbWFuZE9wdGlvbjxQYXJ0aWFsPENPPiwgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+ICYge1xuICAgICAgICBnbG9iYWw6IHRydWU7XG4gICAgICB9LFxuICApOiB0aGlzIHtcbiAgICB0aGlzLl92ZXJzaW9uT3B0aW9ucyA9IGZsYWdzID09PSBmYWxzZSA/IGZsYWdzIDoge1xuICAgICAgZmxhZ3MsXG4gICAgICBkZXNjLFxuICAgICAgb3B0czogdHlwZW9mIG9wdHMgPT09IFwiZnVuY3Rpb25cIiA/IHsgYWN0aW9uOiBvcHRzIH0gOiBvcHRzLFxuICAgIH07XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogRGlzYWJsZSBoZWxwIG9wdGlvbi4gKi9cbiAgcHVibGljIGhlbHBPcHRpb24oZW5hYmxlOiBmYWxzZSk6IHRoaXM7XG5cbiAgLyoqXG4gICAqIFNldCBnbG9iYWwgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBUaGUgZmxhZ3Mgb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZGVzYyAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgIEhlbHAgb3B0aW9uIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgaGVscE9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86IElDb21tYW5kT3B0aW9uPFBhcnRpYWw8Q08+LCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4gJiB7XG4gICAgICBnbG9iYWw6IHRydWU7XG4gICAgfSxcbiAgKTogdGhpcztcblxuICAvKipcbiAgICogU2V0IGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBIZWxwIG9wdGlvbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuXG4gIC8qKlxuICAgKiBTZXQgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBUaGUgZmxhZ3Mgb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZGVzYyAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgIFRoZSBhY3Rpb24gb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKi9cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQWN0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICk6IHRoaXM7XG5cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyB8IGZhbHNlLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86XG4gICAgICB8IElBY3Rpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPlxuICAgICAgfCBJQ29tbWFuZE9wdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+XG4gICAgICB8IElDb21tYW5kT3B0aW9uPFBhcnRpYWw8Q08+LCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4gJiB7XG4gICAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICAgIH0sXG4gICk6IHRoaXMge1xuICAgIHRoaXMuX2hlbHBPcHRpb25zID0gZmxhZ3MgPT09IGZhbHNlID8gZmxhZ3MgOiB7XG4gICAgICBmbGFncyxcbiAgICAgIGRlc2MsXG4gICAgICBvcHRzOiB0eXBlb2Ygb3B0cyA9PT0gXCJmdW5jdGlvblwiID8geyBhY3Rpb246IG9wdHMgfSA6IG9wdHMsXG4gICAgfTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgbmV3IHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgIENvbW1hbmQgZGVmaW5pdGlvbi4gRS5nOiBgbXktY29tbWFuZCA8aW5wdXQtZmlsZTpzdHJpbmc+IDxvdXRwdXQtZmlsZTpzdHJpbmc+YFxuICAgKiBAcGFyYW0gY21kICAgICAgIFRoZSBuZXcgY2hpbGQgY29tbWFuZCB0byByZWdpc3Rlci5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyBjaGlsZCBjb21tYW5kLlxuICAgKi9cbiAgcHVibGljIGNvbW1hbmQ8XG4gICAgQyBleHRlbmRzIENvbW1hbmQ8XG4gICAgICAoRyAmIFJlY29yZDxzdHJpbmcsIHVua25vd24+KSB8IHZvaWQgfCB1bmRlZmluZWQsXG4gICAgICBUIHwgdm9pZCB8IHVuZGVmaW5lZCxcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCxcbiAgICAgIEFycmF5PHVua25vd24+LFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgQ29tbWFuZDxcbiAgICAgICAgRyB8IHZvaWQgfCB1bmRlZmluZWQsXG4gICAgICAgIFQgfCB2b2lkIHwgdW5kZWZpbmVkLFxuICAgICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQsXG4gICAgICAgIEFycmF5PHVua25vd24+LFxuICAgICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQsXG4gICAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCxcbiAgICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgICB1bmRlZmluZWRcbiAgICAgID5cbiAgICA+LFxuICAgIEcgZXh0ZW5kcyAoQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUEcgOiBNZXJnZTxDUEcsIENHPiksXG4gICAgVCBleHRlbmRzIChDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQVCA6IE1lcmdlPENQVCwgQ1Q+KSxcbiAgPihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY21kOiBDLFxuICAgIG92ZXJyaWRlPzogYm9vbGVhbixcbiAgKTogUmV0dXJuVHlwZTxDW1wicmVzZXRcIl0+IGV4dGVuZHMgQ29tbWFuZDxcbiAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQsXG4gICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgIGluZmVyIE9wdGlvbnMsXG4gICAgaW5mZXIgQXJndW1lbnRzLFxuICAgIGluZmVyIEdsb2JhbE9wdGlvbnMsXG4gICAgaW5mZXIgVHlwZXMsXG4gICAgaW5mZXIgR2xvYmFsVHlwZXMsXG4gICAgdW5kZWZpbmVkXG4gID4gPyBDb21tYW5kPFxuICAgICAgRyxcbiAgICAgIFQsXG4gICAgICBPcHRpb25zLFxuICAgICAgQXJndW1lbnRzLFxuICAgICAgR2xvYmFsT3B0aW9ucyxcbiAgICAgIFR5cGVzLFxuICAgICAgR2xvYmFsVHlwZXMsXG4gICAgICBPbmVPZjxDUCwgdGhpcz5cbiAgICA+XG4gICAgOiBuZXZlcjtcblxuICAvKipcbiAgICogQWRkIG5ldyBzdWItY29tbWFuZC5cbiAgICogQHBhcmFtIG5hbWUgICAgICBDb21tYW5kIGRlZmluaXRpb24uIEUuZzogYG15LWNvbW1hbmQgPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPmBcbiAgICogQHBhcmFtIGNtZCAgICAgICBUaGUgbmV3IGNoaWxkIGNvbW1hbmQgdG8gcmVnaXN0ZXIuXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgY2hpbGQgY29tbWFuZC5cbiAgICovXG4gIHB1YmxpYyBjb21tYW5kPFxuICAgIEMgZXh0ZW5kcyBDb21tYW5kPFxuICAgICAgRyB8IHZvaWQgfCB1bmRlZmluZWQsXG4gICAgICBUIHwgdm9pZCB8IHVuZGVmaW5lZCxcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCxcbiAgICAgIEFycmF5PHVua25vd24+LFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkLFxuICAgICAgT25lT2Y8Q1AsIHRoaXM+IHwgdW5kZWZpbmVkXG4gICAgPixcbiAgICBHIGV4dGVuZHMgKENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gQ1BHIDogTWVyZ2U8Q1BHLCBDRz4pLFxuICAgIFQgZXh0ZW5kcyAoQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUFQgOiBNZXJnZTxDUFQsIENUPiksXG4gID4oXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGNtZDogQyxcbiAgICBvdmVycmlkZT86IGJvb2xlYW4sXG4gICk6IEMgZXh0ZW5kcyBDb21tYW5kPFxuICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdm9pZCxcbiAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQsXG4gICAgaW5mZXIgT3B0aW9ucyxcbiAgICBpbmZlciBBcmd1bWVudHMsXG4gICAgaW5mZXIgR2xvYmFsT3B0aW9ucyxcbiAgICBpbmZlciBUeXBlcyxcbiAgICBpbmZlciBHbG9iYWxUeXBlcyxcbiAgICBPbmVPZjxDUCwgdGhpcz4gfCB1bmRlZmluZWRcbiAgPiA/IENvbW1hbmQ8XG4gICAgICBHLFxuICAgICAgVCxcbiAgICAgIE9wdGlvbnMsXG4gICAgICBBcmd1bWVudHMsXG4gICAgICBHbG9iYWxPcHRpb25zLFxuICAgICAgVHlwZXMsXG4gICAgICBHbG9iYWxUeXBlcyxcbiAgICAgIE9uZU9mPENQLCB0aGlzPlxuICAgID5cbiAgICA6IG5ldmVyO1xuXG4gIC8qKlxuICAgKiBBZGQgbmV3IHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgIENvbW1hbmQgZGVmaW5pdGlvbi4gRS5nOiBgbXktY29tbWFuZCA8aW5wdXQtZmlsZTpzdHJpbmc+IDxvdXRwdXQtZmlsZTpzdHJpbmc+YFxuICAgKiBAcGFyYW0gZGVzYyAgICAgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgbmV3IGNoaWxkIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgY2hpbGQgY29tbWFuZC5cbiAgICovXG4gIHB1YmxpYyBjb21tYW5kPFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcsXG4gICAgQSBleHRlbmRzIFR5cGVkQ29tbWFuZEFyZ3VtZW50czxcbiAgICAgIE4sXG4gICAgICBDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQVCA6IE1lcmdlPENQVCwgQ0dUPlxuICAgID4sXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG92ZXJyaWRlPzogYm9vbGVhbixcbiAgKTogQ1BHIGV4dGVuZHMgbnVtYmVyID8gQ29tbWFuZDxhbnk+IDogQ29tbWFuZDxcbiAgICBDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQRyA6IE1lcmdlPENQRywgQ0c+LFxuICAgIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gQ1BUIDogTWVyZ2U8Q1BULCBDR1Q+LFxuICAgIHZvaWQsXG4gICAgQSxcbiAgICB2b2lkLFxuICAgIHZvaWQsXG4gICAgdm9pZCxcbiAgICBPbmVPZjxDUCwgdGhpcz5cbiAgPjtcblxuICAvKipcbiAgICogQWRkIG5ldyBzdWItY29tbWFuZC5cbiAgICogQHBhcmFtIG5hbWVBbmRBcmd1bWVudHMgIENvbW1hbmQgZGVmaW5pdGlvbi4gRS5nOiBgbXktY29tbWFuZCA8aW5wdXQtZmlsZTpzdHJpbmc+IDxvdXRwdXQtZmlsZTpzdHJpbmc+YFxuICAgKiBAcGFyYW0gY21kT3JEZXNjcmlwdGlvbiAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBuZXcgY2hpbGQgY29tbWFuZC5cbiAgICogQHBhcmFtIG92ZXJyaWRlICAgICAgICAgIE92ZXJyaWRlIGV4aXN0aW5nIGNoaWxkIGNvbW1hbmQuXG4gICAqL1xuICBjb21tYW5kKFxuICAgIG5hbWVBbmRBcmd1bWVudHM6IHN0cmluZyxcbiAgICBjbWRPckRlc2NyaXB0aW9uPzogQ29tbWFuZDxhbnk+IHwgc3RyaW5nLFxuICAgIG92ZXJyaWRlPzogYm9vbGVhbixcbiAgKTogQ29tbWFuZDxhbnk+IHtcbiAgICB0aGlzLnJlc2V0KCk7XG5cbiAgICBjb25zdCByZXN1bHQgPSBzcGxpdEFyZ3VtZW50cyhuYW1lQW5kQXJndW1lbnRzKTtcblxuICAgIGNvbnN0IG5hbWU6IHN0cmluZyB8IHVuZGVmaW5lZCA9IHJlc3VsdC5mbGFncy5zaGlmdCgpO1xuICAgIGNvbnN0IGFsaWFzZXM6IHN0cmluZ1tdID0gcmVzdWx0LmZsYWdzO1xuXG4gICAgaWYgKCFuYW1lKSB7XG4gICAgICB0aHJvdyBuZXcgTWlzc2luZ0NvbW1hbmROYW1lKCk7XG4gICAgfVxuXG4gICAgaWYgKHRoaXMuZ2V0QmFzZUNvbW1hbmQobmFtZSwgdHJ1ZSkpIHtcbiAgICAgIGlmICghb3ZlcnJpZGUpIHtcbiAgICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZUNvbW1hbmROYW1lKG5hbWUpO1xuICAgICAgfVxuICAgICAgdGhpcy5yZW1vdmVDb21tYW5kKG5hbWUpO1xuICAgIH1cblxuICAgIGxldCBkZXNjcmlwdGlvbjogc3RyaW5nIHwgdW5kZWZpbmVkO1xuICAgIGxldCBjbWQ6IENvbW1hbmQ8YW55PjtcblxuICAgIGlmICh0eXBlb2YgY21kT3JEZXNjcmlwdGlvbiA9PT0gXCJzdHJpbmdcIikge1xuICAgICAgZGVzY3JpcHRpb24gPSBjbWRPckRlc2NyaXB0aW9uO1xuICAgIH1cblxuICAgIGlmIChjbWRPckRlc2NyaXB0aW9uIGluc3RhbmNlb2YgQ29tbWFuZCkge1xuICAgICAgY21kID0gY21kT3JEZXNjcmlwdGlvbi5yZXNldCgpO1xuICAgIH0gZWxzZSB7XG4gICAgICBjbWQgPSBuZXcgQ29tbWFuZCgpO1xuICAgIH1cblxuICAgIGNtZC5fbmFtZSA9IG5hbWU7XG4gICAgY21kLl9wYXJlbnQgPSB0aGlzO1xuXG4gICAgaWYgKGRlc2NyaXB0aW9uKSB7XG4gICAgICBjbWQuZGVzY3JpcHRpb24oZGVzY3JpcHRpb24pO1xuICAgIH1cblxuICAgIGlmIChyZXN1bHQudHlwZURlZmluaXRpb24pIHtcbiAgICAgIGNtZC5hcmd1bWVudHMocmVzdWx0LnR5cGVEZWZpbml0aW9uKTtcbiAgICB9XG5cbiAgICBhbGlhc2VzLmZvckVhY2goKGFsaWFzOiBzdHJpbmcpID0+IGNtZC5hbGlhcyhhbGlhcykpO1xuXG4gICAgdGhpcy5jb21tYW5kcy5zZXQobmFtZSwgY21kKTtcblxuICAgIHRoaXMuc2VsZWN0KG5hbWUpO1xuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogQWRkIG5ldyBjb21tYW5kIGFsaWFzLlxuICAgKiBAcGFyYW0gYWxpYXMgVGhhIG5hbWUgb2YgdGhlIGFsaWFzLlxuICAgKi9cbiAgcHVibGljIGFsaWFzKGFsaWFzOiBzdHJpbmcpOiB0aGlzIHtcbiAgICBpZiAodGhpcy5jbWQuX25hbWUgPT09IGFsaWFzIHx8IHRoaXMuY21kLmFsaWFzZXMuaW5jbHVkZXMoYWxpYXMpKSB7XG4gICAgICB0aHJvdyBuZXcgRHVwbGljYXRlQ29tbWFuZEFsaWFzKGFsaWFzKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZC5hbGlhc2VzLnB1c2goYWxpYXMpO1xuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogUmVzZXQgaW50ZXJuYWwgY29tbWFuZCByZWZlcmVuY2UgdG8gbWFpbiBjb21tYW5kLiAqL1xuICBwdWJsaWMgcmVzZXQoKTogT25lT2Y8Q1AsIHRoaXM+IHtcbiAgICB0aGlzLl9ncm91cE5hbWUgPSB1bmRlZmluZWQ7XG4gICAgdGhpcy5jbWQgPSB0aGlzO1xuICAgIHJldHVybiB0aGlzIGFzIE9uZU9mPENQLCB0aGlzPjtcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgaW50ZXJuYWwgY29tbWFuZCBwb2ludGVyIHRvIGNoaWxkIGNvbW1hbmQgd2l0aCBnaXZlbiBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBUaGUgbmFtZSBvZiB0aGUgY29tbWFuZCB0byBzZWxlY3QuXG4gICAqL1xuICBwdWJsaWMgc2VsZWN0PFxuICAgIE8gZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQgPSBhbnksXG4gICAgQSBleHRlbmRzIEFycmF5PHVua25vd24+ID0gYW55LFxuICAgIEcgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHZvaWQgPSBhbnksXG4gID4obmFtZTogc3RyaW5nKTogQ29tbWFuZDxDUEcsIENQVCwgTywgQSwgRywgQ1QsIENHVCwgQ1A+IHtcbiAgICBjb25zdCBjbWQgPSB0aGlzLmdldEJhc2VDb21tYW5kKG5hbWUsIHRydWUpO1xuXG4gICAgaWYgKCFjbWQpIHtcbiAgICAgIHRocm93IG5ldyBDb21tYW5kTm90Rm91bmQobmFtZSwgdGhpcy5nZXRCYXNlQ29tbWFuZHModHJ1ZSkpO1xuICAgIH1cblxuICAgIHRoaXMuY21kID0gY21kO1xuXG4gICAgcmV0dXJuIHRoaXMgYXMgQ29tbWFuZDxhbnk+O1xuICB9XG5cbiAgLyoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqIFNVQiBIQU5ETEVSICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKiovXG5cbiAgLyoqIFNldCBjb21tYW5kIG5hbWUuICovXG4gIHB1YmxpYyBuYW1lKG5hbWU6IHN0cmluZyk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9uYW1lID0gbmFtZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgY29tbWFuZCB2ZXJzaW9uLlxuICAgKiBAcGFyYW0gdmVyc2lvbiBTZW1hbnRpYyB2ZXJzaW9uIHN0cmluZyBzdHJpbmcgb3IgbWV0aG9kIHRoYXQgcmV0dXJucyB0aGUgdmVyc2lvbiBzdHJpbmcuXG4gICAqL1xuICBwdWJsaWMgdmVyc2lvbihcbiAgICB2ZXJzaW9uOlxuICAgICAgfCBzdHJpbmdcbiAgICAgIHwgSVZlcnNpb25IYW5kbGVyPFBhcnRpYWw8Q08+LCBQYXJ0aWFsPENBPiwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICk6IHRoaXMge1xuICAgIGlmICh0eXBlb2YgdmVyc2lvbiA9PT0gXCJzdHJpbmdcIikge1xuICAgICAgdGhpcy5jbWQudmVyID0gKCkgPT4gdmVyc2lvbjtcbiAgICB9IGVsc2UgaWYgKHR5cGVvZiB2ZXJzaW9uID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHRoaXMuY21kLnZlciA9IHZlcnNpb247XG4gICAgfVxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgcHVibGljIG1ldGEobmFtZTogc3RyaW5nLCB2YWx1ZTogc3RyaW5nKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX21ldGFbbmFtZV0gPSB2YWx1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIHB1YmxpYyBnZXRNZXRhKCk6IFJlY29yZDxzdHJpbmcsIHN0cmluZz47XG4gIHB1YmxpYyBnZXRNZXRhKG5hbWU6IHN0cmluZyk6IHN0cmluZztcbiAgcHVibGljIGdldE1ldGEobmFtZT86IHN0cmluZyk6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4gfCBzdHJpbmcge1xuICAgIHJldHVybiB0eXBlb2YgbmFtZSA9PT0gXCJ1bmRlZmluZWRcIiA/IHRoaXMuX21ldGEgOiB0aGlzLl9tZXRhW25hbWVdO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjb21tYW5kIGhlbHAuXG4gICAqIEBwYXJhbSBoZWxwIEhlbHAgc3RyaW5nLCBtZXRob2QsIG9yIGNvbmZpZyBmb3IgZ2VuZXJhdG9yIHRoYXQgcmV0dXJucyB0aGUgaGVscCBzdHJpbmcuXG4gICAqL1xuICBwdWJsaWMgaGVscChcbiAgICBoZWxwOlxuICAgICAgfCBzdHJpbmdcbiAgICAgIHwgSUhlbHBIYW5kbGVyPFBhcnRpYWw8Q08+LCBQYXJ0aWFsPENBPiwgQ0csIENQRz5cbiAgICAgIHwgSGVscE9wdGlvbnMsXG4gICk6IHRoaXMge1xuICAgIGlmICh0eXBlb2YgaGVscCA9PT0gXCJzdHJpbmdcIikge1xuICAgICAgdGhpcy5jbWQuX2hlbHAgPSAoKSA9PiBoZWxwO1xuICAgIH0gZWxzZSBpZiAodHlwZW9mIGhlbHAgPT09IFwiZnVuY3Rpb25cIikge1xuICAgICAgdGhpcy5jbWQuX2hlbHAgPSBoZWxwO1xuICAgIH0gZWxzZSB7XG4gICAgICB0aGlzLmNtZC5faGVscCA9IChjbWQ6IENvbW1hbmQsIG9wdGlvbnM6IEhlbHBPcHRpb25zKTogc3RyaW5nID0+XG4gICAgICAgIEhlbHBHZW5lcmF0b3IuZ2VuZXJhdGUoY21kLCB7IC4uLmhlbHAsIC4uLm9wdGlvbnMgfSk7XG4gICAgfVxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCB0aGUgbG9uZyBjb21tYW5kIGRlc2NyaXB0aW9uLlxuICAgKiBAcGFyYW0gZGVzY3JpcHRpb24gVGhlIGNvbW1hbmQgZGVzY3JpcHRpb24uXG4gICAqL1xuICBwdWJsaWMgZGVzY3JpcHRpb24oXG4gICAgZGVzY3JpcHRpb246IElEZXNjcmlwdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5kZXNjID0gZGVzY3JpcHRpb247XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IHRoZSBjb21tYW5kIHVzYWdlLiBEZWZhdWx0cyB0byBhcmd1bWVudHMuXG4gICAqIEBwYXJhbSB1c2FnZSBUaGUgY29tbWFuZCB1c2FnZS5cbiAgICovXG4gIHB1YmxpYyB1c2FnZSh1c2FnZTogc3RyaW5nKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX3VzYWdlID0gdXNhZ2U7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogSGlkZSBjb21tYW5kIGZyb20gaGVscCwgY29tcGxldGlvbnMsIGV0Yy5cbiAgICovXG4gIHB1YmxpYyBoaWRkZW4oKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuaXNIaWRkZW4gPSB0cnVlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqIE1ha2UgY29tbWFuZCBnbG9iYWxseSBhdmFpbGFibGUuICovXG4gIHB1YmxpYyBnbG9iYWwoKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuaXNHbG9iYWwgPSB0cnVlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqIE1ha2UgY29tbWFuZCBleGVjdXRhYmxlLiAqL1xuICBwdWJsaWMgZXhlY3V0YWJsZSgpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5pc0V4ZWN1dGFibGUgPSB0cnVlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjb21tYW5kIGFyZ3VtZW50czpcbiAgICpcbiAgICogICA8cmVxdWlyZWRBcmc6c3RyaW5nPiBbb3B0aW9uYWxBcmc6IG51bWJlcl0gWy4uLnJlc3RBcmdzOnN0cmluZ11cbiAgICovXG4gIHB1YmxpYyBhcmd1bWVudHM8XG4gICAgQSBleHRlbmRzIFR5cGVkQXJndW1lbnRzPE4sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+PixcbiAgICBOIGV4dGVuZHMgc3RyaW5nID0gc3RyaW5nLFxuICA+KFxuICAgIGFyZ3M6IE4sXG4gICk6IENvbW1hbmQ8Q1BHLCBDUFQsIENPLCBBLCBDRywgQ1QsIENHVCwgQ1A+IHtcbiAgICB0aGlzLmNtZC5hcmdzRGVmaW5pdGlvbiA9IGFyZ3M7XG4gICAgcmV0dXJuIHRoaXMgYXMgQ29tbWFuZDxhbnk+O1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjb21tYW5kIGNhbGxiYWNrIG1ldGhvZC5cbiAgICogQHBhcmFtIGZuIENvbW1hbmQgYWN0aW9uIGhhbmRsZXIuXG4gICAqL1xuICBwdWJsaWMgYWN0aW9uKGZuOiBJQWN0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4pOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5mbiA9IGZuO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIERvbid0IHRocm93IGFuIGVycm9yIGlmIHRoZSBjb21tYW5kIHdhcyBjYWxsZWQgd2l0aG91dCBhcmd1bWVudHMuXG4gICAqIEBwYXJhbSBhbGxvd0VtcHR5IEVuYWJsZS9kaXNhYmxlIGFsbG93IGVtcHR5LlxuICAgKi9cbiAgcHVibGljIGFsbG93RW1wdHk8VCBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQ+KFxuICAgIGFsbG93RW1wdHk/OiBULFxuICApOiBmYWxzZSBleHRlbmRzIFQgPyB0aGlzXG4gICAgOiBDb21tYW5kPFBhcnRpYWw8Q1BHPiwgQ1BULCBQYXJ0aWFsPENPPiwgQ0EsIENHLCBDVCwgQ0dULCBDUD4ge1xuICAgIHRoaXMuY21kLl9hbGxvd0VtcHR5ID0gYWxsb3dFbXB0eSAhPT0gZmFsc2U7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogRW5hYmxlIHN0b3AgZWFybHkuIElmIGVuYWJsZWQsIGFsbCBhcmd1bWVudHMgc3RhcnRpbmcgZnJvbSB0aGUgZmlyc3Qgbm9uXG4gICAqIG9wdGlvbiBhcmd1bWVudCB3aWxsIGJlIHBhc3NlZCBhcyBhcmd1bWVudHMgd2l0aCB0eXBlIHN0cmluZyB0byB0aGUgY29tbWFuZFxuICAgKiBhY3Rpb24gaGFuZGxlci5cbiAgICpcbiAgICogRm9yIGV4YW1wbGU6XG4gICAqICAgICBgY29tbWFuZCAtLWRlYnVnLWxldmVsIHdhcm5pbmcgc2VydmVyIC0tcG9ydCA4MGBcbiAgICpcbiAgICogV2lsbCByZXN1bHQgaW46XG4gICAqICAgICAtIG9wdGlvbnM6IGB7ZGVidWdMZXZlbDogJ3dhcm5pbmcnfWBcbiAgICogICAgIC0gYXJnczogYFsnc2VydmVyJywgJy0tcG9ydCcsICc4MCddYFxuICAgKlxuICAgKiBAcGFyYW0gc3RvcEVhcmx5IEVuYWJsZS9kaXNhYmxlIHN0b3AgZWFybHkuXG4gICAqL1xuICBwdWJsaWMgc3RvcEVhcmx5KHN0b3BFYXJseSA9IHRydWUpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fc3RvcEVhcmx5ID0gc3RvcEVhcmx5O1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIERpc2FibGUgcGFyc2luZyBhcmd1bWVudHMuIElmIGVuYWJsZWQgdGhlIHJhdyBhcmd1bWVudHMgd2lsbCBiZSBwYXNzZWQgdG9cbiAgICogdGhlIGFjdGlvbiBoYW5kbGVyLiBUaGlzIGhhcyBubyBlZmZlY3QgZm9yIHBhcmVudCBvciBjaGlsZCBjb21tYW5kcy4gT25seVxuICAgKiBmb3IgdGhlIGNvbW1hbmQgb24gd2hpY2ggdGhpcyBtZXRob2Qgd2FzIGNhbGxlZC5cbiAgICogQHBhcmFtIHVzZVJhd0FyZ3MgRW5hYmxlL2Rpc2FibGUgcmF3IGFyZ3VtZW50cy5cbiAgICovXG4gIHB1YmxpYyB1c2VSYXdBcmdzKFxuICAgIHVzZVJhd0FyZ3MgPSB0cnVlLFxuICApOiBDb21tYW5kPHZvaWQsIHZvaWQsIHZvaWQsIEFycmF5PHN0cmluZz4sIHZvaWQsIHZvaWQsIHZvaWQsIENQPiB7XG4gICAgdGhpcy5jbWQuX3VzZVJhd0FyZ3MgPSB1c2VSYXdBcmdzO1xuICAgIHJldHVybiB0aGlzIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgZGVmYXVsdCBjb21tYW5kLiBUaGUgZGVmYXVsdCBjb21tYW5kIGlzIGV4ZWN1dGVkIHdoZW4gdGhlIHByb2dyYW1cbiAgICogd2FzIGNhbGxlZCB3aXRob3V0IGFueSBhcmd1bWVudCBhbmQgaWYgbm8gYWN0aW9uIGhhbmRsZXIgaXMgcmVnaXN0ZXJlZC5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZGVmYXVsdCBjb21tYW5kLlxuICAgKi9cbiAgcHVibGljIGRlZmF1bHQobmFtZTogc3RyaW5nKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuZGVmYXVsdENvbW1hbmQgPSBuYW1lO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgcHVibGljIGdsb2JhbFR5cGU8XG4gICAgSCBleHRlbmRzIFR5cGVPclR5cGVIYW5kbGVyPHVua25vd24+LFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcgPSBzdHJpbmcsXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBoYW5kbGVyOiBILFxuICAgIG9wdGlvbnM/OiBPbWl0PElUeXBlT3B0aW9ucywgXCJnbG9iYWxcIj4sXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBDTyxcbiAgICBDQSxcbiAgICBDRyxcbiAgICBDVCxcbiAgICBNZXJnZTxDR1QsIFR5cGVkVHlwZTxOLCBIPj4sXG4gICAgQ1BcbiAgPiB7XG4gICAgcmV0dXJuIHRoaXMudHlwZShuYW1lLCBoYW5kbGVyLCB7IC4uLm9wdGlvbnMsIGdsb2JhbDogdHJ1ZSB9KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZWdpc3RlciBjdXN0b20gdHlwZS5cbiAgICogQHBhcmFtIG5hbWUgICAgVGhlIG5hbWUgb2YgdGhlIHR5cGUuXG4gICAqIEBwYXJhbSBoYW5kbGVyIFRoZSBjYWxsYmFjayBtZXRob2QgdG8gcGFyc2UgdGhlIHR5cGUuXG4gICAqIEBwYXJhbSBvcHRpb25zIFR5cGUgb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyB0eXBlPFxuICAgIEggZXh0ZW5kcyBUeXBlT3JUeXBlSGFuZGxlcjx1bmtub3duPixcbiAgICBOIGV4dGVuZHMgc3RyaW5nID0gc3RyaW5nLFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgaGFuZGxlcjogSCxcbiAgICBvcHRpb25zPzogSVR5cGVPcHRpb25zLFxuICApOiBDb21tYW5kPFxuICAgIENQRyxcbiAgICBDUFQsXG4gICAgQ08sXG4gICAgQ0EsXG4gICAgQ0csXG4gICAgTWVyZ2U8Q1QsIFR5cGVkVHlwZTxOLCBIPj4sXG4gICAgQ0dULFxuICAgIENQXG4gID4ge1xuICAgIGlmICh0aGlzLmNtZC50eXBlcy5nZXQobmFtZSkgJiYgIW9wdGlvbnM/Lm92ZXJyaWRlKSB7XG4gICAgICB0aHJvdyBuZXcgRHVwbGljYXRlVHlwZShuYW1lKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZC50eXBlcy5zZXQobmFtZSwgeyAuLi5vcHRpb25zLCBuYW1lLCBoYW5kbGVyIH0pO1xuXG4gICAgaWYgKFxuICAgICAgaGFuZGxlciBpbnN0YW5jZW9mIFR5cGUgJiZcbiAgICAgICh0eXBlb2YgaGFuZGxlci5jb21wbGV0ZSAhPT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICB0eXBlb2YgaGFuZGxlci52YWx1ZXMgIT09IFwidW5kZWZpbmVkXCIpXG4gICAgKSB7XG4gICAgICBjb25zdCBjb21wbGV0ZUhhbmRsZXI6IElDb21wbGV0ZUhhbmRsZXIgPSAoXG4gICAgICAgIGNtZDogQ29tbWFuZCxcbiAgICAgICAgcGFyZW50PzogQ29tbWFuZCxcbiAgICAgICkgPT4gaGFuZGxlci5jb21wbGV0ZT8uKGNtZCwgcGFyZW50KSB8fCBbXTtcbiAgICAgIHRoaXMuY29tcGxldGUobmFtZSwgY29tcGxldGVIYW5kbGVyLCBvcHRpb25zKTtcbiAgICB9XG5cbiAgICByZXR1cm4gdGhpcyBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICBwdWJsaWMgZ2xvYmFsQ29tcGxldGUoXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGNvbXBsZXRlOiBJQ29tcGxldGVIYW5kbGVyLFxuICAgIG9wdGlvbnM/OiBPbWl0PElDb21wbGV0ZU9wdGlvbnMsIFwiZ2xvYmFsXCI+LFxuICApOiB0aGlzIHtcbiAgICByZXR1cm4gdGhpcy5jb21wbGV0ZShuYW1lLCBjb21wbGV0ZSwgeyAuLi5vcHRpb25zLCBnbG9iYWw6IHRydWUgfSk7XG4gIH1cblxuICAvKipcbiAgICogUmVnaXN0ZXIgY29tbWFuZCBzcGVjaWZpYyBjdXN0b20gdHlwZS5cbiAgICogQHBhcmFtIG5hbWUgICAgICBUaGUgbmFtZSBvZiB0aGUgY29tcGxldGlvbi5cbiAgICogQHBhcmFtIGNvbXBsZXRlICBUaGUgY2FsbGJhY2sgbWV0aG9kIHRvIGNvbXBsZXRlIHRoZSB0eXBlLlxuICAgKiBAcGFyYW0gb3B0aW9ucyAgIENvbXBsZXRlIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgY29tcGxldGUoXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGNvbXBsZXRlOiBJQ29tcGxldGVIYW5kbGVyPFxuICAgICAgUGFydGlhbDxDTz4sXG4gICAgICBQYXJ0aWFsPENBPixcbiAgICAgIENHLFxuICAgICAgQ1BHLFxuICAgICAgQ1QsXG4gICAgICBDR1QsXG4gICAgICBDUFQsXG4gICAgICBhbnlcbiAgICA+LFxuICAgIG9wdGlvbnM6IElDb21wbGV0ZU9wdGlvbnMgJiB7IGdsb2JhbDogYm9vbGVhbiB9LFxuICApOiB0aGlzO1xuICBwdWJsaWMgY29tcGxldGUoXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGNvbXBsZXRlOiBJQ29tcGxldGVIYW5kbGVyPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICAgb3B0aW9ucz86IElDb21wbGV0ZU9wdGlvbnMsXG4gICk6IHRoaXM7XG5cbiAgcHVibGljIGNvbXBsZXRlKFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBjb21wbGV0ZTpcbiAgICAgIHwgSUNvbXBsZXRlSGFuZGxlcjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+XG4gICAgICB8IElDb21wbGV0ZUhhbmRsZXI8XG4gICAgICAgIFBhcnRpYWw8Q08+LFxuICAgICAgICBQYXJ0aWFsPENBPixcbiAgICAgICAgQ0csXG4gICAgICAgIENQRyxcbiAgICAgICAgQ1QsXG4gICAgICAgIENHVCxcbiAgICAgICAgQ1BULFxuICAgICAgICBhbnlcbiAgICAgID4sXG4gICAgb3B0aW9ucz86IElDb21wbGV0ZU9wdGlvbnMsXG4gICk6IHRoaXMge1xuICAgIGlmICh0aGlzLmNtZC5jb21wbGV0aW9ucy5oYXMobmFtZSkgJiYgIW9wdGlvbnM/Lm92ZXJyaWRlKSB7XG4gICAgICB0aHJvdyBuZXcgRHVwbGljYXRlQ29tcGxldGlvbihuYW1lKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZC5jb21wbGV0aW9ucy5zZXQobmFtZSwge1xuICAgICAgbmFtZSxcbiAgICAgIGNvbXBsZXRlLFxuICAgICAgLi4ub3B0aW9ucyxcbiAgICB9KTtcblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFRocm93IHZhbGlkYXRpb24gZXJyb3JzIGluc3RlYWQgb2YgY2FsbGluZyBgRGVuby5leGl0KClgIHRvIGhhbmRsZVxuICAgKiB2YWxpZGF0aW9uIGVycm9ycyBtYW51YWxseS5cbiAgICpcbiAgICogQSB2YWxpZGF0aW9uIGVycm9yIGlzIHRocm93biB3aGVuIHRoZSBjb21tYW5kIGlzIHdyb25nbHkgdXNlZCBieSB0aGUgdXNlci5cbiAgICogRm9yIGV4YW1wbGU6IElmIHRoZSB1c2VyIHBhc3NlcyBzb21lIGludmFsaWQgb3B0aW9ucyBvciBhcmd1bWVudHMgdG8gdGhlXG4gICAqIGNvbW1hbmQuXG4gICAqXG4gICAqIFRoaXMgaGFzIG5vIGVmZmVjdCBmb3IgcGFyZW50IGNvbW1hbmRzLiBPbmx5IGZvciB0aGUgY29tbWFuZCBvbiB3aGljaCB0aGlzXG4gICAqIG1ldGhvZCB3YXMgY2FsbGVkIGFuZCBhbGwgY2hpbGQgY29tbWFuZHMuXG4gICAqXG4gICAqICoqRXhhbXBsZToqKlxuICAgKlxuICAgKiBgYGBcbiAgICogdHJ5IHtcbiAgICogICBjbWQucGFyc2UoKTtcbiAgICogfSBjYXRjaChlcnJvcikge1xuICAgKiAgIGlmIChlcnJvciBpbnN0YW5jZW9mIFZhbGlkYXRpb25FcnJvcikge1xuICAgKiAgICAgY21kLnNob3dIZWxwKCk7XG4gICAqICAgICBEZW5vLmV4aXQoMSk7XG4gICAqICAgfVxuICAgKiAgIHRocm93IGVycm9yO1xuICAgKiB9XG4gICAqIGBgYFxuICAgKlxuICAgKiBAc2VlIFZhbGlkYXRpb25FcnJvclxuICAgKi9cbiAgcHVibGljIHRocm93RXJyb3JzKCk6IHRoaXMge1xuICAgIHRoaXMuY21kLnRocm93T25FcnJvciA9IHRydWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2FtZSBhcyBgLnRocm93RXJyb3JzKClgIGJ1dCBhbHNvIHByZXZlbnRzIGNhbGxpbmcgYERlbm8uZXhpdGAgYWZ0ZXJcbiAgICogcHJpbnRpbmcgaGVscCBvciB2ZXJzaW9uIHdpdGggdGhlIC0taGVscCBhbmQgLS12ZXJzaW9uIG9wdGlvbi5cbiAgICovXG4gIHB1YmxpYyBub0V4aXQoKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX3Nob3VsZEV4aXQgPSBmYWxzZTtcbiAgICB0aGlzLnRocm93RXJyb3JzKCk7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogRGlzYWJsZSBpbmhlcml0aW5nIGdsb2JhbCBjb21tYW5kcywgb3B0aW9ucyBhbmQgZW52aXJvbm1lbnQgdmFyaWFibGVzIGZyb21cbiAgICogcGFyZW50IGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIG5vR2xvYmFscygpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fbm9HbG9iYWxzID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBDaGVjayB3aGV0aGVyIHRoZSBjb21tYW5kIHNob3VsZCB0aHJvdyBlcnJvcnMgb3IgZXhpdC4gKi9cbiAgcHJvdGVjdGVkIHNob3VsZFRocm93RXJyb3JzKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLnRocm93T25FcnJvciB8fCAhIXRoaXMuX3BhcmVudD8uc2hvdWxkVGhyb3dFcnJvcnMoKTtcbiAgfVxuXG4gIC8qKiBDaGVjayB3aGV0aGVyIHRoZSBjb21tYW5kIHNob3VsZCBleGl0IGFmdGVyIHByaW50aW5nIGhlbHAgb3IgdmVyc2lvbi4gKi9cbiAgcHJvdGVjdGVkIHNob3VsZEV4aXQoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuX3Nob3VsZEV4aXQgPz8gdGhpcy5fcGFyZW50Py5zaG91bGRFeGl0KCkgPz8gdHJ1ZTtcbiAgfVxuXG4gIHB1YmxpYyBnbG9iYWxPcHRpb248XG4gICAgRiBleHRlbmRzIHN0cmluZyxcbiAgICBHIGV4dGVuZHMgVHlwZWRPcHRpb248XG4gICAgICBGLFxuICAgICAgQ08sXG4gICAgICBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PixcbiAgICAgIHVuZGVmaW5lZCBleHRlbmRzIFggPyBSIDogZmFsc2UsXG4gICAgICBEXG4gICAgPixcbiAgICBNRyBleHRlbmRzIE1hcFZhbHVlPEcsIFYsIEM+LFxuICAgIFIgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIEMgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcImNvbGxlY3RcIl0gPSB1bmRlZmluZWQsXG4gICAgWCBleHRlbmRzIElDb21tYW5kT3B0aW9uW1wiY29uZmxpY3RzXCJdID0gdW5kZWZpbmVkLFxuICAgIEQgPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBmbGFnczogRixcbiAgICBkZXNjOiBzdHJpbmcsXG4gICAgb3B0cz86XG4gICAgICB8IE9taXQ8XG4gICAgICAgIElDb21tYW5kR2xvYmFsT3B0aW9uPFxuICAgICAgICAgIFBhcnRpYWw8Q08+LFxuICAgICAgICAgIENBLFxuICAgICAgICAgIE1lcmdlT3B0aW9uczxGLCBDRywgRz4sXG4gICAgICAgICAgQ1BHLFxuICAgICAgICAgIENULFxuICAgICAgICAgIENHVCxcbiAgICAgICAgICBDUFQsXG4gICAgICAgICAgQ1BcbiAgICAgICAgPixcbiAgICAgICAgXCJ2YWx1ZVwiXG4gICAgICA+XG4gICAgICAgICYge1xuICAgICAgICAgIGRlZmF1bHQ/OiBJRGVmYXVsdFZhbHVlPEQ+O1xuICAgICAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgICAgICBjb2xsZWN0PzogQztcbiAgICAgICAgICB2YWx1ZT86IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPjtcbiAgICAgICAgfVxuICAgICAgfCBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj4sXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBDTyxcbiAgICBDQSxcbiAgICBNZXJnZU9wdGlvbnM8RiwgQ0csIE1HPixcbiAgICBDVCxcbiAgICBDR1QsXG4gICAgQ1BcbiAgPiB7XG4gICAgaWYgKHR5cGVvZiBvcHRzID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHJldHVybiB0aGlzLm9wdGlvbihcbiAgICAgICAgZmxhZ3MsXG4gICAgICAgIGRlc2MsXG4gICAgICAgIHsgdmFsdWU6IG9wdHMsIGdsb2JhbDogdHJ1ZSB9IGFzIElDb21tYW5kT3B0aW9uLFxuICAgICAgKSBhcyBDb21tYW5kPGFueT47XG4gICAgfVxuICAgIHJldHVybiB0aGlzLm9wdGlvbihcbiAgICAgIGZsYWdzLFxuICAgICAgZGVzYyxcbiAgICAgIHsgLi4ub3B0cywgZ2xvYmFsOiB0cnVlIH0gYXMgSUNvbW1hbmRPcHRpb24sXG4gICAgKSBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKipcbiAgICogRW5hYmxlIGdyb3VwaW5nIG9mIG9wdGlvbnMgYW5kIHNldCB0aGUgbmFtZSBvZiB0aGUgZ3JvdXAuXG4gICAqIEFsbCBvcHRpb24gd2hpY2ggYXJlIGFkZGVkIGFmdGVyIGNhbGxpbmcgdGhlIGAuZ3JvdXAoKWAgbWV0aG9kIHdpbGwgYmVcbiAgICogZ3JvdXBlZCBpbiB0aGUgaGVscCBvdXRwdXQuIElmIHRoZSBgLmdyb3VwKClgIG1ldGhvZCBjYW4gYmUgdXNlIG11bHRpcGxlXG4gICAqIHRpbWVzIHRvIGNyZWF0ZSBtb3JlIGdyb3Vwcy5cbiAgICpcbiAgICogQHBhcmFtIG5hbWUgVGhlIG5hbWUgb2YgdGhlIG9wdGlvbiBncm91cC5cbiAgICovXG4gIHB1YmxpYyBncm91cChuYW1lOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fZ3JvdXBOYW1lID0gbmFtZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgYSBuZXcgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgRmxhZ3Mgc3RyaW5nIGUuZzogLWgsIC0taGVscCwgLS1tYW51YWwgPHJlcXVpcmVkQXJnOnN0cmluZz4gW29wdGlvbmFsQXJnOm51bWJlcl0gWy4uLnJlc3RBcmdzOnN0cmluZ11cbiAgICogQHBhcmFtIGRlc2MgRmxhZyBkZXNjcmlwdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgRmxhZyBvcHRpb25zIG9yIGN1c3RvbSBoYW5kbGVyIGZvciBwcm9jZXNzaW5nIGZsYWcgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgb3B0aW9uPFxuICAgIEYgZXh0ZW5kcyBzdHJpbmcsXG4gICAgRyBleHRlbmRzIFR5cGVkT3B0aW9uPFxuICAgICAgRixcbiAgICAgIENPLFxuICAgICAgTWVyZ2U8Q1BULCBNZXJnZTxDR1QsIENUPj4sXG4gICAgICB1bmRlZmluZWQgZXh0ZW5kcyBYID8gUiA6IGZhbHNlLFxuICAgICAgRFxuICAgID4sXG4gICAgTUcgZXh0ZW5kcyBNYXBWYWx1ZTxHLCBWLCBDPixcbiAgICBSIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBDIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJjb2xsZWN0XCJdID0gdW5kZWZpbmVkLFxuICAgIFggZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcImNvbmZsaWN0c1wiXSA9IHVuZGVmaW5lZCxcbiAgICBEID0gdW5kZWZpbmVkLFxuICAgIFYgPSB1bmRlZmluZWQsXG4gID4oXG4gICAgZmxhZ3M6IEYsXG4gICAgZGVzYzogc3RyaW5nLFxuICAgIG9wdHM6XG4gICAgICB8IE9taXQ8XG4gICAgICAgIElDb21tYW5kT3B0aW9uPFxuICAgICAgICAgIFBhcnRpYWw8Q08+LFxuICAgICAgICAgIENBLFxuICAgICAgICAgIE1lcmdlT3B0aW9uczxGLCBDRywgRz4sXG4gICAgICAgICAgQ1BHLFxuICAgICAgICAgIENULFxuICAgICAgICAgIENHVCxcbiAgICAgICAgICBDUFQsXG4gICAgICAgICAgQ1BcbiAgICAgICAgPixcbiAgICAgICAgXCJ2YWx1ZVwiXG4gICAgICA+XG4gICAgICAgICYge1xuICAgICAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICAgICAgICBkZWZhdWx0PzogSURlZmF1bHRWYWx1ZTxEPjtcbiAgICAgICAgICByZXF1aXJlZD86IFI7XG4gICAgICAgICAgY29sbGVjdD86IEM7XG4gICAgICAgICAgdmFsdWU/OiBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj47XG4gICAgICAgIH1cbiAgICAgIHwgSUZsYWdWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxHPj4sIFY+LFxuICApOiBDb21tYW5kPFxuICAgIENQRyxcbiAgICBDUFQsXG4gICAgQ08sXG4gICAgQ0EsXG4gICAgTWVyZ2VPcHRpb25zPEYsIENHLCBNRz4sXG4gICAgQ1QsXG4gICAgQ0dULFxuICAgIENQXG4gID47XG5cbiAgcHVibGljIG9wdGlvbjxcbiAgICBGIGV4dGVuZHMgc3RyaW5nLFxuICAgIE8gZXh0ZW5kcyBUeXBlZE9wdGlvbjxcbiAgICAgIEYsXG4gICAgICBDTyxcbiAgICAgIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LFxuICAgICAgdW5kZWZpbmVkIGV4dGVuZHMgWCA/IFIgOiBmYWxzZSxcbiAgICAgIERcbiAgICA+LFxuICAgIE1PIGV4dGVuZHMgTWFwVmFsdWU8TywgViwgQz4sXG4gICAgUiBleHRlbmRzIElDb21tYW5kT3B0aW9uW1wicmVxdWlyZWRcIl0gPSB1bmRlZmluZWQsXG4gICAgQyBleHRlbmRzIElDb21tYW5kT3B0aW9uW1wiY29sbGVjdFwiXSA9IHVuZGVmaW5lZCxcbiAgICBYIGV4dGVuZHMgSUNvbW1hbmRPcHRpb25bXCJjb25mbGljdHNcIl0gPSB1bmRlZmluZWQsXG4gICAgRCA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIGZsYWdzOiBGLFxuICAgIGRlc2M6IHN0cmluZyxcbiAgICBvcHRzPzpcbiAgICAgIHwgT21pdDxcbiAgICAgICAgSUNvbW1hbmRPcHRpb248TWVyZ2VPcHRpb25zPEYsIENPLCBNTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgICAgICAgXCJ2YWx1ZVwiXG4gICAgICA+XG4gICAgICAgICYge1xuICAgICAgICAgIGRlZmF1bHQ/OiBJRGVmYXVsdFZhbHVlPEQ+O1xuICAgICAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgICAgICBjb2xsZWN0PzogQztcbiAgICAgICAgICBjb25mbGljdHM/OiBYO1xuICAgICAgICAgIHZhbHVlPzogSUZsYWdWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxPPj4sIFY+O1xuICAgICAgICB9XG4gICAgICB8IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Tz4+LCBWPixcbiAgKTogQ29tbWFuZDxcbiAgICBDUEcsXG4gICAgQ1BULFxuICAgIE1lcmdlT3B0aW9uczxGLCBDTywgTU8+LFxuICAgIENBLFxuICAgIENHLFxuICAgIENULFxuICAgIENHVCxcbiAgICBDUFxuICA+O1xuXG4gIHB1YmxpYyBvcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjOiBzdHJpbmcsXG4gICAgb3B0cz86IElDb21tYW5kT3B0aW9uIHwgSUZsYWdWYWx1ZUhhbmRsZXIsXG4gICk6IENvbW1hbmQ8YW55PiB7XG4gICAgaWYgKHR5cGVvZiBvcHRzID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHJldHVybiB0aGlzLm9wdGlvbihmbGFncywgZGVzYywgeyB2YWx1ZTogb3B0cyB9KTtcbiAgICB9XG5cbiAgICBjb25zdCByZXN1bHQgPSBzcGxpdEFyZ3VtZW50cyhmbGFncyk7XG5cbiAgICBjb25zdCBhcmdzOiBJQXJndW1lbnRbXSA9IHJlc3VsdC50eXBlRGVmaW5pdGlvblxuICAgICAgPyBwYXJzZUFyZ3VtZW50c0RlZmluaXRpb24ocmVzdWx0LnR5cGVEZWZpbml0aW9uKVxuICAgICAgOiBbXTtcblxuICAgIGNvbnN0IG9wdGlvbjogSU9wdGlvbiA9IHtcbiAgICAgIC4uLm9wdHMsXG4gICAgICBuYW1lOiBcIlwiLFxuICAgICAgZGVzY3JpcHRpb246IGRlc2MsXG4gICAgICBhcmdzLFxuICAgICAgZmxhZ3M6IHJlc3VsdC5mbGFncyxcbiAgICAgIGVxdWFsc1NpZ246IHJlc3VsdC5lcXVhbHNTaWduLFxuICAgICAgdHlwZURlZmluaXRpb246IHJlc3VsdC50eXBlRGVmaW5pdGlvbixcbiAgICAgIGdyb3VwTmFtZTogdGhpcy5fZ3JvdXBOYW1lLFxuICAgIH07XG5cbiAgICBpZiAob3B0aW9uLnNlcGFyYXRvcikge1xuICAgICAgZm9yIChjb25zdCBhcmcgb2YgYXJncykge1xuICAgICAgICBpZiAoYXJnLmxpc3QpIHtcbiAgICAgICAgICBhcmcuc2VwYXJhdG9yID0gb3B0aW9uLnNlcGFyYXRvcjtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIGZvciAoY29uc3QgcGFydCBvZiBvcHRpb24uZmxhZ3MpIHtcbiAgICAgIGNvbnN0IGFyZyA9IHBhcnQudHJpbSgpO1xuICAgICAgY29uc3QgaXNMb25nID0gL14tLS8udGVzdChhcmcpO1xuICAgICAgY29uc3QgbmFtZSA9IGlzTG9uZyA/IGFyZy5zbGljZSgyKSA6IGFyZy5zbGljZSgxKTtcblxuICAgICAgaWYgKHRoaXMuY21kLmdldEJhc2VPcHRpb24obmFtZSwgdHJ1ZSkpIHtcbiAgICAgICAgaWYgKG9wdHM/Lm92ZXJyaWRlKSB7XG4gICAgICAgICAgdGhpcy5yZW1vdmVPcHRpb24obmFtZSk7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZU9wdGlvbk5hbWUobmFtZSk7XG4gICAgICAgIH1cbiAgICAgIH1cblxuICAgICAgaWYgKCFvcHRpb24ubmFtZSAmJiBpc0xvbmcpIHtcbiAgICAgICAgb3B0aW9uLm5hbWUgPSBuYW1lO1xuICAgICAgfSBlbHNlIGlmICghb3B0aW9uLmFsaWFzZXMpIHtcbiAgICAgICAgb3B0aW9uLmFsaWFzZXMgPSBbbmFtZV07XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBvcHRpb24uYWxpYXNlcy5wdXNoKG5hbWUpO1xuICAgICAgfVxuICAgIH1cblxuICAgIGlmIChvcHRpb24ucHJlcGVuZCkge1xuICAgICAgdGhpcy5jbWQub3B0aW9ucy51bnNoaWZ0KG9wdGlvbik7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRoaXMuY21kLm9wdGlvbnMucHVzaChvcHRpb24pO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEFkZCBuZXcgY29tbWFuZCBleGFtcGxlLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgICAgICBOYW1lIG9mIHRoZSBleGFtcGxlLlxuICAgKiBAcGFyYW0gZGVzY3JpcHRpb24gICBUaGUgY29udGVudCBvZiB0aGUgZXhhbXBsZS5cbiAgICovXG4gIHB1YmxpYyBleGFtcGxlKG5hbWU6IHN0cmluZywgZGVzY3JpcHRpb246IHN0cmluZyk6IHRoaXMge1xuICAgIGlmICh0aGlzLmNtZC5oYXNFeGFtcGxlKG5hbWUpKSB7XG4gICAgICB0aHJvdyBuZXcgRHVwbGljYXRlRXhhbXBsZShuYW1lKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZC5leGFtcGxlcy5wdXNoKHsgbmFtZSwgZGVzY3JpcHRpb24gfSk7XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIHB1YmxpYyBnbG9iYWxFbnY8XG4gICAgTiBleHRlbmRzIHN0cmluZyxcbiAgICBHIGV4dGVuZHMgVHlwZWRFbnY8TiwgUCwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSPixcbiAgICBNRyBleHRlbmRzIE1hcFZhbHVlPEcsIFY+LFxuICAgIFIgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIFAgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInByZWZpeFwiXSA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgZGVzY3JpcHRpb246IHN0cmluZyxcbiAgICBvcHRpb25zPzogT21pdDxJR2xvYmFsRW52VmFyT3B0aW9ucywgXCJ2YWx1ZVwiPiAmIHtcbiAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgIHByZWZpeD86IFA7XG4gICAgICB2YWx1ZT86IElFbnZWYXJWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxHPj4sIFY+O1xuICAgIH0sXG4gICk6IENvbW1hbmQ8Q1BHLCBDUFQsIENPLCBDQSwgTWVyZ2U8Q0csIE1HPiwgQ1QsIENHVCwgQ1A+IHtcbiAgICByZXR1cm4gdGhpcy5lbnYoXG4gICAgICBuYW1lLFxuICAgICAgZGVzY3JpcHRpb24sXG4gICAgICB7IC4uLm9wdGlvbnMsIGdsb2JhbDogdHJ1ZSB9IGFzIElFbnZWYXJPcHRpb25zLFxuICAgICkgYXMgQ29tbWFuZDxhbnk+O1xuICB9XG5cbiAgLyoqXG4gICAqIEFkZCBuZXcgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBuYW1lICAgICAgICAgIE5hbWUgb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gZGVzY3JpcHRpb24gICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gb3B0aW9ucyAgICAgICBFbnZpcm9ubWVudCB2YXJpYWJsZSBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGVudjxcbiAgICBOIGV4dGVuZHMgc3RyaW5nLFxuICAgIEcgZXh0ZW5kcyBUeXBlZEVudjxOLCBQLCBDTywgTWVyZ2U8Q1BULCBNZXJnZTxDR1QsIENUPj4sIFI+LFxuICAgIE1HIGV4dGVuZHMgTWFwVmFsdWU8RywgVj4sXG4gICAgUiBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicmVxdWlyZWRcIl0gPSB1bmRlZmluZWQsXG4gICAgUCBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicHJlZml4XCJdID0gdW5kZWZpbmVkLFxuICAgIFYgPSB1bmRlZmluZWQsXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBkZXNjcmlwdGlvbjogc3RyaW5nLFxuICAgIG9wdGlvbnM6IE9taXQ8SUVudlZhck9wdGlvbnMsIFwidmFsdWVcIj4gJiB7XG4gICAgICBnbG9iYWw6IHRydWU7XG4gICAgICByZXF1aXJlZD86IFI7XG4gICAgICBwcmVmaXg/OiBQO1xuICAgICAgdmFsdWU/OiBJRW52VmFyVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPjtcbiAgICB9LFxuICApOiBDb21tYW5kPENQRywgQ1BULCBDTywgQ0EsIE1lcmdlPENHLCBNRz4sIENULCBDR1QsIENQPjtcblxuICBwdWJsaWMgZW52PFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcsXG4gICAgTyBleHRlbmRzIFR5cGVkRW52PE4sIFAsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUj4sXG4gICAgTU8gZXh0ZW5kcyBNYXBWYWx1ZTxPLCBWPixcbiAgICBSIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBQIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJwcmVmaXhcIl0gPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGRlc2NyaXB0aW9uOiBzdHJpbmcsXG4gICAgb3B0aW9ucz86IE9taXQ8SUVudlZhck9wdGlvbnMsIFwidmFsdWVcIj4gJiB7XG4gICAgICByZXF1aXJlZD86IFI7XG4gICAgICBwcmVmaXg/OiBQO1xuICAgICAgdmFsdWU/OiBJRW52VmFyVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Tz4+LCBWPjtcbiAgICB9LFxuICApOiBDb21tYW5kPENQRywgQ1BULCBNZXJnZTxDTywgTU8+LCBDQSwgQ0csIENULCBDR1QsIENQPjtcblxuICBwdWJsaWMgZW52KFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBkZXNjcmlwdGlvbjogc3RyaW5nLFxuICAgIG9wdGlvbnM/OiBJRW52VmFyT3B0aW9ucyxcbiAgKTogQ29tbWFuZDxhbnk+IHtcbiAgICBjb25zdCByZXN1bHQgPSBzcGxpdEFyZ3VtZW50cyhuYW1lKTtcblxuICAgIGlmICghcmVzdWx0LnR5cGVEZWZpbml0aW9uKSB7XG4gICAgICByZXN1bHQudHlwZURlZmluaXRpb24gPSBcIjx2YWx1ZTpib29sZWFuPlwiO1xuICAgIH1cblxuICAgIGlmIChyZXN1bHQuZmxhZ3Muc29tZSgoZW52TmFtZSkgPT4gdGhpcy5jbWQuZ2V0QmFzZUVudlZhcihlbnZOYW1lLCB0cnVlKSkpIHtcbiAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVFbnZpcm9ubWVudFZhcmlhYmxlKG5hbWUpO1xuICAgIH1cblxuICAgIGNvbnN0IGRldGFpbHM6IElBcmd1bWVudFtdID0gcGFyc2VBcmd1bWVudHNEZWZpbml0aW9uKFxuICAgICAgcmVzdWx0LnR5cGVEZWZpbml0aW9uLFxuICAgICk7XG5cbiAgICBpZiAoZGV0YWlscy5sZW5ndGggPiAxKSB7XG4gICAgICB0aHJvdyBuZXcgRW52aXJvbm1lbnRWYXJpYWJsZVNpbmdsZVZhbHVlKG5hbWUpO1xuICAgIH0gZWxzZSBpZiAoZGV0YWlscy5sZW5ndGggJiYgZGV0YWlsc1swXS5vcHRpb25hbFZhbHVlKSB7XG4gICAgICB0aHJvdyBuZXcgRW52aXJvbm1lbnRWYXJpYWJsZU9wdGlvbmFsVmFsdWUobmFtZSk7XG4gICAgfSBlbHNlIGlmIChkZXRhaWxzLmxlbmd0aCAmJiBkZXRhaWxzWzBdLnZhcmlhZGljKSB7XG4gICAgICB0aHJvdyBuZXcgRW52aXJvbm1lbnRWYXJpYWJsZVZhcmlhZGljVmFsdWUobmFtZSk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQuZW52VmFycy5wdXNoKHtcbiAgICAgIG5hbWU6IHJlc3VsdC5mbGFnc1swXSxcbiAgICAgIG5hbWVzOiByZXN1bHQuZmxhZ3MsXG4gICAgICBkZXNjcmlwdGlvbixcbiAgICAgIHR5cGU6IGRldGFpbHNbMF0udHlwZSxcbiAgICAgIGRldGFpbHM6IGRldGFpbHMuc2hpZnQoKSBhcyBJQXJndW1lbnQsXG4gICAgICAuLi5vcHRpb25zLFxuICAgIH0pO1xuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKiogTUFJTiBIQU5ETEVSICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKi9cblxuICAvKipcbiAgICogUGFyc2UgY29tbWFuZCBsaW5lIGFyZ3VtZW50cyBhbmQgZXhlY3V0ZSBtYXRjaGVkIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBhcmdzIENvbW1hbmQgbGluZSBhcmdzIHRvIHBhcnNlLiBFeDogYGNtZC5wYXJzZSggRGVuby5hcmdzIClgXG4gICAqL1xuICBwdWJsaWMgYXN5bmMgcGFyc2UoXG4gICAgYXJnczogc3RyaW5nW10gPSBEZW5vLmFyZ3MsXG4gICk6IFByb21pc2U8XG4gICAgQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBJUGFyc2VSZXN1bHQ8XG4gICAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgICBBcnJheTx1bmtub3duPixcbiAgICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgICB1bmRlZmluZWRcbiAgICAgID5cbiAgICAgIDogSVBhcnNlUmVzdWx0PFxuICAgICAgICBNYXBUeXBlczxDTz4sXG4gICAgICAgIE1hcFR5cGVzPENBPixcbiAgICAgICAgTWFwVHlwZXM8Q0c+LFxuICAgICAgICBNYXBUeXBlczxDUEc+LFxuICAgICAgICBDVCxcbiAgICAgICAgQ0dULFxuICAgICAgICBDUFQsXG4gICAgICAgIENQXG4gICAgICA+XG4gID4ge1xuICAgIHRyeSB7XG4gICAgICByZXR1cm4gYXdhaXQgdGhpcy5wYXJzZUNvbW1hbmQoeyBhcmdzIH0pIGFzIGFueTtcbiAgICB9IGNhdGNoIChlcnJvcjogdW5rbm93bikge1xuICAgICAgdGhpcy50aHJvdyhcbiAgICAgICAgZXJyb3IgaW5zdGFuY2VvZiBFcnJvclxuICAgICAgICAgID8gZXJyb3JcbiAgICAgICAgICA6IG5ldyBFcnJvcihgW25vbi1lcnJvci10aHJvd25dICR7ZXJyb3J9YCksXG4gICAgICApO1xuICAgIH1cbiAgfVxuXG4gIHByaXZhdGUgYXN5bmMgcGFyc2VDb21tYW5kKGN0eDogUGFyc2VDb250ZXh0KTogUHJvbWlzZTxJUGFyc2VSZXN1bHQ+IHtcbiAgICB0aGlzLnJlc2V0KCk7XG4gICAgdGhpcy5yZWdpc3RlckRlZmF1bHRzKCk7XG4gICAgdGhpcy5yYXdBcmdzID0gY3R4LmFyZ3M7XG5cbiAgICBpZiAodGhpcy5pc0V4ZWN1dGFibGUpIHtcbiAgICAgIGF3YWl0IHRoaXMuZXhlY3V0ZUV4ZWN1dGFibGUoY3R4LmFyZ3MpO1xuICAgICAgcmV0dXJuIHsgb3B0aW9uczoge30sIGFyZ3M6IFtdLCBjbWQ6IHRoaXMsIGxpdGVyYWw6IFtdIH0gYXMgYW55O1xuICAgIH1cblxuICAgIGlmICh0aGlzLl91c2VSYXdBcmdzKSB7XG4gICAgICBjb25zdCBlbnYgPSBhd2FpdCB0aGlzLnBhcnNlRW52VmFycyh0aGlzLmVudlZhcnMpO1xuICAgICAgcmV0dXJuIHRoaXMuZXhlY3V0ZShlbnYsIC4uLmN0eC5hcmdzKSBhcyBhbnk7XG4gICAgfVxuXG4gICAgbGV0IHByZVBhcnNlR2xvYmFscyA9IGZhbHNlO1xuICAgIGxldCBzdWJDb21tYW5kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQ7XG5cbiAgICAvLyBQcmUgcGFyc2UgZ2xvYmFscyB0byBzdXBwb3J0OiBjbWQgLS1nbG9iYWwtb3B0aW9uIHN1Yi1jb21tYW5kIC0tb3B0aW9uXG4gICAgaWYgKGN0eC5hcmdzLmxlbmd0aCA+IDApIHtcbiAgICAgIC8vIERldGVjdCBzdWIgY29tbWFuZC5cbiAgICAgIHN1YkNvbW1hbmQgPSB0aGlzLmdldENvbW1hbmQoY3R4LmFyZ3NbMF0sIHRydWUpO1xuXG4gICAgICBpZiAoc3ViQ29tbWFuZCkge1xuICAgICAgICBjdHguYXJncyA9IGN0eC5hcmdzLnNsaWNlKDEpO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgLy8gT25seSBwcmUgcGFyc2UgZ2xvYmFscyBpZiBmaXJzdCBhcmcgaXN0IGEgZ2xvYmFsIG9wdGlvbi5cbiAgICAgICAgY29uc3Qgb3B0aW9uTmFtZSA9IGN0eC5hcmdzWzBdLnJlcGxhY2UoL14tKy8sIFwiXCIpO1xuICAgICAgICBwcmVQYXJzZUdsb2JhbHMgPSB0aGlzLmdldE9wdGlvbihvcHRpb25OYW1lLCB0cnVlKT8uZ2xvYmFsID09PSB0cnVlO1xuXG4gICAgICAgIC8vIFBhcnNlIGdsb2JhbCBvcHRpb25zICYgZW52IHZhcnMuXG4gICAgICAgIGlmIChwcmVQYXJzZUdsb2JhbHMpIHtcbiAgICAgICAgICBjdHggPSBhd2FpdCB0aGlzLnBhcnNlR2xvYmFsT3B0aW9uc0FuZEVudlZhcnMoY3R4KTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH0gZWxzZSB7XG4gICAgICBwcmVQYXJzZUdsb2JhbHMgPSBmYWxzZTtcbiAgICB9XG5cbiAgICAvLyBQYXJzZSBzdWIgY29tbWFuZC5cbiAgICBpZiAoc3ViQ29tbWFuZCB8fCBjdHguYXJncy5sZW5ndGggPiAwKSB7XG4gICAgICBpZiAoIXN1YkNvbW1hbmQpIHtcbiAgICAgICAgc3ViQ29tbWFuZCA9IHRoaXMuZ2V0Q29tbWFuZChjdHguYXJnc1swXSwgdHJ1ZSk7XG5cbiAgICAgICAgaWYgKHN1YkNvbW1hbmQpIHtcbiAgICAgICAgICBjdHguYXJncyA9IGN0eC5hcmdzLnNsaWNlKDEpO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGlmIChzdWJDb21tYW5kKSB7XG4gICAgICAgIHN1YkNvbW1hbmQuX2dsb2JhbFBhcmVudCA9IHRoaXM7XG5cbiAgICAgICAgcmV0dXJuIHN1YkNvbW1hbmQucGFyc2VDb21tYW5kKGN0eCk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgLy8gUGFyc2UgcmVzdCBvcHRpb25zICYgZW52IHZhcnMuXG4gICAgY3R4ID0gYXdhaXQgdGhpcy5wYXJzZU9wdGlvbnNBbmRFbnZWYXJzKGN0eCwgcHJlUGFyc2VHbG9iYWxzKTtcblxuICAgIHRoaXMubGl0ZXJhbEFyZ3MgPSBjdHgubGl0ZXJhbCA/PyBbXTtcblxuICAgIC8vIE1lcmdlIGVudiBhbmQgZ2xvYmFsIG9wdGlvbnMuXG4gICAgY29uc3Qgb3B0aW9ucyA9IHsgLi4uY3R4LmVudiwgLi4uY3R4Lm9wdGlvbnMgfTtcblxuICAgIC8vIFBhcnNlIGFyZ3VtZW50cy5cbiAgICBjb25zdCBwYXJhbXMgPSB0aGlzLnBhcnNlQXJndW1lbnRzKGN0eC5hcmdzLCBvcHRpb25zKTtcblxuICAgIC8vIEV4ZWN1dGUgb3B0aW9uIGFjdGlvbi5cbiAgICBpZiAoY3R4LmFjdGlvbikge1xuICAgICAgYXdhaXQgY3R4LmFjdGlvbi5hY3Rpb24uY2FsbCh0aGlzLCBvcHRpb25zLCAuLi5wYXJhbXMpO1xuXG4gICAgICBpZiAoY3R4LmFjdGlvbi5zdGFuZGFsb25lKSB7XG4gICAgICAgIHJldHVybiB7XG4gICAgICAgICAgb3B0aW9ucyxcbiAgICAgICAgICBhcmdzOiBwYXJhbXMsXG4gICAgICAgICAgY21kOiB0aGlzLFxuICAgICAgICAgIGxpdGVyYWw6IHRoaXMubGl0ZXJhbEFyZ3MsXG4gICAgICAgIH0gYXMgYW55O1xuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiB0aGlzLmV4ZWN1dGUob3B0aW9ucywgLi4ucGFyYW1zKSBhcyBhbnk7XG4gIH1cblxuICBwcml2YXRlIGFzeW5jIHBhcnNlR2xvYmFsT3B0aW9uc0FuZEVudlZhcnMoXG4gICAgY3R4OiBQYXJzZUNvbnRleHQsXG4gICk6IFByb21pc2U8UGFyc2VDb250ZXh0PiB7XG4gICAgLy8gUGFyc2UgZ2xvYmFsIGVudiB2YXJzLlxuICAgIGNvbnN0IGVudlZhcnMgPSBbXG4gICAgICAuLi50aGlzLmVudlZhcnMuZmlsdGVyKChlbnZWYXIpID0+IGVudlZhci5nbG9iYWwpLFxuICAgICAgLi4udGhpcy5nZXRHbG9iYWxFbnZWYXJzKHRydWUpLFxuICAgIF07XG5cbiAgICBjb25zdCBpc0hlbHBPcHRpb24gPSB0aGlzLmdldEhlbHBPcHRpb24oKT8uZmxhZ3MuaW5jbHVkZXMoY3R4LmFyZ3NbMF0pO1xuICAgIGNvbnN0IGVudiA9IGF3YWl0IHRoaXMucGFyc2VFbnZWYXJzKGVudlZhcnMsICFpc0hlbHBPcHRpb24pO1xuXG4gICAgLy8gUGFyc2UgZ2xvYmFsIG9wdGlvbnMuXG4gICAgY29uc3Qgb3B0aW9ucyA9IFtcbiAgICAgIC4uLnRoaXMub3B0aW9ucy5maWx0ZXIoKG9wdGlvbikgPT4gb3B0aW9uLmdsb2JhbCksXG4gICAgICAuLi50aGlzLmdldEdsb2JhbE9wdGlvbnModHJ1ZSksXG4gICAgXTtcblxuICAgIHJldHVybiB0aGlzLnBhcnNlT3B0aW9ucyhjdHgsIG9wdGlvbnMsIGVudiwgdHJ1ZSk7XG4gIH1cblxuICBwcml2YXRlIGFzeW5jIHBhcnNlT3B0aW9uc0FuZEVudlZhcnMoXG4gICAgY3R4OiBQYXJzZUNvbnRleHQsXG4gICAgcHJlUGFyc2VHbG9iYWxzOiBib29sZWFuLFxuICApOiBQcm9taXNlPFBhcnNlQ29udGV4dD4ge1xuICAgIC8vIFBhcnNlIGVudiB2YXJzLlxuICAgIGNvbnN0IGVudlZhcnMgPSBwcmVQYXJzZUdsb2JhbHNcbiAgICAgID8gdGhpcy5lbnZWYXJzLmZpbHRlcigoZW52VmFyKSA9PiAhZW52VmFyLmdsb2JhbClcbiAgICAgIDogdGhpcy5nZXRFbnZWYXJzKHRydWUpO1xuXG4gICAgY29uc3QgaGVscE9wdGlvbiA9IHRoaXMuZ2V0SGVscE9wdGlvbigpO1xuICAgIGNvbnN0IGlzVmVyc2lvbk9wdGlvbiA9IHRoaXMuX3ZlcnNpb25PcHRpb24/LmZsYWdzLmluY2x1ZGVzKGN0eC5hcmdzWzBdKTtcbiAgICBjb25zdCBpc0hlbHBPcHRpb24gPSBoZWxwT3B0aW9uICYmIGN0eC5vcHRpb25zPy5baGVscE9wdGlvbi5uYW1lXSA9PT0gdHJ1ZTtcbiAgICBjb25zdCBlbnYgPSB7XG4gICAgICAuLi5jdHguZW52LFxuICAgICAgLi4uYXdhaXQgdGhpcy5wYXJzZUVudlZhcnMoZW52VmFycywgIWlzSGVscE9wdGlvbiAmJiAhaXNWZXJzaW9uT3B0aW9uKSxcbiAgICB9O1xuXG4gICAgLy8gUGFyc2Ugb3B0aW9ucy5cbiAgICBjb25zdCBvcHRpb25zID0gcHJlUGFyc2VHbG9iYWxzXG4gICAgICA/IHRoaXMub3B0aW9ucy5maWx0ZXIoKG9wdGlvbikgPT4gIW9wdGlvbi5nbG9iYWwpXG4gICAgICA6IHRoaXMuZ2V0T3B0aW9ucyh0cnVlKTtcblxuICAgIHJldHVybiB0aGlzLnBhcnNlT3B0aW9ucyhjdHgsIG9wdGlvbnMsIGVudik7XG4gIH1cblxuICAvKiogUmVnaXN0ZXIgZGVmYXVsdCBvcHRpb25zIGxpa2UgYC0tdmVyc2lvbmAgYW5kIGAtLWhlbHBgLiAqL1xuICBwcml2YXRlIHJlZ2lzdGVyRGVmYXVsdHMoKTogdGhpcyB7XG4gICAgaWYgKHRoaXMuaGFzRGVmYXVsdHMgfHwgdGhpcy5nZXRQYXJlbnQoKSkge1xuICAgICAgcmV0dXJuIHRoaXM7XG4gICAgfVxuICAgIHRoaXMuaGFzRGVmYXVsdHMgPSB0cnVlO1xuXG4gICAgdGhpcy5yZXNldCgpO1xuXG4gICAgIXRoaXMudHlwZXMuaGFzKFwic3RyaW5nXCIpICYmXG4gICAgICB0aGlzLnR5cGUoXCJzdHJpbmdcIiwgbmV3IFN0cmluZ1R5cGUoKSwgeyBnbG9iYWw6IHRydWUgfSk7XG4gICAgIXRoaXMudHlwZXMuaGFzKFwibnVtYmVyXCIpICYmXG4gICAgICB0aGlzLnR5cGUoXCJudW1iZXJcIiwgbmV3IE51bWJlclR5cGUoKSwgeyBnbG9iYWw6IHRydWUgfSk7XG4gICAgIXRoaXMudHlwZXMuaGFzKFwiaW50ZWdlclwiKSAmJlxuICAgICAgdGhpcy50eXBlKFwiaW50ZWdlclwiLCBuZXcgSW50ZWdlclR5cGUoKSwgeyBnbG9iYWw6IHRydWUgfSk7XG4gICAgIXRoaXMudHlwZXMuaGFzKFwiYm9vbGVhblwiKSAmJlxuICAgICAgdGhpcy50eXBlKFwiYm9vbGVhblwiLCBuZXcgQm9vbGVhblR5cGUoKSwgeyBnbG9iYWw6IHRydWUgfSk7XG4gICAgIXRoaXMudHlwZXMuaGFzKFwiZmlsZVwiKSAmJlxuICAgICAgdGhpcy50eXBlKFwiZmlsZVwiLCBuZXcgRmlsZVR5cGUoKSwgeyBnbG9iYWw6IHRydWUgfSk7XG5cbiAgICBpZiAoIXRoaXMuX2hlbHApIHtcbiAgICAgIHRoaXMuaGVscCh7XG4gICAgICAgIGhpbnRzOiB0cnVlLFxuICAgICAgICB0eXBlczogZmFsc2UsXG4gICAgICB9KTtcbiAgICB9XG5cbiAgICBpZiAodGhpcy5fdmVyc2lvbk9wdGlvbnMgIT09IGZhbHNlICYmICh0aGlzLl92ZXJzaW9uT3B0aW9ucyB8fCB0aGlzLnZlcikpIHtcbiAgICAgIHRoaXMub3B0aW9uKFxuICAgICAgICB0aGlzLl92ZXJzaW9uT3B0aW9ucz8uZmxhZ3MgfHwgXCItViwgLS12ZXJzaW9uXCIsXG4gICAgICAgIHRoaXMuX3ZlcnNpb25PcHRpb25zPy5kZXNjIHx8XG4gICAgICAgICAgXCJTaG93IHRoZSB2ZXJzaW9uIG51bWJlciBmb3IgdGhpcyBwcm9ncmFtLlwiLFxuICAgICAgICB7XG4gICAgICAgICAgc3RhbmRhbG9uZTogdHJ1ZSxcbiAgICAgICAgICBwcmVwZW5kOiB0cnVlLFxuICAgICAgICAgIGFjdGlvbjogYXN5bmMgZnVuY3Rpb24gKCkge1xuICAgICAgICAgICAgY29uc3QgbG9uZyA9IHRoaXMuZ2V0UmF3QXJncygpLmluY2x1ZGVzKFxuICAgICAgICAgICAgICBgLS0ke3RoaXMuX3ZlcnNpb25PcHRpb24/Lm5hbWV9YCxcbiAgICAgICAgICAgICk7XG4gICAgICAgICAgICBpZiAobG9uZykge1xuICAgICAgICAgICAgICBhd2FpdCB0aGlzLmNoZWNrVmVyc2lvbigpO1xuICAgICAgICAgICAgICB0aGlzLnNob3dMb25nVmVyc2lvbigpO1xuICAgICAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICAgICAgdGhpcy5zaG93VmVyc2lvbigpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgdGhpcy5leGl0KCk7XG4gICAgICAgICAgfSxcbiAgICAgICAgICAuLi4odGhpcy5fdmVyc2lvbk9wdGlvbnM/Lm9wdHMgPz8ge30pLFxuICAgICAgICB9LFxuICAgICAgKTtcbiAgICAgIHRoaXMuX3ZlcnNpb25PcHRpb24gPSB0aGlzLm9wdGlvbnNbMF07XG4gICAgfVxuXG4gICAgaWYgKHRoaXMuX2hlbHBPcHRpb25zICE9PSBmYWxzZSkge1xuICAgICAgdGhpcy5vcHRpb24oXG4gICAgICAgIHRoaXMuX2hlbHBPcHRpb25zPy5mbGFncyB8fCBcIi1oLCAtLWhlbHBcIixcbiAgICAgICAgdGhpcy5faGVscE9wdGlvbnM/LmRlc2MgfHwgXCJTaG93IHRoaXMgaGVscC5cIixcbiAgICAgICAge1xuICAgICAgICAgIHN0YW5kYWxvbmU6IHRydWUsXG4gICAgICAgICAgZ2xvYmFsOiB0cnVlLFxuICAgICAgICAgIHByZXBlbmQ6IHRydWUsXG4gICAgICAgICAgYWN0aW9uOiBhc3luYyBmdW5jdGlvbiAoKSB7XG4gICAgICAgICAgICBjb25zdCBsb25nID0gdGhpcy5nZXRSYXdBcmdzKCkuaW5jbHVkZXMoXG4gICAgICAgICAgICAgIGAtLSR7dGhpcy5nZXRIZWxwT3B0aW9uKCk/Lm5hbWV9YCxcbiAgICAgICAgICAgICk7XG4gICAgICAgICAgICBhd2FpdCB0aGlzLmNoZWNrVmVyc2lvbigpO1xuICAgICAgICAgICAgdGhpcy5zaG93SGVscCh7IGxvbmcgfSk7XG4gICAgICAgICAgICB0aGlzLmV4aXQoKTtcbiAgICAgICAgICB9LFxuICAgICAgICAgIC4uLih0aGlzLl9oZWxwT3B0aW9ucz8ub3B0cyA/PyB7fSksXG4gICAgICAgIH0sXG4gICAgICApO1xuICAgICAgdGhpcy5faGVscE9wdGlvbiA9IHRoaXMub3B0aW9uc1swXTtcbiAgICB9XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBFeGVjdXRlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBvcHRpb25zIEEgbWFwIG9mIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBhcmdzIENvbW1hbmQgYXJndW1lbnRzLlxuICAgKi9cbiAgcHJvdGVjdGVkIGFzeW5jIGV4ZWN1dGUoXG4gICAgb3B0aW9uczogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgLi4uYXJnczogQXJyYXk8dW5rbm93bj5cbiAgKTogUHJvbWlzZTxJUGFyc2VSZXN1bHQ+IHtcbiAgICBpZiAodGhpcy5mbikge1xuICAgICAgYXdhaXQgdGhpcy5mbihvcHRpb25zLCAuLi5hcmdzKTtcbiAgICB9IGVsc2UgaWYgKHRoaXMuZGVmYXVsdENvbW1hbmQpIHtcbiAgICAgIGNvbnN0IGNtZCA9IHRoaXMuZ2V0Q29tbWFuZCh0aGlzLmRlZmF1bHRDb21tYW5kLCB0cnVlKTtcblxuICAgICAgaWYgKCFjbWQpIHtcbiAgICAgICAgdGhyb3cgbmV3IERlZmF1bHRDb21tYW5kTm90Rm91bmQoXG4gICAgICAgICAgdGhpcy5kZWZhdWx0Q29tbWFuZCxcbiAgICAgICAgICB0aGlzLmdldENvbW1hbmRzKCksXG4gICAgICAgICk7XG4gICAgICB9XG5cbiAgICAgIGNtZC5fZ2xvYmFsUGFyZW50ID0gdGhpcztcbiAgICAgIGF3YWl0IGNtZC5leGVjdXRlKG9wdGlvbnMsIC4uLmFyZ3MpO1xuICAgIH1cblxuICAgIHJldHVybiB7XG4gICAgICBvcHRpb25zLFxuICAgICAgYXJncyxcbiAgICAgIGNtZDogdGhpcyxcbiAgICAgIGxpdGVyYWw6IHRoaXMubGl0ZXJhbEFyZ3MsXG4gICAgfTtcbiAgfVxuXG4gIC8qKlxuICAgKiBFeGVjdXRlIGV4dGVybmFsIHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gYXJncyBSYXcgY29tbWFuZCBsaW5lIGFyZ3VtZW50cy5cbiAgICovXG4gIHByb3RlY3RlZCBhc3luYyBleGVjdXRlRXhlY3V0YWJsZShhcmdzOiBzdHJpbmdbXSkge1xuICAgIGNvbnN0IGNvbW1hbmQgPSB0aGlzLmdldFBhdGgoKS5yZXBsYWNlKC9cXHMrL2csIFwiLVwiKTtcblxuICAgIGF3YWl0IERlbm8ucGVybWlzc2lvbnMucmVxdWVzdCh7IG5hbWU6IFwicnVuXCIsIGNvbW1hbmQgfSk7XG5cbiAgICB0cnkge1xuICAgICAgY29uc3QgcHJvY2VzczogRGVuby5Qcm9jZXNzID0gRGVuby5ydW4oe1xuICAgICAgICBjbWQ6IFtjb21tYW5kLCAuLi5hcmdzXSxcbiAgICAgIH0pO1xuICAgICAgY29uc3Qgc3RhdHVzOiBEZW5vLlByb2Nlc3NTdGF0dXMgPSBhd2FpdCBwcm9jZXNzLnN0YXR1cygpO1xuXG4gICAgICBpZiAoIXN0YXR1cy5zdWNjZXNzKSB7XG4gICAgICAgIERlbm8uZXhpdChzdGF0dXMuY29kZSk7XG4gICAgICB9XG4gICAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICAgIGlmIChlcnJvciBpbnN0YW5jZW9mIERlbm8uZXJyb3JzLk5vdEZvdW5kKSB7XG4gICAgICAgIHRocm93IG5ldyBDb21tYW5kRXhlY3V0YWJsZU5vdEZvdW5kKGNvbW1hbmQpO1xuICAgICAgfVxuICAgICAgdGhyb3cgZXJyb3I7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFBhcnNlIHJhdyBjb21tYW5kIGxpbmUgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gYXJncyBSYXcgY29tbWFuZCBsaW5lIGFyZ3VtZW50cy5cbiAgICovXG4gIHByb3RlY3RlZCBwYXJzZU9wdGlvbnMoXG4gICAgY3R4OiBQYXJzZUNvbnRleHQsXG4gICAgb3B0aW9uczogSU9wdGlvbltdLFxuICAgIGVudjogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgc3RvcEVhcmx5OiBib29sZWFuID0gdGhpcy5fc3RvcEVhcmx5LFxuICApOiBQYXJzZUNvbnRleHQge1xuICAgIHRyeSB7XG4gICAgICBsZXQgYWN0aW9uOiBBY3Rpb25PcHRpb24gfCB1bmRlZmluZWQ7XG5cbiAgICAgIGNvbnN0IHBhcnNlUmVzdWx0ID0gcGFyc2VGbGFncyhjdHguYXJncywge1xuICAgICAgICBzdG9wRWFybHksXG4gICAgICAgIGFsbG93RW1wdHk6IHRoaXMuX2FsbG93RW1wdHksXG4gICAgICAgIGZsYWdzOiBvcHRpb25zLFxuICAgICAgICBpZ25vcmVEZWZhdWx0czogZW52LFxuICAgICAgICBwYXJzZTogKHR5cGU6IElUeXBlSW5mbykgPT4gdGhpcy5wYXJzZVR5cGUodHlwZSksXG4gICAgICAgIG9wdGlvbjogKG9wdGlvbjogSU9wdGlvbikgPT4ge1xuICAgICAgICAgIGlmICghYWN0aW9uICYmIG9wdGlvbi5hY3Rpb24pIHtcbiAgICAgICAgICAgIGFjdGlvbiA9IG9wdGlvbiBhcyBBY3Rpb25PcHRpb247XG4gICAgICAgICAgfVxuICAgICAgICB9LFxuICAgICAgfSk7XG5cbiAgICAgIC8vIE1lcmdlIGNvbnRleHQuXG4gICAgICByZXR1cm4ge1xuICAgICAgICBhcmdzOiBwYXJzZVJlc3VsdC51bmtub3duLFxuICAgICAgICBvcHRpb25zOiB7IC4uLmN0eC5vcHRpb25zLCAuLi5wYXJzZVJlc3VsdC5mbGFncyB9LFxuICAgICAgICBlbnY6IHsgLi4uY3R4LmVudiwgLi4uZW52IH0sXG4gICAgICAgIGFjdGlvbjogY3R4LmFjdGlvbiA/PyBhY3Rpb24sXG4gICAgICAgIGxpdGVyYWw6IHBhcnNlUmVzdWx0LmxpdGVyYWwsXG4gICAgICB9O1xuICAgIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBGbGFnc1ZhbGlkYXRpb25FcnJvcikge1xuICAgICAgICB0aHJvdyBuZXcgVmFsaWRhdGlvbkVycm9yKGVycm9yLm1lc3NhZ2UpO1xuICAgICAgfVxuXG4gICAgICB0aHJvdyBlcnJvcjtcbiAgICB9XG4gIH1cblxuICAvKiogUGFyc2UgYXJndW1lbnQgdHlwZS4gKi9cbiAgcHJvdGVjdGVkIHBhcnNlVHlwZSh0eXBlOiBJVHlwZUluZm8pOiB1bmtub3duIHtcbiAgICBjb25zdCB0eXBlU2V0dGluZ3M6IElUeXBlIHwgdW5kZWZpbmVkID0gdGhpcy5nZXRUeXBlKHR5cGUudHlwZSk7XG5cbiAgICBpZiAoIXR5cGVTZXR0aW5ncykge1xuICAgICAgdGhyb3cgbmV3IFVua25vd25UeXBlKFxuICAgICAgICB0eXBlLnR5cGUsXG4gICAgICAgIHRoaXMuZ2V0VHlwZXMoKS5tYXAoKHR5cGUpID0+IHR5cGUubmFtZSksXG4gICAgICApO1xuICAgIH1cblxuICAgIHRyeSB7XG4gICAgICByZXR1cm4gdHlwZVNldHRpbmdzLmhhbmRsZXIgaW5zdGFuY2VvZiBUeXBlXG4gICAgICAgID8gdHlwZVNldHRpbmdzLmhhbmRsZXIucGFyc2UodHlwZSlcbiAgICAgICAgOiB0eXBlU2V0dGluZ3MuaGFuZGxlcih0eXBlKTtcbiAgICB9IGNhdGNoIChlcnJvcikge1xuICAgICAgaWYgKGVycm9yIGluc3RhbmNlb2YgRmxhZ3NWYWxpZGF0aW9uRXJyb3IpIHtcbiAgICAgICAgdGhyb3cgbmV3IFZhbGlkYXRpb25FcnJvcihlcnJvci5tZXNzYWdlKTtcbiAgICAgIH1cbiAgICAgIHRocm93IGVycm9yO1xuICAgIH1cbiAgfVxuXG4gIC8qKlxuICAgKiBSZWFkIGFuZCB2YWxpZGF0ZSBlbnZpcm9ubWVudCB2YXJpYWJsZXMuXG4gICAqIEBwYXJhbSBlbnZWYXJzIGVudiB2YXJzIGRlZmluZWQgYnkgdGhlIGNvbW1hbmRcbiAgICogQHBhcmFtIHZhbGlkYXRlIHdoZW4gdHJ1ZSwgdGhyb3dzIGFuIGVycm9yIGlmIGEgcmVxdWlyZWQgZW52IHZhciBpcyBtaXNzaW5nXG4gICAqL1xuICBwcm90ZWN0ZWQgYXN5bmMgcGFyc2VFbnZWYXJzKFxuICAgIGVudlZhcnM6IEFycmF5PElFbnZWYXI+LFxuICAgIHZhbGlkYXRlID0gdHJ1ZSxcbiAgKTogUHJvbWlzZTxSZWNvcmQ8c3RyaW5nLCB1bmtub3duPj4ge1xuICAgIGNvbnN0IHJlc3VsdDogUmVjb3JkPHN0cmluZywgdW5rbm93bj4gPSB7fTtcblxuICAgIGZvciAoY29uc3QgZW52IG9mIGVudlZhcnMpIHtcbiAgICAgIGNvbnN0IGZvdW5kID0gYXdhaXQgdGhpcy5maW5kRW52VmFyKGVudi5uYW1lcyk7XG5cbiAgICAgIGlmIChmb3VuZCkge1xuICAgICAgICBjb25zdCB7IG5hbWUsIHZhbHVlIH0gPSBmb3VuZDtcblxuICAgICAgICBjb25zdCBwcm9wZXJ0eU5hbWUgPSB1bmRlcnNjb3JlVG9DYW1lbENhc2UoXG4gICAgICAgICAgZW52LnByZWZpeFxuICAgICAgICAgICAgPyBlbnYubmFtZXNbMF0ucmVwbGFjZShuZXcgUmVnRXhwKGBeJHtlbnYucHJlZml4fWApLCBcIlwiKVxuICAgICAgICAgICAgOiBlbnYubmFtZXNbMF0sXG4gICAgICAgICk7XG5cbiAgICAgICAgaWYgKGVudi5kZXRhaWxzLmxpc3QpIHtcbiAgICAgICAgICBjb25zdCB2YWx1ZXMgPSB2YWx1ZS5zcGxpdChlbnYuZGV0YWlscy5zZXBhcmF0b3IgPz8gXCIsXCIpO1xuXG4gICAgICAgICAgcmVzdWx0W3Byb3BlcnR5TmFtZV0gPSB2YWx1ZXMubWFwKCh2YWx1ZSkgPT5cbiAgICAgICAgICAgIHRoaXMucGFyc2VUeXBlKHtcbiAgICAgICAgICAgICAgbGFiZWw6IFwiRW52aXJvbm1lbnQgdmFyaWFibGVcIixcbiAgICAgICAgICAgICAgdHlwZTogZW52LnR5cGUsXG4gICAgICAgICAgICAgIG5hbWUsXG4gICAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgICAgfSlcbiAgICAgICAgICApO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdFtwcm9wZXJ0eU5hbWVdID0gdGhpcy5wYXJzZVR5cGUoe1xuICAgICAgICAgICAgbGFiZWw6IFwiRW52aXJvbm1lbnQgdmFyaWFibGVcIixcbiAgICAgICAgICAgIHR5cGU6IGVudi50eXBlLFxuICAgICAgICAgICAgbmFtZSxcbiAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgaWYgKGVudi52YWx1ZSAmJiB0eXBlb2YgcmVzdWx0W3Byb3BlcnR5TmFtZV0gIT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICByZXN1bHRbcHJvcGVydHlOYW1lXSA9IGVudi52YWx1ZShyZXN1bHRbcHJvcGVydHlOYW1lXSk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoZW52LnJlcXVpcmVkICYmIHZhbGlkYXRlKSB7XG4gICAgICAgIHRocm93IG5ldyBNaXNzaW5nUmVxdWlyZWRFbnZWYXIoZW52KTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0O1xuICB9XG5cbiAgcHJvdGVjdGVkIGFzeW5jIGZpbmRFbnZWYXIoXG4gICAgbmFtZXM6IHJlYWRvbmx5IHN0cmluZ1tdLFxuICApOiBQcm9taXNlPHsgbmFtZTogc3RyaW5nOyB2YWx1ZTogc3RyaW5nIH0gfCB1bmRlZmluZWQ+IHtcbiAgICBmb3IgKGNvbnN0IG5hbWUgb2YgbmFtZXMpIHtcbiAgICAgIGNvbnN0IHN0YXR1cyA9IGF3YWl0IERlbm8ucGVybWlzc2lvbnMucXVlcnkoe1xuICAgICAgICBuYW1lOiBcImVudlwiLFxuICAgICAgICB2YXJpYWJsZTogbmFtZSxcbiAgICAgIH0pO1xuXG4gICAgICBpZiAoc3RhdHVzLnN0YXRlID09PSBcImdyYW50ZWRcIikge1xuICAgICAgICBjb25zdCB2YWx1ZSA9IERlbm8uZW52LmdldChuYW1lKTtcblxuICAgICAgICBpZiAodmFsdWUpIHtcbiAgICAgICAgICByZXR1cm4geyBuYW1lLCB2YWx1ZSB9O1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmV0dXJuIHVuZGVmaW5lZDtcbiAgfVxuXG4gIC8qKlxuICAgKiBQYXJzZSBjb21tYW5kLWxpbmUgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gYXJncyAgUmF3IGNvbW1hbmQgbGluZSBhcmd1bWVudHMuXG4gICAqIEBwYXJhbSBvcHRpb25zIFBhcnNlZCBjb21tYW5kIGxpbmUgb3B0aW9ucy5cbiAgICovXG4gIHByb3RlY3RlZCBwYXJzZUFyZ3VtZW50cyhcbiAgICBhcmdzOiBzdHJpbmdbXSxcbiAgICBvcHRpb25zOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgKTogQ0Ege1xuICAgIGNvbnN0IHBhcmFtczogQXJyYXk8dW5rbm93bj4gPSBbXTtcblxuICAgIC8vIHJlbW92ZSBhcnJheSByZWZlcmVuY2VcbiAgICBhcmdzID0gYXJncy5zbGljZSgwKTtcblxuICAgIGlmICghdGhpcy5oYXNBcmd1bWVudHMoKSkge1xuICAgICAgaWYgKGFyZ3MubGVuZ3RoKSB7XG4gICAgICAgIGlmICh0aGlzLmhhc0NvbW1hbmRzKHRydWUpKSB7XG4gICAgICAgICAgdGhyb3cgbmV3IFVua25vd25Db21tYW5kKGFyZ3NbMF0sIHRoaXMuZ2V0Q29tbWFuZHMoKSk7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgdGhyb3cgbmV3IE5vQXJndW1lbnRzQWxsb3dlZCh0aGlzLmdldFBhdGgoKSk7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9IGVsc2Uge1xuICAgICAgaWYgKCFhcmdzLmxlbmd0aCkge1xuICAgICAgICBjb25zdCByZXF1aXJlZCA9IHRoaXMuZ2V0QXJndW1lbnRzKClcbiAgICAgICAgICAuZmlsdGVyKChleHBlY3RlZEFyZykgPT4gIWV4cGVjdGVkQXJnLm9wdGlvbmFsVmFsdWUpXG4gICAgICAgICAgLm1hcCgoZXhwZWN0ZWRBcmcpID0+IGV4cGVjdGVkQXJnLm5hbWUpO1xuXG4gICAgICAgIGlmIChyZXF1aXJlZC5sZW5ndGgpIHtcbiAgICAgICAgICBjb25zdCBvcHRpb25OYW1lczogc3RyaW5nW10gPSBPYmplY3Qua2V5cyhvcHRpb25zKTtcbiAgICAgICAgICBjb25zdCBoYXNTdGFuZGFsb25lT3B0aW9uID0gISFvcHRpb25OYW1lcy5maW5kKChuYW1lKSA9PlxuICAgICAgICAgICAgdGhpcy5nZXRPcHRpb24obmFtZSwgdHJ1ZSk/LnN0YW5kYWxvbmVcbiAgICAgICAgICApO1xuXG4gICAgICAgICAgaWYgKCFoYXNTdGFuZGFsb25lT3B0aW9uKSB7XG4gICAgICAgICAgICB0aHJvdyBuZXcgTWlzc2luZ0FyZ3VtZW50cyhyZXF1aXJlZCk7XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBmb3IgKGNvbnN0IGV4cGVjdGVkQXJnIG9mIHRoaXMuZ2V0QXJndW1lbnRzKCkpIHtcbiAgICAgICAgICBpZiAoIWFyZ3MubGVuZ3RoKSB7XG4gICAgICAgICAgICBpZiAoZXhwZWN0ZWRBcmcub3B0aW9uYWxWYWx1ZSkge1xuICAgICAgICAgICAgICBicmVhaztcbiAgICAgICAgICAgIH1cbiAgICAgICAgICAgIHRocm93IG5ldyBNaXNzaW5nQXJndW1lbnQoYE1pc3NpbmcgYXJndW1lbnQ6ICR7ZXhwZWN0ZWRBcmcubmFtZX1gKTtcbiAgICAgICAgICB9XG5cbiAgICAgICAgICBsZXQgYXJnOiB1bmtub3duO1xuXG4gICAgICAgICAgY29uc3QgcGFyc2VBcmdWYWx1ZSA9ICh2YWx1ZTogc3RyaW5nKSA9PiB7XG4gICAgICAgICAgICByZXR1cm4gZXhwZWN0ZWRBcmcubGlzdFxuICAgICAgICAgICAgICA/IHZhbHVlLnNwbGl0KFwiLFwiKS5tYXAoKHZhbHVlKSA9PiBwYXJzZUFyZ1R5cGUodmFsdWUpKVxuICAgICAgICAgICAgICA6IHBhcnNlQXJnVHlwZSh2YWx1ZSk7XG4gICAgICAgICAgfTtcblxuICAgICAgICAgIGNvbnN0IHBhcnNlQXJnVHlwZSA9ICh2YWx1ZTogc3RyaW5nKSA9PiB7XG4gICAgICAgICAgICByZXR1cm4gdGhpcy5wYXJzZVR5cGUoe1xuICAgICAgICAgICAgICBsYWJlbDogXCJBcmd1bWVudFwiLFxuICAgICAgICAgICAgICB0eXBlOiBleHBlY3RlZEFyZy50eXBlLFxuICAgICAgICAgICAgICBuYW1lOiBleHBlY3RlZEFyZy5uYW1lLFxuICAgICAgICAgICAgICB2YWx1ZSxcbiAgICAgICAgICAgIH0pO1xuICAgICAgICAgIH07XG5cbiAgICAgICAgICBpZiAoZXhwZWN0ZWRBcmcudmFyaWFkaWMpIHtcbiAgICAgICAgICAgIGFyZyA9IGFyZ3Muc3BsaWNlKDAsIGFyZ3MubGVuZ3RoKS5tYXAoKHZhbHVlKSA9PlxuICAgICAgICAgICAgICBwYXJzZUFyZ1ZhbHVlKHZhbHVlKVxuICAgICAgICAgICAgKTtcbiAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgYXJnID0gcGFyc2VBcmdWYWx1ZShhcmdzLnNoaWZ0KCkgYXMgc3RyaW5nKTtcbiAgICAgICAgICB9XG5cbiAgICAgICAgICBpZiAoZXhwZWN0ZWRBcmcudmFyaWFkaWMgJiYgQXJyYXkuaXNBcnJheShhcmcpKSB7XG4gICAgICAgICAgICBwYXJhbXMucHVzaCguLi5hcmcpO1xuICAgICAgICAgIH0gZWxzZSBpZiAodHlwZW9mIGFyZyAhPT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICAgICAgcGFyYW1zLnB1c2goYXJnKTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoYXJncy5sZW5ndGgpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgVG9vTWFueUFyZ3VtZW50cyhhcmdzKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiBwYXJhbXMgYXMgQ0E7XG4gIH1cblxuICAvKipcbiAgICogSGFuZGxlIGVycm9yLiBJZiBgdGhyb3dFcnJvcnNgIGlzIGVuYWJsZWQgdGhlIGVycm9yIHdpbGwgYmUgcmV0dXJuZWQsXG4gICAqIG90aGVyd2lzZSBhIGZvcm1hdHRlZCBlcnJvciBtZXNzYWdlIHdpbGwgYmUgcHJpbnRlZCBhbmQgYERlbm8uZXhpdCgxKWBcbiAgICogd2lsbCBiZSBjYWxsZWQuXG4gICAqIEBwYXJhbSBlcnJvciBFcnJvciB0byBoYW5kbGUuXG4gICAqL1xuICBwcm90ZWN0ZWQgdGhyb3coZXJyb3I6IEVycm9yKTogbmV2ZXIge1xuICAgIGlmICh0aGlzLnNob3VsZFRocm93RXJyb3JzKCkgfHwgIShlcnJvciBpbnN0YW5jZW9mIFZhbGlkYXRpb25FcnJvcikpIHtcbiAgICAgIHRocm93IGVycm9yO1xuICAgIH1cbiAgICB0aGlzLnNob3dIZWxwKCk7XG5cbiAgICBjb25zb2xlLmVycm9yKHJlZChgICAke2JvbGQoXCJlcnJvclwiKX06ICR7ZXJyb3IubWVzc2FnZX1cXG5gKSk7XG5cbiAgICBEZW5vLmV4aXQoZXJyb3IgaW5zdGFuY2VvZiBWYWxpZGF0aW9uRXJyb3IgPyBlcnJvci5leGl0Q29kZSA6IDEpO1xuICB9XG5cbiAgLyoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqIEdFVFRFUiAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKiovXG5cbiAgLyoqIEdldCBjb21tYW5kIG5hbWUuICovXG4gIHB1YmxpYyBnZXROYW1lKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIHRoaXMuX25hbWU7XG4gIH1cblxuICAvKiogR2V0IHBhcmVudCBjb21tYW5kLiAqL1xuICBwdWJsaWMgZ2V0UGFyZW50KCk6IENQIHtcbiAgICByZXR1cm4gdGhpcy5fcGFyZW50IGFzIENQO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBwYXJlbnQgY29tbWFuZCBmcm9tIGdsb2JhbCBleGVjdXRlZCBjb21tYW5kLlxuICAgKiBCZSBzdXJlLCB0byBjYWxsIHRoaXMgbWV0aG9kIG9ubHkgaW5zaWRlIGFuIGFjdGlvbiBoYW5kbGVyLiBVbmxlc3MgdGhpcyBvciBhbnkgY2hpbGQgY29tbWFuZCB3YXMgZXhlY3V0ZWQsXG4gICAqIHRoaXMgbWV0aG9kIHJldHVybnMgYWx3YXlzIHVuZGVmaW5lZC5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxQYXJlbnQoKTogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5fZ2xvYmFsUGFyZW50O1xuICB9XG5cbiAgLyoqIEdldCBtYWluIGNvbW1hbmQuICovXG4gIHB1YmxpYyBnZXRNYWluQ29tbWFuZCgpOiBDb21tYW5kPGFueT4ge1xuICAgIHJldHVybiB0aGlzLl9wYXJlbnQ/LmdldE1haW5Db21tYW5kKCkgPz8gdGhpcztcbiAgfVxuXG4gIC8qKiBHZXQgY29tbWFuZCBuYW1lIGFsaWFzZXMuICovXG4gIHB1YmxpYyBnZXRBbGlhc2VzKCk6IHN0cmluZ1tdIHtcbiAgICByZXR1cm4gdGhpcy5hbGlhc2VzO1xuICB9XG5cbiAgLyoqIEdldCBmdWxsIGNvbW1hbmQgcGF0aC4gKi9cbiAgcHVibGljIGdldFBhdGgoKTogc3RyaW5nIHtcbiAgICByZXR1cm4gdGhpcy5fcGFyZW50XG4gICAgICA/IHRoaXMuX3BhcmVudC5nZXRQYXRoKCkgKyBcIiBcIiArIHRoaXMuX25hbWVcbiAgICAgIDogdGhpcy5fbmFtZTtcbiAgfVxuXG4gIC8qKiBHZXQgYXJndW1lbnRzIGRlZmluaXRpb24uIEUuZzogPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPiAqL1xuICBwdWJsaWMgZ2V0QXJnc0RlZmluaXRpb24oKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5hcmdzRGVmaW5pdGlvbjtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYXJndW1lbnQgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgYXJndW1lbnQuXG4gICAqL1xuICBwdWJsaWMgZ2V0QXJndW1lbnQobmFtZTogc3RyaW5nKTogSUFyZ3VtZW50IHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRBcmd1bWVudHMoKS5maW5kKChhcmcpID0+IGFyZy5uYW1lID09PSBuYW1lKTtcbiAgfVxuXG4gIC8qKiBHZXQgYXJndW1lbnRzLiAqL1xuICBwdWJsaWMgZ2V0QXJndW1lbnRzKCk6IElBcmd1bWVudFtdIHtcbiAgICBpZiAoIXRoaXMuYXJncy5sZW5ndGggJiYgdGhpcy5hcmdzRGVmaW5pdGlvbikge1xuICAgICAgdGhpcy5hcmdzID0gcGFyc2VBcmd1bWVudHNEZWZpbml0aW9uKHRoaXMuYXJnc0RlZmluaXRpb24pO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzLmFyZ3M7XG4gIH1cblxuICAvKiogQ2hlY2sgaWYgY29tbWFuZCBoYXMgYXJndW1lbnRzLiAqL1xuICBwdWJsaWMgaGFzQXJndW1lbnRzKCkge1xuICAgIHJldHVybiAhIXRoaXMuYXJnc0RlZmluaXRpb247XG4gIH1cblxuICAvKiogR2V0IGNvbW1hbmQgdmVyc2lvbi4gKi9cbiAgcHVibGljIGdldFZlcnNpb24oKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRWZXJzaW9uSGFuZGxlcigpPy5jYWxsKHRoaXMsIHRoaXMpO1xuICB9XG5cbiAgLyoqIEdldCBoZWxwIGhhbmRsZXIgbWV0aG9kLiAqL1xuICBwcml2YXRlIGdldFZlcnNpb25IYW5kbGVyKCk6IElWZXJzaW9uSGFuZGxlciB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMudmVyID8/IHRoaXMuX3BhcmVudD8uZ2V0VmVyc2lvbkhhbmRsZXIoKTtcbiAgfVxuXG4gIC8qKiBHZXQgY29tbWFuZCBkZXNjcmlwdGlvbi4gKi9cbiAgcHVibGljIGdldERlc2NyaXB0aW9uKCk6IHN0cmluZyB7XG4gICAgLy8gY2FsbCBkZXNjcmlwdGlvbiBtZXRob2Qgb25seSBvbmNlXG4gICAgcmV0dXJuIHR5cGVvZiB0aGlzLmRlc2MgPT09IFwiZnVuY3Rpb25cIlxuICAgICAgPyB0aGlzLmRlc2MgPSB0aGlzLmRlc2MoKVxuICAgICAgOiB0aGlzLmRlc2M7XG4gIH1cblxuICBwdWJsaWMgZ2V0VXNhZ2UoKSB7XG4gICAgcmV0dXJuIHRoaXMuX3VzYWdlID8/IHRoaXMuZ2V0QXJnc0RlZmluaXRpb24oKTtcbiAgfVxuXG4gIC8qKiBHZXQgc2hvcnQgY29tbWFuZCBkZXNjcmlwdGlvbi4gVGhpcyBpcyB0aGUgZmlyc3QgbGluZSBvZiB0aGUgZGVzY3JpcHRpb24uICovXG4gIHB1YmxpYyBnZXRTaG9ydERlc2NyaXB0aW9uKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIGdldERlc2NyaXB0aW9uKHRoaXMuZ2V0RGVzY3JpcHRpb24oKSwgdHJ1ZSk7XG4gIH1cblxuICAvKiogR2V0IG9yaWdpbmFsIGNvbW1hbmQtbGluZSBhcmd1bWVudHMuICovXG4gIHB1YmxpYyBnZXRSYXdBcmdzKCk6IHN0cmluZ1tdIHtcbiAgICByZXR1cm4gdGhpcy5yYXdBcmdzO1xuICB9XG5cbiAgLyoqIEdldCBhbGwgYXJndW1lbnRzIGRlZmluZWQgYWZ0ZXIgdGhlIGRvdWJsZSBkYXNoLiAqL1xuICBwdWJsaWMgZ2V0TGl0ZXJhbEFyZ3MoKTogc3RyaW5nW10ge1xuICAgIHJldHVybiB0aGlzLmxpdGVyYWxBcmdzO1xuICB9XG5cbiAgLyoqIE91dHB1dCBnZW5lcmF0ZWQgaGVscCB3aXRob3V0IGV4aXRpbmcuICovXG4gIHB1YmxpYyBzaG93VmVyc2lvbigpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldFZlcnNpb24oKSk7XG4gIH1cblxuICAvKiogUmV0dXJucyBjb21tYW5kIG5hbWUsIHZlcnNpb24gYW5kIG1ldGEgZGF0YS4gKi9cbiAgcHVibGljIGdldExvbmdWZXJzaW9uKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIGAke2JvbGQodGhpcy5nZXRNYWluQ29tbWFuZCgpLmdldE5hbWUoKSl9ICR7XG4gICAgICBibHVlKHRoaXMuZ2V0VmVyc2lvbigpID8/IFwiXCIpXG4gICAgfWAgK1xuICAgICAgT2JqZWN0LmVudHJpZXModGhpcy5nZXRNZXRhKCkpLm1hcChcbiAgICAgICAgKFtrLCB2XSkgPT4gYFxcbiR7Ym9sZChrKX0gJHtibHVlKHYpfWAsXG4gICAgICApLmpvaW4oXCJcIik7XG4gIH1cblxuICAvKiogT3V0cHV0cyBjb21tYW5kIG5hbWUsIHZlcnNpb24gYW5kIG1ldGEgZGF0YS4gKi9cbiAgcHVibGljIHNob3dMb25nVmVyc2lvbigpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldExvbmdWZXJzaW9uKCkpO1xuICB9XG5cbiAgLyoqIE91dHB1dCBnZW5lcmF0ZWQgaGVscCB3aXRob3V0IGV4aXRpbmcuICovXG4gIHB1YmxpYyBzaG93SGVscChvcHRpb25zPzogSGVscE9wdGlvbnMpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldEhlbHAob3B0aW9ucykpO1xuICB9XG5cbiAgLyoqIEdldCBnZW5lcmF0ZWQgaGVscC4gKi9cbiAgcHVibGljIGdldEhlbHAob3B0aW9ucz86IEhlbHBPcHRpb25zKTogc3RyaW5nIHtcbiAgICB0aGlzLnJlZ2lzdGVyRGVmYXVsdHMoKTtcbiAgICByZXR1cm4gdGhpcy5nZXRIZWxwSGFuZGxlcigpLmNhbGwodGhpcywgdGhpcywgb3B0aW9ucyA/PyB7fSk7XG4gIH1cblxuICAvKiogR2V0IGhlbHAgaGFuZGxlciBtZXRob2QuICovXG4gIHByaXZhdGUgZ2V0SGVscEhhbmRsZXIoKTogSUhlbHBIYW5kbGVyIHtcbiAgICByZXR1cm4gdGhpcy5faGVscCA/PyB0aGlzLl9wYXJlbnQ/LmdldEhlbHBIYW5kbGVyKCkgYXMgSUhlbHBIYW5kbGVyO1xuICB9XG5cbiAgcHJpdmF0ZSBleGl0KGNvZGUgPSAwKSB7XG4gICAgaWYgKHRoaXMuc2hvdWxkRXhpdCgpKSB7XG4gICAgICBEZW5vLmV4aXQoY29kZSk7XG4gICAgfVxuICB9XG5cbiAgLyoqIENoZWNrIGlmIG5ldyB2ZXJzaW9uIGlzIGF2YWlsYWJsZSBhbmQgYWRkIGhpbnQgdG8gdmVyc2lvbi4gKi9cbiAgcHVibGljIGFzeW5jIGNoZWNrVmVyc2lvbigpOiBQcm9taXNlPHZvaWQ+IHtcbiAgICBjb25zdCBtYWluQ29tbWFuZCA9IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICBjb25zdCB1cGdyYWRlQ29tbWFuZCA9IG1haW5Db21tYW5kLmdldENvbW1hbmQoXCJ1cGdyYWRlXCIpO1xuXG4gICAgaWYgKCFpc1VwZ3JhZGVDb21tYW5kKHVwZ3JhZGVDb21tYW5kKSkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cbiAgICBjb25zdCBsYXRlc3RWZXJzaW9uID0gYXdhaXQgdXBncmFkZUNvbW1hbmQuZ2V0TGF0ZXN0VmVyc2lvbigpO1xuICAgIGNvbnN0IGN1cnJlbnRWZXJzaW9uID0gbWFpbkNvbW1hbmQuZ2V0VmVyc2lvbigpO1xuXG4gICAgaWYgKGN1cnJlbnRWZXJzaW9uID09PSBsYXRlc3RWZXJzaW9uKSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuICAgIGNvbnN0IHZlcnNpb25IZWxwVGV4dCA9XG4gICAgICBgKE5ldyB2ZXJzaW9uIGF2YWlsYWJsZTogJHtsYXRlc3RWZXJzaW9ufS4gUnVuICcke21haW5Db21tYW5kLmdldE5hbWUoKX0gdXBncmFkZScgdG8gdXBncmFkZSB0byB0aGUgbGF0ZXN0IHZlcnNpb24hKWA7XG5cbiAgICBtYWluQ29tbWFuZC52ZXJzaW9uKGAke2N1cnJlbnRWZXJzaW9ufSAgJHtib2xkKHllbGxvdyh2ZXJzaW9uSGVscFRleHQpKX1gKTtcbiAgfVxuXG4gIC8qKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKiBPcHRpb25zIEdFVFRFUiAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqL1xuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgb3B0aW9ucyBvciBub3QuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBoYXNPcHRpb25zKGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gdGhpcy5nZXRPcHRpb25zKGhpZGRlbikubGVuZ3RoID4gMDtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgb3B0aW9ucy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldE9wdGlvbnMoaGlkZGVuPzogYm9vbGVhbik6IElPcHRpb25bXSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsT3B0aW9ucyhoaWRkZW4pLmNvbmNhdCh0aGlzLmdldEJhc2VPcHRpb25zKGhpZGRlbikpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBnZXRCYXNlT3B0aW9ucyhoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbltdIHtcbiAgICBpZiAoIXRoaXMub3B0aW9ucy5sZW5ndGgpIHtcbiAgICAgIHJldHVybiBbXTtcbiAgICB9XG5cbiAgICByZXR1cm4gaGlkZGVuXG4gICAgICA/IHRoaXMub3B0aW9ucy5zbGljZSgwKVxuICAgICAgOiB0aGlzLm9wdGlvbnMuZmlsdGVyKChvcHQpID0+ICFvcHQuaGlkZGVuKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZ2xvYmFsIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxPcHRpb25zKGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uW10ge1xuICAgIGNvbnN0IGhlbHBPcHRpb24gPSB0aGlzLmdldEhlbHBPcHRpb24oKTtcbiAgICBjb25zdCBnZXRHbG9iYWxzID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4sXG4gICAgICBub0dsb2JhbHM6IGJvb2xlYW4sXG4gICAgICBvcHRpb25zOiBJT3B0aW9uW10gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IElPcHRpb25bXSA9PiB7XG4gICAgICBpZiAoY21kLm9wdGlvbnMubGVuZ3RoKSB7XG4gICAgICAgIGZvciAoY29uc3Qgb3B0aW9uIG9mIGNtZC5vcHRpb25zKSB7XG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgb3B0aW9uLmdsb2JhbCAmJlxuICAgICAgICAgICAgIXRoaXMub3B0aW9ucy5maW5kKChvcHQpID0+IG9wdC5uYW1lID09PSBvcHRpb24ubmFtZSkgJiZcbiAgICAgICAgICAgIG5hbWVzLmluZGV4T2Yob3B0aW9uLm5hbWUpID09PSAtMSAmJlxuICAgICAgICAgICAgKGhpZGRlbiB8fCAhb3B0aW9uLmhpZGRlbilcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGlmIChub0dsb2JhbHMgJiYgb3B0aW9uICE9PSBoZWxwT3B0aW9uKSB7XG4gICAgICAgICAgICAgIGNvbnRpbnVlO1xuICAgICAgICAgICAgfVxuXG4gICAgICAgICAgICBuYW1lcy5wdXNoKG9wdGlvbi5uYW1lKTtcbiAgICAgICAgICAgIG9wdGlvbnMucHVzaChvcHRpb24pO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuICAgICAgfVxuXG4gICAgICByZXR1cm4gY21kLl9wYXJlbnRcbiAgICAgICAgPyBnZXRHbG9iYWxzKFxuICAgICAgICAgIGNtZC5fcGFyZW50LFxuICAgICAgICAgIG5vR2xvYmFscyB8fCBjbWQuX25vR2xvYmFscyxcbiAgICAgICAgICBvcHRpb25zLFxuICAgICAgICAgIG5hbWVzLFxuICAgICAgICApXG4gICAgICAgIDogb3B0aW9ucztcbiAgICB9O1xuXG4gICAgcmV0dXJuIHRoaXMuX3BhcmVudCA/IGdldEdsb2JhbHModGhpcy5fcGFyZW50LCB0aGlzLl9ub0dsb2JhbHMpIDogW107XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGFuIG9wdGlvbiB3aXRoIGdpdmVuIG5hbWUgb3Igbm90LlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhhc09wdGlvbihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gISF0aGlzLmdldE9wdGlvbihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBvcHRpb24gYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgb3B0aW9uLiBNdXN0IGJlIGluIHBhcmFtLWNhc2UuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBnZXRPcHRpb24obmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZU9wdGlvbihuYW1lLCBoaWRkZW4pID8/XG4gICAgICB0aGlzLmdldEdsb2JhbE9wdGlvbihuYW1lLCBoaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIG9wdGlvbiBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VPcHRpb24obmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgY29uc3Qgb3B0aW9uID0gdGhpcy5vcHRpb25zLmZpbmQoKG9wdGlvbikgPT5cbiAgICAgIG9wdGlvbi5uYW1lID09PSBuYW1lIHx8IG9wdGlvbi5hbGlhc2VzPy5pbmNsdWRlcyhuYW1lKVxuICAgICk7XG5cbiAgICByZXR1cm4gb3B0aW9uICYmIChoaWRkZW4gfHwgIW9wdGlvbi5oaWRkZW4pID8gb3B0aW9uIDogdW5kZWZpbmVkO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgb3B0aW9uIGZyb20gcGFyZW50IGNvbW1hbmRzIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsT3B0aW9uKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IElPcHRpb24gfCB1bmRlZmluZWQge1xuICAgIGNvbnN0IGhlbHBPcHRpb24gPSB0aGlzLmdldEhlbHBPcHRpb24oKTtcbiAgICBjb25zdCBnZXRHbG9iYWxPcHRpb24gPSAoXG4gICAgICBwYXJlbnQ6IENvbW1hbmQsXG4gICAgICBub0dsb2JhbHM6IGJvb2xlYW4sXG4gICAgKTogSU9wdGlvbiB8IHVuZGVmaW5lZCA9PiB7XG4gICAgICBjb25zdCBvcHRpb246IElPcHRpb24gfCB1bmRlZmluZWQgPSBwYXJlbnQuZ2V0QmFzZU9wdGlvbihcbiAgICAgICAgbmFtZSxcbiAgICAgICAgaGlkZGVuLFxuICAgICAgKTtcblxuICAgICAgaWYgKCFvcHRpb24/Lmdsb2JhbCkge1xuICAgICAgICByZXR1cm4gcGFyZW50Ll9wYXJlbnQgJiYgZ2V0R2xvYmFsT3B0aW9uKFxuICAgICAgICAgIHBhcmVudC5fcGFyZW50LFxuICAgICAgICAgIG5vR2xvYmFscyB8fCBwYXJlbnQuX25vR2xvYmFscyxcbiAgICAgICAgKTtcbiAgICAgIH1cbiAgICAgIGlmIChub0dsb2JhbHMgJiYgb3B0aW9uICE9PSBoZWxwT3B0aW9uKSB7XG4gICAgICAgIHJldHVybjtcbiAgICAgIH1cblxuICAgICAgcmV0dXJuIG9wdGlvbjtcbiAgICB9O1xuXG4gICAgcmV0dXJuIHRoaXMuX3BhcmVudCAmJiBnZXRHbG9iYWxPcHRpb24oXG4gICAgICB0aGlzLl9wYXJlbnQsXG4gICAgICB0aGlzLl9ub0dsb2JhbHMsXG4gICAgKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW1vdmUgb3B0aW9uIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKi9cbiAgcHVibGljIHJlbW92ZU9wdGlvbihuYW1lOiBzdHJpbmcpOiBJT3B0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBpbmRleCA9IHRoaXMub3B0aW9ucy5maW5kSW5kZXgoKG9wdGlvbikgPT4gb3B0aW9uLm5hbWUgPT09IG5hbWUpO1xuXG4gICAgaWYgKGluZGV4ID09PSAtMSkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzLm9wdGlvbnMuc3BsaWNlKGluZGV4LCAxKVswXTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgc3ViLWNvbW1hbmRzIG9yIG5vdC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBoYXNDb21tYW5kcyhoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0Q29tbWFuZHMoaGlkZGVuKS5sZW5ndGggPiAwO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBjb21tYW5kcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRDb21tYW5kcyhoaWRkZW4/OiBib29sZWFuKTogQXJyYXk8Q29tbWFuZDxhbnk+PiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsQ29tbWFuZHMoaGlkZGVuKS5jb25jYXQodGhpcy5nZXRCYXNlQ29tbWFuZHMoaGlkZGVuKSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgY29tbWFuZHMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbW1hbmRzKGhpZGRlbj86IGJvb2xlYW4pOiBBcnJheTxDb21tYW5kPGFueT4+IHtcbiAgICBjb25zdCBjb21tYW5kcyA9IEFycmF5LmZyb20odGhpcy5jb21tYW5kcy52YWx1ZXMoKSk7XG4gICAgcmV0dXJuIGhpZGRlbiA/IGNvbW1hbmRzIDogY29tbWFuZHMuZmlsdGVyKChjbWQpID0+ICFjbWQuaXNIaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgY29tbWFuZHMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tbWFuZHMoaGlkZGVuPzogYm9vbGVhbik6IEFycmF5PENvbW1hbmQ8YW55Pj4ge1xuICAgIGNvbnN0IGdldENvbW1hbmRzID0gKFxuICAgICAgY29tbWFuZDogQ29tbWFuZDxhbnk+LFxuICAgICAgbm9HbG9iYWxzOiBib29sZWFuLFxuICAgICAgY29tbWFuZHM6IEFycmF5PENvbW1hbmQ8YW55Pj4gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IEFycmF5PENvbW1hbmQ8YW55Pj4gPT4ge1xuICAgICAgaWYgKGNvbW1hbmQuY29tbWFuZHMuc2l6ZSkge1xuICAgICAgICBmb3IgKGNvbnN0IFtfLCBjbWRdIG9mIGNvbW1hbmQuY29tbWFuZHMpIHtcbiAgICAgICAgICBpZiAoXG4gICAgICAgICAgICBjbWQuaXNHbG9iYWwgJiZcbiAgICAgICAgICAgIHRoaXMgIT09IGNtZCAmJlxuICAgICAgICAgICAgIXRoaXMuY29tbWFuZHMuaGFzKGNtZC5fbmFtZSkgJiZcbiAgICAgICAgICAgIG5hbWVzLmluZGV4T2YoY21kLl9uYW1lKSA9PT0gLTEgJiZcbiAgICAgICAgICAgIChoaWRkZW4gfHwgIWNtZC5pc0hpZGRlbilcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGlmIChub0dsb2JhbHMgJiYgY21kPy5nZXROYW1lKCkgIT09IFwiaGVscFwiKSB7XG4gICAgICAgICAgICAgIGNvbnRpbnVlO1xuICAgICAgICAgICAgfVxuXG4gICAgICAgICAgICBuYW1lcy5wdXNoKGNtZC5fbmFtZSk7XG4gICAgICAgICAgICBjb21tYW5kcy5wdXNoKGNtZCk7XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIHJldHVybiBjb21tYW5kLl9wYXJlbnRcbiAgICAgICAgPyBnZXRDb21tYW5kcyhcbiAgICAgICAgICBjb21tYW5kLl9wYXJlbnQsXG4gICAgICAgICAgbm9HbG9iYWxzIHx8IGNvbW1hbmQuX25vR2xvYmFscyxcbiAgICAgICAgICBjb21tYW5kcyxcbiAgICAgICAgICBuYW1lcyxcbiAgICAgICAgKVxuICAgICAgICA6IGNvbW1hbmRzO1xuICAgIH07XG5cbiAgICByZXR1cm4gdGhpcy5fcGFyZW50ID8gZ2V0Q29tbWFuZHModGhpcy5fcGFyZW50LCB0aGlzLl9ub0dsb2JhbHMpIDogW107XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgYSBjaGlsZCBjb21tYW5kIGV4aXN0cyBieSBnaXZlbiBuYW1lIG9yIGFsaWFzLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9yIGFsaWFzIG9mIHRoZSBjb21tYW5kLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGhhc0NvbW1hbmQobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuICEhdGhpcy5nZXRDb21tYW5kKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGNvbW1hbmQgYnkgbmFtZSBvciBhbGlhcy5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvciBhbGlhcyBvZiB0aGUgY29tbWFuZC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRDb21tYW5kPEMgZXh0ZW5kcyBDb21tYW5kPGFueT4+KFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBoaWRkZW4/OiBib29sZWFuLFxuICApOiBDIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlQ29tbWFuZChuYW1lLCBoaWRkZW4pID8/XG4gICAgICB0aGlzLmdldEdsb2JhbENvbW1hbmQobmFtZSwgaGlkZGVuKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBjb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbW1hbmQ8QyBleHRlbmRzIENvbW1hbmQ8YW55Pj4oXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGhpZGRlbj86IGJvb2xlYW4sXG4gICk6IEMgfCB1bmRlZmluZWQge1xuICAgIGZvciAoY29uc3QgY21kIG9mIHRoaXMuY29tbWFuZHMudmFsdWVzKCkpIHtcbiAgICAgIGlmIChjbWQuX25hbWUgPT09IG5hbWUgfHwgY21kLmFsaWFzZXMuaW5jbHVkZXMobmFtZSkpIHtcbiAgICAgICAgcmV0dXJuIChjbWQgJiYgKGhpZGRlbiB8fCAhY21kLmlzSGlkZGVuKSA/IGNtZCA6IHVuZGVmaW5lZCkgYXNcbiAgICAgICAgICB8IENcbiAgICAgICAgICB8IHVuZGVmaW5lZDtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBjb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tbWFuZDxDIGV4dGVuZHMgQ29tbWFuZDxhbnk+PihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgaGlkZGVuPzogYm9vbGVhbixcbiAgKTogQyB8IHVuZGVmaW5lZCB7XG4gICAgY29uc3QgZ2V0R2xvYmFsQ29tbWFuZCA9IChcbiAgICAgIHBhcmVudDogQ29tbWFuZCxcbiAgICAgIG5vR2xvYmFsczogYm9vbGVhbixcbiAgICApOiBDb21tYW5kIHwgdW5kZWZpbmVkID0+IHtcbiAgICAgIGNvbnN0IGNtZDogQ29tbWFuZCB8IHVuZGVmaW5lZCA9IHBhcmVudC5nZXRCYXNlQ29tbWFuZChuYW1lLCBoaWRkZW4pO1xuXG4gICAgICBpZiAoIWNtZD8uaXNHbG9iYWwpIHtcbiAgICAgICAgcmV0dXJuIHBhcmVudC5fcGFyZW50ICYmXG4gICAgICAgICAgZ2V0R2xvYmFsQ29tbWFuZChwYXJlbnQuX3BhcmVudCwgbm9HbG9iYWxzIHx8IHBhcmVudC5fbm9HbG9iYWxzKTtcbiAgICAgIH1cbiAgICAgIGlmIChub0dsb2JhbHMgJiYgY21kLmdldE5hbWUoKSAhPT0gXCJoZWxwXCIpIHtcbiAgICAgICAgcmV0dXJuO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gY21kO1xuICAgIH07XG5cbiAgICByZXR1cm4gdGhpcy5fcGFyZW50ICYmIGdldEdsb2JhbENvbW1hbmQodGhpcy5fcGFyZW50LCB0aGlzLl9ub0dsb2JhbHMpIGFzIEM7XG4gIH1cblxuICAvKipcbiAgICogUmVtb3ZlIHN1Yi1jb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqL1xuICBwdWJsaWMgcmVtb3ZlQ29tbWFuZChuYW1lOiBzdHJpbmcpOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQge1xuICAgIGNvbnN0IGNvbW1hbmQgPSB0aGlzLmdldEJhc2VDb21tYW5kKG5hbWUsIHRydWUpO1xuXG4gICAgaWYgKGNvbW1hbmQpIHtcbiAgICAgIHRoaXMuY29tbWFuZHMuZGVsZXRlKGNvbW1hbmQuX25hbWUpO1xuICAgIH1cblxuICAgIHJldHVybiBjb21tYW5kO1xuICB9XG5cbiAgLyoqIEdldCB0eXBlcy4gKi9cbiAgcHVibGljIGdldFR5cGVzKCk6IElUeXBlW10ge1xuICAgIHJldHVybiB0aGlzLmdldEdsb2JhbFR5cGVzKCkuY29uY2F0KHRoaXMuZ2V0QmFzZVR5cGVzKCkpO1xuICB9XG5cbiAgLyoqIEdldCBiYXNlIHR5cGVzLiAqL1xuICBwdWJsaWMgZ2V0QmFzZVR5cGVzKCk6IElUeXBlW10ge1xuICAgIHJldHVybiBBcnJheS5mcm9tKHRoaXMudHlwZXMudmFsdWVzKCkpO1xuICB9XG5cbiAgLyoqIEdldCBnbG9iYWwgdHlwZXMuICovXG4gIHB1YmxpYyBnZXRHbG9iYWxUeXBlcygpOiBJVHlwZVtdIHtcbiAgICBjb25zdCBnZXRUeXBlcyA9IChcbiAgICAgIGNtZDogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkLFxuICAgICAgdHlwZXM6IElUeXBlW10gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IElUeXBlW10gPT4ge1xuICAgICAgaWYgKGNtZCkge1xuICAgICAgICBpZiAoY21kLnR5cGVzLnNpemUpIHtcbiAgICAgICAgICBjbWQudHlwZXMuZm9yRWFjaCgodHlwZTogSVR5cGUpID0+IHtcbiAgICAgICAgICAgIGlmIChcbiAgICAgICAgICAgICAgdHlwZS5nbG9iYWwgJiZcbiAgICAgICAgICAgICAgIXRoaXMudHlwZXMuaGFzKHR5cGUubmFtZSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZih0eXBlLm5hbWUpID09PSAtMVxuICAgICAgICAgICAgKSB7XG4gICAgICAgICAgICAgIG5hbWVzLnB1c2godHlwZS5uYW1lKTtcbiAgICAgICAgICAgICAgdHlwZXMucHVzaCh0eXBlKTtcbiAgICAgICAgICAgIH1cbiAgICAgICAgICB9KTtcbiAgICAgICAgfVxuXG4gICAgICAgIHJldHVybiBnZXRUeXBlcyhjbWQuX3BhcmVudCwgdHlwZXMsIG5hbWVzKTtcbiAgICAgIH1cblxuICAgICAgcmV0dXJuIHR5cGVzO1xuICAgIH07XG5cbiAgICByZXR1cm4gZ2V0VHlwZXModGhpcy5fcGFyZW50KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgdHlwZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSB0eXBlLlxuICAgKi9cbiAgcHVibGljIGdldFR5cGUobmFtZTogc3RyaW5nKTogSVR5cGUgfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLmdldEJhc2VUeXBlKG5hbWUpID8/IHRoaXMuZ2V0R2xvYmFsVHlwZShuYW1lKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSB0eXBlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIHR5cGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZVR5cGUobmFtZTogc3RyaW5nKTogSVR5cGUgfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLnR5cGVzLmdldChuYW1lKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZ2xvYmFsIHR5cGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgdHlwZS5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxUeXBlKG5hbWU6IHN0cmluZyk6IElUeXBlIHwgdW5kZWZpbmVkIHtcbiAgICBpZiAoIXRoaXMuX3BhcmVudCkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIGNvbnN0IGNtZDogSVR5cGUgfCB1bmRlZmluZWQgPSB0aGlzLl9wYXJlbnQuZ2V0QmFzZVR5cGUobmFtZSk7XG5cbiAgICBpZiAoIWNtZD8uZ2xvYmFsKSB7XG4gICAgICByZXR1cm4gdGhpcy5fcGFyZW50LmdldEdsb2JhbFR5cGUobmFtZSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIGNtZDtcbiAgfVxuXG4gIC8qKiBHZXQgY29tcGxldGlvbnMuICovXG4gIHB1YmxpYyBnZXRDb21wbGV0aW9ucygpIHtcbiAgICByZXR1cm4gdGhpcy5nZXRHbG9iYWxDb21wbGV0aW9ucygpLmNvbmNhdCh0aGlzLmdldEJhc2VDb21wbGV0aW9ucygpKTtcbiAgfVxuXG4gIC8qKiBHZXQgYmFzZSBjb21wbGV0aW9ucy4gKi9cbiAgcHVibGljIGdldEJhc2VDb21wbGV0aW9ucygpOiBJQ29tcGxldGlvbltdIHtcbiAgICByZXR1cm4gQXJyYXkuZnJvbSh0aGlzLmNvbXBsZXRpb25zLnZhbHVlcygpKTtcbiAgfVxuXG4gIC8qKiBHZXQgZ2xvYmFsIGNvbXBsZXRpb25zLiAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tcGxldGlvbnMoKTogSUNvbXBsZXRpb25bXSB7XG4gICAgY29uc3QgZ2V0Q29tcGxldGlvbnMgPSAoXG4gICAgICBjbWQ6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCxcbiAgICAgIGNvbXBsZXRpb25zOiBJQ29tcGxldGlvbltdID0gW10sXG4gICAgICBuYW1lczogc3RyaW5nW10gPSBbXSxcbiAgICApOiBJQ29tcGxldGlvbltdID0+IHtcbiAgICAgIGlmIChjbWQpIHtcbiAgICAgICAgaWYgKGNtZC5jb21wbGV0aW9ucy5zaXplKSB7XG4gICAgICAgICAgY21kLmNvbXBsZXRpb25zLmZvckVhY2goKGNvbXBsZXRpb246IElDb21wbGV0aW9uKSA9PiB7XG4gICAgICAgICAgICBpZiAoXG4gICAgICAgICAgICAgIGNvbXBsZXRpb24uZ2xvYmFsICYmXG4gICAgICAgICAgICAgICF0aGlzLmNvbXBsZXRpb25zLmhhcyhjb21wbGV0aW9uLm5hbWUpICYmXG4gICAgICAgICAgICAgIG5hbWVzLmluZGV4T2YoY29tcGxldGlvbi5uYW1lKSA9PT0gLTFcbiAgICAgICAgICAgICkge1xuICAgICAgICAgICAgICBuYW1lcy5wdXNoKGNvbXBsZXRpb24ubmFtZSk7XG4gICAgICAgICAgICAgIGNvbXBsZXRpb25zLnB1c2goY29tcGxldGlvbik7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0Q29tcGxldGlvbnMoY21kLl9wYXJlbnQsIGNvbXBsZXRpb25zLCBuYW1lcyk7XG4gICAgICB9XG5cbiAgICAgIHJldHVybiBjb21wbGV0aW9ucztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldENvbXBsZXRpb25zKHRoaXMuX3BhcmVudCk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGNvbXBsZXRpb24gYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgY29tcGxldGlvbi5cbiAgICovXG4gIHB1YmxpYyBnZXRDb21wbGV0aW9uKG5hbWU6IHN0cmluZyk6IElDb21wbGV0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlQ29tcGxldGlvbihuYW1lKSA/PyB0aGlzLmdldEdsb2JhbENvbXBsZXRpb24obmFtZSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgY29tcGxldGlvbiBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBjb21wbGV0aW9uLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VDb21wbGV0aW9uKG5hbWU6IHN0cmluZyk6IElDb21wbGV0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5jb21wbGV0aW9ucy5nZXQobmFtZSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBjb21wbGV0aW9ucyBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBjb21wbGV0aW9uLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbENvbXBsZXRpb24obmFtZTogc3RyaW5nKTogSUNvbXBsZXRpb24gfCB1bmRlZmluZWQge1xuICAgIGlmICghdGhpcy5fcGFyZW50KSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3QgY29tcGxldGlvbjogSUNvbXBsZXRpb24gfCB1bmRlZmluZWQgPSB0aGlzLl9wYXJlbnQuZ2V0QmFzZUNvbXBsZXRpb24oXG4gICAgICBuYW1lLFxuICAgICk7XG5cbiAgICBpZiAoIWNvbXBsZXRpb24/Lmdsb2JhbCkge1xuICAgICAgcmV0dXJuIHRoaXMuX3BhcmVudC5nZXRHbG9iYWxDb21wbGV0aW9uKG5hbWUpO1xuICAgIH1cblxuICAgIHJldHVybiBjb21wbGV0aW9uO1xuICB9XG5cbiAgLyoqXG4gICAqIENoZWNrcyB3aGV0aGVyIHRoZSBjb21tYW5kIGhhcyBlbnZpcm9ubWVudCB2YXJpYWJsZXMgb3Igbm90LlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGhhc0VudlZhcnMoaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmdldEVudlZhcnMoaGlkZGVuKS5sZW5ndGggPiAwO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBlbnZpcm9ubWVudCB2YXJpYWJsZXMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0RW52VmFycyhoaWRkZW4/OiBib29sZWFuKTogSUVudlZhcltdIHtcbiAgICByZXR1cm4gdGhpcy5nZXRHbG9iYWxFbnZWYXJzKGhpZGRlbikuY29uY2F0KHRoaXMuZ2V0QmFzZUVudlZhcnMoaGlkZGVuKSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgZW52aXJvbm1lbnQgdmFyaWFibGVzLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VFbnZWYXJzKGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyW10ge1xuICAgIGlmICghdGhpcy5lbnZWYXJzLmxlbmd0aCkge1xuICAgICAgcmV0dXJuIFtdO1xuICAgIH1cblxuICAgIHJldHVybiBoaWRkZW5cbiAgICAgID8gdGhpcy5lbnZWYXJzLnNsaWNlKDApXG4gICAgICA6IHRoaXMuZW52VmFycy5maWx0ZXIoKGVudikgPT4gIWVudi5oaWRkZW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgZW52aXJvbm1lbnQgdmFyaWFibGVzLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEdsb2JhbEVudlZhcnMoaGlkZGVuPzogYm9vbGVhbik6IElFbnZWYXJbXSB7XG4gICAgaWYgKHRoaXMuX25vR2xvYmFscykge1xuICAgICAgcmV0dXJuIFtdO1xuICAgIH1cblxuICAgIGNvbnN0IGdldEVudlZhcnMgPSAoXG4gICAgICBjbWQ6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCxcbiAgICAgIGVudlZhcnM6IElFbnZWYXJbXSA9IFtdLFxuICAgICAgbmFtZXM6IHN0cmluZ1tdID0gW10sXG4gICAgKTogSUVudlZhcltdID0+IHtcbiAgICAgIGlmIChjbWQpIHtcbiAgICAgICAgaWYgKGNtZC5lbnZWYXJzLmxlbmd0aCkge1xuICAgICAgICAgIGNtZC5lbnZWYXJzLmZvckVhY2goKGVudlZhcjogSUVudlZhcikgPT4ge1xuICAgICAgICAgICAgaWYgKFxuICAgICAgICAgICAgICBlbnZWYXIuZ2xvYmFsICYmXG4gICAgICAgICAgICAgICF0aGlzLmVudlZhcnMuZmluZCgoZW52KSA9PiBlbnYubmFtZXNbMF0gPT09IGVudlZhci5uYW1lc1swXSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZihlbnZWYXIubmFtZXNbMF0pID09PSAtMSAmJlxuICAgICAgICAgICAgICAoaGlkZGVuIHx8ICFlbnZWYXIuaGlkZGVuKVxuICAgICAgICAgICAgKSB7XG4gICAgICAgICAgICAgIG5hbWVzLnB1c2goZW52VmFyLm5hbWVzWzBdKTtcbiAgICAgICAgICAgICAgZW52VmFycy5wdXNoKGVudlZhcik7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0RW52VmFycyhjbWQuX3BhcmVudCwgZW52VmFycywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gZW52VmFycztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldEVudlZhcnModGhpcy5fcGFyZW50KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgYW4gZW52aXJvbm1lbnQgdmFyaWFibGUgd2l0aCBnaXZlbiBuYW1lIG9yIG5vdC5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgaGFzRW52VmFyKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0RW52VmFyKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGVudmlyb25tZW50IHZhcmlhYmxlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEVudlZhcihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlRW52VmFyKG5hbWUsIGhpZGRlbikgPz9cbiAgICAgIHRoaXMuZ2V0R2xvYmFsRW52VmFyKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgZW52aXJvbm1lbnQgdmFyaWFibGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUVudlZhcihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyIHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBlbnZWYXI6IElFbnZWYXIgfCB1bmRlZmluZWQgPSB0aGlzLmVudlZhcnMuZmluZCgoZW52KSA9PlxuICAgICAgZW52Lm5hbWVzLmluZGV4T2YobmFtZSkgIT09IC0xXG4gICAgKTtcblxuICAgIHJldHVybiBlbnZWYXIgJiYgKGhpZGRlbiB8fCAhZW52VmFyLmhpZGRlbikgPyBlbnZWYXIgOiB1bmRlZmluZWQ7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBlbnZpcm9ubWVudCB2YXJpYWJsZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxFbnZWYXIobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSUVudlZhciB8IHVuZGVmaW5lZCB7XG4gICAgaWYgKCF0aGlzLl9wYXJlbnQgfHwgdGhpcy5fbm9HbG9iYWxzKSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3QgZW52VmFyOiBJRW52VmFyIHwgdW5kZWZpbmVkID0gdGhpcy5fcGFyZW50LmdldEJhc2VFbnZWYXIoXG4gICAgICBuYW1lLFxuICAgICAgaGlkZGVuLFxuICAgICk7XG5cbiAgICBpZiAoIWVudlZhcj8uZ2xvYmFsKSB7XG4gICAgICByZXR1cm4gdGhpcy5fcGFyZW50LmdldEdsb2JhbEVudlZhcihuYW1lLCBoaWRkZW4pO1xuICAgIH1cblxuICAgIHJldHVybiBlbnZWYXI7XG4gIH1cblxuICAvKiogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGV4YW1wbGVzIG9yIG5vdC4gKi9cbiAgcHVibGljIGhhc0V4YW1wbGVzKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmV4YW1wbGVzLmxlbmd0aCA+IDA7XG4gIH1cblxuICAvKiogR2V0IGFsbCBleGFtcGxlcy4gKi9cbiAgcHVibGljIGdldEV4YW1wbGVzKCk6IElFeGFtcGxlW10ge1xuICAgIHJldHVybiB0aGlzLmV4YW1wbGVzO1xuICB9XG5cbiAgLyoqIENoZWNrcyB3aGV0aGVyIHRoZSBjb21tYW5kIGhhcyBhbiBleGFtcGxlIHdpdGggZ2l2ZW4gbmFtZSBvciBub3QuICovXG4gIHB1YmxpYyBoYXNFeGFtcGxlKG5hbWU6IHN0cmluZyk6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0RXhhbXBsZShuYW1lKTtcbiAgfVxuXG4gIC8qKiBHZXQgZXhhbXBsZSB3aXRoIGdpdmVuIG5hbWUuICovXG4gIHB1YmxpYyBnZXRFeGFtcGxlKG5hbWU6IHN0cmluZyk6IElFeGFtcGxlIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5leGFtcGxlcy5maW5kKChleGFtcGxlKSA9PiBleGFtcGxlLm5hbWUgPT09IG5hbWUpO1xuICB9XG5cbiAgcHJpdmF0ZSBnZXRIZWxwT3B0aW9uKCk6IElPcHRpb24gfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLl9oZWxwT3B0aW9uID8/IHRoaXMuX3BhcmVudD8uZ2V0SGVscE9wdGlvbigpO1xuICB9XG59XG5cbmZ1bmN0aW9uIGlzVXBncmFkZUNvbW1hbmQoY29tbWFuZDogdW5rbm93bik6IGNvbW1hbmQgaXMgVXBncmFkZUNvbW1hbmRJbXBsIHtcbiAgcmV0dXJuIGNvbW1hbmQgaW5zdGFuY2VvZiBDb21tYW5kICYmIFwiZ2V0TGF0ZXN0VmVyc2lvblwiIGluIGNvbW1hbmQ7XG59XG5cbmludGVyZmFjZSBVcGdyYWRlQ29tbWFuZEltcGwge1xuICBnZXRMYXRlc3RWZXJzaW9uKCk6IFByb21pc2U8c3RyaW5nPjtcbn1cblxuaW50ZXJmYWNlIElEZWZhdWx0T3B0aW9uIHtcbiAgZmxhZ3M6IHN0cmluZztcbiAgZGVzYz86IHN0cmluZztcbiAgb3B0cz86IElDb21tYW5kT3B0aW9uO1xufVxuXG50eXBlIEFjdGlvbk9wdGlvbiA9IElPcHRpb24gJiB7IGFjdGlvbjogSUFjdGlvbiB9O1xuXG5pbnRlcmZhY2UgUGFyc2VDb250ZXh0IHtcbiAgYXJnczogc3RyaW5nW107XG4gIG9wdGlvbnM/OiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPjtcbiAgZW52PzogUmVjb3JkPHN0cmluZywgdW5rbm93bj47XG4gIGFjdGlvbj86IEFjdGlvbk9wdGlvbjtcbiAgbGl0ZXJhbD86IHN0cmluZ1tdO1xufVxuXG50eXBlIFRyaW1MZWZ0PFQgZXh0ZW5kcyBzdHJpbmcsIFYgZXh0ZW5kcyBzdHJpbmcgfCB1bmRlZmluZWQ+ID0gVCBleHRlbmRzXG4gIGAke1Z9JHtpbmZlciBVfWAgPyBVXG4gIDogVDtcblxudHlwZSBUcmltUmlnaHQ8VCBleHRlbmRzIHN0cmluZywgViBleHRlbmRzIHN0cmluZz4gPSBUIGV4dGVuZHMgYCR7aW5mZXIgVX0ke1Z9YFxuICA/IFVcbiAgOiBUO1xuXG50eXBlIExvd2VyPFYgZXh0ZW5kcyBzdHJpbmc+ID0gViBleHRlbmRzIFVwcGVyY2FzZTxWPiA/IExvd2VyY2FzZTxWPlxuICA6IFVuY2FwaXRhbGl6ZTxWPjtcblxudHlwZSBDYW1lbENhc2U8VCBleHRlbmRzIHN0cmluZz4gPSBUIGV4dGVuZHMgYCR7aW5mZXIgVn1fJHtpbmZlciBSZXN0fWBcbiAgPyBgJHtMb3dlcjxWPn0ke0NhcGl0YWxpemU8Q2FtZWxDYXNlPFJlc3Q+Pn1gXG4gIDogVCBleHRlbmRzIGAke2luZmVyIFZ9LSR7aW5mZXIgUmVzdH1gXG4gICAgPyBgJHtMb3dlcjxWPn0ke0NhcGl0YWxpemU8Q2FtZWxDYXNlPFJlc3Q+Pn1gXG4gIDogTG93ZXI8VD47XG5cbnR5cGUgT25lT2Y8VCwgVj4gPSBUIGV4dGVuZHMgdm9pZCA/IFYgOiBUO1xuXG50eXBlIE1lcmdlPEwsIFI+ID0gTCBleHRlbmRzIHZvaWQgPyBSXG4gIDogUiBleHRlbmRzIHZvaWQgPyBMXG4gIDogTCAmIFI7XG5cbi8vIHR5cGUgTWVyZ2U8TCwgUj4gPSBMIGV4dGVuZHMgdm9pZCA/IFJcbi8vICAgOiBSIGV4dGVuZHMgdm9pZCA/IExcbi8vICAgOiBPbWl0PEwsIGtleW9mIFI+ICYgUjtcblxudHlwZSBNZXJnZVJlY3Vyc2l2ZTxMLCBSPiA9IEwgZXh0ZW5kcyB2b2lkID8gUlxuICA6IFIgZXh0ZW5kcyB2b2lkID8gTFxuICA6IEwgJiBSO1xuXG50eXBlIE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFQgZXh0ZW5kcyBzdHJpbmc+ID0gYFske1R9XWAgfCBgPCR7VH0+YDtcbnR5cGUgUmVzdFZhbHVlID0gYC4uLiR7c3RyaW5nfWAgfCBgJHtzdHJpbmd9Li4uYDtcblxuLyoqXG4gKiBSZXN0IGFyZ3Mgd2l0aCBsaXN0IHR5cGUgYW5kIGNvbXBsZXRpb25zLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGVbXTpjb21wbGV0aW9uXWBcbiAqIC0gYDwuLi5uYW1lOnR5cGVbXTpjb21wbGV0aW9uPmBcbiAqIC0gYFtuYW1lLi4uOnR5cGVbXTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGVbXTpjb21wbGV0aW9uPmBcbiAqL1xudHlwZSBSZXN0QXJnc0xpc3RUeXBlQ29tcGxldGlvbjxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH1bXToke3N0cmluZ31gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIGxpc3QgdHlwZS5cbiAqXG4gKiAtIGBbLi4ubmFtZTp0eXBlW11dYFxuICogLSBgPC4uLm5hbWU6dHlwZVtdPmBcbiAqIC0gYFtuYW1lLi4uOnR5cGVbXV1gXG4gKiAtIGA8bmFtZS4uLjp0eXBlW10+YFxuICovXG50eXBlIFJlc3RBcmdzTGlzdFR5cGU8VCBleHRlbmRzIHN0cmluZz4gPSBPcHRpb25hbE9yUmVxdWlyZWRWYWx1ZTxcbiAgYCR7UmVzdFZhbHVlfToke1R9W11gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIHR5cGUgYW5kIGNvbXBsZXRpb25zLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGU6Y29tcGxldGlvbl1gXG4gKiAtIGA8Li4ubmFtZTp0eXBlOmNvbXBsZXRpb24+YFxuICogLSBgW25hbWUuLi46dHlwZTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGU6Y29tcGxldGlvbj5gXG4gKi9cbnR5cGUgUmVzdEFyZ3NUeXBlQ29tcGxldGlvbjxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH06JHtzdHJpbmd9YFxuPjtcblxuLyoqXG4gKiBSZXN0IGFyZ3Mgd2l0aCB0eXBlLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGVdYFxuICogLSBgPC4uLm5hbWU6dHlwZT5gXG4gKiAtIGBbbmFtZS4uLjp0eXBlXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGU+YFxuICovXG50eXBlIFJlc3RBcmdzVHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH1gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncy5cbiAqIC0gYFsuLi5uYW1lXWBcbiAqIC0gYDwuLi5uYW1lPmBcbiAqIC0gYFtuYW1lLi4uXWBcbiAqIC0gYDxuYW1lLi4uPmBcbiAqL1xudHlwZSBSZXN0QXJncyA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9YFxuPjtcblxuLyoqXG4gKiBTaW5nbGUgYXJnIHdpdGggbGlzdCB0eXBlIGFuZCBjb21wbGV0aW9ucy5cbiAqXG4gKiAtIGBbbmFtZTp0eXBlW106Y29tcGxldGlvbl1gXG4gKiAtIGA8bmFtZTp0eXBlW106Y29tcGxldGlvbj5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnTGlzdFR5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfVtdOiR7c3RyaW5nfWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZyB3aXRoIGxpc3QgdHlwZS5cbiAqXG4gKiAtIGBbbmFtZTp0eXBlW11dYFxuICogLSBgPG5hbWU6dHlwZVtdPmBcbiAqL1xudHlwZSBTaW5nbGVBcmdMaXN0VHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9OiR7VH1bXWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZyAgd2l0aCB0eXBlIGFuZCBjb21wbGV0aW9uLlxuICpcbiAqIC0gYFtuYW1lOnR5cGU6Y29tcGxldGlvbl1gXG4gKiAtIGA8bmFtZTp0eXBlOmNvbXBsZXRpb24+YFxuICovXG50eXBlIFNpbmdsZUFyZ1R5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfToke3N0cmluZ31gXG4+O1xuXG4vKipcbiAqIFNpbmdsZSBhcmcgd2l0aCB0eXBlLlxuICpcbiAqIC0gYFtuYW1lOnR5cGVdYFxuICogLSBgPG5hbWU6dHlwZT5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnVHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9OiR7VH1gXG4+O1xuXG4vKipcbiAqIFNpbmdsZSBhcmcuXG4gKlxuICogLSBgW25hbWVdYFxuICogLSBgPG5hbWU+YFxuICovXG50eXBlIFNpbmdsZUFyZyA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtzdHJpbmd9YFxuPjtcblxudHlwZSBEZWZhdWx0VHlwZXMgPSB7XG4gIG51bWJlcjogTnVtYmVyVHlwZTtcbiAgaW50ZWdlcjogSW50ZWdlclR5cGU7XG4gIHN0cmluZzogU3RyaW5nVHlwZTtcbiAgYm9vbGVhbjogQm9vbGVhblR5cGU7XG4gIGZpbGU6IEZpbGVUeXBlO1xufTtcblxudHlwZSBBcmd1bWVudFR5cGU8QSBleHRlbmRzIHN0cmluZywgVSwgVCA9IE1lcmdlPERlZmF1bHRUeXBlcywgVT4+ID0gQSBleHRlbmRzXG4gIFJlc3RBcmdzTGlzdFR5cGVDb21wbGV0aW9uPGluZmVyIFR5cGU+XG4gID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PEFycmF5PFI+PiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3NMaXN0VHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PEFycmF5PFI+PiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3NUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBSZXN0QXJnc1R5cGU8aW5mZXIgVHlwZT5cbiAgICA/IFQgZXh0ZW5kcyBSZWNvcmQ8VHlwZSwgaW5mZXIgUj4gPyBBcnJheTxSPiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgUmVzdEFyZ3MgPyBBcnJheTxzdHJpbmc+XG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ0xpc3RUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmdMaXN0VHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmdUeXBlQ29tcGxldGlvbjxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IFIgOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ1R5cGU8aW5mZXIgVHlwZT5cbiAgICA/IFQgZXh0ZW5kcyBSZWNvcmQ8VHlwZSwgaW5mZXIgUj4gPyBSIDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBTaW5nbGVBcmcgPyBzdHJpbmdcbiAgOiB1bmtub3duO1xuXG50eXBlIEFyZ3VtZW50VHlwZXM8QSBleHRlbmRzIHN0cmluZywgVD4gPSBBIGV4dGVuZHMgYCR7c3RyaW5nfSAke3N0cmluZ31gXG4gID8gVHlwZWRBcmd1bWVudHM8QSwgVD5cbiAgOiBBcmd1bWVudFR5cGU8QSwgVD47XG5cbnR5cGUgR2V0QXJndW1lbnRzPEEgZXh0ZW5kcyBzdHJpbmc+ID0gQSBleHRlbmRzIGAtJHtzdHJpbmd9PSR7aW5mZXIgUmVzdH1gXG4gID8gR2V0QXJndW1lbnRzPFJlc3Q+XG4gIDogQSBleHRlbmRzIGAtJHtzdHJpbmd9ICR7aW5mZXIgUmVzdH1gID8gR2V0QXJndW1lbnRzPFJlc3Q+XG4gIDogQTtcblxudHlwZSBPcHRpb25OYW1lPE5hbWUgZXh0ZW5kcyBzdHJpbmc+ID0gTmFtZSBleHRlbmRzIFwiKlwiID8gc3RyaW5nXG4gIDogQ2FtZWxDYXNlPFRyaW1SaWdodDxOYW1lLCBcIixcIj4+O1xuXG50eXBlIElzUmVxdWlyZWQ8UiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQsIEQ+ID0gUiBleHRlbmRzIHRydWUgPyB0cnVlXG4gIDogRCBleHRlbmRzIHVuZGVmaW5lZCA/IGZhbHNlXG4gIDogdHJ1ZTtcblxudHlwZSBOZWdhdGFibGVPcHRpb248XG4gIEYgZXh0ZW5kcyBzdHJpbmcsXG4gIENPLFxuICBELFxuICBOIGV4dGVuZHMgc3RyaW5nID0gT3B0aW9uTmFtZTxGPixcbj4gPSBEIGV4dGVuZHMgdW5kZWZpbmVkXG4gID8gTiBleHRlbmRzIGtleW9mIENPID8geyBbSyBpbiBOXT86IGZhbHNlIH0gOiB7IFtLIGluIE5dOiBib29sZWFuIH1cbiAgOiB7IFtLIGluIE5dOiBOb25OdWxsYWJsZTxEPiB8IGZhbHNlIH07XG5cbnR5cGUgQm9vbGVhbk9wdGlvbjxcbiAgTiBleHRlbmRzIHN0cmluZyxcbiAgQ08sXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IE4gZXh0ZW5kcyBgbm8tJHtpbmZlciBOYW1lfWAgPyBOZWdhdGFibGVPcHRpb248TmFtZSwgQ08sIEQ+XG4gIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9LiR7aW5mZXIgUmVzdH1gXG4gICAgPyAoUiBleHRlbmRzIHRydWVcbiAgICAgID8geyBbSyBpbiBPcHRpb25OYW1lPE5hbWU+XTogQm9vbGVhbk9wdGlvbjxSZXN0LCBDTywgUiwgRD4gfVxuICAgICAgOiB7IFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dPzogQm9vbGVhbk9wdGlvbjxSZXN0LCBDTywgUiwgRD4gfSlcbiAgOiAoUiBleHRlbmRzIHRydWUgPyB7IFtLIGluIE9wdGlvbk5hbWU8Tj5dOiB0cnVlIHwgRCB9XG4gICAgOiB7IFtLIGluIE9wdGlvbk5hbWU8Tj5dPzogdHJ1ZSB8IEQgfSk7XG5cbnR5cGUgVmFsdWVPcHRpb248XG4gIE4gZXh0ZW5kcyBzdHJpbmcsXG4gIEYgZXh0ZW5kcyBzdHJpbmcsXG4gIFYsXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IE4gZXh0ZW5kcyBgJHtpbmZlciBOYW1lfS4ke2luZmVyIFJlc3ROYW1lfWAgPyAoUiBleHRlbmRzIHRydWUgPyB7XG4gICAgICBbSyBpbiBPcHRpb25OYW1lPE5hbWU+XTogVmFsdWVPcHRpb248UmVzdE5hbWUsIEYsIFYsIFIsIEQ+O1xuICAgIH1cbiAgICA6IHtcbiAgICAgIFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dPzogVmFsdWVPcHRpb248UmVzdE5hbWUsIEYsIFYsIFIsIEQ+O1xuICAgIH0pXG4gIDogKFIgZXh0ZW5kcyB0cnVlID8ge1xuICAgICAgW0sgaW4gT3B0aW9uTmFtZTxOPl06IEdldEFyZ3VtZW50czxGPiBleHRlbmRzIGBbJHtzdHJpbmd9XWBcbiAgICAgICAgPyBOb25OdWxsYWJsZTxEPiB8IHRydWUgfCBBcmd1bWVudFR5cGU8R2V0QXJndW1lbnRzPEY+LCBWPlxuICAgICAgICA6IE5vbk51bGxhYmxlPEQ+IHwgQXJndW1lbnRUeXBlPEdldEFyZ3VtZW50czxGPiwgVj47XG4gICAgfVxuICAgIDoge1xuICAgICAgW0sgaW4gT3B0aW9uTmFtZTxOPl0/OiBHZXRBcmd1bWVudHM8Rj4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICAgID8gTm9uTnVsbGFibGU8RD4gfCB0cnVlIHwgQXJndW1lbnRUeXBlPEdldEFyZ3VtZW50czxGPiwgVj5cbiAgICAgICAgOiBOb25OdWxsYWJsZTxEPiB8IEFyZ3VtZW50VHlwZTxHZXRBcmd1bWVudHM8Rj4sIFY+O1xuICAgIH0pO1xuXG50eXBlIFZhbHVlc09wdGlvbjxcbiAgVCBleHRlbmRzIHN0cmluZyxcbiAgUmVzdCBleHRlbmRzIHN0cmluZyxcbiAgVixcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gVCBleHRlbmRzIGAke2luZmVyIE5hbWV9LiR7aW5mZXIgUmVzdE5hbWV9YCA/IChSIGV4dGVuZHMgdHJ1ZSA/IHtcbiAgICAgIFtOIGluIE9wdGlvbk5hbWU8TmFtZT5dOiBWYWx1ZXNPcHRpb248UmVzdE5hbWUsIFJlc3QsIFYsIFIsIEQ+O1xuICAgIH1cbiAgICA6IHtcbiAgICAgIFtOIGluIE9wdGlvbk5hbWU8TmFtZT5dPzogVmFsdWVzT3B0aW9uPFJlc3ROYW1lLCBSZXN0LCBWLCBSLCBEPjtcbiAgICB9KVxuICA6IChSIGV4dGVuZHMgdHJ1ZSA/IHtcbiAgICAgIFtOIGluIE9wdGlvbk5hbWU8VD5dOiBHZXRBcmd1bWVudHM8UmVzdD4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICAgID8gTm9uTnVsbGFibGU8RD4gfCB0cnVlIHwgQXJndW1lbnRUeXBlczxHZXRBcmd1bWVudHM8UmVzdD4sIFY+XG4gICAgICAgIDogTm9uTnVsbGFibGU8RD4gfCBBcmd1bWVudFR5cGVzPEdldEFyZ3VtZW50czxSZXN0PiwgVj47XG4gICAgfVxuICAgIDoge1xuICAgICAgW04gaW4gT3B0aW9uTmFtZTxUPl0/OiBHZXRBcmd1bWVudHM8UmVzdD4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICAgID8gTm9uTnVsbGFibGU8RD4gfCB0cnVlIHwgQXJndW1lbnRUeXBlczxHZXRBcmd1bWVudHM8UmVzdD4sIFY+XG4gICAgICAgIDogTm9uTnVsbGFibGU8RD4gfCBBcmd1bWVudFR5cGVzPEdldEFyZ3VtZW50czxSZXN0PiwgVj47XG4gICAgfSk7XG5cbnR5cGUgTWFwVmFsdWU8TywgViwgQyA9IHVuZGVmaW5lZD4gPSBWIGV4dGVuZHMgdW5kZWZpbmVkID8gQyBleHRlbmRzIHRydWUgPyB7XG4gICAgICBbSyBpbiBrZXlvZiBPXTogT1tLXSBleHRlbmRzIChSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHVuZGVmaW5lZClcbiAgICAgICAgPyBNYXBWYWx1ZTxPW0tdLCBWPlxuICAgICAgICA6IEFycmF5PE5vbk51bGxhYmxlPE9bS10+PjtcbiAgICB9XG4gIDogT1xuICA6IHtcbiAgICBbSyBpbiBrZXlvZiBPXTogT1tLXSBleHRlbmRzIChSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHVuZGVmaW5lZClcbiAgICAgID8gTWFwVmFsdWU8T1tLXSwgVj5cbiAgICAgIDogVjtcbiAgfTtcblxudHlwZSBHZXRPcHRpb25OYW1lPFQ+ID0gVCBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX09JHtzdHJpbmd9YFxuICA/IFRyaW1SaWdodDxOYW1lLCBcIixcIj5cbiAgOiBUIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfSAke3N0cmluZ31gID8gVHJpbVJpZ2h0PE5hbWUsIFwiLFwiPlxuICA6IFQgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9YCA/IE5hbWVcbiAgOiBUIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9PSR7c3RyaW5nfWAgPyBUcmltUmlnaHQ8TmFtZSwgXCIsXCI+XG4gIDogVCBleHRlbmRzIGAtJHtpbmZlciBOYW1lfSAke3N0cmluZ31gID8gVHJpbVJpZ2h0PE5hbWUsIFwiLFwiPlxuICA6IFQgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX1gID8gTmFtZVxuICA6IHVua25vd247XG5cbnR5cGUgTWVyZ2VPcHRpb25zPFQsIENPLCBPLCBOID0gR2V0T3B0aW9uTmFtZTxUPj4gPSBOIGV4dGVuZHMgYG5vLSR7c3RyaW5nfWBcbiAgPyBTcHJlYWQ8Q08sIE8+XG4gIDogTiBleHRlbmRzIGAke3N0cmluZ30uJHtzdHJpbmd9YCA/IE1lcmdlUmVjdXJzaXZlPENPLCBPPlxuICA6IE1lcmdlPENPLCBPPjtcblxuLy8gdHlwZSBNZXJnZU9wdGlvbnM8VCwgQ08sIE8sIE4gPSBHZXRPcHRpb25OYW1lPFQ+PiA9IE4gZXh0ZW5kcyBgbm8tJHtzdHJpbmd9YFxuLy8gICA/IFNwcmVhZDxDTywgTz5cbi8vICAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0uJHtpbmZlciBDaGlsZH1gXG4vLyAgICAgPyAoT3B0aW9uTmFtZTxOYW1lPiBleHRlbmRzIGtleW9mIE1lcmdlPENPLCBPPlxuLy8gICAgICAgPyBPcHRpb25OYW1lPENoaWxkPiBleHRlbmRzXG4vLyAgICAgICAgIGtleW9mIE5vbk51bGxhYmxlPE1lcmdlPENPLCBPPltPcHRpb25OYW1lPE5hbWU+XT4gPyBTcHJlYWRUd288Q08sIE8+XG4vLyAgICAgICA6IE1lcmdlUmVjdXJzaXZlPENPLCBPPlxuLy8gICAgICAgOiBNZXJnZVJlY3Vyc2l2ZTxDTywgTz4pXG4vLyAgIDogTWVyZ2U8Q08sIE8+O1xuXG50eXBlIFR5cGVkT3B0aW9uPFxuICBGIGV4dGVuZHMgc3RyaW5nLFxuICBDTyxcbiAgVCxcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gbnVtYmVyIGV4dGVuZHMgVCA/IGFueVxuICA6IEYgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9PSR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZXNPcHRpb248TmFtZSwgUmVzdCwgVCwgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBGIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfSAke2luZmVyIFJlc3R9YFxuICAgID8gVmFsdWVzT3B0aW9uPE5hbWUsIFJlc3QsIFQsIElzUmVxdWlyZWQ8UiwgRD4sIEQ+XG4gIDogRiBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX1gXG4gICAgPyBCb29sZWFuT3B0aW9uPE5hbWUsIENPLCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX09JHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlc09wdGlvbjxOYW1lLCBSZXN0LCBULCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX0gJHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlc09wdGlvbjxOYW1lLCBSZXN0LCBULCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX1gID8gQm9vbGVhbk9wdGlvbjxOYW1lLCBDTywgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPjtcblxudHlwZSBUeXBlZEFyZ3VtZW50czxBIGV4dGVuZHMgc3RyaW5nLCBUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQ+ID1cbiAgbnVtYmVyIGV4dGVuZHMgVCA/IGFueVxuICAgIDogQSBleHRlbmRzIGAke2luZmVyIEFyZ30gJHtpbmZlciBSZXN0fWBcbiAgICAgID8gQXJnIGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgICA/IFtBcmd1bWVudFR5cGU8QXJnLCBUPj8sIC4uLlR5cGVkQXJndW1lbnRzPFJlc3QsIFQ+XVxuICAgICAgOiBbQXJndW1lbnRUeXBlPEFyZywgVD4sIC4uLlR5cGVkQXJndW1lbnRzPFJlc3QsIFQ+XVxuICAgIDogQSBleHRlbmRzIGAke3N0cmluZ30uLi4ke3N0cmluZ31gID8gW1xuICAgICAgICAuLi5Bcmd1bWVudFR5cGU8QSwgVD4gZXh0ZW5kcyBBcnJheTxpbmZlciBVPlxuICAgICAgICAgID8gQSBleHRlbmRzIGBbJHtzdHJpbmd9XWAgPyBBcnJheTxVPiA6IFtVLCAuLi5BcnJheTxVPl1cbiAgICAgICAgICA6IG5ldmVyLFxuICAgICAgXVxuICAgIDogQSBleHRlbmRzIGBbJHtzdHJpbmd9XWAgPyBbQXJndW1lbnRUeXBlPEEsIFQ+P11cbiAgICA6IFtBcmd1bWVudFR5cGU8QSwgVD5dO1xuXG50eXBlIFR5cGVkQ29tbWFuZEFyZ3VtZW50czxOIGV4dGVuZHMgc3RyaW5nLCBUPiA9IG51bWJlciBleHRlbmRzIFQgPyBhbnlcbiAgOiBOIGV4dGVuZHMgYCR7c3RyaW5nfSAke2luZmVyIEFyZ3N9YCA/IFR5cGVkQXJndW1lbnRzPEFyZ3MsIFQ+XG4gIDogW107XG5cbnR5cGUgVHlwZWRFbnY8XG4gIE4gZXh0ZW5kcyBzdHJpbmcsXG4gIFAgZXh0ZW5kcyBzdHJpbmcgfCB1bmRlZmluZWQsXG4gIENPLFxuICBULFxuICBSIGV4dGVuZHMgYm9vbGVhbiB8IHVuZGVmaW5lZCA9IHVuZGVmaW5lZCxcbiAgRCA9IHVuZGVmaW5lZCxcbj4gPSBudW1iZXIgZXh0ZW5kcyBUID8gYW55XG4gIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9PSR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZU9wdGlvbjxUcmltTGVmdDxOYW1lLCBQPiwgUmVzdCwgVCwgUiwgRD5cbiAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0gJHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlT3B0aW9uPFRyaW1MZWZ0PE5hbWUsIFA+LCBSZXN0LCBULCBSLCBEPlxuICA6IE4gZXh0ZW5kcyBgJHtpbmZlciBOYW1lfWAgPyBCb29sZWFuT3B0aW9uPFRyaW1MZWZ0PE5hbWUsIFA+LCBDTywgUiwgRD5cbiAgOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPjtcblxudHlwZSBUeXBlZFR5cGU8XG4gIE5hbWUgZXh0ZW5kcyBzdHJpbmcsXG4gIEhhbmRsZXIgZXh0ZW5kcyBUeXBlT3JUeXBlSGFuZGxlcjx1bmtub3duPixcbj4gPSB7IFtOIGluIE5hbWVdOiBIYW5kbGVyIH07XG5cbnR5cGUgUmVxdWlyZWRLZXlzPFQ+ID0ge1xuICAvLyBkZW5vLWxpbnQtaWdub3JlIGJhbi10eXBlc1xuICBbSyBpbiBrZXlvZiBUXS0/OiB7fSBleHRlbmRzIFBpY2s8VCwgSz4gPyBuZXZlciA6IEs7XG59W2tleW9mIFRdO1xuXG50eXBlIE9wdGlvbmFsS2V5czxUPiA9IHtcbiAgLy8gZGVuby1saW50LWlnbm9yZSBiYW4tdHlwZXNcbiAgW0sgaW4ga2V5b2YgVF0tPzoge30gZXh0ZW5kcyBQaWNrPFQsIEs+ID8gSyA6IG5ldmVyO1xufVtrZXlvZiBUXTtcblxudHlwZSBTcHJlYWRSZXF1aXJlZFByb3BlcnRpZXM8XG4gIEwsXG4gIFIsXG4gIEsgZXh0ZW5kcyBrZXlvZiBMICYga2V5b2YgUixcbj4gPSB7XG4gIFtQIGluIEtdOiBFeGNsdWRlPExbUF0sIHVuZGVmaW5lZD4gfCBFeGNsdWRlPFJbUF0sIHVuZGVmaW5lZD47XG59O1xuXG50eXBlIFNwcmVhZE9wdGlvbmFsUHJvcGVydGllczxcbiAgTCxcbiAgUixcbiAgSyBleHRlbmRzIGtleW9mIEwgJiBrZXlvZiBSLFxuPiA9IHtcbiAgW1AgaW4gS10/OiBMW1BdIHwgUltQXTtcbn07XG5cbi8qKiBNZXJnZSB0eXBlcyBvZiB0d28gb2JqZWN0cy4gKi9cbnR5cGUgU3ByZWFkPEwsIFI+ID0gTCBleHRlbmRzIHZvaWQgPyBSIDogUiBleHRlbmRzIHZvaWQgPyBMXG46IC8vIFByb3BlcnRpZXMgaW4gTCB0aGF0IGRvbid0IGV4aXN0IGluIFIuXG4mIE9taXQ8TCwga2V5b2YgUj5cbi8vIFByb3BlcnRpZXMgaW4gUiB0aGF0IGRvbid0IGV4aXN0IGluIEwuXG4mIE9taXQ8Uiwga2V5b2YgTD5cbi8vIFJlcXVpcmVkIHByb3BlcnRpZXMgaW4gUiB0aGF0IGV4aXN0IGluIEwuXG4mIFNwcmVhZFJlcXVpcmVkUHJvcGVydGllczxMLCBSLCBSZXF1aXJlZEtleXM8Uj4gJiBrZXlvZiBMPlxuLy8gUmVxdWlyZWQgcHJvcGVydGllcyBpbiBMIHRoYXQgZXhpc3QgaW4gUi5cbiYgU3ByZWFkUmVxdWlyZWRQcm9wZXJ0aWVzPEwsIFIsIFJlcXVpcmVkS2V5czxMPiAmIGtleW9mIFI+XG4vLyBPcHRpb25hbCBwcm9wZXJ0aWVzIGluIEwgYW5kIFIuXG4mIFNwcmVhZE9wdGlvbmFsUHJvcGVydGllczxcbiAgTCxcbiAgUixcbiAgT3B0aW9uYWxLZXlzPEw+ICYgT3B0aW9uYWxLZXlzPFI+XG4+O1xuXG50eXBlIFZhbHVlT2Y8VD4gPSBUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgaW5mZXIgVj4gPyBWYWx1ZU9mPFY+IDogVDtcbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSx3Q0FBd0M7QUFDeEMsU0FDRSxXQUFXLEVBQ1gsZUFBZSxJQUFJLG9CQUFvQixRQUNsQyxxQkFBcUIsQ0FBQztBQUM3QixTQUFTLHFCQUFxQixRQUFRLGNBQWMsQ0FBQztBQUNyRCxTQUFTLFVBQVUsUUFBUSxtQkFBbUIsQ0FBQztBQUUvQyxTQUNFLGNBQWMsRUFDZCx3QkFBd0IsRUFDeEIsY0FBYyxRQUNULGFBQWEsQ0FBQztBQUNyQixTQUFTLElBQUksRUFBRSxJQUFJLEVBQUUsR0FBRyxFQUFFLE1BQU0sUUFBUSxXQUFXLENBQUM7QUFDcEQsU0FDRSx5QkFBeUIsRUFDekIs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