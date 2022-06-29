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
            opts: typeof opts === "function" ? {
                action: opts
            } : opts
        };
        return this;
    }
    helpOption(flags, desc, opts) {
        this._helpOption = flags === false ? flags : {
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
   */ allowEmpty(allowEmpty = true) {
        this.cmd._allowEmpty = allowEmpty;
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
    /** Check whether the command should throw errors or exit. */ shouldThrowErrors() {
        return this.cmd.throwOnError || !!this.cmd._parent?.shouldThrowErrors();
    }
    /** Check whether the command should exit after printing help or version. */ shouldExit() {
        return (this.cmd._shouldExit ?? this.cmd._parent?.shouldExit()) ?? true;
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
                    literal: []
                };
            } else if (this._useRawArgs) {
                const env = await this.parseEnvVars();
                return this.execute(env, ...this.rawArgs);
            } else {
                const env1 = await this.parseEnvVars();
                const { actionOption , flags , unknown , literal  } = this.parseFlags(this.rawArgs, env1);
                this.literalArgs = literal;
                const options = {
                    ...env1,
                    ...flags
                };
                const params = this.parseArguments(unknown, options);
                if (actionOption) {
                    await actionOption.action.call(this, options, ...params);
                    if (actionOption.standalone) {
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
        } catch (error) {
            if (error instanceof Error) {
                throw this.error(error);
            } else {
                throw this.error(new Error(`[non-error-thrown] ${error}`));
            }
        }
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
        if (this._versionOption !== false && (this._versionOption || this.ver)) {
            this.option(this._versionOption?.flags || "-V, --version", this._versionOption?.desc || "Show the version number for this program.", {
                standalone: true,
                prepend: true,
                action: async function() {
                    const long = this.getRawArgs().includes(`--${versionOption.name}`);
                    if (long) {
                        await this.checkVersion();
                        this.showLongVersion();
                    } else {
                        this.showVersion();
                    }
                    this.exit();
                },
                ...this._versionOption?.opts ?? {}
            });
            const versionOption = this.options[0];
        }
        if (this._helpOption !== false) {
            this.option(this._helpOption?.flags || "-h, --help", this._helpOption?.desc || "Show this help.", {
                standalone: true,
                global: true,
                prepend: true,
                action: async function() {
                    const long = this.getRawArgs().includes(`--${helpOption.name}`);
                    await this.checkVersion();
                    this.showHelp({
                        long
                    });
                    this.exit();
                },
                ...this._helpOption?.opts ?? {}
            });
            const helpOption = this.options[0];
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
   */ parseFlags(args, env) {
        try {
            let actionOption;
            const result = parseFlags(args, {
                stopEarly: this._stopEarly,
                allowEmpty: this._allowEmpty,
                flags: this.getOptions(true),
                ignoreDefaults: env,
                parse: (type)=>this.parseType(type),
                option: (option)=>{
                    if (!actionOption && option.action) {
                        actionOption = option;
                    }
                }
            });
            return {
                ...result,
                actionOption
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
        return typeSettings.handler instanceof Type ? typeSettings.handler.parse(type) : typeSettings.handler(type);
    }
    /** Validate environment variables. */ async parseEnvVars() {
        const envVars = this.getEnvVars(true);
        const result = {};
        if (!envVars.length) {
            return result;
        }
        const hasEnvPermissions = (await Deno.permissions.query({
            name: "env"
        })).state === "granted";
        for (const env of envVars){
            const name = hasEnvPermissions && env.names.find((name)=>!!Deno.env.get(name));
            if (name) {
                const propertyName = underscoreToCamelCase(env.prefix ? env.names[0].replace(new RegExp(`^${env.prefix}`), "") : env.names[0]);
                if (env.details.list) {
                    const values = Deno.env.get(name)?.split(env.details.separator ?? ",") ?? [
                        ""
                    ];
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
                        value: Deno.env.get(name) ?? ""
                    });
                }
                if (env.value && typeof result[propertyName] !== "undefined") {
                    result[propertyName] = env.value(result[propertyName]);
                }
            } else if (env.required) {
                throw new MissingRequiredEnvVar(env);
            }
        }
        return result;
    }
    /**
   * Parse command-line arguments.
   * @param args  Raw command line arguments.
   * @param flags Parsed command line options.
   */ parseArguments(args, flags) {
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
                    const flagNames = Object.keys(flags);
                    const hasStandaloneOption = !!flagNames.find((name)=>this.getOption(name, true)?.standalone);
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
                    if (expectedArg.variadic) {
                        arg = args.splice(0, args.length).map((value)=>this.parseType({
                                label: "Argument",
                                type: expectedArg.type,
                                name: expectedArg.name,
                                value
                            }));
                    } else {
                        arg = this.parseType({
                            label: "Argument",
                            type: expectedArg.type,
                            name: expectedArg.name,
                            value: args.shift()
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
    /**
   * Handle error. If `throwErrors` is enabled the error will be returned,
   * otherwise a formatted error message will be printed and `Deno.exit(1)`
   * will be called.
   * @param error Error to handle.
   */ error(error) {
        if (this.shouldThrowErrors() || !(error instanceof ValidationError)) {
            return error;
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
        if (isUpgradeCommand(upgradeCommand)) {
            const latestVersion = await upgradeCommand.getLatestVersion();
            const currentVersion = mainCommand.getVersion();
            if (currentVersion !== latestVersion) {
                mainCommand.version(`${currentVersion}  ${bold(yellow(`(New version available: ${latestVersion}. Run '${mainCommand.getName()} upgrade' to upgrade to the latest version!)`))}`);
            }
        }
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
        const getOptions = (cmd, options = [], names = [])=>{
            if (cmd) {
                if (cmd.options.length) {
                    cmd.options.forEach((option)=>{
                        if (option.global && !this.options.find((opt)=>opt.name === option.name) && names.indexOf(option.name) === -1 && (hidden || !option.hidden)) {
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
        const option = this.options.find((option)=>option.name === name);
        return option && (hidden || !option.hidden) ? option : undefined;
    }
    /**
   * Get global option from parent commands by name.
   * @param name Name of the option. Must be in param-case.
   * @param hidden Include hidden options.
   */ getGlobalOption(name, hidden) {
        if (!this._parent) {
            return;
        }
        const option = this._parent.getBaseOption(name, hidden);
        if (!option || !option.global) {
            return this._parent.getGlobalOption(name, hidden);
        }
        return option;
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
        const getCommands = (cmd, commands = [], names = [])=>{
            if (cmd) {
                if (cmd.commands.size) {
                    cmd.commands.forEach((cmd)=>{
                        if (cmd.isGlobal && this !== cmd && !this.commands.has(cmd._name) && names.indexOf(cmd._name) === -1 && (hidden || !cmd.isHidden)) {
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
        if (!this._parent) {
            return;
        }
        const cmd = this._parent.getBaseCommand(name, hidden);
        if (!cmd?.isGlobal) {
            return this._parent.getGlobalCommand(name, hidden);
        }
        return cmd;
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
        if (!this._parent) {
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
}
function isUpgradeCommand(command) {
    return command instanceof Command && "getLatestVersion" in command;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvY29tbWFuZC9jb21tYW5kLnRzIl0sInNvdXJjZXNDb250ZW50IjpbIi8vIGRlbm8tbGludC1pZ25vcmUtZmlsZSBuby1leHBsaWNpdC1hbnlcbmltcG9ydCB7XG4gIFVua25vd25UeXBlLFxuICBWYWxpZGF0aW9uRXJyb3IgYXMgRmxhZ3NWYWxpZGF0aW9uRXJyb3IsXG59IGZyb20gXCIuLi9mbGFncy9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBNaXNzaW5nUmVxdWlyZWRFbnZWYXIgfSBmcm9tIFwiLi9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBwYXJzZUZsYWdzIH0gZnJvbSBcIi4uL2ZsYWdzL2ZsYWdzLnRzXCI7XG5pbXBvcnQgdHlwZSB7IElEZWZhdWx0VmFsdWUsIElGbGFnc1Jlc3VsdCB9IGZyb20gXCIuLi9mbGFncy90eXBlcy50c1wiO1xuaW1wb3J0IHtcbiAgZ2V0RGVzY3JpcHRpb24sXG4gIHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbixcbiAgc3BsaXRBcmd1bWVudHMsXG59IGZyb20gXCIuL191dGlscy50c1wiO1xuaW1wb3J0IHsgYmx1ZSwgYm9sZCwgcmVkLCB5ZWxsb3cgfSBmcm9tIFwiLi9kZXBzLnRzXCI7XG5pbXBvcnQge1xuICBDb21tYW5kRXhlY3V0YWJsZU5vdEZvdW5kLFxuICBDb21tYW5kTm90Rm91bmQsXG4gIERlZmF1bHRDb21tYW5kTm90Rm91bmQsXG4gIER1cGxpY2F0ZUNvbW1hbmRBbGlhcyxcbiAgRHVwbGljYXRlQ29tbWFuZE5hbWUsXG4gIER1cGxpY2F0ZUNvbXBsZXRpb24sXG4gIER1cGxpY2F0ZUVudmlyb25tZW50VmFyaWFibGUsXG4gIER1cGxpY2F0ZUV4YW1wbGUsXG4gIER1cGxpY2F0ZU9wdGlvbk5hbWUsXG4gIER1cGxpY2F0ZVR5cGUsXG4gIEVudmlyb25tZW50VmFyaWFibGVPcHRpb25hbFZhbHVlLFxuICBFbnZpcm9ubWVudFZhcmlhYmxlU2luZ2xlVmFsdWUsXG4gIEVudmlyb25tZW50VmFyaWFibGVWYXJpYWRpY1ZhbHVlLFxuICBNaXNzaW5nQXJndW1lbnQsXG4gIE1pc3NpbmdBcmd1bWVudHMsXG4gIE1pc3NpbmdDb21tYW5kTmFtZSxcbiAgTm9Bcmd1bWVudHNBbGxvd2VkLFxuICBUb29NYW55QXJndW1lbnRzLFxuICBVbmtub3duQ29tbWFuZCxcbiAgVmFsaWRhdGlvbkVycm9yLFxufSBmcm9tIFwiLi9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBCb29sZWFuVHlwZSB9IGZyb20gXCIuL3R5cGVzL2Jvb2xlYW4udHNcIjtcbmltcG9ydCB7IEZpbGVUeXBlIH0gZnJvbSBcIi4vdHlwZXMvZmlsZS50c1wiO1xuaW1wb3J0IHsgTnVtYmVyVHlwZSB9IGZyb20gXCIuL3R5cGVzL251bWJlci50c1wiO1xuaW1wb3J0IHsgU3RyaW5nVHlwZSB9IGZyb20gXCIuL3R5cGVzL3N0cmluZy50c1wiO1xuaW1wb3J0IHsgVHlwZSB9IGZyb20gXCIuL3R5cGUudHNcIjtcbmltcG9ydCB7IEhlbHBHZW5lcmF0b3IgfSBmcm9tIFwiLi9oZWxwL19oZWxwX2dlbmVyYXRvci50c1wiO1xuaW1wb3J0IHR5cGUgeyBIZWxwT3B0aW9ucyB9IGZyb20gXCIuL2hlbHAvX2hlbHBfZ2VuZXJhdG9yLnRzXCI7XG5pbXBvcnQgdHlwZSB7XG4gIElBY3Rpb24sXG4gIElBcmd1bWVudCxcbiAgSUNvbW1hbmRHbG9iYWxPcHRpb24sXG4gIElDb21tYW5kT3B0aW9uLFxuICBJQ29tcGxldGVIYW5kbGVyLFxuICBJQ29tcGxldGVPcHRpb25zLFxuICBJQ29tcGxldGlvbixcbiAgSURlc2NyaXB0aW9uLFxuICBJRW52VmFyLFxuICBJRW52VmFyT3B0aW9ucyxcbiAgSUVudlZhclZhbHVlSGFuZGxlcixcbiAgSUV4YW1wbGUsXG4gIElGbGFnVmFsdWVIYW5kbGVyLFxuICBJR2xvYmFsRW52VmFyT3B0aW9ucyxcbiAgSUhlbHBIYW5kbGVyLFxuICBJT3B0aW9uLFxuICBJUGFyc2VSZXN1bHQsXG4gIElUeXBlLFxuICBJVHlwZUluZm8sXG4gIElUeXBlT3B0aW9ucyxcbiAgSVZlcnNpb25IYW5kbGVyLFxuICBNYXBUeXBlcyxcbiAgVHlwZU9yVHlwZUhhbmRsZXIsXG59IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBJbnRlZ2VyVHlwZSB9IGZyb20gXCIuL3R5cGVzL2ludGVnZXIudHNcIjtcbmltcG9ydCB7IHVuZGVyc2NvcmVUb0NhbWVsQ2FzZSB9IGZyb20gXCIuLi9mbGFncy9fdXRpbHMudHNcIjtcblxuZXhwb3J0IGNsYXNzIENvbW1hbmQ8XG4gIENQRyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkID0gdm9pZCxcbiAgQ1BUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQgPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiB2b2lkLFxuICBDTyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdm9pZCxcbiAgQ0EgZXh0ZW5kcyBBcnJheTx1bmtub3duPiA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IFtdLFxuICBDRyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdm9pZCxcbiAgQ1QgZXh0ZW5kcyBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCA9IENQRyBleHRlbmRzIG51bWJlciA/IGFueSA6IHtcbiAgICBudW1iZXI6IG51bWJlcjtcbiAgICBpbnRlZ2VyOiBudW1iZXI7XG4gICAgc3RyaW5nOiBzdHJpbmc7XG4gICAgYm9vbGVhbjogYm9vbGVhbjtcbiAgICBmaWxlOiBzdHJpbmc7XG4gIH0sXG4gIENHVCBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkID0gQ1BHIGV4dGVuZHMgbnVtYmVyID8gYW55IDogdm9pZCxcbiAgQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQgPSBDUEcgZXh0ZW5kcyBudW1iZXIgPyBhbnkgOiB1bmRlZmluZWQsXG4+IHtcbiAgcHJpdmF0ZSB0eXBlczogTWFwPHN0cmluZywgSVR5cGU+ID0gbmV3IE1hcCgpO1xuICBwcml2YXRlIHJhd0FyZ3M6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgcHJpdmF0ZSBsaXRlcmFsQXJnczogQXJyYXk8c3RyaW5nPiA9IFtdO1xuICAvLyBAVE9ETzogZ2V0IHNjcmlwdCBuYW1lOiBodHRwczovL2dpdGh1Yi5jb20vZGVub2xhbmQvZGVuby9wdWxsLzUwMzRcbiAgLy8gcHJpdmF0ZSBuYW1lOiBzdHJpbmcgPSBsb2NhdGlvbi5wYXRobmFtZS5zcGxpdCggJy8nICkucG9wKCkgYXMgc3RyaW5nO1xuICBwcml2YXRlIF9uYW1lID0gXCJDT01NQU5EXCI7XG4gIHByaXZhdGUgX3BhcmVudD86IENQO1xuICBwcml2YXRlIF9nbG9iYWxQYXJlbnQ/OiBDb21tYW5kPGFueT47XG4gIHByaXZhdGUgdmVyPzogSVZlcnNpb25IYW5kbGVyO1xuICBwcml2YXRlIGRlc2M6IElEZXNjcmlwdGlvbiA9IFwiXCI7XG4gIHByaXZhdGUgX3VzYWdlPzogc3RyaW5nO1xuICBwcml2YXRlIGZuPzogSUFjdGlvbjtcbiAgcHJpdmF0ZSBvcHRpb25zOiBBcnJheTxJT3B0aW9uPiA9IFtdO1xuICBwcml2YXRlIGNvbW1hbmRzOiBNYXA8c3RyaW5nLCBDb21tYW5kPGFueT4+ID0gbmV3IE1hcCgpO1xuICBwcml2YXRlIGV4YW1wbGVzOiBBcnJheTxJRXhhbXBsZT4gPSBbXTtcbiAgcHJpdmF0ZSBlbnZWYXJzOiBBcnJheTxJRW52VmFyPiA9IFtdO1xuICBwcml2YXRlIGFsaWFzZXM6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgcHJpdmF0ZSBjb21wbGV0aW9uczogTWFwPHN0cmluZywgSUNvbXBsZXRpb24+ID0gbmV3IE1hcCgpO1xuICBwcml2YXRlIGNtZDogQ29tbWFuZDxhbnk+ID0gdGhpcztcbiAgcHJpdmF0ZSBhcmdzRGVmaW5pdGlvbj86IHN0cmluZztcbiAgcHJpdmF0ZSBpc0V4ZWN1dGFibGUgPSBmYWxzZTtcbiAgcHJpdmF0ZSB0aHJvd09uRXJyb3IgPSBmYWxzZTtcbiAgcHJpdmF0ZSBfYWxsb3dFbXB0eSA9IGZhbHNlO1xuICBwcml2YXRlIF9zdG9wRWFybHkgPSBmYWxzZTtcbiAgcHJpdmF0ZSBkZWZhdWx0Q29tbWFuZD86IHN0cmluZztcbiAgcHJpdmF0ZSBfdXNlUmF3QXJncyA9IGZhbHNlO1xuICBwcml2YXRlIGFyZ3M6IEFycmF5PElBcmd1bWVudD4gPSBbXTtcbiAgcHJpdmF0ZSBpc0hpZGRlbiA9IGZhbHNlO1xuICBwcml2YXRlIGlzR2xvYmFsID0gZmFsc2U7XG4gIHByaXZhdGUgaGFzRGVmYXVsdHMgPSBmYWxzZTtcbiAgcHJpdmF0ZSBfdmVyc2lvbk9wdGlvbj86IElEZWZhdWx0T3B0aW9uIHwgZmFsc2U7XG4gIHByaXZhdGUgX2hlbHBPcHRpb24/OiBJRGVmYXVsdE9wdGlvbiB8IGZhbHNlO1xuICBwcml2YXRlIF9oZWxwPzogSUhlbHBIYW5kbGVyO1xuICBwcml2YXRlIF9zaG91bGRFeGl0PzogYm9vbGVhbjtcbiAgcHJpdmF0ZSBfbWV0YTogUmVjb3JkPHN0cmluZywgc3RyaW5nPiA9IHt9O1xuICBwcml2YXRlIF9ncm91cE5hbWU/OiBzdHJpbmc7XG5cbiAgLyoqIERpc2FibGUgdmVyc2lvbiBvcHRpb24uICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKGVuYWJsZTogZmFsc2UpOiB0aGlzO1xuICAvKipcbiAgICogU2V0IGdsb2JhbCB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGZsYWdzIFRoZSBmbGFncyBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBkZXNjICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyAgVmVyc2lvbiBvcHRpb24gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUNvbW1hbmRPcHRpb248UGFydGlhbDxDTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPiAmIHtcbiAgICAgIGdsb2JhbDogdHJ1ZTtcbiAgICB9LFxuICApOiB0aGlzO1xuICAvKipcbiAgICogU2V0IHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBWZXJzaW9uIG9wdGlvbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIHZlcnNpb25PcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuICAvKipcbiAgICogU2V0IHZlcnNpb24gb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgdmVyc2lvbiBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBUaGUgYWN0aW9uIG9mIHRoZSB2ZXJzaW9uIG9wdGlvbi5cbiAgICovXG4gIHB1YmxpYyB2ZXJzaW9uT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzogSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzO1xuICBwdWJsaWMgdmVyc2lvbk9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nIHwgZmFsc2UsXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvcHRzPzpcbiAgICAgIHwgSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+XG4gICAgICB8IElDb21tYW5kT3B0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD5cbiAgICAgIHwgSUNvbW1hbmRPcHRpb248UGFydGlhbDxDTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPiAmIHtcbiAgICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgICAgfSxcbiAgKTogdGhpcyB7XG4gICAgdGhpcy5fdmVyc2lvbk9wdGlvbiA9IGZsYWdzID09PSBmYWxzZSA/IGZsYWdzIDoge1xuICAgICAgZmxhZ3MsXG4gICAgICBkZXNjLFxuICAgICAgb3B0czogdHlwZW9mIG9wdHMgPT09IFwiZnVuY3Rpb25cIiA/IHsgYWN0aW9uOiBvcHRzIH0gOiBvcHRzLFxuICAgIH07XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogRGlzYWJsZSBoZWxwIG9wdGlvbi4gKi9cbiAgcHVibGljIGhlbHBPcHRpb24oZW5hYmxlOiBmYWxzZSk6IHRoaXM7XG4gIC8qKlxuICAgKiBTZXQgZ2xvYmFsIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZmxhZ3MgVGhlIGZsYWdzIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIGRlc2MgIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBvcHRzICBIZWxwIG9wdGlvbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbjxQYXJ0aWFsPENPPiwgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+ICYge1xuICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgIH0sXG4gICk6IHRoaXM7XG4gIC8qKlxuICAgKiBTZXQgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBUaGUgZmxhZ3Mgb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZGVzYyAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgIEhlbHAgb3B0aW9uIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgaGVscE9wdGlvbihcbiAgICBmbGFnczogc3RyaW5nLFxuICAgIGRlc2M/OiBzdHJpbmcsXG4gICAgb3B0cz86IElDb21tYW5kT3B0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICk6IHRoaXM7XG4gIC8qKlxuICAgKiBTZXQgaGVscCBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBUaGUgZmxhZ3Mgb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKiBAcGFyYW0gZGVzYyAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBoZWxwIG9wdGlvbi5cbiAgICogQHBhcmFtIG9wdHMgIFRoZSBhY3Rpb24gb2YgdGhlIGhlbHAgb3B0aW9uLlxuICAgKi9cbiAgcHVibGljIGhlbHBPcHRpb24oXG4gICAgZmxhZ3M6IHN0cmluZyxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQWN0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD4sXG4gICk6IHRoaXM7XG4gIHB1YmxpYyBoZWxwT3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcgfCBmYWxzZSxcbiAgICBkZXNjPzogc3RyaW5nLFxuICAgIG9wdHM/OlxuICAgICAgfCBJQWN0aW9uPENPLCBDQSwgQ0csIENQRywgQ1QsIENHVCwgQ1BULCBDUD5cbiAgICAgIHwgSUNvbW1hbmRPcHRpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPlxuICAgICAgfCBJQ29tbWFuZE9wdGlvbjxQYXJ0aWFsPENPPiwgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+ICYge1xuICAgICAgICBnbG9iYWw6IHRydWU7XG4gICAgICB9LFxuICApOiB0aGlzIHtcbiAgICB0aGlzLl9oZWxwT3B0aW9uID0gZmxhZ3MgPT09IGZhbHNlID8gZmxhZ3MgOiB7XG4gICAgICBmbGFncyxcbiAgICAgIGRlc2MsXG4gICAgICBvcHRzOiB0eXBlb2Ygb3B0cyA9PT0gXCJmdW5jdGlvblwiID8geyBhY3Rpb246IG9wdHMgfSA6IG9wdHMsXG4gICAgfTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgbmV3IHN1Yi1jb21tYW5kLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgIENvbW1hbmQgZGVmaW5pdGlvbi4gRS5nOiBgbXktY29tbWFuZCA8aW5wdXQtZmlsZTpzdHJpbmc+IDxvdXRwdXQtZmlsZTpzdHJpbmc+YFxuICAgKiBAcGFyYW0gY21kICAgICAgIFRoZSBuZXcgY2hpbGQgY29tbWFuZCB0byByZWdpc3Rlci5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyBjaGlsZCBjb21tYW5kLlxuICAgKi9cbiAgcHVibGljIGNvbW1hbmQ8XG4gICAgQyBleHRlbmRzIENvbW1hbmQ8XG4gICAgICBHIHwgdm9pZCB8IHVuZGVmaW5lZCxcbiAgICAgIFQgfCB2b2lkIHwgdW5kZWZpbmVkLFxuICAgICAgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQsXG4gICAgICBBcnJheTx1bmtub3duPixcbiAgICAgIFJlY29yZDxzdHJpbmcsIGFueT4gfCB2b2lkLFxuICAgICAgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQsXG4gICAgICBSZWNvcmQ8c3RyaW5nLCBhbnk+IHwgdm9pZCxcbiAgICAgIE9uZU9mPENQLCB0aGlzPiB8IHVuZGVmaW5lZFxuICAgID4sXG4gICAgRyBleHRlbmRzIChDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQRyA6IE1lcmdlPENQRywgQ0c+KSxcbiAgICBUIGV4dGVuZHMgKENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gQ1BUIDogTWVyZ2U8Q1BULCBDVD4pLFxuICA+KFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBjbWQ6IEMsXG4gICAgb3ZlcnJpZGU/OiBib29sZWFuLFxuICApOiBDIGV4dGVuZHMgQ29tbWFuZDxcbiAgICBhbnksXG4gICAgYW55LFxuICAgIGluZmVyIE9wdGlvbnMsXG4gICAgaW5mZXIgQXJndW1lbnRzLFxuICAgIGluZmVyIEdsb2JhbE9wdGlvbnMsXG4gICAgaW5mZXIgVHlwZXMsXG4gICAgaW5mZXIgR2xvYmFsVHlwZXMsXG4gICAgYW55XG4gID4gPyBDb21tYW5kPFxuICAgIEcsXG4gICAgVCxcbiAgICBPcHRpb25zLFxuICAgIEFyZ3VtZW50cyxcbiAgICBHbG9iYWxPcHRpb25zLFxuICAgIFR5cGVzLFxuICAgIEdsb2JhbFR5cGVzLFxuICAgIE9uZU9mPENQLCB0aGlzPlxuICA+XG4gICAgOiBuZXZlcjtcblxuICAvKipcbiAgICogQWRkIG5ldyBzdWItY29tbWFuZC5cbiAgICogQHBhcmFtIG5hbWUgICAgICBDb21tYW5kIGRlZmluaXRpb24uIEUuZzogYG15LWNvbW1hbmQgPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPmBcbiAgICogQHBhcmFtIGRlc2MgICAgICBUaGUgZGVzY3JpcHRpb24gb2YgdGhlIG5ldyBjaGlsZCBjb21tYW5kLlxuICAgKiBAcGFyYW0gb3ZlcnJpZGUgIE92ZXJyaWRlIGV4aXN0aW5nIGNoaWxkIGNvbW1hbmQuXG4gICAqL1xuICBwdWJsaWMgY29tbWFuZDxcbiAgICBOIGV4dGVuZHMgc3RyaW5nLFxuICAgIEEgZXh0ZW5kcyBUeXBlZENvbW1hbmRBcmd1bWVudHM8XG4gICAgICBOLFxuICAgICAgQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUFQgOiBNZXJnZTxDUFQsIENHVD5cbiAgICA+LFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgZGVzYz86IHN0cmluZyxcbiAgICBvdmVycmlkZT86IGJvb2xlYW4sXG4gICk6IENQRyBleHRlbmRzIG51bWJlciA/IENvbW1hbmQ8YW55PiA6IENvbW1hbmQ8XG4gICAgQ1AgZXh0ZW5kcyBDb21tYW5kPGFueT4gPyBDUEcgOiBNZXJnZTxDUEcsIENHPixcbiAgICBDUCBleHRlbmRzIENvbW1hbmQ8YW55PiA/IENQVCA6IE1lcmdlPENQVCwgQ0dUPixcbiAgICB2b2lkLFxuICAgIEEsXG4gICAgdm9pZCxcbiAgICB2b2lkLFxuICAgIHZvaWQsXG4gICAgT25lT2Y8Q1AsIHRoaXM+XG4gID47XG5cbiAgLyoqXG4gICAqIEFkZCBuZXcgc3ViLWNvbW1hbmQuXG4gICAqIEBwYXJhbSBuYW1lQW5kQXJndW1lbnRzICBDb21tYW5kIGRlZmluaXRpb24uIEUuZzogYG15LWNvbW1hbmQgPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPmBcbiAgICogQHBhcmFtIGNtZE9yRGVzY3JpcHRpb24gIFRoZSBkZXNjcmlwdGlvbiBvZiB0aGUgbmV3IGNoaWxkIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgICAgICAgICBPdmVycmlkZSBleGlzdGluZyBjaGlsZCBjb21tYW5kLlxuICAgKi9cbiAgY29tbWFuZChcbiAgICBuYW1lQW5kQXJndW1lbnRzOiBzdHJpbmcsXG4gICAgY21kT3JEZXNjcmlwdGlvbj86IENvbW1hbmQ8YW55PiB8IHN0cmluZyxcbiAgICBvdmVycmlkZT86IGJvb2xlYW4sXG4gICk6IENvbW1hbmQ8YW55PiB7XG4gICAgdGhpcy5yZXNldCgpO1xuXG4gICAgY29uc3QgcmVzdWx0ID0gc3BsaXRBcmd1bWVudHMobmFtZUFuZEFyZ3VtZW50cyk7XG5cbiAgICBjb25zdCBuYW1lOiBzdHJpbmcgfCB1bmRlZmluZWQgPSByZXN1bHQuZmxhZ3Muc2hpZnQoKTtcbiAgICBjb25zdCBhbGlhc2VzOiBzdHJpbmdbXSA9IHJlc3VsdC5mbGFncztcblxuICAgIGlmICghbmFtZSkge1xuICAgICAgdGhyb3cgbmV3IE1pc3NpbmdDb21tYW5kTmFtZSgpO1xuICAgIH1cblxuICAgIGlmICh0aGlzLmdldEJhc2VDb21tYW5kKG5hbWUsIHRydWUpKSB7XG4gICAgICBpZiAoIW92ZXJyaWRlKSB7XG4gICAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVDb21tYW5kTmFtZShuYW1lKTtcbiAgICAgIH1cbiAgICAgIHRoaXMucmVtb3ZlQ29tbWFuZChuYW1lKTtcbiAgICB9XG5cbiAgICBsZXQgZGVzY3JpcHRpb246IHN0cmluZyB8IHVuZGVmaW5lZDtcbiAgICBsZXQgY21kOiBDb21tYW5kPGFueT47XG5cbiAgICBpZiAodHlwZW9mIGNtZE9yRGVzY3JpcHRpb24gPT09IFwic3RyaW5nXCIpIHtcbiAgICAgIGRlc2NyaXB0aW9uID0gY21kT3JEZXNjcmlwdGlvbjtcbiAgICB9XG5cbiAgICBpZiAoY21kT3JEZXNjcmlwdGlvbiBpbnN0YW5jZW9mIENvbW1hbmQpIHtcbiAgICAgIGNtZCA9IGNtZE9yRGVzY3JpcHRpb24ucmVzZXQoKTtcbiAgICB9IGVsc2Uge1xuICAgICAgY21kID0gbmV3IENvbW1hbmQoKTtcbiAgICB9XG5cbiAgICBjbWQuX25hbWUgPSBuYW1lO1xuICAgIGNtZC5fcGFyZW50ID0gdGhpcztcblxuICAgIGlmIChkZXNjcmlwdGlvbikge1xuICAgICAgY21kLmRlc2NyaXB0aW9uKGRlc2NyaXB0aW9uKTtcbiAgICB9XG5cbiAgICBpZiAocmVzdWx0LnR5cGVEZWZpbml0aW9uKSB7XG4gICAgICBjbWQuYXJndW1lbnRzKHJlc3VsdC50eXBlRGVmaW5pdGlvbik7XG4gICAgfVxuXG4gICAgYWxpYXNlcy5mb3JFYWNoKChhbGlhczogc3RyaW5nKSA9PiBjbWQuYWxpYXMoYWxpYXMpKTtcblxuICAgIHRoaXMuY29tbWFuZHMuc2V0KG5hbWUsIGNtZCk7XG5cbiAgICB0aGlzLnNlbGVjdChuYW1lKTtcblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEFkZCBuZXcgY29tbWFuZCBhbGlhcy5cbiAgICogQHBhcmFtIGFsaWFzIFRoYSBuYW1lIG9mIHRoZSBhbGlhcy5cbiAgICovXG4gIHB1YmxpYyBhbGlhcyhhbGlhczogc3RyaW5nKTogdGhpcyB7XG4gICAgaWYgKHRoaXMuY21kLl9uYW1lID09PSBhbGlhcyB8fCB0aGlzLmNtZC5hbGlhc2VzLmluY2x1ZGVzKGFsaWFzKSkge1xuICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZUNvbW1hbmRBbGlhcyhhbGlhcyk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQuYWxpYXNlcy5wdXNoKGFsaWFzKTtcblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqIFJlc2V0IGludGVybmFsIGNvbW1hbmQgcmVmZXJlbmNlIHRvIG1haW4gY29tbWFuZC4gKi9cbiAgcHVibGljIHJlc2V0KCk6IE9uZU9mPENQLCB0aGlzPiB7XG4gICAgdGhpcy5fZ3JvdXBOYW1lID0gdW5kZWZpbmVkO1xuICAgIHRoaXMuY21kID0gdGhpcztcbiAgICByZXR1cm4gdGhpcyBhcyBPbmVPZjxDUCwgdGhpcz47XG4gIH1cblxuICAvKipcbiAgICogU2V0IGludGVybmFsIGNvbW1hbmQgcG9pbnRlciB0byBjaGlsZCBjb21tYW5kIHdpdGggZ2l2ZW4gbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgVGhlIG5hbWUgb2YgdGhlIGNvbW1hbmQgdG8gc2VsZWN0LlxuICAgKi9cbiAgcHVibGljIHNlbGVjdDxcbiAgICBPIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkID0gYW55LFxuICAgIEEgZXh0ZW5kcyBBcnJheTx1bmtub3duPiA9IGFueSxcbiAgICBHIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgdW5rbm93bj4gfCB2b2lkID0gYW55LFxuICA+KG5hbWU6IHN0cmluZyk6IENvbW1hbmQ8Q1BHLCBDUFQsIE8sIEEsIEcsIENULCBDR1QsIENQPiB7XG4gICAgY29uc3QgY21kID0gdGhpcy5nZXRCYXNlQ29tbWFuZChuYW1lLCB0cnVlKTtcblxuICAgIGlmICghY21kKSB7XG4gICAgICB0aHJvdyBuZXcgQ29tbWFuZE5vdEZvdW5kKG5hbWUsIHRoaXMuZ2V0QmFzZUNvbW1hbmRzKHRydWUpKTtcbiAgICB9XG5cbiAgICB0aGlzLmNtZCA9IGNtZDtcblxuICAgIHJldHVybiB0aGlzIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIC8qKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKiBTVUIgSEFORExFUiAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqL1xuXG4gIC8qKiBTZXQgY29tbWFuZCBuYW1lLiAqL1xuICBwdWJsaWMgbmFtZShuYW1lOiBzdHJpbmcpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC5fbmFtZSA9IG5hbWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IGNvbW1hbmQgdmVyc2lvbi5cbiAgICogQHBhcmFtIHZlcnNpb24gU2VtYW50aWMgdmVyc2lvbiBzdHJpbmcgc3RyaW5nIG9yIG1ldGhvZCB0aGF0IHJldHVybnMgdGhlIHZlcnNpb24gc3RyaW5nLlxuICAgKi9cbiAgcHVibGljIHZlcnNpb24oXG4gICAgdmVyc2lvbjpcbiAgICAgIHwgc3RyaW5nXG4gICAgICB8IElWZXJzaW9uSGFuZGxlcjxQYXJ0aWFsPENPPiwgUGFydGlhbDxDQT4sIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICApOiB0aGlzIHtcbiAgICBpZiAodHlwZW9mIHZlcnNpb24gPT09IFwic3RyaW5nXCIpIHtcbiAgICAgIHRoaXMuY21kLnZlciA9ICgpID0+IHZlcnNpb247XG4gICAgfSBlbHNlIGlmICh0eXBlb2YgdmVyc2lvbiA9PT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICB0aGlzLmNtZC52ZXIgPSB2ZXJzaW9uO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIHB1YmxpYyBtZXRhKG5hbWU6IHN0cmluZywgdmFsdWU6IHN0cmluZyk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9tZXRhW25hbWVdID0gdmFsdWU7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICBwdWJsaWMgZ2V0TWV0YSgpOiBSZWNvcmQ8c3RyaW5nLCBzdHJpbmc+O1xuICBwdWJsaWMgZ2V0TWV0YShuYW1lOiBzdHJpbmcpOiBzdHJpbmc7XG4gIHB1YmxpYyBnZXRNZXRhKG5hbWU/OiBzdHJpbmcpOiBSZWNvcmQ8c3RyaW5nLCBzdHJpbmc+IHwgc3RyaW5nIHtcbiAgICByZXR1cm4gdHlwZW9mIG5hbWUgPT09IFwidW5kZWZpbmVkXCIgPyB0aGlzLl9tZXRhIDogdGhpcy5fbWV0YVtuYW1lXTtcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgY29tbWFuZCBoZWxwLlxuICAgKiBAcGFyYW0gaGVscCBIZWxwIHN0cmluZywgbWV0aG9kLCBvciBjb25maWcgZm9yIGdlbmVyYXRvciB0aGF0IHJldHVybnMgdGhlIGhlbHAgc3RyaW5nLlxuICAgKi9cbiAgcHVibGljIGhlbHAoXG4gICAgaGVscDpcbiAgICAgIHwgc3RyaW5nXG4gICAgICB8IElIZWxwSGFuZGxlcjxQYXJ0aWFsPENPPiwgUGFydGlhbDxDQT4sIENHLCBDUEc+XG4gICAgICB8IEhlbHBPcHRpb25zLFxuICApOiB0aGlzIHtcbiAgICBpZiAodHlwZW9mIGhlbHAgPT09IFwic3RyaW5nXCIpIHtcbiAgICAgIHRoaXMuY21kLl9oZWxwID0gKCkgPT4gaGVscDtcbiAgICB9IGVsc2UgaWYgKHR5cGVvZiBoZWxwID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHRoaXMuY21kLl9oZWxwID0gaGVscDtcbiAgICB9IGVsc2Uge1xuICAgICAgdGhpcy5jbWQuX2hlbHAgPSAoY21kOiBDb21tYW5kLCBvcHRpb25zOiBIZWxwT3B0aW9ucyk6IHN0cmluZyA9PlxuICAgICAgICBIZWxwR2VuZXJhdG9yLmdlbmVyYXRlKGNtZCwgeyAuLi5oZWxwLCAuLi5vcHRpb25zIH0pO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgdGhlIGxvbmcgY29tbWFuZCBkZXNjcmlwdGlvbi5cbiAgICogQHBhcmFtIGRlc2NyaXB0aW9uIFRoZSBjb21tYW5kIGRlc2NyaXB0aW9uLlxuICAgKi9cbiAgcHVibGljIGRlc2NyaXB0aW9uKFxuICAgIGRlc2NyaXB0aW9uOiBJRGVzY3JpcHRpb248Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuZGVzYyA9IGRlc2NyaXB0aW9uO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCB0aGUgY29tbWFuZCB1c2FnZS4gRGVmYXVsdHMgdG8gYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gdXNhZ2UgVGhlIGNvbW1hbmQgdXNhZ2UuXG4gICAqL1xuICBwdWJsaWMgdXNhZ2UodXNhZ2U6IHN0cmluZyk6IHRoaXMge1xuICAgIHRoaXMuY21kLl91c2FnZSA9IHVzYWdlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEhpZGUgY29tbWFuZCBmcm9tIGhlbHAsIGNvbXBsZXRpb25zLCBldGMuXG4gICAqL1xuICBwdWJsaWMgaGlkZGVuKCk6IHRoaXMge1xuICAgIHRoaXMuY21kLmlzSGlkZGVuID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBNYWtlIGNvbW1hbmQgZ2xvYmFsbHkgYXZhaWxhYmxlLiAqL1xuICBwdWJsaWMgZ2xvYmFsKCk6IHRoaXMge1xuICAgIHRoaXMuY21kLmlzR2xvYmFsID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBNYWtlIGNvbW1hbmQgZXhlY3V0YWJsZS4gKi9cbiAgcHVibGljIGV4ZWN1dGFibGUoKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuaXNFeGVjdXRhYmxlID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgY29tbWFuZCBhcmd1bWVudHM6XG4gICAqXG4gICAqICAgPHJlcXVpcmVkQXJnOnN0cmluZz4gW29wdGlvbmFsQXJnOiBudW1iZXJdIFsuLi5yZXN0QXJnczpzdHJpbmddXG4gICAqL1xuICBwdWJsaWMgYXJndW1lbnRzPFxuICAgIEEgZXh0ZW5kcyBUeXBlZEFyZ3VtZW50czxOLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+Pj4sXG4gICAgTiBleHRlbmRzIHN0cmluZyA9IHN0cmluZyxcbiAgPihcbiAgICBhcmdzOiBOLFxuICApOiBDb21tYW5kPENQRywgQ1BULCBDTywgQSwgQ0csIENULCBDR1QsIENQPiB7XG4gICAgdGhpcy5jbWQuYXJnc0RlZmluaXRpb24gPSBhcmdzO1xuICAgIHJldHVybiB0aGlzIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgY29tbWFuZCBjYWxsYmFjayBtZXRob2QuXG4gICAqIEBwYXJhbSBmbiBDb21tYW5kIGFjdGlvbiBoYW5kbGVyLlxuICAgKi9cbiAgcHVibGljIGFjdGlvbihmbjogSUFjdGlvbjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+KTogdGhpcyB7XG4gICAgdGhpcy5jbWQuZm4gPSBmbjtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBEb24ndCB0aHJvdyBhbiBlcnJvciBpZiB0aGUgY29tbWFuZCB3YXMgY2FsbGVkIHdpdGhvdXQgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gYWxsb3dFbXB0eSBFbmFibGUvZGlzYWJsZSBhbGxvdyBlbXB0eS5cbiAgICovXG4gIHB1YmxpYyBhbGxvd0VtcHR5KGFsbG93RW1wdHkgPSB0cnVlKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX2FsbG93RW1wdHkgPSBhbGxvd0VtcHR5O1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEVuYWJsZSBzdG9wIGVhcmx5LiBJZiBlbmFibGVkLCBhbGwgYXJndW1lbnRzIHN0YXJ0aW5nIGZyb20gdGhlIGZpcnN0IG5vblxuICAgKiBvcHRpb24gYXJndW1lbnQgd2lsbCBiZSBwYXNzZWQgYXMgYXJndW1lbnRzIHdpdGggdHlwZSBzdHJpbmcgdG8gdGhlIGNvbW1hbmRcbiAgICogYWN0aW9uIGhhbmRsZXIuXG4gICAqXG4gICAqIEZvciBleGFtcGxlOlxuICAgKiAgICAgYGNvbW1hbmQgLS1kZWJ1Zy1sZXZlbCB3YXJuaW5nIHNlcnZlciAtLXBvcnQgODBgXG4gICAqXG4gICAqIFdpbGwgcmVzdWx0IGluOlxuICAgKiAgICAgLSBvcHRpb25zOiBge2RlYnVnTGV2ZWw6ICd3YXJuaW5nJ31gXG4gICAqICAgICAtIGFyZ3M6IGBbJ3NlcnZlcicsICctLXBvcnQnLCAnODAnXWBcbiAgICpcbiAgICogQHBhcmFtIHN0b3BFYXJseSBFbmFibGUvZGlzYWJsZSBzdG9wIGVhcmx5LlxuICAgKi9cbiAgcHVibGljIHN0b3BFYXJseShzdG9wRWFybHkgPSB0cnVlKTogdGhpcyB7XG4gICAgdGhpcy5jbWQuX3N0b3BFYXJseSA9IHN0b3BFYXJseTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBEaXNhYmxlIHBhcnNpbmcgYXJndW1lbnRzLiBJZiBlbmFibGVkIHRoZSByYXcgYXJndW1lbnRzIHdpbGwgYmUgcGFzc2VkIHRvXG4gICAqIHRoZSBhY3Rpb24gaGFuZGxlci4gVGhpcyBoYXMgbm8gZWZmZWN0IGZvciBwYXJlbnQgb3IgY2hpbGQgY29tbWFuZHMuIE9ubHlcbiAgICogZm9yIHRoZSBjb21tYW5kIG9uIHdoaWNoIHRoaXMgbWV0aG9kIHdhcyBjYWxsZWQuXG4gICAqIEBwYXJhbSB1c2VSYXdBcmdzIEVuYWJsZS9kaXNhYmxlIHJhdyBhcmd1bWVudHMuXG4gICAqL1xuICBwdWJsaWMgdXNlUmF3QXJncyhcbiAgICB1c2VSYXdBcmdzID0gdHJ1ZSxcbiAgKTogQ29tbWFuZDx2b2lkLCB2b2lkLCB2b2lkLCBBcnJheTxzdHJpbmc+LCB2b2lkLCB2b2lkLCB2b2lkLCBDUD4ge1xuICAgIHRoaXMuY21kLl91c2VSYXdBcmdzID0gdXNlUmF3QXJncztcbiAgICByZXR1cm4gdGhpcyBhcyBDb21tYW5kPGFueT47XG4gIH1cblxuICAvKipcbiAgICogU2V0IGRlZmF1bHQgY29tbWFuZC4gVGhlIGRlZmF1bHQgY29tbWFuZCBpcyBleGVjdXRlZCB3aGVuIHRoZSBwcm9ncmFtXG4gICAqIHdhcyBjYWxsZWQgd2l0aG91dCBhbnkgYXJndW1lbnQgYW5kIGlmIG5vIGFjdGlvbiBoYW5kbGVyIGlzIHJlZ2lzdGVyZWQuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGRlZmF1bHQgY29tbWFuZC5cbiAgICovXG4gIHB1YmxpYyBkZWZhdWx0KG5hbWU6IHN0cmluZyk6IHRoaXMge1xuICAgIHRoaXMuY21kLmRlZmF1bHRDb21tYW5kID0gbmFtZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIHB1YmxpYyBnbG9iYWxUeXBlPFxuICAgIEggZXh0ZW5kcyBUeXBlT3JUeXBlSGFuZGxlcjx1bmtub3duPixcbiAgICBOIGV4dGVuZHMgc3RyaW5nID0gc3RyaW5nLFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgaGFuZGxlcjogSCxcbiAgICBvcHRpb25zPzogT21pdDxJVHlwZU9wdGlvbnMsIFwiZ2xvYmFsXCI+LFxuICApOiBDb21tYW5kPFxuICAgIENQRyxcbiAgICBDUFQsXG4gICAgQ08sXG4gICAgQ0EsXG4gICAgQ0csXG4gICAgQ1QsXG4gICAgTWVyZ2U8Q0dULCBUeXBlZFR5cGU8TiwgSD4+LFxuICAgIENQXG4gID4ge1xuICAgIHJldHVybiB0aGlzLnR5cGUobmFtZSwgaGFuZGxlciwgeyAuLi5vcHRpb25zLCBnbG9iYWw6IHRydWUgfSk7XG4gIH1cblxuICAvKipcbiAgICogUmVnaXN0ZXIgY3VzdG9tIHR5cGUuXG4gICAqIEBwYXJhbSBuYW1lICAgIFRoZSBuYW1lIG9mIHRoZSB0eXBlLlxuICAgKiBAcGFyYW0gaGFuZGxlciBUaGUgY2FsbGJhY2sgbWV0aG9kIHRvIHBhcnNlIHRoZSB0eXBlLlxuICAgKiBAcGFyYW0gb3B0aW9ucyBUeXBlIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgdHlwZTxcbiAgICBIIGV4dGVuZHMgVHlwZU9yVHlwZUhhbmRsZXI8dW5rbm93bj4sXG4gICAgTiBleHRlbmRzIHN0cmluZyA9IHN0cmluZyxcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGhhbmRsZXI6IEgsXG4gICAgb3B0aW9ucz86IElUeXBlT3B0aW9ucyxcbiAgKTogQ29tbWFuZDxcbiAgICBDUEcsXG4gICAgQ1BULFxuICAgIENPLFxuICAgIENBLFxuICAgIENHLFxuICAgIE1lcmdlPENULCBUeXBlZFR5cGU8TiwgSD4+LFxuICAgIENHVCxcbiAgICBDUFxuICA+IHtcbiAgICBpZiAodGhpcy5jbWQudHlwZXMuZ2V0KG5hbWUpICYmICFvcHRpb25zPy5vdmVycmlkZSkge1xuICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZVR5cGUobmFtZSk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQudHlwZXMuc2V0KG5hbWUsIHsgLi4ub3B0aW9ucywgbmFtZSwgaGFuZGxlciB9KTtcblxuICAgIGlmIChcbiAgICAgIGhhbmRsZXIgaW5zdGFuY2VvZiBUeXBlICYmXG4gICAgICAodHlwZW9mIGhhbmRsZXIuY29tcGxldGUgIT09IFwidW5kZWZpbmVkXCIgfHxcbiAgICAgICAgdHlwZW9mIGhhbmRsZXIudmFsdWVzICE9PSBcInVuZGVmaW5lZFwiKVxuICAgICkge1xuICAgICAgY29uc3QgY29tcGxldGVIYW5kbGVyOiBJQ29tcGxldGVIYW5kbGVyID0gKFxuICAgICAgICBjbWQ6IENvbW1hbmQsXG4gICAgICAgIHBhcmVudD86IENvbW1hbmQsXG4gICAgICApID0+IGhhbmRsZXIuY29tcGxldGU/LihjbWQsIHBhcmVudCkgfHwgW107XG4gICAgICB0aGlzLmNvbXBsZXRlKG5hbWUsIGNvbXBsZXRlSGFuZGxlciwgb3B0aW9ucyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHRoaXMgYXMgQ29tbWFuZDxhbnk+O1xuICB9XG5cbiAgcHVibGljIGdsb2JhbENvbXBsZXRlKFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBjb21wbGV0ZTogSUNvbXBsZXRlSGFuZGxlcixcbiAgICBvcHRpb25zPzogT21pdDxJQ29tcGxldGVPcHRpb25zLCBcImdsb2JhbFwiPixcbiAgKTogdGhpcyB7XG4gICAgcmV0dXJuIHRoaXMuY29tcGxldGUobmFtZSwgY29tcGxldGUsIHsgLi4ub3B0aW9ucywgZ2xvYmFsOiB0cnVlIH0pO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlZ2lzdGVyIGNvbW1hbmQgc3BlY2lmaWMgY3VzdG9tIHR5cGUuXG4gICAqIEBwYXJhbSBuYW1lICAgICAgVGhlIG5hbWUgb2YgdGhlIGNvbXBsZXRpb24uXG4gICAqIEBwYXJhbSBjb21wbGV0ZSAgVGhlIGNhbGxiYWNrIG1ldGhvZCB0byBjb21wbGV0ZSB0aGUgdHlwZS5cbiAgICogQHBhcmFtIG9wdGlvbnMgICBDb21wbGV0ZSBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGNvbXBsZXRlKFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBjb21wbGV0ZTogSUNvbXBsZXRlSGFuZGxlcjxcbiAgICAgIFBhcnRpYWw8Q08+LFxuICAgICAgUGFydGlhbDxDQT4sXG4gICAgICBDRyxcbiAgICAgIENQRyxcbiAgICAgIENULFxuICAgICAgQ0dULFxuICAgICAgQ1BULFxuICAgICAgYW55XG4gICAgPixcbiAgICBvcHRpb25zOiBJQ29tcGxldGVPcHRpb25zICYgeyBnbG9iYWw6IGJvb2xlYW4gfSxcbiAgKTogdGhpcztcbiAgcHVibGljIGNvbXBsZXRlKFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBjb21wbGV0ZTogSUNvbXBsZXRlSGFuZGxlcjxDTywgQ0EsIENHLCBDUEcsIENULCBDR1QsIENQVCwgQ1A+LFxuICAgIG9wdGlvbnM/OiBJQ29tcGxldGVPcHRpb25zLFxuICApOiB0aGlzO1xuICBjb21wbGV0ZShcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgY29tcGxldGU6XG4gICAgICB8IElDb21wbGV0ZUhhbmRsZXI8Q08sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPlxuICAgICAgfCBJQ29tcGxldGVIYW5kbGVyPFxuICAgICAgICBQYXJ0aWFsPENPPixcbiAgICAgICAgUGFydGlhbDxDQT4sXG4gICAgICAgIENHLFxuICAgICAgICBDUEcsXG4gICAgICAgIENULFxuICAgICAgICBDR1QsXG4gICAgICAgIENQVCxcbiAgICAgICAgYW55XG4gICAgICA+LFxuICAgIG9wdGlvbnM/OiBJQ29tcGxldGVPcHRpb25zLFxuICApOiB0aGlzIHtcbiAgICBpZiAodGhpcy5jbWQuY29tcGxldGlvbnMuaGFzKG5hbWUpICYmICFvcHRpb25zPy5vdmVycmlkZSkge1xuICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZUNvbXBsZXRpb24obmFtZSk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQuY29tcGxldGlvbnMuc2V0KG5hbWUsIHtcbiAgICAgIG5hbWUsXG4gICAgICBjb21wbGV0ZSxcbiAgICAgIC4uLm9wdGlvbnMsXG4gICAgfSk7XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBUaHJvdyB2YWxpZGF0aW9uIGVycm9ycyBpbnN0ZWFkIG9mIGNhbGxpbmcgYERlbm8uZXhpdCgpYCB0byBoYW5kbGVcbiAgICogdmFsaWRhdGlvbiBlcnJvcnMgbWFudWFsbHkuXG4gICAqXG4gICAqIEEgdmFsaWRhdGlvbiBlcnJvciBpcyB0aHJvd24gd2hlbiB0aGUgY29tbWFuZCBpcyB3cm9uZ2x5IHVzZWQgYnkgdGhlIHVzZXIuXG4gICAqIEZvciBleGFtcGxlOiBJZiB0aGUgdXNlciBwYXNzZXMgc29tZSBpbnZhbGlkIG9wdGlvbnMgb3IgYXJndW1lbnRzIHRvIHRoZVxuICAgKiBjb21tYW5kLlxuICAgKlxuICAgKiBUaGlzIGhhcyBubyBlZmZlY3QgZm9yIHBhcmVudCBjb21tYW5kcy4gT25seSBmb3IgdGhlIGNvbW1hbmQgb24gd2hpY2ggdGhpc1xuICAgKiBtZXRob2Qgd2FzIGNhbGxlZCBhbmQgYWxsIGNoaWxkIGNvbW1hbmRzLlxuICAgKlxuICAgKiAqKkV4YW1wbGU6KipcbiAgICpcbiAgICogYGBgXG4gICAqIHRyeSB7XG4gICAqICAgY21kLnBhcnNlKCk7XG4gICAqIH0gY2F0Y2goZXJyb3IpIHtcbiAgICogICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBWYWxpZGF0aW9uRXJyb3IpIHtcbiAgICogICAgIGNtZC5zaG93SGVscCgpO1xuICAgKiAgICAgRGVuby5leGl0KDEpO1xuICAgKiAgIH1cbiAgICogICB0aHJvdyBlcnJvcjtcbiAgICogfVxuICAgKiBgYGBcbiAgICpcbiAgICogQHNlZSBWYWxpZGF0aW9uRXJyb3JcbiAgICovXG4gIHB1YmxpYyB0aHJvd0Vycm9ycygpOiB0aGlzIHtcbiAgICB0aGlzLmNtZC50aHJvd09uRXJyb3IgPSB0cnVlO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNhbWUgYXMgYC50aHJvd0Vycm9ycygpYCBidXQgYWxzbyBwcmV2ZW50cyBjYWxsaW5nIGBEZW5vLmV4aXRgIGFmdGVyXG4gICAqIHByaW50aW5nIGhlbHAgb3IgdmVyc2lvbiB3aXRoIHRoZSAtLWhlbHAgYW5kIC0tdmVyc2lvbiBvcHRpb24uXG4gICAqL1xuICBwdWJsaWMgbm9FeGl0KCk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9zaG91bGRFeGl0ID0gZmFsc2U7XG4gICAgdGhpcy50aHJvd0Vycm9ycygpO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqIENoZWNrIHdoZXRoZXIgdGhlIGNvbW1hbmQgc2hvdWxkIHRocm93IGVycm9ycyBvciBleGl0LiAqL1xuICBwcm90ZWN0ZWQgc2hvdWxkVGhyb3dFcnJvcnMoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuY21kLnRocm93T25FcnJvciB8fCAhIXRoaXMuY21kLl9wYXJlbnQ/LnNob3VsZFRocm93RXJyb3JzKCk7XG4gIH1cblxuICAvKiogQ2hlY2sgd2hldGhlciB0aGUgY29tbWFuZCBzaG91bGQgZXhpdCBhZnRlciBwcmludGluZyBoZWxwIG9yIHZlcnNpb24uICovXG4gIHByb3RlY3RlZCBzaG91bGRFeGl0KCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmNtZC5fc2hvdWxkRXhpdCA/PyB0aGlzLmNtZC5fcGFyZW50Py5zaG91bGRFeGl0KCkgPz8gdHJ1ZTtcbiAgfVxuXG4gIHB1YmxpYyBnbG9iYWxPcHRpb248XG4gICAgRiBleHRlbmRzIHN0cmluZyxcbiAgICBHIGV4dGVuZHMgVHlwZWRPcHRpb248RiwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSLCBEPixcbiAgICBNRyBleHRlbmRzIE1hcFZhbHVlPEcsIFYsIEM+LFxuICAgIFIgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIEMgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcImNvbGxlY3RcIl0gPSB1bmRlZmluZWQsXG4gICAgRCA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIGZsYWdzOiBGLFxuICAgIGRlc2M6IHN0cmluZyxcbiAgICBvcHRzPzpcbiAgICAgIHwgT21pdDxcbiAgICAgICAgSUNvbW1hbmRHbG9iYWxPcHRpb248XG4gICAgICAgICAgUGFydGlhbDxDTz4sXG4gICAgICAgICAgQ0EsXG4gICAgICAgICAgTWVyZ2VPcHRpb25zPEYsIENHLCBHPixcbiAgICAgICAgICBDUEcsXG4gICAgICAgICAgQ1QsXG4gICAgICAgICAgQ0dULFxuICAgICAgICAgIENQVCxcbiAgICAgICAgICBDUFxuICAgICAgICA+LFxuICAgICAgICBcInZhbHVlXCJcbiAgICAgID5cbiAgICAgICAgJiB7XG4gICAgICAgICAgZGVmYXVsdD86IElEZWZhdWx0VmFsdWU8RD47XG4gICAgICAgICAgcmVxdWlyZWQ/OiBSO1xuICAgICAgICAgIGNvbGxlY3Q/OiBDO1xuICAgICAgICAgIHZhbHVlPzogSUZsYWdWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxHPj4sIFY+O1xuICAgICAgICB9XG4gICAgICB8IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPixcbiAgKTogQ29tbWFuZDxcbiAgICBDUEcsXG4gICAgQ1BULFxuICAgIENPLFxuICAgIENBLFxuICAgIE1lcmdlT3B0aW9uczxGLCBDRywgTUc+LFxuICAgIENULFxuICAgIENHVCxcbiAgICBDUFxuICA+IHtcbiAgICBpZiAodHlwZW9mIG9wdHMgPT09IFwiZnVuY3Rpb25cIikge1xuICAgICAgcmV0dXJuIHRoaXMub3B0aW9uKFxuICAgICAgICBmbGFncyxcbiAgICAgICAgZGVzYyxcbiAgICAgICAgeyB2YWx1ZTogb3B0cywgZ2xvYmFsOiB0cnVlIH0gYXMgSUNvbW1hbmRPcHRpb24sXG4gICAgICApIGFzIENvbW1hbmQ8YW55PjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXMub3B0aW9uKFxuICAgICAgZmxhZ3MsXG4gICAgICBkZXNjLFxuICAgICAgeyAuLi5vcHRzLCBnbG9iYWw6IHRydWUgfSBhcyBJQ29tbWFuZE9wdGlvbixcbiAgICApIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIC8qKlxuICAgKiBFbmFibGUgZ3JvdXBpbmcgb2Ygb3B0aW9ucyBhbmQgc2V0IHRoZSBuYW1lIG9mIHRoZSBncm91cC5cbiAgICogQWxsIG9wdGlvbiB3aGljaCBhcmUgYWRkZWQgYWZ0ZXIgY2FsbGluZyB0aGUgYC5ncm91cCgpYCBtZXRob2Qgd2lsbCBiZVxuICAgKiBncm91cGVkIGluIHRoZSBoZWxwIG91dHB1dC4gSWYgdGhlIGAuZ3JvdXAoKWAgbWV0aG9kIGNhbiBiZSB1c2UgbXVsdGlwbGVcbiAgICogdGltZXMgdG8gY3JlYXRlIG1vcmUgZ3JvdXBzLlxuICAgKlxuICAgKiBAcGFyYW0gbmFtZSBUaGUgbmFtZSBvZiB0aGUgb3B0aW9uIGdyb3VwLlxuICAgKi9cbiAgcHVibGljIGdyb3VwKG5hbWU6IHN0cmluZyk6IHRoaXMge1xuICAgIHRoaXMuY21kLl9ncm91cE5hbWUgPSBuYW1lO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEFkZCBhIG5ldyBvcHRpb24uXG4gICAqIEBwYXJhbSBmbGFncyBGbGFncyBzdHJpbmcgZS5nOiAtaCwgLS1oZWxwLCAtLW1hbnVhbCA8cmVxdWlyZWRBcmc6c3RyaW5nPiBbb3B0aW9uYWxBcmc6bnVtYmVyXSBbLi4ucmVzdEFyZ3M6c3RyaW5nXVxuICAgKiBAcGFyYW0gZGVzYyBGbGFnIGRlc2NyaXB0aW9uLlxuICAgKiBAcGFyYW0gb3B0cyBGbGFnIG9wdGlvbnMgb3IgY3VzdG9tIGhhbmRsZXIgZm9yIHByb2Nlc3NpbmcgZmxhZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBvcHRpb248XG4gICAgRiBleHRlbmRzIHN0cmluZyxcbiAgICBHIGV4dGVuZHMgVHlwZWRPcHRpb248RiwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSLCBEPixcbiAgICBNRyBleHRlbmRzIE1hcFZhbHVlPEcsIFYsIEM+LFxuICAgIFIgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIEMgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcImNvbGxlY3RcIl0gPSB1bmRlZmluZWQsXG4gICAgRCA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIGZsYWdzOiBGLFxuICAgIGRlc2M6IHN0cmluZyxcbiAgICBvcHRzOlxuICAgICAgfCBPbWl0PFxuICAgICAgICBJQ29tbWFuZE9wdGlvbjxcbiAgICAgICAgICBQYXJ0aWFsPENPPixcbiAgICAgICAgICBDQSxcbiAgICAgICAgICBNZXJnZU9wdGlvbnM8RiwgQ0csIEc+LFxuICAgICAgICAgIENQRyxcbiAgICAgICAgICBDVCxcbiAgICAgICAgICBDR1QsXG4gICAgICAgICAgQ1BULFxuICAgICAgICAgIENQXG4gICAgICAgID4sXG4gICAgICAgIFwidmFsdWVcIlxuICAgICAgPlxuICAgICAgICAmIHtcbiAgICAgICAgICBnbG9iYWw6IHRydWU7XG4gICAgICAgICAgZGVmYXVsdD86IElEZWZhdWx0VmFsdWU8RD47XG4gICAgICAgICAgcmVxdWlyZWQ/OiBSO1xuICAgICAgICAgIGNvbGxlY3Q/OiBDO1xuICAgICAgICAgIHZhbHVlPzogSUZsYWdWYWx1ZUhhbmRsZXI8TWFwVHlwZXM8VmFsdWVPZjxHPj4sIFY+O1xuICAgICAgICB9XG4gICAgICB8IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPixcbiAgKTogQ29tbWFuZDxcbiAgICBDUEcsXG4gICAgQ1BULFxuICAgIENPLFxuICAgIENBLFxuICAgIE1lcmdlT3B0aW9uczxGLCBDRywgTUc+LFxuICAgIENULFxuICAgIENHVCxcbiAgICBDUFxuICA+O1xuXG4gIHB1YmxpYyBvcHRpb248XG4gICAgRiBleHRlbmRzIHN0cmluZyxcbiAgICBPIGV4dGVuZHMgVHlwZWRPcHRpb248RiwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSLCBEPixcbiAgICBNTyBleHRlbmRzIE1hcFZhbHVlPE8sIFYsIEM+LFxuICAgIFIgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIEMgZXh0ZW5kcyBJQ29tbWFuZE9wdGlvbltcImNvbGxlY3RcIl0gPSB1bmRlZmluZWQsXG4gICAgRCA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIGZsYWdzOiBGLFxuICAgIGRlc2M6IHN0cmluZyxcbiAgICBvcHRzPzpcbiAgICAgIHwgT21pdDxcbiAgICAgICAgSUNvbW1hbmRPcHRpb248TWVyZ2VPcHRpb25zPEYsIENPLCBNTz4sIENBLCBDRywgQ1BHLCBDVCwgQ0dULCBDUFQsIENQPixcbiAgICAgICAgXCJ2YWx1ZVwiXG4gICAgICA+XG4gICAgICAgICYge1xuICAgICAgICAgIGRlZmF1bHQ/OiBJRGVmYXVsdFZhbHVlPEQ+O1xuICAgICAgICAgIHJlcXVpcmVkPzogUjtcbiAgICAgICAgICBjb2xsZWN0PzogQztcbiAgICAgICAgICB2YWx1ZT86IElGbGFnVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Tz4+LCBWPjtcbiAgICAgICAgfVxuICAgICAgfCBJRmxhZ1ZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPE8+PiwgVj4sXG4gICk6IENvbW1hbmQ8XG4gICAgQ1BHLFxuICAgIENQVCxcbiAgICBNZXJnZU9wdGlvbnM8RiwgQ08sIE1PPixcbiAgICBDQSxcbiAgICBDRyxcbiAgICBDVCxcbiAgICBDR1QsXG4gICAgQ1BcbiAgPjtcblxuICBwdWJsaWMgb3B0aW9uKFxuICAgIGZsYWdzOiBzdHJpbmcsXG4gICAgZGVzYzogc3RyaW5nLFxuICAgIG9wdHM/OiBJQ29tbWFuZE9wdGlvbiB8IElGbGFnVmFsdWVIYW5kbGVyLFxuICApOiBDb21tYW5kPGFueT4ge1xuICAgIGlmICh0eXBlb2Ygb3B0cyA9PT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICByZXR1cm4gdGhpcy5vcHRpb24oZmxhZ3MsIGRlc2MsIHsgdmFsdWU6IG9wdHMgfSk7XG4gICAgfVxuXG4gICAgY29uc3QgcmVzdWx0ID0gc3BsaXRBcmd1bWVudHMoZmxhZ3MpO1xuXG4gICAgY29uc3QgYXJnczogSUFyZ3VtZW50W10gPSByZXN1bHQudHlwZURlZmluaXRpb25cbiAgICAgID8gcGFyc2VBcmd1bWVudHNEZWZpbml0aW9uKHJlc3VsdC50eXBlRGVmaW5pdGlvbilcbiAgICAgIDogW107XG5cbiAgICBjb25zdCBvcHRpb246IElPcHRpb24gPSB7XG4gICAgICAuLi5vcHRzLFxuICAgICAgbmFtZTogXCJcIixcbiAgICAgIGRlc2NyaXB0aW9uOiBkZXNjLFxuICAgICAgYXJncyxcbiAgICAgIGZsYWdzOiByZXN1bHQuZmxhZ3MsXG4gICAgICBlcXVhbHNTaWduOiByZXN1bHQuZXF1YWxzU2lnbixcbiAgICAgIHR5cGVEZWZpbml0aW9uOiByZXN1bHQudHlwZURlZmluaXRpb24sXG4gICAgICBncm91cE5hbWU6IHRoaXMuX2dyb3VwTmFtZSxcbiAgICB9O1xuXG4gICAgaWYgKG9wdGlvbi5zZXBhcmF0b3IpIHtcbiAgICAgIGZvciAoY29uc3QgYXJnIG9mIGFyZ3MpIHtcbiAgICAgICAgaWYgKGFyZy5saXN0KSB7XG4gICAgICAgICAgYXJnLnNlcGFyYXRvciA9IG9wdGlvbi5zZXBhcmF0b3I7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG5cbiAgICBmb3IgKGNvbnN0IHBhcnQgb2Ygb3B0aW9uLmZsYWdzKSB7XG4gICAgICBjb25zdCBhcmcgPSBwYXJ0LnRyaW0oKTtcbiAgICAgIGNvbnN0IGlzTG9uZyA9IC9eLS0vLnRlc3QoYXJnKTtcbiAgICAgIGNvbnN0IG5hbWUgPSBpc0xvbmcgPyBhcmcuc2xpY2UoMikgOiBhcmcuc2xpY2UoMSk7XG5cbiAgICAgIGlmICh0aGlzLmNtZC5nZXRCYXNlT3B0aW9uKG5hbWUsIHRydWUpKSB7XG4gICAgICAgIGlmIChvcHRzPy5vdmVycmlkZSkge1xuICAgICAgICAgIHRoaXMucmVtb3ZlT3B0aW9uKG5hbWUpO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVPcHRpb25OYW1lKG5hbWUpO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGlmICghb3B0aW9uLm5hbWUgJiYgaXNMb25nKSB7XG4gICAgICAgIG9wdGlvbi5uYW1lID0gbmFtZTtcbiAgICAgIH0gZWxzZSBpZiAoIW9wdGlvbi5hbGlhc2VzKSB7XG4gICAgICAgIG9wdGlvbi5hbGlhc2VzID0gW25hbWVdO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgb3B0aW9uLmFsaWFzZXMucHVzaChuYW1lKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBpZiAob3B0aW9uLnByZXBlbmQpIHtcbiAgICAgIHRoaXMuY21kLm9wdGlvbnMudW5zaGlmdChvcHRpb24pO1xuICAgIH0gZWxzZSB7XG4gICAgICB0aGlzLmNtZC5vcHRpb25zLnB1c2gob3B0aW9uKTtcbiAgICB9XG5cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgbmV3IGNvbW1hbmQgZXhhbXBsZS5cbiAgICogQHBhcmFtIG5hbWUgICAgICAgICAgTmFtZSBvZiB0aGUgZXhhbXBsZS5cbiAgICogQHBhcmFtIGRlc2NyaXB0aW9uICAgVGhlIGNvbnRlbnQgb2YgdGhlIGV4YW1wbGUuXG4gICAqL1xuICBwdWJsaWMgZXhhbXBsZShuYW1lOiBzdHJpbmcsIGRlc2NyaXB0aW9uOiBzdHJpbmcpOiB0aGlzIHtcbiAgICBpZiAodGhpcy5jbWQuaGFzRXhhbXBsZShuYW1lKSkge1xuICAgICAgdGhyb3cgbmV3IER1cGxpY2F0ZUV4YW1wbGUobmFtZSk7XG4gICAgfVxuXG4gICAgdGhpcy5jbWQuZXhhbXBsZXMucHVzaCh7IG5hbWUsIGRlc2NyaXB0aW9uIH0pO1xuXG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICBwdWJsaWMgZ2xvYmFsRW52PFxuICAgIE4gZXh0ZW5kcyBzdHJpbmcsXG4gICAgRyBleHRlbmRzIFR5cGVkRW52PE4sIFAsIENPLCBNZXJnZTxDUFQsIE1lcmdlPENHVCwgQ1Q+PiwgUj4sXG4gICAgTUcgZXh0ZW5kcyBNYXBWYWx1ZTxHLCBWPixcbiAgICBSIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJyZXF1aXJlZFwiXSA9IHVuZGVmaW5lZCxcbiAgICBQIGV4dGVuZHMgSUVudlZhck9wdGlvbnNbXCJwcmVmaXhcIl0gPSB1bmRlZmluZWQsXG4gICAgViA9IHVuZGVmaW5lZCxcbiAgPihcbiAgICBuYW1lOiBOLFxuICAgIGRlc2NyaXB0aW9uOiBzdHJpbmcsXG4gICAgb3B0aW9ucz86IE9taXQ8SUdsb2JhbEVudlZhck9wdGlvbnMsIFwidmFsdWVcIj4gJiB7XG4gICAgICByZXF1aXJlZD86IFI7XG4gICAgICBwcmVmaXg/OiBQO1xuICAgICAgdmFsdWU/OiBJRW52VmFyVmFsdWVIYW5kbGVyPE1hcFR5cGVzPFZhbHVlT2Y8Rz4+LCBWPjtcbiAgICB9LFxuICApOiBDb21tYW5kPENQRywgQ1BULCBDTywgQ0EsIE1lcmdlPENHLCBNRz4sIENULCBDR1QsIENQPiB7XG4gICAgcmV0dXJuIHRoaXMuZW52KFxuICAgICAgbmFtZSxcbiAgICAgIGRlc2NyaXB0aW9uLFxuICAgICAgeyAuLi5vcHRpb25zLCBnbG9iYWw6IHRydWUgfSBhcyBJRW52VmFyT3B0aW9ucyxcbiAgICApIGFzIENvbW1hbmQ8YW55PjtcbiAgfVxuXG4gIC8qKlxuICAgKiBBZGQgbmV3IGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gbmFtZSAgICAgICAgICBOYW1lIG9mIHRoZSBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIGRlc2NyaXB0aW9uICAgVGhlIGRlc2NyaXB0aW9uIG9mIHRoZSBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIG9wdGlvbnMgICAgICAgRW52aXJvbm1lbnQgdmFyaWFibGUgb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBlbnY8XG4gICAgTiBleHRlbmRzIHN0cmluZyxcbiAgICBHIGV4dGVuZHMgVHlwZWRFbnY8TiwgUCwgQ08sIE1lcmdlPENQVCwgTWVyZ2U8Q0dULCBDVD4+LCBSPixcbiAgICBNRyBleHRlbmRzIE1hcFZhbHVlPEcsIFY+LFxuICAgIFIgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInJlcXVpcmVkXCJdID0gdW5kZWZpbmVkLFxuICAgIFAgZXh0ZW5kcyBJRW52VmFyT3B0aW9uc1tcInByZWZpeFwiXSA9IHVuZGVmaW5lZCxcbiAgICBWID0gdW5kZWZpbmVkLFxuICA+KFxuICAgIG5hbWU6IE4sXG4gICAgZGVzY3JpcHRpb246IHN0cmluZyxcbiAgICBvcHRpb25zOiBPbWl0PElFbnZWYXJPcHRpb25zLCBcInZhbHVlXCI+ICYge1xuICAgICAgZ2xvYmFsOiB0cnVlO1xuICAgICAgcmVxdWlyZWQ/OiBSO1xuICAgICAgcHJlZml4PzogUDtcbiAgICAgIHZhbHVlPzogSUVudlZhclZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPEc+PiwgVj47XG4gICAgfSxcbiAgKTogQ29tbWFuZDxDUEcsIENQVCwgQ08sIENBLCBNZXJnZTxDRywgTUc+LCBDVCwgQ0dULCBDUD47XG5cbiAgcHVibGljIGVudjxcbiAgICBOIGV4dGVuZHMgc3RyaW5nLFxuICAgIE8gZXh0ZW5kcyBUeXBlZEVudjxOLCBQLCBDTywgTWVyZ2U8Q1BULCBNZXJnZTxDR1QsIENUPj4sIFI+LFxuICAgIE1PIGV4dGVuZHMgTWFwVmFsdWU8TywgVj4sXG4gICAgUiBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicmVxdWlyZWRcIl0gPSB1bmRlZmluZWQsXG4gICAgUCBleHRlbmRzIElFbnZWYXJPcHRpb25zW1wicHJlZml4XCJdID0gdW5kZWZpbmVkLFxuICAgIFYgPSB1bmRlZmluZWQsXG4gID4oXG4gICAgbmFtZTogTixcbiAgICBkZXNjcmlwdGlvbjogc3RyaW5nLFxuICAgIG9wdGlvbnM/OiBPbWl0PElFbnZWYXJPcHRpb25zLCBcInZhbHVlXCI+ICYge1xuICAgICAgcmVxdWlyZWQ/OiBSO1xuICAgICAgcHJlZml4PzogUDtcbiAgICAgIHZhbHVlPzogSUVudlZhclZhbHVlSGFuZGxlcjxNYXBUeXBlczxWYWx1ZU9mPE8+PiwgVj47XG4gICAgfSxcbiAgKTogQ29tbWFuZDxDUEcsIENQVCwgTWVyZ2U8Q08sIE1PPiwgQ0EsIENHLCBDVCwgQ0dULCBDUD47XG5cbiAgcHVibGljIGVudihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgZGVzY3JpcHRpb246IHN0cmluZyxcbiAgICBvcHRpb25zPzogSUVudlZhck9wdGlvbnMsXG4gICk6IENvbW1hbmQ8YW55PiB7XG4gICAgY29uc3QgcmVzdWx0ID0gc3BsaXRBcmd1bWVudHMobmFtZSk7XG5cbiAgICBpZiAoIXJlc3VsdC50eXBlRGVmaW5pdGlvbikge1xuICAgICAgcmVzdWx0LnR5cGVEZWZpbml0aW9uID0gXCI8dmFsdWU6Ym9vbGVhbj5cIjtcbiAgICB9XG5cbiAgICBpZiAocmVzdWx0LmZsYWdzLnNvbWUoKGVudk5hbWUpID0+IHRoaXMuY21kLmdldEJhc2VFbnZWYXIoZW52TmFtZSwgdHJ1ZSkpKSB7XG4gICAgICB0aHJvdyBuZXcgRHVwbGljYXRlRW52aXJvbm1lbnRWYXJpYWJsZShuYW1lKTtcbiAgICB9XG5cbiAgICBjb25zdCBkZXRhaWxzOiBJQXJndW1lbnRbXSA9IHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbihcbiAgICAgIHJlc3VsdC50eXBlRGVmaW5pdGlvbixcbiAgICApO1xuXG4gICAgaWYgKGRldGFpbHMubGVuZ3RoID4gMSkge1xuICAgICAgdGhyb3cgbmV3IEVudmlyb25tZW50VmFyaWFibGVTaW5nbGVWYWx1ZShuYW1lKTtcbiAgICB9IGVsc2UgaWYgKGRldGFpbHMubGVuZ3RoICYmIGRldGFpbHNbMF0ub3B0aW9uYWxWYWx1ZSkge1xuICAgICAgdGhyb3cgbmV3IEVudmlyb25tZW50VmFyaWFibGVPcHRpb25hbFZhbHVlKG5hbWUpO1xuICAgIH0gZWxzZSBpZiAoZGV0YWlscy5sZW5ndGggJiYgZGV0YWlsc1swXS52YXJpYWRpYykge1xuICAgICAgdGhyb3cgbmV3IEVudmlyb25tZW50VmFyaWFibGVWYXJpYWRpY1ZhbHVlKG5hbWUpO1xuICAgIH1cblxuICAgIHRoaXMuY21kLmVudlZhcnMucHVzaCh7XG4gICAgICBuYW1lOiByZXN1bHQuZmxhZ3NbMF0sXG4gICAgICBuYW1lczogcmVzdWx0LmZsYWdzLFxuICAgICAgZGVzY3JpcHRpb24sXG4gICAgICB0eXBlOiBkZXRhaWxzWzBdLnR5cGUsXG4gICAgICBkZXRhaWxzOiBkZXRhaWxzLnNoaWZ0KCkgYXMgSUFyZ3VtZW50LFxuICAgICAgLi4ub3B0aW9ucyxcbiAgICB9KTtcblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqIE1BSU4gSEFORExFUiAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKiovXG5cbiAgLyoqXG4gICAqIFBhcnNlIGNvbW1hbmQgbGluZSBhcmd1bWVudHMgYW5kIGV4ZWN1dGUgbWF0Y2hlZCBjb21tYW5kLlxuICAgKiBAcGFyYW0gYXJncyBDb21tYW5kIGxpbmUgYXJncyB0byBwYXJzZS4gRXg6IGBjbWQucGFyc2UoIERlbm8uYXJncyApYFxuICAgKi9cbiAgcHVibGljIGFzeW5jIHBhcnNlKFxuICAgIGFyZ3M6IHN0cmluZ1tdID0gRGVuby5hcmdzLFxuICApOiBQcm9taXNlPFxuICAgIENQIGV4dGVuZHMgQ29tbWFuZDxhbnk+ID8gSVBhcnNlUmVzdWx0PFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICBBcnJheTx1bmtub3duPixcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgICAgIFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICAgICAgUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICAgICB1bmRlZmluZWRcbiAgICA+XG4gICAgICA6IElQYXJzZVJlc3VsdDxcbiAgICAgICAgTWFwVHlwZXM8Q08+LFxuICAgICAgICBNYXBUeXBlczxDQT4sXG4gICAgICAgIE1hcFR5cGVzPENHPixcbiAgICAgICAgTWFwVHlwZXM8Q1BHPixcbiAgICAgICAgQ1QsXG4gICAgICAgIENHVCxcbiAgICAgICAgQ1BULFxuICAgICAgICBDUFxuICAgICAgPlxuICA+IHtcbiAgICB0cnkge1xuICAgICAgdGhpcy5yZXNldCgpO1xuICAgICAgdGhpcy5yZWdpc3RlckRlZmF1bHRzKCk7XG4gICAgICB0aGlzLnJhd0FyZ3MgPSBhcmdzO1xuXG4gICAgICBpZiAoYXJncy5sZW5ndGggPiAwKSB7XG4gICAgICAgIGNvbnN0IHN1YkNvbW1hbmQgPSB0aGlzLmdldENvbW1hbmQoYXJnc1swXSwgdHJ1ZSk7XG4gICAgICAgIGlmIChzdWJDb21tYW5kKSB7XG4gICAgICAgICAgc3ViQ29tbWFuZC5fZ2xvYmFsUGFyZW50ID0gdGhpcztcbiAgICAgICAgICByZXR1cm4gc3ViQ29tbWFuZC5wYXJzZShcbiAgICAgICAgICAgIHRoaXMucmF3QXJncy5zbGljZSgxKSxcbiAgICAgICAgICApIGFzIGFueTtcbiAgICAgICAgfVxuICAgICAgfVxuXG4gICAgICBpZiAodGhpcy5pc0V4ZWN1dGFibGUpIHtcbiAgICAgICAgYXdhaXQgdGhpcy5leGVjdXRlRXhlY3V0YWJsZSh0aGlzLnJhd0FyZ3MpO1xuICAgICAgICByZXR1cm4ge1xuICAgICAgICAgIG9wdGlvbnM6IHt9LFxuICAgICAgICAgIGFyZ3M6IFtdLFxuICAgICAgICAgIGNtZDogdGhpcyxcbiAgICAgICAgICBsaXRlcmFsOiBbXSxcbiAgICAgICAgfSBhcyBhbnk7XG4gICAgICB9IGVsc2UgaWYgKHRoaXMuX3VzZVJhd0FyZ3MpIHtcbiAgICAgICAgY29uc3QgZW52OiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiA9IGF3YWl0IHRoaXMucGFyc2VFbnZWYXJzKCk7XG4gICAgICAgIHJldHVybiB0aGlzLmV4ZWN1dGUoZW52LCAuLi50aGlzLnJhd0FyZ3MpIGFzIGFueTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIGNvbnN0IGVudjogUmVjb3JkPHN0cmluZywgdW5rbm93bj4gPSBhd2FpdCB0aGlzLnBhcnNlRW52VmFycygpO1xuICAgICAgICBjb25zdCB7IGFjdGlvbk9wdGlvbiwgZmxhZ3MsIHVua25vd24sIGxpdGVyYWwgfSA9IHRoaXNcbiAgICAgICAgICAucGFyc2VGbGFncyhcbiAgICAgICAgICAgIHRoaXMucmF3QXJncyxcbiAgICAgICAgICAgIGVudixcbiAgICAgICAgICApO1xuXG4gICAgICAgIHRoaXMubGl0ZXJhbEFyZ3MgPSBsaXRlcmFsO1xuXG4gICAgICAgIGNvbnN0IG9wdGlvbnM6IFJlY29yZDxzdHJpbmcsIHVua25vd24+ID0geyAuLi5lbnYsIC4uLmZsYWdzIH07XG4gICAgICAgIGNvbnN0IHBhcmFtcyA9IHRoaXMucGFyc2VBcmd1bWVudHModW5rbm93biwgb3B0aW9ucyk7XG5cbiAgICAgICAgaWYgKGFjdGlvbk9wdGlvbikge1xuICAgICAgICAgIGF3YWl0IGFjdGlvbk9wdGlvbi5hY3Rpb24uY2FsbCh0aGlzLCBvcHRpb25zLCAuLi5wYXJhbXMpO1xuICAgICAgICAgIGlmIChhY3Rpb25PcHRpb24uc3RhbmRhbG9uZSkge1xuICAgICAgICAgICAgcmV0dXJuIHtcbiAgICAgICAgICAgICAgb3B0aW9ucyxcbiAgICAgICAgICAgICAgYXJnczogcGFyYW1zLFxuICAgICAgICAgICAgICBjbWQ6IHRoaXMsXG4gICAgICAgICAgICAgIGxpdGVyYWw6IHRoaXMubGl0ZXJhbEFyZ3MsXG4gICAgICAgICAgICB9IGFzIGFueTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gdGhpcy5leGVjdXRlKG9wdGlvbnMsIC4uLnBhcmFtcykgYXMgYW55O1xuICAgICAgfVxuICAgIH0gY2F0Y2ggKGVycm9yOiB1bmtub3duKSB7XG4gICAgICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBFcnJvcikge1xuICAgICAgICB0aHJvdyB0aGlzLmVycm9yKGVycm9yKTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHRocm93IHRoaXMuZXJyb3IobmV3IEVycm9yKGBbbm9uLWVycm9yLXRocm93bl0gJHtlcnJvcn1gKSk7XG4gICAgICB9XG4gICAgfVxuICB9XG5cbiAgLyoqIFJlZ2lzdGVyIGRlZmF1bHQgb3B0aW9ucyBsaWtlIGAtLXZlcnNpb25gIGFuZCBgLS1oZWxwYC4gKi9cbiAgcHJpdmF0ZSByZWdpc3RlckRlZmF1bHRzKCk6IHRoaXMge1xuICAgIGlmICh0aGlzLmhhc0RlZmF1bHRzIHx8IHRoaXMuZ2V0UGFyZW50KCkpIHtcbiAgICAgIHJldHVybiB0aGlzO1xuICAgIH1cbiAgICB0aGlzLmhhc0RlZmF1bHRzID0gdHJ1ZTtcblxuICAgIHRoaXMucmVzZXQoKTtcblxuICAgICF0aGlzLnR5cGVzLmhhcyhcInN0cmluZ1wiKSAmJlxuICAgICAgdGhpcy50eXBlKFwic3RyaW5nXCIsIG5ldyBTdHJpbmdUeXBlKCksIHsgZ2xvYmFsOiB0cnVlIH0pO1xuICAgICF0aGlzLnR5cGVzLmhhcyhcIm51bWJlclwiKSAmJlxuICAgICAgdGhpcy50eXBlKFwibnVtYmVyXCIsIG5ldyBOdW1iZXJUeXBlKCksIHsgZ2xvYmFsOiB0cnVlIH0pO1xuICAgICF0aGlzLnR5cGVzLmhhcyhcImludGVnZXJcIikgJiZcbiAgICAgIHRoaXMudHlwZShcImludGVnZXJcIiwgbmV3IEludGVnZXJUeXBlKCksIHsgZ2xvYmFsOiB0cnVlIH0pO1xuICAgICF0aGlzLnR5cGVzLmhhcyhcImJvb2xlYW5cIikgJiZcbiAgICAgIHRoaXMudHlwZShcImJvb2xlYW5cIiwgbmV3IEJvb2xlYW5UeXBlKCksIHsgZ2xvYmFsOiB0cnVlIH0pO1xuICAgICF0aGlzLnR5cGVzLmhhcyhcImZpbGVcIikgJiZcbiAgICAgIHRoaXMudHlwZShcImZpbGVcIiwgbmV3IEZpbGVUeXBlKCksIHsgZ2xvYmFsOiB0cnVlIH0pO1xuXG4gICAgaWYgKCF0aGlzLl9oZWxwKSB7XG4gICAgICB0aGlzLmhlbHAoe1xuICAgICAgICBoaW50czogdHJ1ZSxcbiAgICAgICAgdHlwZXM6IGZhbHNlLFxuICAgICAgfSk7XG4gICAgfVxuXG4gICAgaWYgKHRoaXMuX3ZlcnNpb25PcHRpb24gIT09IGZhbHNlICYmICh0aGlzLl92ZXJzaW9uT3B0aW9uIHx8IHRoaXMudmVyKSkge1xuICAgICAgdGhpcy5vcHRpb24oXG4gICAgICAgIHRoaXMuX3ZlcnNpb25PcHRpb24/LmZsYWdzIHx8IFwiLVYsIC0tdmVyc2lvblwiLFxuICAgICAgICB0aGlzLl92ZXJzaW9uT3B0aW9uPy5kZXNjIHx8XG4gICAgICAgICAgXCJTaG93IHRoZSB2ZXJzaW9uIG51bWJlciBmb3IgdGhpcyBwcm9ncmFtLlwiLFxuICAgICAgICB7XG4gICAgICAgICAgc3RhbmRhbG9uZTogdHJ1ZSxcbiAgICAgICAgICBwcmVwZW5kOiB0cnVlLFxuICAgICAgICAgIGFjdGlvbjogYXN5bmMgZnVuY3Rpb24gKCkge1xuICAgICAgICAgICAgY29uc3QgbG9uZyA9IHRoaXMuZ2V0UmF3QXJncygpLmluY2x1ZGVzKGAtLSR7dmVyc2lvbk9wdGlvbi5uYW1lfWApO1xuICAgICAgICAgICAgaWYgKGxvbmcpIHtcbiAgICAgICAgICAgICAgYXdhaXQgdGhpcy5jaGVja1ZlcnNpb24oKTtcbiAgICAgICAgICAgICAgdGhpcy5zaG93TG9uZ1ZlcnNpb24oKTtcbiAgICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICAgIHRoaXMuc2hvd1ZlcnNpb24oKTtcbiAgICAgICAgICAgIH1cbiAgICAgICAgICAgIHRoaXMuZXhpdCgpO1xuICAgICAgICAgIH0sXG4gICAgICAgICAgLi4uKHRoaXMuX3ZlcnNpb25PcHRpb24/Lm9wdHMgPz8ge30pLFxuICAgICAgICB9LFxuICAgICAgKTtcbiAgICAgIGNvbnN0IHZlcnNpb25PcHRpb24gPSB0aGlzLm9wdGlvbnNbMF07XG4gICAgfVxuXG4gICAgaWYgKHRoaXMuX2hlbHBPcHRpb24gIT09IGZhbHNlKSB7XG4gICAgICB0aGlzLm9wdGlvbihcbiAgICAgICAgdGhpcy5faGVscE9wdGlvbj8uZmxhZ3MgfHwgXCItaCwgLS1oZWxwXCIsXG4gICAgICAgIHRoaXMuX2hlbHBPcHRpb24/LmRlc2MgfHwgXCJTaG93IHRoaXMgaGVscC5cIixcbiAgICAgICAge1xuICAgICAgICAgIHN0YW5kYWxvbmU6IHRydWUsXG4gICAgICAgICAgZ2xvYmFsOiB0cnVlLFxuICAgICAgICAgIHByZXBlbmQ6IHRydWUsXG4gICAgICAgICAgYWN0aW9uOiBhc3luYyBmdW5jdGlvbiAoKSB7XG4gICAgICAgICAgICBjb25zdCBsb25nID0gdGhpcy5nZXRSYXdBcmdzKCkuaW5jbHVkZXMoYC0tJHtoZWxwT3B0aW9uLm5hbWV9YCk7XG4gICAgICAgICAgICBhd2FpdCB0aGlzLmNoZWNrVmVyc2lvbigpO1xuICAgICAgICAgICAgdGhpcy5zaG93SGVscCh7IGxvbmcgfSk7XG4gICAgICAgICAgICB0aGlzLmV4aXQoKTtcbiAgICAgICAgICB9LFxuICAgICAgICAgIC4uLih0aGlzLl9oZWxwT3B0aW9uPy5vcHRzID8/IHt9KSxcbiAgICAgICAgfSxcbiAgICAgICk7XG4gICAgICBjb25zdCBoZWxwT3B0aW9uID0gdGhpcy5vcHRpb25zWzBdO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEV4ZWN1dGUgY29tbWFuZC5cbiAgICogQHBhcmFtIG9wdGlvbnMgQSBtYXAgb2Ygb3B0aW9ucy5cbiAgICogQHBhcmFtIGFyZ3MgQ29tbWFuZCBhcmd1bWVudHMuXG4gICAqL1xuICBwcm90ZWN0ZWQgYXN5bmMgZXhlY3V0ZShcbiAgICBvcHRpb25zOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgICAuLi5hcmdzOiBBcnJheTx1bmtub3duPlxuICApOiBQcm9taXNlPElQYXJzZVJlc3VsdD4ge1xuICAgIGlmICh0aGlzLmZuKSB7XG4gICAgICBhd2FpdCB0aGlzLmZuKG9wdGlvbnMsIC4uLmFyZ3MpO1xuICAgIH0gZWxzZSBpZiAodGhpcy5kZWZhdWx0Q29tbWFuZCkge1xuICAgICAgY29uc3QgY21kID0gdGhpcy5nZXRDb21tYW5kKHRoaXMuZGVmYXVsdENvbW1hbmQsIHRydWUpO1xuXG4gICAgICBpZiAoIWNtZCkge1xuICAgICAgICB0aHJvdyBuZXcgRGVmYXVsdENvbW1hbmROb3RGb3VuZChcbiAgICAgICAgICB0aGlzLmRlZmF1bHRDb21tYW5kLFxuICAgICAgICAgIHRoaXMuZ2V0Q29tbWFuZHMoKSxcbiAgICAgICAgKTtcbiAgICAgIH1cblxuICAgICAgY21kLl9nbG9iYWxQYXJlbnQgPSB0aGlzO1xuICAgICAgYXdhaXQgY21kLmV4ZWN1dGUob3B0aW9ucywgLi4uYXJncyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHtcbiAgICAgIG9wdGlvbnMsXG4gICAgICBhcmdzLFxuICAgICAgY21kOiB0aGlzLFxuICAgICAgbGl0ZXJhbDogdGhpcy5saXRlcmFsQXJncyxcbiAgICB9O1xuICB9XG5cbiAgLyoqXG4gICAqIEV4ZWN1dGUgZXh0ZXJuYWwgc3ViLWNvbW1hbmQuXG4gICAqIEBwYXJhbSBhcmdzIFJhdyBjb21tYW5kIGxpbmUgYXJndW1lbnRzLlxuICAgKi9cbiAgcHJvdGVjdGVkIGFzeW5jIGV4ZWN1dGVFeGVjdXRhYmxlKGFyZ3M6IHN0cmluZ1tdKSB7XG4gICAgY29uc3QgY29tbWFuZCA9IHRoaXMuZ2V0UGF0aCgpLnJlcGxhY2UoL1xccysvZywgXCItXCIpO1xuXG4gICAgYXdhaXQgRGVuby5wZXJtaXNzaW9ucy5yZXF1ZXN0KHsgbmFtZTogXCJydW5cIiwgY29tbWFuZCB9KTtcblxuICAgIHRyeSB7XG4gICAgICBjb25zdCBwcm9jZXNzOiBEZW5vLlByb2Nlc3MgPSBEZW5vLnJ1bih7XG4gICAgICAgIGNtZDogW2NvbW1hbmQsIC4uLmFyZ3NdLFxuICAgICAgfSk7XG4gICAgICBjb25zdCBzdGF0dXM6IERlbm8uUHJvY2Vzc1N0YXR1cyA9IGF3YWl0IHByb2Nlc3Muc3RhdHVzKCk7XG4gICAgICBpZiAoIXN0YXR1cy5zdWNjZXNzKSB7XG4gICAgICAgIERlbm8uZXhpdChzdGF0dXMuY29kZSk7XG4gICAgICB9XG4gICAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICAgIGlmIChlcnJvciBpbnN0YW5jZW9mIERlbm8uZXJyb3JzLk5vdEZvdW5kKSB7XG4gICAgICAgIHRocm93IG5ldyBDb21tYW5kRXhlY3V0YWJsZU5vdEZvdW5kKGNvbW1hbmQpO1xuICAgICAgfVxuICAgICAgdGhyb3cgZXJyb3I7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFBhcnNlIHJhdyBjb21tYW5kIGxpbmUgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gYXJncyBSYXcgY29tbWFuZCBsaW5lIGFyZ3VtZW50cy5cbiAgICovXG4gIHByb3RlY3RlZCBwYXJzZUZsYWdzKFxuICAgIGFyZ3M6IHN0cmluZ1tdLFxuICAgIGVudjogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gICk6IElGbGFnc1Jlc3VsdCAmIHsgYWN0aW9uT3B0aW9uPzogSU9wdGlvbiAmIHsgYWN0aW9uOiBJQWN0aW9uIH0gfSB7XG4gICAgdHJ5IHtcbiAgICAgIGxldCBhY3Rpb25PcHRpb246IElPcHRpb24gJiB7IGFjdGlvbjogSUFjdGlvbiB9IHwgdW5kZWZpbmVkO1xuICAgICAgY29uc3QgcmVzdWx0ID0gcGFyc2VGbGFncyhhcmdzLCB7XG4gICAgICAgIHN0b3BFYXJseTogdGhpcy5fc3RvcEVhcmx5LFxuICAgICAgICBhbGxvd0VtcHR5OiB0aGlzLl9hbGxvd0VtcHR5LFxuICAgICAgICBmbGFnczogdGhpcy5nZXRPcHRpb25zKHRydWUpLFxuICAgICAgICBpZ25vcmVEZWZhdWx0czogZW52LFxuICAgICAgICBwYXJzZTogKHR5cGU6IElUeXBlSW5mbykgPT4gdGhpcy5wYXJzZVR5cGUodHlwZSksXG4gICAgICAgIG9wdGlvbjogKG9wdGlvbjogSU9wdGlvbikgPT4ge1xuICAgICAgICAgIGlmICghYWN0aW9uT3B0aW9uICYmIG9wdGlvbi5hY3Rpb24pIHtcbiAgICAgICAgICAgIGFjdGlvbk9wdGlvbiA9IG9wdGlvbiBhcyBJT3B0aW9uICYgeyBhY3Rpb246IElBY3Rpb24gfTtcbiAgICAgICAgICB9XG4gICAgICAgIH0sXG4gICAgICB9KTtcbiAgICAgIHJldHVybiB7IC4uLnJlc3VsdCwgYWN0aW9uT3B0aW9uIH07XG4gICAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICAgIGlmIChlcnJvciBpbnN0YW5jZW9mIEZsYWdzVmFsaWRhdGlvbkVycm9yKSB7XG4gICAgICAgIHRocm93IG5ldyBWYWxpZGF0aW9uRXJyb3IoZXJyb3IubWVzc2FnZSk7XG4gICAgICB9XG4gICAgICB0aHJvdyBlcnJvcjtcbiAgICB9XG4gIH1cblxuICAvKiogUGFyc2UgYXJndW1lbnQgdHlwZS4gKi9cbiAgcHJvdGVjdGVkIHBhcnNlVHlwZSh0eXBlOiBJVHlwZUluZm8pOiB1bmtub3duIHtcbiAgICBjb25zdCB0eXBlU2V0dGluZ3M6IElUeXBlIHwgdW5kZWZpbmVkID0gdGhpcy5nZXRUeXBlKHR5cGUudHlwZSk7XG5cbiAgICBpZiAoIXR5cGVTZXR0aW5ncykge1xuICAgICAgdGhyb3cgbmV3IFVua25vd25UeXBlKFxuICAgICAgICB0eXBlLnR5cGUsXG4gICAgICAgIHRoaXMuZ2V0VHlwZXMoKS5tYXAoKHR5cGUpID0+IHR5cGUubmFtZSksXG4gICAgICApO1xuICAgIH1cblxuICAgIHJldHVybiB0eXBlU2V0dGluZ3MuaGFuZGxlciBpbnN0YW5jZW9mIFR5cGVcbiAgICAgID8gdHlwZVNldHRpbmdzLmhhbmRsZXIucGFyc2UodHlwZSlcbiAgICAgIDogdHlwZVNldHRpbmdzLmhhbmRsZXIodHlwZSk7XG4gIH1cblxuICAvKiogVmFsaWRhdGUgZW52aXJvbm1lbnQgdmFyaWFibGVzLiAqL1xuICBwcm90ZWN0ZWQgYXN5bmMgcGFyc2VFbnZWYXJzKCk6IFByb21pc2U8UmVjb3JkPHN0cmluZywgdW5rbm93bj4+IHtcbiAgICBjb25zdCBlbnZWYXJzID0gdGhpcy5nZXRFbnZWYXJzKHRydWUpO1xuICAgIGNvbnN0IHJlc3VsdDogUmVjb3JkPHN0cmluZywgdW5rbm93bj4gPSB7fTtcblxuICAgIGlmICghZW52VmFycy5sZW5ndGgpIHtcbiAgICAgIHJldHVybiByZXN1bHQ7XG4gICAgfVxuXG4gICAgY29uc3QgaGFzRW52UGVybWlzc2lvbnMgPSAoYXdhaXQgRGVuby5wZXJtaXNzaW9ucy5xdWVyeSh7XG4gICAgICBuYW1lOiBcImVudlwiLFxuICAgIH0pKS5zdGF0ZSA9PT0gXCJncmFudGVkXCI7XG5cbiAgICBmb3IgKGNvbnN0IGVudiBvZiBlbnZWYXJzKSB7XG4gICAgICBjb25zdCBuYW1lID0gaGFzRW52UGVybWlzc2lvbnMgJiYgZW52Lm5hbWVzLmZpbmQoXG4gICAgICAgIChuYW1lOiBzdHJpbmcpID0+ICEhRGVuby5lbnYuZ2V0KG5hbWUpLFxuICAgICAgKTtcblxuICAgICAgaWYgKG5hbWUpIHtcbiAgICAgICAgY29uc3QgcHJvcGVydHlOYW1lID0gdW5kZXJzY29yZVRvQ2FtZWxDYXNlKFxuICAgICAgICAgIGVudi5wcmVmaXhcbiAgICAgICAgICAgID8gZW52Lm5hbWVzWzBdLnJlcGxhY2UobmV3IFJlZ0V4cChgXiR7ZW52LnByZWZpeH1gKSwgXCJcIilcbiAgICAgICAgICAgIDogZW52Lm5hbWVzWzBdLFxuICAgICAgICApO1xuXG4gICAgICAgIGlmIChlbnYuZGV0YWlscy5saXN0KSB7XG4gICAgICAgICAgY29uc3QgdmFsdWVzID0gRGVuby5lbnYuZ2V0KG5hbWUpXG4gICAgICAgICAgICA/LnNwbGl0KGVudi5kZXRhaWxzLnNlcGFyYXRvciA/PyBcIixcIikgPz8gW1wiXCJdO1xuXG4gICAgICAgICAgcmVzdWx0W3Byb3BlcnR5TmFtZV0gPSB2YWx1ZXMubWFwKCh2YWx1ZSkgPT5cbiAgICAgICAgICAgIHRoaXMucGFyc2VUeXBlKHtcbiAgICAgICAgICAgICAgbGFiZWw6IFwiRW52aXJvbm1lbnQgdmFyaWFibGVcIixcbiAgICAgICAgICAgICAgdHlwZTogZW52LnR5cGUsXG4gICAgICAgICAgICAgIG5hbWUsXG4gICAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgICAgfSlcbiAgICAgICAgICApO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdFtwcm9wZXJ0eU5hbWVdID0gdGhpcy5wYXJzZVR5cGUoe1xuICAgICAgICAgICAgbGFiZWw6IFwiRW52aXJvbm1lbnQgdmFyaWFibGVcIixcbiAgICAgICAgICAgIHR5cGU6IGVudi50eXBlLFxuICAgICAgICAgICAgbmFtZSxcbiAgICAgICAgICAgIHZhbHVlOiBEZW5vLmVudi5nZXQobmFtZSkgPz8gXCJcIixcbiAgICAgICAgICB9KTtcbiAgICAgICAgfVxuXG4gICAgICAgIGlmIChlbnYudmFsdWUgJiYgdHlwZW9mIHJlc3VsdFtwcm9wZXJ0eU5hbWVdICE9PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICAgICAgcmVzdWx0W3Byb3BlcnR5TmFtZV0gPSBlbnYudmFsdWUocmVzdWx0W3Byb3BlcnR5TmFtZV0pO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGVudi5yZXF1aXJlZCkge1xuICAgICAgICB0aHJvdyBuZXcgTWlzc2luZ1JlcXVpcmVkRW52VmFyKGVudik7XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxuXG4gIC8qKlxuICAgKiBQYXJzZSBjb21tYW5kLWxpbmUgYXJndW1lbnRzLlxuICAgKiBAcGFyYW0gYXJncyAgUmF3IGNvbW1hbmQgbGluZSBhcmd1bWVudHMuXG4gICAqIEBwYXJhbSBmbGFncyBQYXJzZWQgY29tbWFuZCBsaW5lIG9wdGlvbnMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcGFyc2VBcmd1bWVudHMoYXJnczogc3RyaW5nW10sIGZsYWdzOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPik6IENBIHtcbiAgICBjb25zdCBwYXJhbXM6IEFycmF5PHVua25vd24+ID0gW107XG5cbiAgICAvLyByZW1vdmUgYXJyYXkgcmVmZXJlbmNlXG4gICAgYXJncyA9IGFyZ3Muc2xpY2UoMCk7XG5cbiAgICBpZiAoIXRoaXMuaGFzQXJndW1lbnRzKCkpIHtcbiAgICAgIGlmIChhcmdzLmxlbmd0aCkge1xuICAgICAgICBpZiAodGhpcy5oYXNDb21tYW5kcyh0cnVlKSkge1xuICAgICAgICAgIHRocm93IG5ldyBVbmtub3duQ29tbWFuZChhcmdzWzBdLCB0aGlzLmdldENvbW1hbmRzKCkpO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHRocm93IG5ldyBOb0FyZ3VtZW50c0FsbG93ZWQodGhpcy5nZXRQYXRoKCkpO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfSBlbHNlIHtcbiAgICAgIGlmICghYXJncy5sZW5ndGgpIHtcbiAgICAgICAgY29uc3QgcmVxdWlyZWQgPSB0aGlzLmdldEFyZ3VtZW50cygpXG4gICAgICAgICAgLmZpbHRlcigoZXhwZWN0ZWRBcmcpID0+ICFleHBlY3RlZEFyZy5vcHRpb25hbFZhbHVlKVxuICAgICAgICAgIC5tYXAoKGV4cGVjdGVkQXJnKSA9PiBleHBlY3RlZEFyZy5uYW1lKTtcblxuICAgICAgICBpZiAocmVxdWlyZWQubGVuZ3RoKSB7XG4gICAgICAgICAgY29uc3QgZmxhZ05hbWVzOiBzdHJpbmdbXSA9IE9iamVjdC5rZXlzKGZsYWdzKTtcbiAgICAgICAgICBjb25zdCBoYXNTdGFuZGFsb25lT3B0aW9uID0gISFmbGFnTmFtZXMuZmluZCgobmFtZSkgPT5cbiAgICAgICAgICAgIHRoaXMuZ2V0T3B0aW9uKG5hbWUsIHRydWUpPy5zdGFuZGFsb25lXG4gICAgICAgICAgKTtcblxuICAgICAgICAgIGlmICghaGFzU3RhbmRhbG9uZU9wdGlvbikge1xuICAgICAgICAgICAgdGhyb3cgbmV3IE1pc3NpbmdBcmd1bWVudHMocmVxdWlyZWQpO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuICAgICAgfSBlbHNlIHtcbiAgICAgICAgZm9yIChjb25zdCBleHBlY3RlZEFyZyBvZiB0aGlzLmdldEFyZ3VtZW50cygpKSB7XG4gICAgICAgICAgaWYgKCFhcmdzLmxlbmd0aCkge1xuICAgICAgICAgICAgaWYgKGV4cGVjdGVkQXJnLm9wdGlvbmFsVmFsdWUpIHtcbiAgICAgICAgICAgICAgYnJlYWs7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgICB0aHJvdyBuZXcgTWlzc2luZ0FyZ3VtZW50KGBNaXNzaW5nIGFyZ3VtZW50OiAke2V4cGVjdGVkQXJnLm5hbWV9YCk7XG4gICAgICAgICAgfVxuXG4gICAgICAgICAgbGV0IGFyZzogdW5rbm93bjtcblxuICAgICAgICAgIGlmIChleHBlY3RlZEFyZy52YXJpYWRpYykge1xuICAgICAgICAgICAgYXJnID0gYXJncy5zcGxpY2UoMCwgYXJncy5sZW5ndGgpXG4gICAgICAgICAgICAgIC5tYXAoKHZhbHVlKSA9PlxuICAgICAgICAgICAgICAgIHRoaXMucGFyc2VUeXBlKHtcbiAgICAgICAgICAgICAgICAgIGxhYmVsOiBcIkFyZ3VtZW50XCIsXG4gICAgICAgICAgICAgICAgICB0eXBlOiBleHBlY3RlZEFyZy50eXBlLFxuICAgICAgICAgICAgICAgICAgbmFtZTogZXhwZWN0ZWRBcmcubmFtZSxcbiAgICAgICAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgICAgICAgIH0pXG4gICAgICAgICAgICAgICk7XG4gICAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICAgIGFyZyA9IHRoaXMucGFyc2VUeXBlKHtcbiAgICAgICAgICAgICAgbGFiZWw6IFwiQXJndW1lbnRcIixcbiAgICAgICAgICAgICAgdHlwZTogZXhwZWN0ZWRBcmcudHlwZSxcbiAgICAgICAgICAgICAgbmFtZTogZXhwZWN0ZWRBcmcubmFtZSxcbiAgICAgICAgICAgICAgdmFsdWU6IGFyZ3Muc2hpZnQoKSBhcyBzdHJpbmcsXG4gICAgICAgICAgICB9KTtcbiAgICAgICAgICB9XG5cbiAgICAgICAgICBpZiAodHlwZW9mIGFyZyAhPT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICAgICAgcGFyYW1zLnB1c2goYXJnKTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoYXJncy5sZW5ndGgpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgVG9vTWFueUFyZ3VtZW50cyhhcmdzKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiBwYXJhbXMgYXMgQ0E7XG4gIH1cblxuICAvKipcbiAgICogSGFuZGxlIGVycm9yLiBJZiBgdGhyb3dFcnJvcnNgIGlzIGVuYWJsZWQgdGhlIGVycm9yIHdpbGwgYmUgcmV0dXJuZWQsXG4gICAqIG90aGVyd2lzZSBhIGZvcm1hdHRlZCBlcnJvciBtZXNzYWdlIHdpbGwgYmUgcHJpbnRlZCBhbmQgYERlbm8uZXhpdCgxKWBcbiAgICogd2lsbCBiZSBjYWxsZWQuXG4gICAqIEBwYXJhbSBlcnJvciBFcnJvciB0byBoYW5kbGUuXG4gICAqL1xuICBwcm90ZWN0ZWQgZXJyb3IoZXJyb3I6IEVycm9yKTogRXJyb3Ige1xuICAgIGlmICh0aGlzLnNob3VsZFRocm93RXJyb3JzKCkgfHwgIShlcnJvciBpbnN0YW5jZW9mIFZhbGlkYXRpb25FcnJvcikpIHtcbiAgICAgIHJldHVybiBlcnJvcjtcbiAgICB9XG4gICAgdGhpcy5zaG93SGVscCgpO1xuICAgIGNvbnNvbGUuZXJyb3IocmVkKGAgICR7Ym9sZChcImVycm9yXCIpfTogJHtlcnJvci5tZXNzYWdlfVxcbmApKTtcbiAgICBEZW5vLmV4aXQoZXJyb3IgaW5zdGFuY2VvZiBWYWxpZGF0aW9uRXJyb3IgPyBlcnJvci5leGl0Q29kZSA6IDEpO1xuICB9XG5cbiAgLyoqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqIEdFVFRFUiAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKlxuICAgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKiovXG5cbiAgLyoqIEdldCBjb21tYW5kIG5hbWUuICovXG4gIHB1YmxpYyBnZXROYW1lKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIHRoaXMuX25hbWU7XG4gIH1cblxuICAvKiogR2V0IHBhcmVudCBjb21tYW5kLiAqL1xuICBwdWJsaWMgZ2V0UGFyZW50KCk6IENQIHtcbiAgICByZXR1cm4gdGhpcy5fcGFyZW50IGFzIENQO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBwYXJlbnQgY29tbWFuZCBmcm9tIGdsb2JhbCBleGVjdXRlZCBjb21tYW5kLlxuICAgKiBCZSBzdXJlLCB0byBjYWxsIHRoaXMgbWV0aG9kIG9ubHkgaW5zaWRlIGFuIGFjdGlvbiBoYW5kbGVyLiBVbmxlc3MgdGhpcyBvciBhbnkgY2hpbGQgY29tbWFuZCB3YXMgZXhlY3V0ZWQsXG4gICAqIHRoaXMgbWV0aG9kIHJldHVybnMgYWx3YXlzIHVuZGVmaW5lZC5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxQYXJlbnQoKTogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5fZ2xvYmFsUGFyZW50O1xuICB9XG5cbiAgLyoqIEdldCBtYWluIGNvbW1hbmQuICovXG4gIHB1YmxpYyBnZXRNYWluQ29tbWFuZCgpOiBDb21tYW5kPGFueT4ge1xuICAgIHJldHVybiB0aGlzLl9wYXJlbnQ/LmdldE1haW5Db21tYW5kKCkgPz8gdGhpcztcbiAgfVxuXG4gIC8qKiBHZXQgY29tbWFuZCBuYW1lIGFsaWFzZXMuICovXG4gIHB1YmxpYyBnZXRBbGlhc2VzKCk6IHN0cmluZ1tdIHtcbiAgICByZXR1cm4gdGhpcy5hbGlhc2VzO1xuICB9XG5cbiAgLyoqIEdldCBmdWxsIGNvbW1hbmQgcGF0aC4gKi9cbiAgcHVibGljIGdldFBhdGgoKTogc3RyaW5nIHtcbiAgICByZXR1cm4gdGhpcy5fcGFyZW50XG4gICAgICA/IHRoaXMuX3BhcmVudC5nZXRQYXRoKCkgKyBcIiBcIiArIHRoaXMuX25hbWVcbiAgICAgIDogdGhpcy5fbmFtZTtcbiAgfVxuXG4gIC8qKiBHZXQgYXJndW1lbnRzIGRlZmluaXRpb24uIEUuZzogPGlucHV0LWZpbGU6c3RyaW5nPiA8b3V0cHV0LWZpbGU6c3RyaW5nPiAqL1xuICBwdWJsaWMgZ2V0QXJnc0RlZmluaXRpb24oKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5hcmdzRGVmaW5pdGlvbjtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYXJndW1lbnQgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgYXJndW1lbnQuXG4gICAqL1xuICBwdWJsaWMgZ2V0QXJndW1lbnQobmFtZTogc3RyaW5nKTogSUFyZ3VtZW50IHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRBcmd1bWVudHMoKS5maW5kKChhcmcpID0+IGFyZy5uYW1lID09PSBuYW1lKTtcbiAgfVxuXG4gIC8qKiBHZXQgYXJndW1lbnRzLiAqL1xuICBwdWJsaWMgZ2V0QXJndW1lbnRzKCk6IElBcmd1bWVudFtdIHtcbiAgICBpZiAoIXRoaXMuYXJncy5sZW5ndGggJiYgdGhpcy5hcmdzRGVmaW5pdGlvbikge1xuICAgICAgdGhpcy5hcmdzID0gcGFyc2VBcmd1bWVudHNEZWZpbml0aW9uKHRoaXMuYXJnc0RlZmluaXRpb24pO1xuICAgIH1cblxuICAgIHJldHVybiB0aGlzLmFyZ3M7XG4gIH1cblxuICAvKiogQ2hlY2sgaWYgY29tbWFuZCBoYXMgYXJndW1lbnRzLiAqL1xuICBwdWJsaWMgaGFzQXJndW1lbnRzKCkge1xuICAgIHJldHVybiAhIXRoaXMuYXJnc0RlZmluaXRpb247XG4gIH1cblxuICAvKiogR2V0IGNvbW1hbmQgdmVyc2lvbi4gKi9cbiAgcHVibGljIGdldFZlcnNpb24oKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRWZXJzaW9uSGFuZGxlcigpPy5jYWxsKHRoaXMsIHRoaXMpO1xuICB9XG5cbiAgLyoqIEdldCBoZWxwIGhhbmRsZXIgbWV0aG9kLiAqL1xuICBwcml2YXRlIGdldFZlcnNpb25IYW5kbGVyKCk6IElWZXJzaW9uSGFuZGxlciB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMudmVyID8/IHRoaXMuX3BhcmVudD8uZ2V0VmVyc2lvbkhhbmRsZXIoKTtcbiAgfVxuXG4gIC8qKiBHZXQgY29tbWFuZCBkZXNjcmlwdGlvbi4gKi9cbiAgcHVibGljIGdldERlc2NyaXB0aW9uKCk6IHN0cmluZyB7XG4gICAgLy8gY2FsbCBkZXNjcmlwdGlvbiBtZXRob2Qgb25seSBvbmNlXG4gICAgcmV0dXJuIHR5cGVvZiB0aGlzLmRlc2MgPT09IFwiZnVuY3Rpb25cIlxuICAgICAgPyB0aGlzLmRlc2MgPSB0aGlzLmRlc2MoKVxuICAgICAgOiB0aGlzLmRlc2M7XG4gIH1cblxuICBwdWJsaWMgZ2V0VXNhZ2UoKSB7XG4gICAgcmV0dXJuIHRoaXMuX3VzYWdlID8/IHRoaXMuZ2V0QXJnc0RlZmluaXRpb24oKTtcbiAgfVxuXG4gIC8qKiBHZXQgc2hvcnQgY29tbWFuZCBkZXNjcmlwdGlvbi4gVGhpcyBpcyB0aGUgZmlyc3QgbGluZSBvZiB0aGUgZGVzY3JpcHRpb24uICovXG4gIHB1YmxpYyBnZXRTaG9ydERlc2NyaXB0aW9uKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIGdldERlc2NyaXB0aW9uKHRoaXMuZ2V0RGVzY3JpcHRpb24oKSwgdHJ1ZSk7XG4gIH1cblxuICAvKiogR2V0IG9yaWdpbmFsIGNvbW1hbmQtbGluZSBhcmd1bWVudHMuICovXG4gIHB1YmxpYyBnZXRSYXdBcmdzKCk6IHN0cmluZ1tdIHtcbiAgICByZXR1cm4gdGhpcy5yYXdBcmdzO1xuICB9XG5cbiAgLyoqIEdldCBhbGwgYXJndW1lbnRzIGRlZmluZWQgYWZ0ZXIgdGhlIGRvdWJsZSBkYXNoLiAqL1xuICBwdWJsaWMgZ2V0TGl0ZXJhbEFyZ3MoKTogc3RyaW5nW10ge1xuICAgIHJldHVybiB0aGlzLmxpdGVyYWxBcmdzO1xuICB9XG5cbiAgLyoqIE91dHB1dCBnZW5lcmF0ZWQgaGVscCB3aXRob3V0IGV4aXRpbmcuICovXG4gIHB1YmxpYyBzaG93VmVyc2lvbigpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldFZlcnNpb24oKSk7XG4gIH1cblxuICAvKiogUmV0dXJucyBjb21tYW5kIG5hbWUsIHZlcnNpb24gYW5kIG1ldGEgZGF0YS4gKi9cbiAgcHVibGljIGdldExvbmdWZXJzaW9uKCk6IHN0cmluZyB7XG4gICAgcmV0dXJuIGAke2JvbGQodGhpcy5nZXRNYWluQ29tbWFuZCgpLmdldE5hbWUoKSl9ICR7XG4gICAgICBibHVlKHRoaXMuZ2V0VmVyc2lvbigpID8/IFwiXCIpXG4gICAgfWAgK1xuICAgICAgT2JqZWN0LmVudHJpZXModGhpcy5nZXRNZXRhKCkpLm1hcChcbiAgICAgICAgKFtrLCB2XSkgPT4gYFxcbiR7Ym9sZChrKX0gJHtibHVlKHYpfWAsXG4gICAgICApLmpvaW4oXCJcIik7XG4gIH1cblxuICAvKiogT3V0cHV0cyBjb21tYW5kIG5hbWUsIHZlcnNpb24gYW5kIG1ldGEgZGF0YS4gKi9cbiAgcHVibGljIHNob3dMb25nVmVyc2lvbigpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldExvbmdWZXJzaW9uKCkpO1xuICB9XG5cbiAgLyoqIE91dHB1dCBnZW5lcmF0ZWQgaGVscCB3aXRob3V0IGV4aXRpbmcuICovXG4gIHB1YmxpYyBzaG93SGVscChvcHRpb25zPzogSGVscE9wdGlvbnMpOiB2b2lkIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLmdldEhlbHAob3B0aW9ucykpO1xuICB9XG5cbiAgLyoqIEdldCBnZW5lcmF0ZWQgaGVscC4gKi9cbiAgcHVibGljIGdldEhlbHAob3B0aW9ucz86IEhlbHBPcHRpb25zKTogc3RyaW5nIHtcbiAgICB0aGlzLnJlZ2lzdGVyRGVmYXVsdHMoKTtcbiAgICByZXR1cm4gdGhpcy5nZXRIZWxwSGFuZGxlcigpLmNhbGwodGhpcywgdGhpcywgb3B0aW9ucyA/PyB7fSk7XG4gIH1cblxuICAvKiogR2V0IGhlbHAgaGFuZGxlciBtZXRob2QuICovXG4gIHByaXZhdGUgZ2V0SGVscEhhbmRsZXIoKTogSUhlbHBIYW5kbGVyIHtcbiAgICByZXR1cm4gdGhpcy5faGVscCA/PyB0aGlzLl9wYXJlbnQ/LmdldEhlbHBIYW5kbGVyKCkgYXMgSUhlbHBIYW5kbGVyO1xuICB9XG5cbiAgcHJpdmF0ZSBleGl0KGNvZGUgPSAwKSB7XG4gICAgaWYgKHRoaXMuc2hvdWxkRXhpdCgpKSB7XG4gICAgICBEZW5vLmV4aXQoY29kZSk7XG4gICAgfVxuICB9XG5cbiAgLyoqIENoZWNrIGlmIG5ldyB2ZXJzaW9uIGlzIGF2YWlsYWJsZSBhbmQgYWRkIGhpbnQgdG8gdmVyc2lvbi4gKi9cbiAgcHVibGljIGFzeW5jIGNoZWNrVmVyc2lvbigpOiBQcm9taXNlPHZvaWQ+IHtcbiAgICBjb25zdCBtYWluQ29tbWFuZCA9IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICBjb25zdCB1cGdyYWRlQ29tbWFuZCA9IG1haW5Db21tYW5kLmdldENvbW1hbmQoXCJ1cGdyYWRlXCIpO1xuICAgIGlmIChpc1VwZ3JhZGVDb21tYW5kKHVwZ3JhZGVDb21tYW5kKSkge1xuICAgICAgY29uc3QgbGF0ZXN0VmVyc2lvbiA9IGF3YWl0IHVwZ3JhZGVDb21tYW5kLmdldExhdGVzdFZlcnNpb24oKTtcbiAgICAgIGNvbnN0IGN1cnJlbnRWZXJzaW9uID0gbWFpbkNvbW1hbmQuZ2V0VmVyc2lvbigpO1xuICAgICAgaWYgKGN1cnJlbnRWZXJzaW9uICE9PSBsYXRlc3RWZXJzaW9uKSB7XG4gICAgICAgIG1haW5Db21tYW5kLnZlcnNpb24oXG4gICAgICAgICAgYCR7Y3VycmVudFZlcnNpb259ICAke1xuICAgICAgICAgICAgYm9sZChcbiAgICAgICAgICAgICAgeWVsbG93KFxuICAgICAgICAgICAgICAgIGAoTmV3IHZlcnNpb24gYXZhaWxhYmxlOiAke2xhdGVzdFZlcnNpb259LiBSdW4gJyR7bWFpbkNvbW1hbmQuZ2V0TmFtZSgpfSB1cGdyYWRlJyB0byB1cGdyYWRlIHRvIHRoZSBsYXRlc3QgdmVyc2lvbiEpYCxcbiAgICAgICAgICAgICAgKSxcbiAgICAgICAgICAgIClcbiAgICAgICAgICB9YCxcbiAgICAgICAgKTtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICAvKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKipcbiAgICoqKiogT3B0aW9ucyBHRVRURVIgKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqXG4gICAqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKi9cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIG9wdGlvbnMgb3Igbm90LlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgaGFzT3B0aW9ucyhoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0T3B0aW9ucyhoaWRkZW4pLmxlbmd0aCA+IDA7XG4gIH1cblxuICAvKipcbiAgICogR2V0IG9wdGlvbnMuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gb3B0aW9ucy5cbiAgICovXG4gIHB1YmxpYyBnZXRPcHRpb25zKGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uW10ge1xuICAgIHJldHVybiB0aGlzLmdldEdsb2JhbE9wdGlvbnMoaGlkZGVuKS5jb25jYXQodGhpcy5nZXRCYXNlT3B0aW9ucyhoaWRkZW4pKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBvcHRpb25zLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZU9wdGlvbnMoaGlkZGVuPzogYm9vbGVhbik6IElPcHRpb25bXSB7XG4gICAgaWYgKCF0aGlzLm9wdGlvbnMubGVuZ3RoKSB7XG4gICAgICByZXR1cm4gW107XG4gICAgfVxuXG4gICAgcmV0dXJuIGhpZGRlblxuICAgICAgPyB0aGlzLm9wdGlvbnMuc2xpY2UoMClcbiAgICAgIDogdGhpcy5vcHRpb25zLmZpbHRlcigob3B0KSA9PiAhb3B0LmhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBvcHRpb25zLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsT3B0aW9ucyhoaWRkZW4/OiBib29sZWFuKTogSU9wdGlvbltdIHtcbiAgICBjb25zdCBnZXRPcHRpb25zID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQsXG4gICAgICBvcHRpb25zOiBJT3B0aW9uW10gPSBbXSxcbiAgICAgIG5hbWVzOiBzdHJpbmdbXSA9IFtdLFxuICAgICk6IElPcHRpb25bXSA9PiB7XG4gICAgICBpZiAoY21kKSB7XG4gICAgICAgIGlmIChjbWQub3B0aW9ucy5sZW5ndGgpIHtcbiAgICAgICAgICBjbWQub3B0aW9ucy5mb3JFYWNoKChvcHRpb246IElPcHRpb24pID0+IHtcbiAgICAgICAgICAgIGlmIChcbiAgICAgICAgICAgICAgb3B0aW9uLmdsb2JhbCAmJlxuICAgICAgICAgICAgICAhdGhpcy5vcHRpb25zLmZpbmQoKG9wdCkgPT4gb3B0Lm5hbWUgPT09IG9wdGlvbi5uYW1lKSAmJlxuICAgICAgICAgICAgICBuYW1lcy5pbmRleE9mKG9wdGlvbi5uYW1lKSA9PT0gLTEgJiZcbiAgICAgICAgICAgICAgKGhpZGRlbiB8fCAhb3B0aW9uLmhpZGRlbilcbiAgICAgICAgICAgICkge1xuICAgICAgICAgICAgICBuYW1lcy5wdXNoKG9wdGlvbi5uYW1lKTtcbiAgICAgICAgICAgICAgb3B0aW9ucy5wdXNoKG9wdGlvbik7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0T3B0aW9ucyhjbWQuX3BhcmVudCwgb3B0aW9ucywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gb3B0aW9ucztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldE9wdGlvbnModGhpcy5fcGFyZW50KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgYW4gb3B0aW9uIHdpdGggZ2l2ZW4gbmFtZSBvciBub3QuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgaGFzT3B0aW9uKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0T3B0aW9uKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IG9wdGlvbiBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBvcHRpb24uIE11c3QgYmUgaW4gcGFyYW0tY2FzZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGdldE9wdGlvbihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlT3B0aW9uKG5hbWUsIGhpZGRlbikgPz9cbiAgICAgIHRoaXMuZ2V0R2xvYmFsT3B0aW9uKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2Ugb3B0aW9uIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZU9wdGlvbihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJT3B0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBvcHRpb24gPSB0aGlzLm9wdGlvbnMuZmluZCgob3B0aW9uKSA9PiBvcHRpb24ubmFtZSA9PT0gbmFtZSk7XG5cbiAgICByZXR1cm4gb3B0aW9uICYmIChoaWRkZW4gfHwgIW9wdGlvbi5oaWRkZW4pID8gb3B0aW9uIDogdW5kZWZpbmVkO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgb3B0aW9uIGZyb20gcGFyZW50IGNvbW1hbmRzIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIG9wdGlvbi4gTXVzdCBiZSBpbiBwYXJhbS1jYXNlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsT3B0aW9uKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IElPcHRpb24gfCB1bmRlZmluZWQge1xuICAgIGlmICghdGhpcy5fcGFyZW50KSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgY29uc3Qgb3B0aW9uOiBJT3B0aW9uIHwgdW5kZWZpbmVkID0gdGhpcy5fcGFyZW50LmdldEJhc2VPcHRpb24oXG4gICAgICBuYW1lLFxuICAgICAgaGlkZGVuLFxuICAgICk7XG5cbiAgICBpZiAoIW9wdGlvbiB8fCAhb3B0aW9uLmdsb2JhbCkge1xuICAgICAgcmV0dXJuIHRoaXMuX3BhcmVudC5nZXRHbG9iYWxPcHRpb24obmFtZSwgaGlkZGVuKTtcbiAgICB9XG5cbiAgICByZXR1cm4gb3B0aW9uO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbW92ZSBvcHRpb24gYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgb3B0aW9uLiBNdXN0IGJlIGluIHBhcmFtLWNhc2UuXG4gICAqL1xuICBwdWJsaWMgcmVtb3ZlT3B0aW9uKG5hbWU6IHN0cmluZyk6IElPcHRpb24gfCB1bmRlZmluZWQge1xuICAgIGNvbnN0IGluZGV4ID0gdGhpcy5vcHRpb25zLmZpbmRJbmRleCgob3B0aW9uKSA9PiBvcHRpb24ubmFtZSA9PT0gbmFtZSk7XG5cbiAgICBpZiAoaW5kZXggPT09IC0xKSB7XG4gICAgICByZXR1cm47XG4gICAgfVxuXG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5zcGxpY2UoaW5kZXgsIDEpWzBdO1xuICB9XG5cbiAgLyoqXG4gICAqIENoZWNrcyB3aGV0aGVyIHRoZSBjb21tYW5kIGhhcyBzdWItY29tbWFuZHMgb3Igbm90LlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGhhc0NvbW1hbmRzKGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gdGhpcy5nZXRDb21tYW5kcyhoaWRkZW4pLmxlbmd0aCA+IDA7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGNvbW1hbmRzLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGdldENvbW1hbmRzKGhpZGRlbj86IGJvb2xlYW4pOiBBcnJheTxDb21tYW5kPGFueT4+IHtcbiAgICByZXR1cm4gdGhpcy5nZXRHbG9iYWxDb21tYW5kcyhoaWRkZW4pLmNvbmNhdCh0aGlzLmdldEJhc2VDb21tYW5kcyhoaWRkZW4pKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBjb21tYW5kcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRCYXNlQ29tbWFuZHMoaGlkZGVuPzogYm9vbGVhbik6IEFycmF5PENvbW1hbmQ8YW55Pj4ge1xuICAgIGNvbnN0IGNvbW1hbmRzID0gQXJyYXkuZnJvbSh0aGlzLmNvbW1hbmRzLnZhbHVlcygpKTtcbiAgICByZXR1cm4gaGlkZGVuID8gY29tbWFuZHMgOiBjb21tYW5kcy5maWx0ZXIoKGNtZCkgPT4gIWNtZC5pc0hpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBjb21tYW5kcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxDb21tYW5kcyhoaWRkZW4/OiBib29sZWFuKTogQXJyYXk8Q29tbWFuZDxhbnk+PiB7XG4gICAgY29uc3QgZ2V0Q29tbWFuZHMgPSAoXG4gICAgICBjbWQ6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCxcbiAgICAgIGNvbW1hbmRzOiBBcnJheTxDb21tYW5kPGFueT4+ID0gW10sXG4gICAgICBuYW1lczogc3RyaW5nW10gPSBbXSxcbiAgICApOiBBcnJheTxDb21tYW5kPGFueT4+ID0+IHtcbiAgICAgIGlmIChjbWQpIHtcbiAgICAgICAgaWYgKGNtZC5jb21tYW5kcy5zaXplKSB7XG4gICAgICAgICAgY21kLmNvbW1hbmRzLmZvckVhY2goKGNtZDogQ29tbWFuZDxhbnk+KSA9PiB7XG4gICAgICAgICAgICBpZiAoXG4gICAgICAgICAgICAgIGNtZC5pc0dsb2JhbCAmJlxuICAgICAgICAgICAgICB0aGlzICE9PSBjbWQgJiZcbiAgICAgICAgICAgICAgIXRoaXMuY29tbWFuZHMuaGFzKGNtZC5fbmFtZSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZihjbWQuX25hbWUpID09PSAtMSAmJlxuICAgICAgICAgICAgICAoaGlkZGVuIHx8ICFjbWQuaXNIaWRkZW4pXG4gICAgICAgICAgICApIHtcbiAgICAgICAgICAgICAgbmFtZXMucHVzaChjbWQuX25hbWUpO1xuICAgICAgICAgICAgICBjb21tYW5kcy5wdXNoKGNtZCk7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0Q29tbWFuZHMoY21kLl9wYXJlbnQsIGNvbW1hbmRzLCBuYW1lcyk7XG4gICAgICB9XG5cbiAgICAgIHJldHVybiBjb21tYW5kcztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldENvbW1hbmRzKHRoaXMuX3BhcmVudCk7XG4gIH1cblxuICAvKipcbiAgICogQ2hlY2tzIHdoZXRoZXIgYSBjaGlsZCBjb21tYW5kIGV4aXN0cyBieSBnaXZlbiBuYW1lIG9yIGFsaWFzLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9yIGFsaWFzIG9mIHRoZSBjb21tYW5kLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGNvbW1hbmRzLlxuICAgKi9cbiAgcHVibGljIGhhc0NvbW1hbmQobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuICEhdGhpcy5nZXRDb21tYW5kKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGNvbW1hbmQgYnkgbmFtZSBvciBhbGlhcy5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvciBhbGlhcyBvZiB0aGUgY29tbWFuZC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBjb21tYW5kcy5cbiAgICovXG4gIHB1YmxpYyBnZXRDb21tYW5kPEMgZXh0ZW5kcyBDb21tYW5kPGFueT4+KFxuICAgIG5hbWU6IHN0cmluZyxcbiAgICBoaWRkZW4/OiBib29sZWFuLFxuICApOiBDIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlQ29tbWFuZChuYW1lLCBoaWRkZW4pID8/XG4gICAgICB0aGlzLmdldEdsb2JhbENvbW1hbmQobmFtZSwgaGlkZGVuKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgYmFzZSBjb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUNvbW1hbmQ8QyBleHRlbmRzIENvbW1hbmQ8YW55Pj4oXG4gICAgbmFtZTogc3RyaW5nLFxuICAgIGhpZGRlbj86IGJvb2xlYW4sXG4gICk6IEMgfCB1bmRlZmluZWQge1xuICAgIGZvciAoY29uc3QgY21kIG9mIHRoaXMuY29tbWFuZHMudmFsdWVzKCkpIHtcbiAgICAgIGlmIChjbWQuX25hbWUgPT09IG5hbWUgfHwgY21kLmFsaWFzZXMuaW5jbHVkZXMobmFtZSkpIHtcbiAgICAgICAgcmV0dXJuIChjbWQgJiYgKGhpZGRlbiB8fCAhY21kLmlzSGlkZGVuKSA/IGNtZCA6IHVuZGVmaW5lZCkgYXNcbiAgICAgICAgICB8IENcbiAgICAgICAgICB8IHVuZGVmaW5lZDtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBjb21tYW5kIGJ5IG5hbWUgb3IgYWxpYXMuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb3IgYWxpYXMgb2YgdGhlIGNvbW1hbmQuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gY29tbWFuZHMuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsQ29tbWFuZDxDIGV4dGVuZHMgQ29tbWFuZDxhbnk+PihcbiAgICBuYW1lOiBzdHJpbmcsXG4gICAgaGlkZGVuPzogYm9vbGVhbixcbiAgKTogQyB8IHVuZGVmaW5lZCB7XG4gICAgaWYgKCF0aGlzLl9wYXJlbnQpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG5cbiAgICBjb25zdCBjbWQgPSB0aGlzLl9wYXJlbnQuZ2V0QmFzZUNvbW1hbmQobmFtZSwgaGlkZGVuKTtcblxuICAgIGlmICghY21kPy5pc0dsb2JhbCkge1xuICAgICAgcmV0dXJuIHRoaXMuX3BhcmVudC5nZXRHbG9iYWxDb21tYW5kKG5hbWUsIGhpZGRlbik7XG4gICAgfVxuXG4gICAgcmV0dXJuIGNtZCBhcyBDO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbW92ZSBzdWItY29tbWFuZCBieSBuYW1lIG9yIGFsaWFzLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9yIGFsaWFzIG9mIHRoZSBjb21tYW5kLlxuICAgKi9cbiAgcHVibGljIHJlbW92ZUNvbW1hbmQobmFtZTogc3RyaW5nKTogQ29tbWFuZDxhbnk+IHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBjb21tYW5kID0gdGhpcy5nZXRCYXNlQ29tbWFuZChuYW1lLCB0cnVlKTtcblxuICAgIGlmIChjb21tYW5kKSB7XG4gICAgICB0aGlzLmNvbW1hbmRzLmRlbGV0ZShjb21tYW5kLl9uYW1lKTtcbiAgICB9XG5cbiAgICByZXR1cm4gY29tbWFuZDtcbiAgfVxuXG4gIC8qKiBHZXQgdHlwZXMuICovXG4gIHB1YmxpYyBnZXRUeXBlcygpOiBJVHlwZVtdIHtcbiAgICByZXR1cm4gdGhpcy5nZXRHbG9iYWxUeXBlcygpLmNvbmNhdCh0aGlzLmdldEJhc2VUeXBlcygpKTtcbiAgfVxuXG4gIC8qKiBHZXQgYmFzZSB0eXBlcy4gKi9cbiAgcHVibGljIGdldEJhc2VUeXBlcygpOiBJVHlwZVtdIHtcbiAgICByZXR1cm4gQXJyYXkuZnJvbSh0aGlzLnR5cGVzLnZhbHVlcygpKTtcbiAgfVxuXG4gIC8qKiBHZXQgZ2xvYmFsIHR5cGVzLiAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsVHlwZXMoKTogSVR5cGVbXSB7XG4gICAgY29uc3QgZ2V0VHlwZXMgPSAoXG4gICAgICBjbWQ6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCxcbiAgICAgIHR5cGVzOiBJVHlwZVtdID0gW10sXG4gICAgICBuYW1lczogc3RyaW5nW10gPSBbXSxcbiAgICApOiBJVHlwZVtdID0+IHtcbiAgICAgIGlmIChjbWQpIHtcbiAgICAgICAgaWYgKGNtZC50eXBlcy5zaXplKSB7XG4gICAgICAgICAgY21kLnR5cGVzLmZvckVhY2goKHR5cGU6IElUeXBlKSA9PiB7XG4gICAgICAgICAgICBpZiAoXG4gICAgICAgICAgICAgIHR5cGUuZ2xvYmFsICYmXG4gICAgICAgICAgICAgICF0aGlzLnR5cGVzLmhhcyh0eXBlLm5hbWUpICYmXG4gICAgICAgICAgICAgIG5hbWVzLmluZGV4T2YodHlwZS5uYW1lKSA9PT0gLTFcbiAgICAgICAgICAgICkge1xuICAgICAgICAgICAgICBuYW1lcy5wdXNoKHR5cGUubmFtZSk7XG4gICAgICAgICAgICAgIHR5cGVzLnB1c2godHlwZSk7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0VHlwZXMoY21kLl9wYXJlbnQsIHR5cGVzLCBuYW1lcyk7XG4gICAgICB9XG5cbiAgICAgIHJldHVybiB0eXBlcztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldFR5cGVzKHRoaXMuX3BhcmVudCk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IHR5cGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgdHlwZS5cbiAgICovXG4gIHB1YmxpYyBnZXRUeXBlKG5hbWU6IHN0cmluZyk6IElUeXBlIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlVHlwZShuYW1lKSA/PyB0aGlzLmdldEdsb2JhbFR5cGUobmFtZSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgdHlwZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSB0eXBlLlxuICAgKi9cbiAgcHVibGljIGdldEJhc2VUeXBlKG5hbWU6IHN0cmluZyk6IElUeXBlIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy50eXBlcy5nZXQobmFtZSk7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCB0eXBlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIHR5cGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0R2xvYmFsVHlwZShuYW1lOiBzdHJpbmcpOiBJVHlwZSB8IHVuZGVmaW5lZCB7XG4gICAgaWYgKCF0aGlzLl9wYXJlbnQpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG5cbiAgICBjb25zdCBjbWQ6IElUeXBlIHwgdW5kZWZpbmVkID0gdGhpcy5fcGFyZW50LmdldEJhc2VUeXBlKG5hbWUpO1xuXG4gICAgaWYgKCFjbWQ/Lmdsb2JhbCkge1xuICAgICAgcmV0dXJuIHRoaXMuX3BhcmVudC5nZXRHbG9iYWxUeXBlKG5hbWUpO1xuICAgIH1cblxuICAgIHJldHVybiBjbWQ7XG4gIH1cblxuICAvKiogR2V0IGNvbXBsZXRpb25zLiAqL1xuICBwdWJsaWMgZ2V0Q29tcGxldGlvbnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsQ29tcGxldGlvbnMoKS5jb25jYXQodGhpcy5nZXRCYXNlQ29tcGxldGlvbnMoKSk7XG4gIH1cblxuICAvKiogR2V0IGJhc2UgY29tcGxldGlvbnMuICovXG4gIHB1YmxpYyBnZXRCYXNlQ29tcGxldGlvbnMoKTogSUNvbXBsZXRpb25bXSB7XG4gICAgcmV0dXJuIEFycmF5LmZyb20odGhpcy5jb21wbGV0aW9ucy52YWx1ZXMoKSk7XG4gIH1cblxuICAvKiogR2V0IGdsb2JhbCBjb21wbGV0aW9ucy4gKi9cbiAgcHVibGljIGdldEdsb2JhbENvbXBsZXRpb25zKCk6IElDb21wbGV0aW9uW10ge1xuICAgIGNvbnN0IGdldENvbXBsZXRpb25zID0gKFxuICAgICAgY21kOiBDb21tYW5kPGFueT4gfCB1bmRlZmluZWQsXG4gICAgICBjb21wbGV0aW9uczogSUNvbXBsZXRpb25bXSA9IFtdLFxuICAgICAgbmFtZXM6IHN0cmluZ1tdID0gW10sXG4gICAgKTogSUNvbXBsZXRpb25bXSA9PiB7XG4gICAgICBpZiAoY21kKSB7XG4gICAgICAgIGlmIChjbWQuY29tcGxldGlvbnMuc2l6ZSkge1xuICAgICAgICAgIGNtZC5jb21wbGV0aW9ucy5mb3JFYWNoKChjb21wbGV0aW9uOiBJQ29tcGxldGlvbikgPT4ge1xuICAgICAgICAgICAgaWYgKFxuICAgICAgICAgICAgICBjb21wbGV0aW9uLmdsb2JhbCAmJlxuICAgICAgICAgICAgICAhdGhpcy5jb21wbGV0aW9ucy5oYXMoY29tcGxldGlvbi5uYW1lKSAmJlxuICAgICAgICAgICAgICBuYW1lcy5pbmRleE9mKGNvbXBsZXRpb24ubmFtZSkgPT09IC0xXG4gICAgICAgICAgICApIHtcbiAgICAgICAgICAgICAgbmFtZXMucHVzaChjb21wbGV0aW9uLm5hbWUpO1xuICAgICAgICAgICAgICBjb21wbGV0aW9ucy5wdXNoKGNvbXBsZXRpb24pO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG5cbiAgICAgICAgcmV0dXJuIGdldENvbXBsZXRpb25zKGNtZC5fcGFyZW50LCBjb21wbGV0aW9ucywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gY29tcGxldGlvbnM7XG4gICAgfTtcblxuICAgIHJldHVybiBnZXRDb21wbGV0aW9ucyh0aGlzLl9wYXJlbnQpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBjb21wbGV0aW9uIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGNvbXBsZXRpb24uXG4gICAqL1xuICBwdWJsaWMgZ2V0Q29tcGxldGlvbihuYW1lOiBzdHJpbmcpOiBJQ29tcGxldGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0QmFzZUNvbXBsZXRpb24obmFtZSkgPz8gdGhpcy5nZXRHbG9iYWxDb21wbGV0aW9uKG5hbWUpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIGNvbXBsZXRpb24gYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgY29tcGxldGlvbi5cbiAgICovXG4gIHB1YmxpYyBnZXRCYXNlQ29tcGxldGlvbihuYW1lOiBzdHJpbmcpOiBJQ29tcGxldGlvbiB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuY29tcGxldGlvbnMuZ2V0KG5hbWUpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBnbG9iYWwgY29tcGxldGlvbnMgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgY29tcGxldGlvbi5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxDb21wbGV0aW9uKG5hbWU6IHN0cmluZyk6IElDb21wbGV0aW9uIHwgdW5kZWZpbmVkIHtcbiAgICBpZiAoIXRoaXMuX3BhcmVudCkge1xuICAgICAgcmV0dXJuO1xuICAgIH1cblxuICAgIGNvbnN0IGNvbXBsZXRpb246IElDb21wbGV0aW9uIHwgdW5kZWZpbmVkID0gdGhpcy5fcGFyZW50LmdldEJhc2VDb21wbGV0aW9uKFxuICAgICAgbmFtZSxcbiAgICApO1xuXG4gICAgaWYgKCFjb21wbGV0aW9uPy5nbG9iYWwpIHtcbiAgICAgIHJldHVybiB0aGlzLl9wYXJlbnQuZ2V0R2xvYmFsQ29tcGxldGlvbihuYW1lKTtcbiAgICB9XG5cbiAgICByZXR1cm4gY29tcGxldGlvbjtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgZW52aXJvbm1lbnQgdmFyaWFibGVzIG9yIG5vdC5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBoYXNFbnZWYXJzKGhpZGRlbj86IGJvb2xlYW4pOiBib29sZWFuIHtcbiAgICByZXR1cm4gdGhpcy5nZXRFbnZWYXJzKGhpZGRlbikubGVuZ3RoID4gMDtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZW52aXJvbm1lbnQgdmFyaWFibGVzLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEVudlZhcnMoaGlkZGVuPzogYm9vbGVhbik6IElFbnZWYXJbXSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0R2xvYmFsRW52VmFycyhoaWRkZW4pLmNvbmNhdCh0aGlzLmdldEJhc2VFbnZWYXJzKGhpZGRlbikpO1xuICB9XG5cbiAgLyoqXG4gICAqIEdldCBiYXNlIGVudmlyb25tZW50IHZhcmlhYmxlcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRCYXNlRW52VmFycyhoaWRkZW4/OiBib29sZWFuKTogSUVudlZhcltdIHtcbiAgICBpZiAoIXRoaXMuZW52VmFycy5sZW5ndGgpIHtcbiAgICAgIHJldHVybiBbXTtcbiAgICB9XG5cbiAgICByZXR1cm4gaGlkZGVuXG4gICAgICA/IHRoaXMuZW52VmFycy5zbGljZSgwKVxuICAgICAgOiB0aGlzLmVudlZhcnMuZmlsdGVyKChlbnYpID0+ICFlbnYuaGlkZGVuKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgZ2xvYmFsIGVudmlyb25tZW50IHZhcmlhYmxlcy5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxFbnZWYXJzKGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyW10ge1xuICAgIGNvbnN0IGdldEVudlZhcnMgPSAoXG4gICAgICBjbWQ6IENvbW1hbmQ8YW55PiB8IHVuZGVmaW5lZCxcbiAgICAgIGVudlZhcnM6IElFbnZWYXJbXSA9IFtdLFxuICAgICAgbmFtZXM6IHN0cmluZ1tdID0gW10sXG4gICAgKTogSUVudlZhcltdID0+IHtcbiAgICAgIGlmIChjbWQpIHtcbiAgICAgICAgaWYgKGNtZC5lbnZWYXJzLmxlbmd0aCkge1xuICAgICAgICAgIGNtZC5lbnZWYXJzLmZvckVhY2goKGVudlZhcjogSUVudlZhcikgPT4ge1xuICAgICAgICAgICAgaWYgKFxuICAgICAgICAgICAgICBlbnZWYXIuZ2xvYmFsICYmXG4gICAgICAgICAgICAgICF0aGlzLmVudlZhcnMuZmluZCgoZW52KSA9PiBlbnYubmFtZXNbMF0gPT09IGVudlZhci5uYW1lc1swXSkgJiZcbiAgICAgICAgICAgICAgbmFtZXMuaW5kZXhPZihlbnZWYXIubmFtZXNbMF0pID09PSAtMSAmJlxuICAgICAgICAgICAgICAoaGlkZGVuIHx8ICFlbnZWYXIuaGlkZGVuKVxuICAgICAgICAgICAgKSB7XG4gICAgICAgICAgICAgIG5hbWVzLnB1c2goZW52VmFyLm5hbWVzWzBdKTtcbiAgICAgICAgICAgICAgZW52VmFycy5wdXNoKGVudlZhcik7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cblxuICAgICAgICByZXR1cm4gZ2V0RW52VmFycyhjbWQuX3BhcmVudCwgZW52VmFycywgbmFtZXMpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gZW52VmFycztcbiAgICB9O1xuXG4gICAgcmV0dXJuIGdldEVudlZhcnModGhpcy5fcGFyZW50KTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgYW4gZW52aXJvbm1lbnQgdmFyaWFibGUgd2l0aCBnaXZlbiBuYW1lIG9yIG5vdC5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgaGFzRW52VmFyKG5hbWU6IHN0cmluZywgaGlkZGVuPzogYm9vbGVhbik6IGJvb2xlYW4ge1xuICAgIHJldHVybiAhIXRoaXMuZ2V0RW52VmFyKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGVudmlyb25tZW50IHZhcmlhYmxlIGJ5IG5hbWUuXG4gICAqIEBwYXJhbSBuYW1lIE5hbWUgb2YgdGhlIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKiBAcGFyYW0gaGlkZGVuIEluY2x1ZGUgaGlkZGVuIGVudmlyb25tZW50IHZhcmlhYmxlLlxuICAgKi9cbiAgcHVibGljIGdldEVudlZhcihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyIHwgdW5kZWZpbmVkIHtcbiAgICByZXR1cm4gdGhpcy5nZXRCYXNlRW52VmFyKG5hbWUsIGhpZGRlbikgPz9cbiAgICAgIHRoaXMuZ2V0R2xvYmFsRW52VmFyKG5hbWUsIGhpZGRlbik7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGJhc2UgZW52aXJvbm1lbnQgdmFyaWFibGUgYnkgbmFtZS5cbiAgICogQHBhcmFtIG5hbWUgTmFtZSBvZiB0aGUgZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqIEBwYXJhbSBoaWRkZW4gSW5jbHVkZSBoaWRkZW4gZW52aXJvbm1lbnQgdmFyaWFibGUuXG4gICAqL1xuICBwdWJsaWMgZ2V0QmFzZUVudlZhcihuYW1lOiBzdHJpbmcsIGhpZGRlbj86IGJvb2xlYW4pOiBJRW52VmFyIHwgdW5kZWZpbmVkIHtcbiAgICBjb25zdCBlbnZWYXI6IElFbnZWYXIgfCB1bmRlZmluZWQgPSB0aGlzLmVudlZhcnMuZmluZCgoZW52KSA9PlxuICAgICAgZW52Lm5hbWVzLmluZGV4T2YobmFtZSkgIT09IC0xXG4gICAgKTtcblxuICAgIHJldHVybiBlbnZWYXIgJiYgKGhpZGRlbiB8fCAhZW52VmFyLmhpZGRlbikgPyBlbnZWYXIgOiB1bmRlZmluZWQ7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGdsb2JhbCBlbnZpcm9ubWVudCB2YXJpYWJsZSBieSBuYW1lLlxuICAgKiBAcGFyYW0gbmFtZSBOYW1lIG9mIHRoZSBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICogQHBhcmFtIGhpZGRlbiBJbmNsdWRlIGhpZGRlbiBlbnZpcm9ubWVudCB2YXJpYWJsZS5cbiAgICovXG4gIHB1YmxpYyBnZXRHbG9iYWxFbnZWYXIobmFtZTogc3RyaW5nLCBoaWRkZW4/OiBib29sZWFuKTogSUVudlZhciB8IHVuZGVmaW5lZCB7XG4gICAgaWYgKCF0aGlzLl9wYXJlbnQpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG5cbiAgICBjb25zdCBlbnZWYXI6IElFbnZWYXIgfCB1bmRlZmluZWQgPSB0aGlzLl9wYXJlbnQuZ2V0QmFzZUVudlZhcihcbiAgICAgIG5hbWUsXG4gICAgICBoaWRkZW4sXG4gICAgKTtcblxuICAgIGlmICghZW52VmFyPy5nbG9iYWwpIHtcbiAgICAgIHJldHVybiB0aGlzLl9wYXJlbnQuZ2V0R2xvYmFsRW52VmFyKG5hbWUsIGhpZGRlbik7XG4gICAgfVxuXG4gICAgcmV0dXJuIGVudlZhcjtcbiAgfVxuXG4gIC8qKiBDaGVja3Mgd2hldGhlciB0aGUgY29tbWFuZCBoYXMgZXhhbXBsZXMgb3Igbm90LiAqL1xuICBwdWJsaWMgaGFzRXhhbXBsZXMoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZXhhbXBsZXMubGVuZ3RoID4gMDtcbiAgfVxuXG4gIC8qKiBHZXQgYWxsIGV4YW1wbGVzLiAqL1xuICBwdWJsaWMgZ2V0RXhhbXBsZXMoKTogSUV4YW1wbGVbXSB7XG4gICAgcmV0dXJuIHRoaXMuZXhhbXBsZXM7XG4gIH1cblxuICAvKiogQ2hlY2tzIHdoZXRoZXIgdGhlIGNvbW1hbmQgaGFzIGFuIGV4YW1wbGUgd2l0aCBnaXZlbiBuYW1lIG9yIG5vdC4gKi9cbiAgcHVibGljIGhhc0V4YW1wbGUobmFtZTogc3RyaW5nKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuICEhdGhpcy5nZXRFeGFtcGxlKG5hbWUpO1xuICB9XG5cbiAgLyoqIEdldCBleGFtcGxlIHdpdGggZ2l2ZW4gbmFtZS4gKi9cbiAgcHVibGljIGdldEV4YW1wbGUobmFtZTogc3RyaW5nKTogSUV4YW1wbGUgfCB1bmRlZmluZWQge1xuICAgIHJldHVybiB0aGlzLmV4YW1wbGVzLmZpbmQoKGV4YW1wbGUpID0+IGV4YW1wbGUubmFtZSA9PT0gbmFtZSk7XG4gIH1cbn1cblxuZnVuY3Rpb24gaXNVcGdyYWRlQ29tbWFuZChjb21tYW5kOiB1bmtub3duKTogY29tbWFuZCBpcyBVcGdyYWRlQ29tbWFuZEltcGwge1xuICByZXR1cm4gY29tbWFuZCBpbnN0YW5jZW9mIENvbW1hbmQgJiYgXCJnZXRMYXRlc3RWZXJzaW9uXCIgaW4gY29tbWFuZDtcbn1cblxuaW50ZXJmYWNlIFVwZ3JhZGVDb21tYW5kSW1wbCB7XG4gIGdldExhdGVzdFZlcnNpb24oKTogUHJvbWlzZTxzdHJpbmc+O1xufVxuXG5pbnRlcmZhY2UgSURlZmF1bHRPcHRpb24ge1xuICBmbGFnczogc3RyaW5nO1xuICBkZXNjPzogc3RyaW5nO1xuICBvcHRzPzogSUNvbW1hbmRPcHRpb247XG59XG5cbnR5cGUgVHJpbUxlZnQ8VCBleHRlbmRzIHN0cmluZywgViBleHRlbmRzIHN0cmluZyB8IHVuZGVmaW5lZD4gPSBUIGV4dGVuZHNcbiAgYCR7Vn0ke2luZmVyIFV9YCA/IFVcbiAgOiBUO1xuXG50eXBlIFRyaW1SaWdodDxUIGV4dGVuZHMgc3RyaW5nLCBWIGV4dGVuZHMgc3RyaW5nPiA9IFQgZXh0ZW5kcyBgJHtpbmZlciBVfSR7Vn1gXG4gID8gVVxuICA6IFQ7XG5cbnR5cGUgTG93ZXI8ViBleHRlbmRzIHN0cmluZz4gPSBWIGV4dGVuZHMgVXBwZXJjYXNlPFY+ID8gTG93ZXJjYXNlPFY+XG4gIDogVW5jYXBpdGFsaXplPFY+O1xuXG50eXBlIENhbWVsQ2FzZTxUIGV4dGVuZHMgc3RyaW5nPiA9IFQgZXh0ZW5kcyBgJHtpbmZlciBWfV8ke2luZmVyIFJlc3R9YFxuICA/IGAke0xvd2VyPFY+fSR7Q2FwaXRhbGl6ZTxDYW1lbENhc2U8UmVzdD4+fWBcbiAgOiBUIGV4dGVuZHMgYCR7aW5mZXIgVn0tJHtpbmZlciBSZXN0fWBcbiAgICA/IGAke0xvd2VyPFY+fSR7Q2FwaXRhbGl6ZTxDYW1lbENhc2U8UmVzdD4+fWBcbiAgOiBMb3dlcjxUPjtcblxudHlwZSBPbmVPZjxULCBWPiA9IFQgZXh0ZW5kcyB2b2lkID8gViA6IFQ7XG5cbnR5cGUgTWVyZ2U8TCwgUj4gPSBMIGV4dGVuZHMgdm9pZCA/IFJcbiAgOiBSIGV4dGVuZHMgdm9pZCA/IExcbiAgOiBMICYgUjtcblxuLy8gdHlwZSBNZXJnZTxMLCBSPiA9IEwgZXh0ZW5kcyB2b2lkID8gUlxuLy8gICA6IFIgZXh0ZW5kcyB2b2lkID8gTFxuLy8gICA6IE9taXQ8TCwga2V5b2YgUj4gJiBSO1xuXG50eXBlIE1lcmdlUmVjdXJzaXZlPEwsIFI+ID0gTCBleHRlbmRzIHZvaWQgPyBSXG4gIDogUiBleHRlbmRzIHZvaWQgPyBMXG4gIDogTCAmIFI7XG5cbnR5cGUgT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8VCBleHRlbmRzIHN0cmluZz4gPSBgWyR7VH1dYCB8IGA8JHtUfT5gO1xudHlwZSBSZXN0VmFsdWUgPSBgLi4uJHtzdHJpbmd9YCB8IGAke3N0cmluZ30uLi5gO1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIGxpc3QgdHlwZSBhbmQgY29tcGxldGlvbnMuXG4gKlxuICogLSBgWy4uLm5hbWU6dHlwZVtdOmNvbXBsZXRpb25dYFxuICogLSBgPC4uLm5hbWU6dHlwZVtdOmNvbXBsZXRpb24+YFxuICogLSBgW25hbWUuLi46dHlwZVtdOmNvbXBsZXRpb25dYFxuICogLSBgPG5hbWUuLi46dHlwZVtdOmNvbXBsZXRpb24+YFxuICovXG50eXBlIFJlc3RBcmdzTGlzdFR5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke1Jlc3RWYWx1ZX06JHtUfVtdOiR7c3RyaW5nfWBcbj47XG5cbi8qKlxuICogUmVzdCBhcmdzIHdpdGggbGlzdCB0eXBlLlxuICpcbiAqIC0gYFsuLi5uYW1lOnR5cGVbXV1gXG4gKiAtIGA8Li4ubmFtZTp0eXBlW10+YFxuICogLSBgW25hbWUuLi46dHlwZVtdXWBcbiAqIC0gYDxuYW1lLi4uOnR5cGVbXT5gXG4gKi9cbnR5cGUgUmVzdEFyZ3NMaXN0VHlwZTxUIGV4dGVuZHMgc3RyaW5nPiA9IE9wdGlvbmFsT3JSZXF1aXJlZFZhbHVlPFxuICBgJHtSZXN0VmFsdWV9OiR7VH1bXWBcbj47XG5cbi8qKlxuICogUmVzdCBhcmdzIHdpdGggdHlwZSBhbmQgY29tcGxldGlvbnMuXG4gKlxuICogLSBgWy4uLm5hbWU6dHlwZTpjb21wbGV0aW9uXWBcbiAqIC0gYDwuLi5uYW1lOnR5cGU6Y29tcGxldGlvbj5gXG4gKiAtIGBbbmFtZS4uLjp0eXBlOmNvbXBsZXRpb25dYFxuICogLSBgPG5hbWUuLi46dHlwZTpjb21wbGV0aW9uPmBcbiAqL1xudHlwZSBSZXN0QXJnc1R5cGVDb21wbGV0aW9uPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke1Jlc3RWYWx1ZX06JHtUfToke3N0cmluZ31gXG4+O1xuXG4vKipcbiAqIFJlc3QgYXJncyB3aXRoIHR5cGUuXG4gKlxuICogLSBgWy4uLm5hbWU6dHlwZV1gXG4gKiAtIGA8Li4ubmFtZTp0eXBlPmBcbiAqIC0gYFtuYW1lLi4uOnR5cGVdYFxuICogLSBgPG5hbWUuLi46dHlwZT5gXG4gKi9cbnR5cGUgUmVzdEFyZ3NUeXBlPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke1Jlc3RWYWx1ZX06JHtUfWBcbj47XG5cbi8qKlxuICogUmVzdCBhcmdzLlxuICogLSBgWy4uLm5hbWVdYFxuICogLSBgPC4uLm5hbWU+YFxuICogLSBgW25hbWUuLi5dYFxuICogLSBgPG5hbWUuLi4+YFxuICovXG50eXBlIFJlc3RBcmdzID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke1Jlc3RWYWx1ZX1gXG4+O1xuXG4vKipcbiAqIFNpbmdsZSBhcmcgd2l0aCBsaXN0IHR5cGUgYW5kIGNvbXBsZXRpb25zLlxuICpcbiAqIC0gYFtuYW1lOnR5cGVbXTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lOnR5cGVbXTpjb21wbGV0aW9uPmBcbiAqL1xudHlwZSBTaW5nbGVBcmdMaXN0VHlwZUNvbXBsZXRpb248VCBleHRlbmRzIHN0cmluZz4gPSBPcHRpb25hbE9yUmVxdWlyZWRWYWx1ZTxcbiAgYCR7c3RyaW5nfToke1R9W106JHtzdHJpbmd9YFxuPjtcblxuLyoqXG4gKiBTaW5nbGUgYXJnIHdpdGggbGlzdCB0eXBlLlxuICpcbiAqIC0gYFtuYW1lOnR5cGVbXV1gXG4gKiAtIGA8bmFtZTp0eXBlW10+YFxuICovXG50eXBlIFNpbmdsZUFyZ0xpc3RUeXBlPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfVtdYFxuPjtcblxuLyoqXG4gKiBTaW5nbGUgYXJnICB3aXRoIHR5cGUgYW5kIGNvbXBsZXRpb24uXG4gKlxuICogLSBgW25hbWU6dHlwZTpjb21wbGV0aW9uXWBcbiAqIC0gYDxuYW1lOnR5cGU6Y29tcGxldGlvbj5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnVHlwZUNvbXBsZXRpb248VCBleHRlbmRzIHN0cmluZz4gPSBPcHRpb25hbE9yUmVxdWlyZWRWYWx1ZTxcbiAgYCR7c3RyaW5nfToke1R9OiR7c3RyaW5nfWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZyB3aXRoIHR5cGUuXG4gKlxuICogLSBgW25hbWU6dHlwZV1gXG4gKiAtIGA8bmFtZTp0eXBlPmBcbiAqL1xudHlwZSBTaW5nbGVBcmdUeXBlPFQgZXh0ZW5kcyBzdHJpbmc+ID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ306JHtUfWBcbj47XG5cbi8qKlxuICogU2luZ2xlIGFyZy5cbiAqXG4gKiAtIGBbbmFtZV1gXG4gKiAtIGA8bmFtZT5gXG4gKi9cbnR5cGUgU2luZ2xlQXJnID0gT3B0aW9uYWxPclJlcXVpcmVkVmFsdWU8XG4gIGAke3N0cmluZ31gXG4+O1xuXG50eXBlIERlZmF1bHRUeXBlcyA9IHtcbiAgbnVtYmVyOiBOdW1iZXJUeXBlO1xuICBpbnRlZ2VyOiBJbnRlZ2VyVHlwZTtcbiAgc3RyaW5nOiBTdHJpbmdUeXBlO1xuICBib29sZWFuOiBCb29sZWFuVHlwZTtcbiAgZmlsZTogRmlsZVR5cGU7XG59O1xuXG50eXBlIEFyZ3VtZW50VHlwZTxBIGV4dGVuZHMgc3RyaW5nLCBVLCBUID0gTWVyZ2U8RGVmYXVsdFR5cGVzLCBVPj4gPSBBIGV4dGVuZHNcbiAgUmVzdEFyZ3NMaXN0VHlwZUNvbXBsZXRpb248aW5mZXIgVHlwZT5cbiAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gQXJyYXk8QXJyYXk8Uj4+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBSZXN0QXJnc0xpc3RUeXBlPGluZmVyIFR5cGU+XG4gICAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gQXJyYXk8QXJyYXk8Uj4+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBSZXN0QXJnc1R5cGVDb21wbGV0aW9uPGluZmVyIFR5cGU+XG4gICAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gQXJyYXk8Uj4gOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFJlc3RBcmdzVHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IEFycmF5PFI+IDogdW5rbm93blxuICA6IEEgZXh0ZW5kcyBSZXN0QXJncyA/IEFycmF5PHN0cmluZz5cbiAgOiBBIGV4dGVuZHMgU2luZ2xlQXJnTGlzdFR5cGVDb21wbGV0aW9uPGluZmVyIFR5cGU+XG4gICAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gQXJyYXk8Uj4gOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ0xpc3RUeXBlPGluZmVyIFR5cGU+XG4gICAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gQXJyYXk8Uj4gOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZ1R5cGVDb21wbGV0aW9uPGluZmVyIFR5cGU+XG4gICAgPyBUIGV4dGVuZHMgUmVjb3JkPFR5cGUsIGluZmVyIFI+ID8gUiA6IHVua25vd25cbiAgOiBBIGV4dGVuZHMgU2luZ2xlQXJnVHlwZTxpbmZlciBUeXBlPlxuICAgID8gVCBleHRlbmRzIFJlY29yZDxUeXBlLCBpbmZlciBSPiA/IFIgOiB1bmtub3duXG4gIDogQSBleHRlbmRzIFNpbmdsZUFyZyA/IHN0cmluZ1xuICA6IHVua25vd247XG5cbnR5cGUgQXJndW1lbnRUeXBlczxBIGV4dGVuZHMgc3RyaW5nLCBUPiA9IEEgZXh0ZW5kcyBgJHtzdHJpbmd9ICR7c3RyaW5nfWBcbiAgPyBUeXBlZEFyZ3VtZW50czxBLCBUPlxuICA6IEFyZ3VtZW50VHlwZTxBLCBUPjtcblxudHlwZSBHZXRBcmd1bWVudHM8QSBleHRlbmRzIHN0cmluZz4gPSBBIGV4dGVuZHMgYC0ke3N0cmluZ309JHtpbmZlciBSZXN0fWBcbiAgPyBHZXRBcmd1bWVudHM8UmVzdD5cbiAgOiBBIGV4dGVuZHMgYC0ke3N0cmluZ30gJHtpbmZlciBSZXN0fWAgPyBHZXRBcmd1bWVudHM8UmVzdD5cbiAgOiBBO1xuXG50eXBlIE9wdGlvbk5hbWU8TmFtZSBleHRlbmRzIHN0cmluZz4gPSBOYW1lIGV4dGVuZHMgXCIqXCIgPyBzdHJpbmdcbiAgOiBDYW1lbENhc2U8VHJpbVJpZ2h0PE5hbWUsIFwiLFwiPj47XG5cbnR5cGUgSXNSZXF1aXJlZDxSIGV4dGVuZHMgYm9vbGVhbiB8IHVuZGVmaW5lZCwgRD4gPSBSIGV4dGVuZHMgdHJ1ZSA/IHRydWVcbiAgOiBEIGV4dGVuZHMgdW5kZWZpbmVkID8gZmFsc2VcbiAgOiB0cnVlO1xuXG50eXBlIE5lZ2F0YWJsZU9wdGlvbjxcbiAgRiBleHRlbmRzIHN0cmluZyxcbiAgQ08sXG4gIEQsXG4gIE4gZXh0ZW5kcyBzdHJpbmcgPSBPcHRpb25OYW1lPEY+LFxuPiA9IEQgZXh0ZW5kcyB1bmRlZmluZWRcbiAgPyBOIGV4dGVuZHMga2V5b2YgQ08gPyB7IFtLIGluIE5dPzogZmFsc2UgfSA6IHsgW0sgaW4gTl06IGJvb2xlYW4gfVxuICA6IHsgW0sgaW4gTl06IE5vbk51bGxhYmxlPEQ+IHwgZmFsc2UgfTtcblxudHlwZSBCb29sZWFuT3B0aW9uPFxuICBOIGV4dGVuZHMgc3RyaW5nLFxuICBDTyxcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gTiBleHRlbmRzIGBuby0ke2luZmVyIE5hbWV9YCA/IE5lZ2F0YWJsZU9wdGlvbjxOYW1lLCBDTywgRD5cbiAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0uJHtpbmZlciBSZXN0fWBcbiAgICA/IChSIGV4dGVuZHMgdHJ1ZVxuICAgICAgPyB7IFtLIGluIE9wdGlvbk5hbWU8TmFtZT5dOiBCb29sZWFuT3B0aW9uPFJlc3QsIENPLCBSLCBEPiB9XG4gICAgICA6IHsgW0sgaW4gT3B0aW9uTmFtZTxOYW1lPl0/OiBCb29sZWFuT3B0aW9uPFJlc3QsIENPLCBSLCBEPiB9KVxuICA6IChSIGV4dGVuZHMgdHJ1ZSA/IHsgW0sgaW4gT3B0aW9uTmFtZTxOPl06IHRydWUgfCBEIH1cbiAgICA6IHsgW0sgaW4gT3B0aW9uTmFtZTxOPl0/OiB0cnVlIHwgRCB9KTtcblxudHlwZSBWYWx1ZU9wdGlvbjxcbiAgTiBleHRlbmRzIHN0cmluZyxcbiAgRiBleHRlbmRzIHN0cmluZyxcbiAgVixcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gTiBleHRlbmRzIGAke2luZmVyIE5hbWV9LiR7aW5mZXIgUmVzdE5hbWV9YCA/IChSIGV4dGVuZHMgdHJ1ZSA/IHtcbiAgW0sgaW4gT3B0aW9uTmFtZTxOYW1lPl06IFZhbHVlT3B0aW9uPFJlc3ROYW1lLCBGLCBWLCBSLCBEPjtcbn1cbiAgOiB7XG4gICAgW0sgaW4gT3B0aW9uTmFtZTxOYW1lPl0/OiBWYWx1ZU9wdGlvbjxSZXN0TmFtZSwgRiwgViwgUiwgRD47XG4gIH0pXG4gIDogKFIgZXh0ZW5kcyB0cnVlID8ge1xuICAgIFtLIGluIE9wdGlvbk5hbWU8Tj5dOiBHZXRBcmd1bWVudHM8Rj4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICA/IE5vbk51bGxhYmxlPEQ+IHwgdHJ1ZSB8IEFyZ3VtZW50VHlwZTxHZXRBcmd1bWVudHM8Rj4sIFY+XG4gICAgICA6IE5vbk51bGxhYmxlPEQ+IHwgQXJndW1lbnRUeXBlPEdldEFyZ3VtZW50czxGPiwgVj47XG4gIH1cbiAgICA6IHtcbiAgICAgIFtLIGluIE9wdGlvbk5hbWU8Tj5dPzogR2V0QXJndW1lbnRzPEY+IGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgICA/IE5vbk51bGxhYmxlPEQ+IHwgdHJ1ZSB8IEFyZ3VtZW50VHlwZTxHZXRBcmd1bWVudHM8Rj4sIFY+XG4gICAgICAgIDogTm9uTnVsbGFibGU8RD4gfCBBcmd1bWVudFR5cGU8R2V0QXJndW1lbnRzPEY+LCBWPjtcbiAgICB9KTtcblxudHlwZSBWYWx1ZXNPcHRpb248XG4gIFQgZXh0ZW5kcyBzdHJpbmcsXG4gIFJlc3QgZXh0ZW5kcyBzdHJpbmcsXG4gIFYsXG4gIFIgZXh0ZW5kcyBib29sZWFuIHwgdW5kZWZpbmVkID0gdW5kZWZpbmVkLFxuICBEID0gdW5kZWZpbmVkLFxuPiA9IFQgZXh0ZW5kcyBgJHtpbmZlciBOYW1lfS4ke2luZmVyIFJlc3ROYW1lfWAgPyAoUiBleHRlbmRzIHRydWUgPyB7XG4gIFtOIGluIE9wdGlvbk5hbWU8TmFtZT5dOiBWYWx1ZXNPcHRpb248UmVzdE5hbWUsIFJlc3QsIFYsIFIsIEQ+O1xufVxuICA6IHtcbiAgICBbTiBpbiBPcHRpb25OYW1lPE5hbWU+XT86IFZhbHVlc09wdGlvbjxSZXN0TmFtZSwgUmVzdCwgViwgUiwgRD47XG4gIH0pXG4gIDogKFIgZXh0ZW5kcyB0cnVlID8ge1xuICAgIFtOIGluIE9wdGlvbk5hbWU8VD5dOiBHZXRBcmd1bWVudHM8UmVzdD4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICA/IE5vbk51bGxhYmxlPEQ+IHwgdHJ1ZSB8IEFyZ3VtZW50VHlwZXM8R2V0QXJndW1lbnRzPFJlc3Q+LCBWPlxuICAgICAgOiBOb25OdWxsYWJsZTxEPiB8IEFyZ3VtZW50VHlwZXM8R2V0QXJndW1lbnRzPFJlc3Q+LCBWPjtcbiAgfVxuICAgIDoge1xuICAgICAgW04gaW4gT3B0aW9uTmFtZTxUPl0/OiBHZXRBcmd1bWVudHM8UmVzdD4gZXh0ZW5kcyBgWyR7c3RyaW5nfV1gXG4gICAgICAgID8gTm9uTnVsbGFibGU8RD4gfCB0cnVlIHwgQXJndW1lbnRUeXBlczxHZXRBcmd1bWVudHM8UmVzdD4sIFY+XG4gICAgICAgIDogTm9uTnVsbGFibGU8RD4gfCBBcmd1bWVudFR5cGVzPEdldEFyZ3VtZW50czxSZXN0PiwgVj47XG4gICAgfSk7XG5cbnR5cGUgTWFwVmFsdWU8TywgViwgQyA9IHVuZGVmaW5lZD4gPSBWIGV4dGVuZHMgdW5kZWZpbmVkID8gQyBleHRlbmRzIHRydWUgPyB7XG4gIFtLIGluIGtleW9mIE9dOiBPW0tdIGV4dGVuZHMgKFJlY29yZDxzdHJpbmcsIHVua25vd24+IHwgdW5kZWZpbmVkKVxuICAgID8gTWFwVmFsdWU8T1tLXSwgVj5cbiAgICA6IEFycmF5PE5vbk51bGxhYmxlPE9bS10+Pjtcbn1cbjogT1xuICA6IHtcbiAgICBbSyBpbiBrZXlvZiBPXTogT1tLXSBleHRlbmRzIChSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiB8IHVuZGVmaW5lZClcbiAgICAgID8gTWFwVmFsdWU8T1tLXSwgVj5cbiAgICAgIDogVjtcbiAgfTtcblxudHlwZSBHZXRPcHRpb25OYW1lPFQ+ID0gVCBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX09JHtzdHJpbmd9YFxuICA/IFRyaW1SaWdodDxOYW1lLCBcIixcIj5cbiAgOiBUIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfSAke3N0cmluZ31gID8gVHJpbVJpZ2h0PE5hbWUsIFwiLFwiPlxuICA6IFQgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9YCA/IE5hbWVcbiAgOiBUIGV4dGVuZHMgYC0ke2luZmVyIE5hbWV9PSR7c3RyaW5nfWAgPyBUcmltUmlnaHQ8TmFtZSwgXCIsXCI+XG4gIDogVCBleHRlbmRzIGAtJHtpbmZlciBOYW1lfSAke3N0cmluZ31gID8gVHJpbVJpZ2h0PE5hbWUsIFwiLFwiPlxuICA6IFQgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX1gID8gTmFtZVxuICA6IHVua25vd247XG5cbnR5cGUgTWVyZ2VPcHRpb25zPFQsIENPLCBPLCBOID0gR2V0T3B0aW9uTmFtZTxUPj4gPSBOIGV4dGVuZHMgYG5vLSR7c3RyaW5nfWBcbiAgPyBTcHJlYWQ8Q08sIE8+XG4gIDogTiBleHRlbmRzIGAke3N0cmluZ30uJHtzdHJpbmd9YCA/IE1lcmdlUmVjdXJzaXZlPENPLCBPPlxuICA6IE1lcmdlPENPLCBPPjtcblxuLy8gdHlwZSBNZXJnZU9wdGlvbnM8VCwgQ08sIE8sIE4gPSBHZXRPcHRpb25OYW1lPFQ+PiA9IE4gZXh0ZW5kcyBgbm8tJHtzdHJpbmd9YFxuLy8gICA/IFNwcmVhZDxDTywgTz5cbi8vICAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0uJHtpbmZlciBDaGlsZH1gXG4vLyAgICAgPyAoT3B0aW9uTmFtZTxOYW1lPiBleHRlbmRzIGtleW9mIE1lcmdlPENPLCBPPlxuLy8gICAgICAgPyBPcHRpb25OYW1lPENoaWxkPiBleHRlbmRzXG4vLyAgICAgICAgIGtleW9mIE5vbk51bGxhYmxlPE1lcmdlPENPLCBPPltPcHRpb25OYW1lPE5hbWU+XT4gPyBTcHJlYWRUd288Q08sIE8+XG4vLyAgICAgICA6IE1lcmdlUmVjdXJzaXZlPENPLCBPPlxuLy8gICAgICAgOiBNZXJnZVJlY3Vyc2l2ZTxDTywgTz4pXG4vLyAgIDogTWVyZ2U8Q08sIE8+O1xuXG50eXBlIFR5cGVkT3B0aW9uPFxuICBGIGV4dGVuZHMgc3RyaW5nLFxuICBDTyxcbiAgVCxcbiAgUiBleHRlbmRzIGJvb2xlYW4gfCB1bmRlZmluZWQgPSB1bmRlZmluZWQsXG4gIEQgPSB1bmRlZmluZWQsXG4+ID0gbnVtYmVyIGV4dGVuZHMgVCA/IGFueVxuICA6IEYgZXh0ZW5kcyBgJHtzdHJpbmd9LS0ke2luZmVyIE5hbWV9PSR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZXNPcHRpb248TmFtZSwgUmVzdCwgVCwgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBGIGV4dGVuZHMgYCR7c3RyaW5nfS0tJHtpbmZlciBOYW1lfSAke2luZmVyIFJlc3R9YFxuICAgID8gVmFsdWVzT3B0aW9uPE5hbWUsIFJlc3QsIFQsIElzUmVxdWlyZWQ8UiwgRD4sIEQ+XG4gIDogRiBleHRlbmRzIGAke3N0cmluZ30tLSR7aW5mZXIgTmFtZX1gXG4gICAgPyBCb29sZWFuT3B0aW9uPE5hbWUsIENPLCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX09JHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlc09wdGlvbjxOYW1lLCBSZXN0LCBULCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX0gJHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlc09wdGlvbjxOYW1lLCBSZXN0LCBULCBJc1JlcXVpcmVkPFIsIEQ+LCBEPlxuICA6IEYgZXh0ZW5kcyBgLSR7aW5mZXIgTmFtZX1gID8gQm9vbGVhbk9wdGlvbjxOYW1lLCBDTywgSXNSZXF1aXJlZDxSLCBEPiwgRD5cbiAgOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPjtcblxudHlwZSBUeXBlZEFyZ3VtZW50czxBIGV4dGVuZHMgc3RyaW5nLCBUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgYW55PiB8IHZvaWQ+ID1cbiAgbnVtYmVyIGV4dGVuZHMgVCA/IGFueVxuICAgIDogQSBleHRlbmRzIGAke2luZmVyIEFyZ30gJHtpbmZlciBSZXN0fWBcbiAgICAgID8gQXJnIGV4dGVuZHMgYFske3N0cmluZ31dYFxuICAgICAgICA/IFtBcmd1bWVudFR5cGU8QXJnLCBUPj8sIC4uLlR5cGVkQXJndW1lbnRzPFJlc3QsIFQ+XVxuICAgICAgOiBbQXJndW1lbnRUeXBlPEFyZywgVD4sIC4uLlR5cGVkQXJndW1lbnRzPFJlc3QsIFQ+XVxuICAgIDogQSBleHRlbmRzIGBbJHtzdHJpbmd9XWAgPyBbQXJndW1lbnRUeXBlPEEsIFQ+P11cbiAgICA6IFtBcmd1bWVudFR5cGU8QSwgVD5dO1xuXG50eXBlIFR5cGVkQ29tbWFuZEFyZ3VtZW50czxOIGV4dGVuZHMgc3RyaW5nLCBUPiA9IG51bWJlciBleHRlbmRzIFQgPyBhbnlcbiAgOiBOIGV4dGVuZHMgYCR7c3RyaW5nfSAke2luZmVyIEFyZ3N9YCA/IFR5cGVkQXJndW1lbnRzPEFyZ3MsIFQ+XG4gIDogW107XG5cbnR5cGUgVHlwZWRFbnY8XG4gIE4gZXh0ZW5kcyBzdHJpbmcsXG4gIFAgZXh0ZW5kcyBzdHJpbmcgfCB1bmRlZmluZWQsXG4gIENPLFxuICBULFxuICBSIGV4dGVuZHMgYm9vbGVhbiB8IHVuZGVmaW5lZCA9IHVuZGVmaW5lZCxcbiAgRCA9IHVuZGVmaW5lZCxcbj4gPSBudW1iZXIgZXh0ZW5kcyBUID8gYW55XG4gIDogTiBleHRlbmRzIGAke2luZmVyIE5hbWV9PSR7aW5mZXIgUmVzdH1gXG4gICAgPyBWYWx1ZU9wdGlvbjxUcmltTGVmdDxOYW1lLCBQPiwgUmVzdCwgVCwgUiwgRD5cbiAgOiBOIGV4dGVuZHMgYCR7aW5mZXIgTmFtZX0gJHtpbmZlciBSZXN0fWBcbiAgICA/IFZhbHVlT3B0aW9uPFRyaW1MZWZ0PE5hbWUsIFA+LCBSZXN0LCBULCBSLCBEPlxuICA6IE4gZXh0ZW5kcyBgJHtpbmZlciBOYW1lfWAgPyBCb29sZWFuT3B0aW9uPFRyaW1MZWZ0PE5hbWUsIFA+LCBDTywgUiwgRD5cbiAgOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPjtcblxudHlwZSBUeXBlZFR5cGU8XG4gIE5hbWUgZXh0ZW5kcyBzdHJpbmcsXG4gIEhhbmRsZXIgZXh0ZW5kcyBUeXBlT3JUeXBlSGFuZGxlcjx1bmtub3duPixcbj4gPSB7IFtOIGluIE5hbWVdOiBIYW5kbGVyIH07XG5cbnR5cGUgUmVxdWlyZWRLZXlzPFQ+ID0ge1xuICAvLyBkZW5vLWxpbnQtaWdub3JlIGJhbi10eXBlc1xuICBbSyBpbiBrZXlvZiBUXS0/OiB7fSBleHRlbmRzIFBpY2s8VCwgSz4gPyBuZXZlciA6IEs7XG59W2tleW9mIFRdO1xuXG50eXBlIE9wdGlvbmFsS2V5czxUPiA9IHtcbiAgLy8gZGVuby1saW50LWlnbm9yZSBiYW4tdHlwZXNcbiAgW0sgaW4ga2V5b2YgVF0tPzoge30gZXh0ZW5kcyBQaWNrPFQsIEs+ID8gSyA6IG5ldmVyO1xufVtrZXlvZiBUXTtcblxudHlwZSBTcHJlYWRSZXF1aXJlZFByb3BlcnRpZXM8XG4gIEwsXG4gIFIsXG4gIEsgZXh0ZW5kcyBrZXlvZiBMICYga2V5b2YgUixcbj4gPSB7XG4gIFtQIGluIEtdOiBFeGNsdWRlPExbUF0sIHVuZGVmaW5lZD4gfCBFeGNsdWRlPFJbUF0sIHVuZGVmaW5lZD47XG59O1xuXG50eXBlIFNwcmVhZE9wdGlvbmFsUHJvcGVydGllczxcbiAgTCxcbiAgUixcbiAgSyBleHRlbmRzIGtleW9mIEwgJiBrZXlvZiBSLFxuPiA9IHtcbiAgW1AgaW4gS10/OiBMW1BdIHwgUltQXTtcbn07XG5cbi8qKiBNZXJnZSB0eXBlcyBvZiB0d28gb2JqZWN0cy4gKi9cbnR5cGUgU3ByZWFkPEwsIFI+ID0gTCBleHRlbmRzIHZvaWQgPyBSIDogUiBleHRlbmRzIHZvaWQgPyBMXG46IC8vIFByb3BlcnRpZXMgaW4gTCB0aGF0IGRvbid0IGV4aXN0IGluIFIuXG4mIE9taXQ8TCwga2V5b2YgUj5cbi8vIFByb3BlcnRpZXMgaW4gUiB0aGF0IGRvbid0IGV4aXN0IGluIEwuXG4mIE9taXQ8Uiwga2V5b2YgTD5cbi8vIFJlcXVpcmVkIHByb3BlcnRpZXMgaW4gUiB0aGF0IGV4aXN0IGluIEwuXG4mIFNwcmVhZFJlcXVpcmVkUHJvcGVydGllczxMLCBSLCBSZXF1aXJlZEtleXM8Uj4gJiBrZXlvZiBMPlxuLy8gUmVxdWlyZWQgcHJvcGVydGllcyBpbiBMIHRoYXQgZXhpc3QgaW4gUi5cbiYgU3ByZWFkUmVxdWlyZWRQcm9wZXJ0aWVzPEwsIFIsIFJlcXVpcmVkS2V5czxMPiAmIGtleW9mIFI+XG4vLyBPcHRpb25hbCBwcm9wZXJ0aWVzIGluIEwgYW5kIFIuXG4mIFNwcmVhZE9wdGlvbmFsUHJvcGVydGllczxcbiAgTCxcbiAgUixcbiAgT3B0aW9uYWxLZXlzPEw+ICYgT3B0aW9uYWxLZXlzPFI+XG4+O1xuXG50eXBlIFZhbHVlT2Y8VD4gPSBUIGV4dGVuZHMgUmVjb3JkPHN0cmluZywgaW5mZXIgVj4gPyBWYWx1ZU9mPFY+IDogVDtcbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSx3Q0FBd0M7QUFDeEMsU0FDRSxXQUFXLEVBQ1gsZUFBZSxJQUFJLG9CQUFvQixRQUNsQyxxQkFBcUIsQ0FBQztBQUM3QixTQUFTLHFCQUFxQixRQUFRLGNBQWMsQ0FBQztBQUNyRCxTQUFTLFVBQVUsUUFBUSxtQkFBbUIsQ0FBQztBQUUvQyxTQUNFLGNBQWMsRUFDZCx3QkFBd0IsRUFDeEIsY0FBYyxRQUNULGFBQWEsQ0FBQztBQUNyQixTQUFTLElBQUksRUFBRSxJQUFJLEVBQUUsR0FBRyxFQUFFLE1BQU0sUUFBUSxXQUFXLENBQUM7QUFDcEQsU0FDRSx5QkFBeUIsRUFDekIsZUFBZSxFQUNmLHNCQUFzQixFQUN0QixxQkFBcUIsRUFDckIsb0JBQW9CLEVBQ3BCLG1CQUFtQixFQUNuQiw0QkFBNEIsRUFDNUIsZ0JBQWdCLEVBQ2hCLG1CQUFtQixFQUNuQixhQUFhLEVBQ2IsZ0NBQWdDLEVBQ2hDLDhCQUE4QixFQUM5QixnQ0FBZ0MsRUFDaEMsZUFBZSxFQUNmLGdCQUFnQixFQUNoQixrQkFBa0IsRUFDbEIsa0JBQWtCLEVBQ2xCLGdCQUFnQixFQUNoQixjQUFjLEVBQ2QsZUFBZSxRQUNWLGNBQWMsQ0FBQztBQUN0QixTQUFTLFdBQVcsUUFBUSxvQkFBb0IsQ0FBQztBQUNqRCxTQUFTLFFBQVEsUUFBUSxpQkFBaUIsQ0FBQztBQUMzQyxTQUFTLFVBQVUsUUFBUSxtQkFBbUIsQ0FBQztBQUMvQyxTQUFTLFVBQVUsUUFBUSxtQkFBbUIsQ0FBQztBQUMvQyxTQUFTLElBQUksUUFBUSxXQUFXLENBQUM7QUFDakMsU0FBUyxhQUFhLFFBQVEsMkJBQTJCLENBQUM7QUEyQjFELFNBQVMsV0FBVyxRQUFRLG9CQUFvQixDQUFDO0FB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