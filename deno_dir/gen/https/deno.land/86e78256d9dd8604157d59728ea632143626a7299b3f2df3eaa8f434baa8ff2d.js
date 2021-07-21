import { didYouMeanOption, didYouMeanType, getFlag } from "./_utils.ts";
export class FlagsError extends Error {
    constructor(message) {
        super(message);
        Object.setPrototypeOf(this, FlagsError.prototype);
    }
}
export class UnknownRequiredOption extends FlagsError {
    constructor(option, options) {
        super(`Unknown required option "${getFlag(option)}".${didYouMeanOption(option, options)}`);
        Object.setPrototypeOf(this, UnknownRequiredOption.prototype);
    }
}
export class UnknownConflictingOption extends FlagsError {
    constructor(option, options) {
        super(`Unknown conflicting option "${getFlag(option)}".${didYouMeanOption(option, options)}`);
        Object.setPrototypeOf(this, UnknownConflictingOption.prototype);
    }
}
export class UnknownType extends FlagsError {
    constructor(type, types) {
        super(`Unknown type "${type}".${didYouMeanType(type, types)}`);
        Object.setPrototypeOf(this, UnknownType.prototype);
    }
}
export class ValidationError extends FlagsError {
    constructor(message) {
        super(message);
        Object.setPrototypeOf(this, ValidationError.prototype);
    }
}
export class DuplicateOption extends ValidationError {
    constructor(name) {
        super(`Option "${getFlag(name).replace(/^--no-/, "--")}" can only occur once, but was found several times.`);
        Object.setPrototypeOf(this, DuplicateOption.prototype);
    }
}
export class UnknownOption extends ValidationError {
    constructor(option, options) {
        super(`Unknown option "${getFlag(option)}".${didYouMeanOption(option, options)}`);
        Object.setPrototypeOf(this, UnknownOption.prototype);
    }
}
export class MissingOptionValue extends ValidationError {
    constructor(option) {
        super(`Missing value for option "${getFlag(option)}".`);
        Object.setPrototypeOf(this, MissingOptionValue.prototype);
    }
}
export class InvalidOptionValue extends ValidationError {
    constructor(option, expected, value) {
        super(`Option "${getFlag(option)}" must be of type "${expected}", but got "${value}".`);
        Object.setPrototypeOf(this, InvalidOptionValue.prototype);
    }
}
export class OptionNotCombinable extends ValidationError {
    constructor(option) {
        super(`Option "${getFlag(option)}" cannot be combined with other options.`);
        Object.setPrototypeOf(this, OptionNotCombinable.prototype);
    }
}
export class ConflictingOption extends ValidationError {
    constructor(option, conflictingOption) {
        super(`Option "${getFlag(option)}" conflicts with option "${getFlag(conflictingOption)}".`);
        Object.setPrototypeOf(this, ConflictingOption.prototype);
    }
}
export class DependingOption extends ValidationError {
    constructor(option, dependingOption) {
        super(`Option "${getFlag(option)}" depends on option "${getFlag(dependingOption)}".`);
        Object.setPrototypeOf(this, DependingOption.prototype);
    }
}
export class MissingRequiredOption extends ValidationError {
    constructor(option) {
        super(`Missing required option "${getFlag(option)}".`);
        Object.setPrototypeOf(this, MissingRequiredOption.prototype);
    }
}
export class RequiredArgumentFollowsOptionalArgument extends ValidationError {
    constructor(arg) {
        super(`An required argument cannot follow an optional argument, but "${arg}"  is defined as required.`);
        Object.setPrototypeOf(this, RequiredArgumentFollowsOptionalArgument.prototype);
    }
}
export class ArgumentFollowsVariadicArgument extends ValidationError {
    constructor(arg) {
        super(`An argument cannot follow an variadic argument, but got "${arg}".`);
        Object.setPrototypeOf(this, ArgumentFollowsVariadicArgument.prototype);
    }
}
export class NoArguments extends ValidationError {
    constructor() {
        super(`No arguments.`);
        Object.setPrototypeOf(this, NoArguments.prototype);
    }
}
export class InvalidTypeError extends ValidationError {
    constructor({ label, name, value, type }, expected) {
        super(`${label} "${name}" must be of type "${type}", but got "${value}".` + (expected
            ? ` Expected values: ${expected.map((value) => `"${value}"`).join(", ")}`
            : ""));
        Object.setPrototypeOf(this, MissingOptionValue.prototype);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Vycm9ycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGdCQUFnQixFQUFFLGNBQWMsRUFBRSxPQUFPLEVBQUUsTUFBTSxhQUFhLENBQUM7QUFHeEUsTUFBTSxPQUFPLFVBQVcsU0FBUSxLQUFLO0lBQ25DLFlBQVksT0FBZTtRQUN6QixLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDZixNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxVQUFVLENBQUMsU0FBUyxDQUFDLENBQUM7SUFDcEQsQ0FBQztDQUNGO0FBRUQsTUFBTSxPQUFPLHFCQUFzQixTQUFRLFVBQVU7SUFDbkQsWUFBWSxNQUFjLEVBQUUsT0FBNEI7UUFDdEQsS0FBSyxDQUNILDRCQUE0QixPQUFPLENBQUMsTUFBTSxDQUFDLEtBQ3pDLGdCQUFnQixDQUFDLE1BQU0sRUFBRSxPQUFPLENBQ2xDLEVBQUUsQ0FDSCxDQUFDO1FBQ0YsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUscUJBQXFCLENBQUMsU0FBUyxDQUFDLENBQUM7SUFDL0QsQ0FBQztDQUNGO0FBRUQsTUFBTSxPQUFPLHdCQUF5QixTQUFRLFVBQVU7SUFDdEQsWUFBWSxNQUFjLEVBQUUsT0FBNEI7UUFDdEQsS0FBSyxDQUNILCtCQUErQixPQUFPLENBQUMsTUFBTSxDQUFDLEtBQzVDLGdCQUFnQixDQUFDLE1BQU0sRUFBRSxPQUFPLENBQ2xDLEVBQUUsQ0FDSCxDQUFDO1FBQ0YsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUsd0JBQXdCLENBQUMsU0FBUyxDQUFDLENBQUM7SUFDbEUsQ0FBQztDQUNGO0FBRUQsTUFBTSxPQUFPLFdBQVksU0FBUSxVQUFVO0lBQ3pDLFlBQVksSUFBWSxFQUFFLEtBQW9CO1FBQzVDLEtBQUssQ0FBQyxpQkFBaUIsSUFBSSxLQUFLLGNBQWMsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFD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