import { didYouMeanCommand } from "./_utils.ts";
import { getFlag } from "../flags/_utils.ts";
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
export class DuplicateOptionName extends CommandError {
    constructor(name) {
        super(`Option with name "${getFlag(name)}" already exists.`);
        Object.setPrototypeOf(this, DuplicateOptionName.prototype);
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Vycm9ycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGlCQUFpQixFQUFFLE1BQU0sYUFBYSxDQUFDO0FBRWhELE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUU3QyxNQUFNLE9BQU8sWUFBYSxTQUFRLEtBQUs7SUFDckMsWUFBWSxPQUFlO1FBQ3pCLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztRQUNmLE1BQU0sQ0FBQyxjQUFjLENBQUMsSUFBSSxFQUFFLFlBQVksQ0FBQyxTQUFTLENBQUMsQ0FBQztJQUN0RCxDQUFDO0NBQ0Y7QUFNRCxNQUFNLE9BQU8sZUFBZ0IsU0FBUSxZQUFZO0lBQy9CLFFBQVEsQ0FBUztJQUVqQyxZQUFZLE9BQWUsRUFBRSxFQUFFLFFBQVEsS0FBNkIsRUFBRTtRQUNwRSxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDZixNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxlQUFlLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDdkQsSUFBSSxDQUFDLFFBQVEsR0FBRyxRQUFRLElBQUksQ0FBQyxDQUFDO0lBQ2hDLENBQUM7Q0FDRjtBQUVELE1BQU0sT0FBTyxtQkFBb0IsU0FBUSxZQUFZO0lBQ25ELFlBQVksSUFBWTtRQUN0QixLQUFLLENBQUMscUJBQXFCLE9BQU8sQ0FBQyxJQUFJLENBQUMsbUJBQW1CLENBQUMsQ0FBQztRQUM3RCxNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxtQkFBbUIsQ0FBQyxTQUFTLENBQUMsQ0FBQztJQUM3RCxDQUFDO0NBQ0Y7QUFFRCxNQUFNLE9BQU8sa0JBQW1CLFNBQVEsWUFBWTtJQUNsRDtRQUNFLEtBQUssQ0FBQyx1QkFBdUIsQ0FBQyxDQUFDO1FBQy9CLE1BQU0sQ0FBQyxjQUFjLENBQUMsSUFBSSxFQUFFLGtCQUFrQixDQUFDLFNBQVMsQ0FBQyxDQUFDO0lB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