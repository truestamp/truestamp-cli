import { didYouMeanCommand } from "./_utils.ts";
export class CommandError extends Error {
    constructor(message) {
        super(message);
        Object.setPrototypeOf(this, CommandError.prototype);
    }
}
export class ValidationError extends CommandError {
    exitCode;
    constructor(message, { exitCode } = {}) {
        super(message);
        Object.setPrototypeOf(this, ValidationError.prototype);
        this.exitCode = exitCode ?? 1;
    }
}
export class MissingCommandName extends CommandError {
    constructor() {
        super("Missing command name.");
        Object.setPrototypeOf(this, MissingCommandName.prototype);
    }
}
export class DuplicateCommandName extends CommandError {
    constructor(name) {
        super(`Duplicate command name "${name}".`);
        Object.setPrototypeOf(this, DuplicateCommandName.prototype);
    }
}
export class DuplicateCommandAlias extends CommandError {
    constructor(alias) {
        super(`Duplicate command alias "${alias}".`);
        Object.setPrototypeOf(this, DuplicateCommandAlias.prototype);
    }
}
export class CommandNotFound extends CommandError {
    constructor(name, commands, excluded) {
        super(`Unknown command "${name}".${didYouMeanCommand(name, commands, excluded)}`);
        Object.setPrototypeOf(this, UnknownCommand.prototype);
    }
}
export class DuplicateType extends CommandError {
    constructor(name) {
        super(`Type with name "${name}" already exists.`);
        Object.setPrototypeOf(this, DuplicateType.prototype);
    }
}
export class DuplicateCompletion extends CommandError {
    constructor(name) {
        super(`Completion with name "${name}" already exists.`);
        Object.setPrototypeOf(this, DuplicateCompletion.prototype);
    }
}
export class DuplicateExample extends CommandError {
    constructor(name) {
        super(`Example with name "${name}" already exists.`);
        Object.setPrototypeOf(this, DuplicateExample.prototype);
    }
}
export class DuplicateEnvironmentVariable extends CommandError {
    constructor(name) {
        super(`Environment variable with name "${name}" already exists.`);
        Object.setPrototypeOf(this, DuplicateEnvironmentVariable.prototype);
    }
}
export class EnvironmentVariableSingleValue extends CommandError {
    constructor(name) {
        super(`An environment variable can only have one value, but "${name}" has more than one.`);
        Object.setPrototypeOf(this, EnvironmentVariableSingleValue.prototype);
    }
}
export class EnvironmentVariableOptionalValue extends CommandError {
    constructor(name) {
        super(`An environment variable cannot have an optional value, but "${name}" is defined as optional.`);
        Object.setPrototypeOf(this, EnvironmentVariableOptionalValue.prototype);
    }
}
export class EnvironmentVariableVariadicValue extends CommandError {
    constructor(name) {
        super(`An environment variable cannot have an variadic value, but "${name}" is defined as variadic.`);
        Object.setPrototypeOf(this, EnvironmentVariableVariadicValue.prototype);
    }
}
export class DefaultCommandNotFound extends CommandError {
    constructor(name, commands) {
        super(`Default command "${name}" not found.${didYouMeanCommand(name, commands)}`);
        Object.setPrototypeOf(this, DefaultCommandNotFound.prototype);
    }
}
export class CommandExecutableNotFound extends CommandError {
    constructor(name, files) {
        super(`Command executable not found: ${name}:\n    - ${files.join("\\n    - ")}`);
        Object.setPrototypeOf(this, CommandExecutableNotFound.prototype);
    }
}
export class UnknownCompletionCommand extends CommandError {
    constructor(name, commands) {
        super(`Auto-completion failed. Unknown command "${name}".${didYouMeanCommand(name, commands)}`);
        Object.setPrototypeOf(this, UnknownCompletionCommand.prototype);
    }
}
export class UnknownCommand extends ValidationError {
    constructor(name, commands, excluded) {
        super(`Unknown command "${name}".${didYouMeanCommand(name, commands, excluded)}`);
        Object.setPrototypeOf(this, UnknownCommand.prototype);
    }
}
export class NoArgumentsAllowed extends ValidationError {
    constructor(name) {
        super(`No arguments allowed for command "${name}".`);
        Object.setPrototypeOf(this, NoArgumentsAllowed.prototype);
    }
}
export class MissingArguments extends ValidationError {
    constructor(args) {
        super("Missing argument(s): " + args.join(", "));
        Object.setPrototypeOf(this, MissingArguments.prototype);
    }
}
export class MissingArgument extends ValidationError {
    constructor(arg) {
        super(`Missing argument "${arg}".`);
        Object.setPrototypeOf(this, MissingArgument.prototype);
    }
}
export class TooManyArguments extends ValidationError {
    constructor(args) {
        super(`Too many arguments: ${args.join(" ")}`);
        Object.setPrototypeOf(this, TooManyArguments.prototype);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Vycm9ycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGlCQUFpQixFQUFFLE1BQU0sYUFBYSxDQUFDO0FBR2hELE1BQU0sT0FBTyxZQUFhLFNBQVEsS0FBSztJQUNyQyxZQUFZLE9BQWU7UUFDekIsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBQ2YsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUsWUFBWSxDQUFDLFNBQVMsQ0FBQyxDQUFDO0lBQ3RELENBQUM7Q0FDRjtBQU1ELE1BQU0sT0FBTyxlQUFnQixTQUFRLFlBQVk7SUFDL0IsUUFBUSxDQUFTO0lBRWpDLFlBQVksT0FBZSxFQUFFLEVBQUUsUUFBUSxLQUE2QixFQUFFO1FBQ3BFLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztRQUNmLE1BQU0sQ0FBQyxjQUFjLENBQUMsSUFBSSxFQUFFLGVBQWUsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUN2RCxJQUFJLENBQUMsUUFBUSxHQUFHLFFBQVEsSUFBSSxDQUFDLENBQUM7SUFDaEMsQ0FBQztDQUNGO0FBRUQsTUFBTSxPQUFPLGtCQUFtQixTQUFRLFlBQVk7SUFDbEQ7UUFDRSxLQUFLLENBQUMsdUJBQXVCLENBQUMsQ0FBQztRQUMvQixNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxrQkFBa0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztJQUM1RCxDQUFDO0NBQ0Y7QUFFRCxNQUFNLE9BQU8sb0JBQXFCLFNBQVEsWUFBWTtJQUNwRCxZQUFZLElBQVk7UUFDdEIsS0FBSyxDQUFDLDJCQUEyQixJQUFJLElBQUksQ0FBQyxDQUFDO1FBQzNDLE1BQU0sQ0FBQyxjQUFjLENBQUMsSUFBSSxFQUFFLG9CQUFvQixDQUFDLFNBQVMsQ0FBQyxDQUFDO0lBQzlELENBQUM7Q0FDRjtBQUVELE1BQU0sT0FBTyxxQkFBc0IsU0FBUSxZQUFZO0lBQ3JELFlBQVksS0FB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