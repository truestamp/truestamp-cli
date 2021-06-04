import { getDefaultValue, getOption, paramCaseToCamelCase } from "./_utils.ts";
import { ArgumentFollowsVariadicArgument, DuplicateOptionName, InvalidOptionValue, MissingOptionValue, RequiredArgumentFollowsOptionalArgument, UnknownConflictingOption, UnknownOption, UnknownRequiredOption, UnknownType, } from "./_errors.ts";
import { normalize } from "./normalize.ts";
import { OptionType } from "./types.ts";
import { boolean } from "./types/boolean.ts";
import { number } from "./types/number.ts";
import { string } from "./types/string.ts";
import { validateFlags } from "./validate_flags.ts";
import { integer } from "./types/integer.ts";
const Types = {
    [OptionType.STRING]: string,
    [OptionType.NUMBER]: number,
    [OptionType.INTEGER]: integer,
    [OptionType.BOOLEAN]: boolean,
};
export function parseFlags(args, opts = {}) {
    !opts.flags && (opts.flags = []);
    const normalized = normalize(args);
    let inLiteral = false;
    let negate = false;
    const flags = {};
    const optionNames = {};
    const literal = [];
    const unknown = [];
    let stopEarly = false;
    opts.flags.forEach((opt) => {
        opt.depends?.forEach((flag) => {
            if (!opts.flags || !getOption(opts.flags, flag)) {
                throw new UnknownRequiredOption(flag, opts.flags ?? []);
            }
        });
        opt.conflicts?.forEach((flag) => {
            if (!opts.flags || !getOption(opts.flags, flag)) {
                throw new UnknownConflictingOption(flag, opts.flags ?? []);
            }
        });
    });
    for (let i = 0; i < normalized.length; i++) {
        let option;
        let args;
        const current = normalized[i];
        if (inLiteral) {
            literal.push(current);
            continue;
        }
        if (current === "--") {
            inLiteral = true;
            continue;
        }
        const isFlag = current.length > 1 && current[0] === "-";
        const next = () => normalized[i + 1];
        if (isFlag && !stopEarly) {
            if (current[2] === "-" || (current[1] === "-" && current.length === 3)) {
                throw new UnknownOption(current, opts.flags);
            }
            negate = current.startsWith("--no-");
            option = getOption(opts.flags, current);
            if (!option) {
                if (opts.flags.length) {
                    throw new UnknownOption(current, opts.flags);
                }
                option = {
                    name: current.replace(/^-+/, ""),
                    optionalValue: true,
                    type: OptionType.STRING,
                };
            }
            const positiveName = option.name.replace(/^no-?/, "");
            const propName = paramCaseToCamelCase(positiveName);
            if (typeof flags[propName] !== "undefined" && !option.collect) {
                throw new DuplicateOptionName(current);
            }
            args = option.args?.length ? option.args : [{
                    type: option.type,
                    requiredValue: option.requiredValue,
                    optionalValue: option.optionalValue,
                    variadic: option.variadic,
                    list: option.list,
                    separator: option.separator,
                }];
            let argIndex = 0;
            let inOptionalArg = false;
            const previous = flags[propName];
            parseNext(option, args);
            if (typeof flags[propName] === "undefined") {
                if (typeof option.default !== "undefined") {
                    flags[propName] = getDefaultValue(option);
                }
                else if (args[argIndex].requiredValue) {
                    throw new MissingOptionValue(option.name);
                }
                else {
                    flags[propName] = true;
                }
            }
            if (option.value) {
                flags[propName] = option.value(flags[propName], previous);
            }
            else if (option.collect) {
                const value = Array.isArray(previous) ? previous : [];
                value.push(flags[propName]);
                flags[propName] = value;
            }
            optionNames[propName] = option.name;
            opts.option?.(option, flags[propName]);
            function parseNext(option, args) {
                const arg = args[argIndex];
                if (!arg) {
                    const flag = next();
                    throw new UnknownOption(flag, opts.flags ?? []);
                }
                if (!arg.type) {
                    arg.type = OptionType.BOOLEAN;
                }
                if (option.args?.length) {
                    if ((typeof arg.optionalValue === "undefined" ||
                        arg.optionalValue === false) &&
                        typeof arg.requiredValue === "undefined") {
                        arg.requiredValue = true;
                    }
                }
                else {
                    if (arg.type !== OptionType.BOOLEAN &&
                        (typeof arg.optionalValue === "undefined" ||
                            arg.optionalValue === false) &&
                        typeof arg.requiredValue === "undefined") {
                        arg.requiredValue = true;
                    }
                }
                if (arg.requiredValue) {
                    if (inOptionalArg) {
                        throw new RequiredArgumentFollowsOptionalArgument(option.name);
                    }
                }
                else {
                    inOptionalArg = true;
                }
                if (negate) {
                    flags[propName] = false;
                    return;
                }
                let result;
                let increase = false;
                if (arg.list && hasNext(arg)) {
                    const parsed = next()
                        .split(arg.separator || ",")
                        .map((nextValue) => {
                        const value = parseValue(option, arg, nextValue);
                        if (typeof value === "undefined") {
                            throw new InvalidOptionValue(option.name, arg.type ?? "?", nextValue);
                        }
                        return value;
                    });
                    if (parsed?.length) {
                        result = parsed;
                    }
                }
                else {
                    if (hasNext(arg)) {
                        result = parseValue(option, arg, next());
                    }
                    else if (arg.optionalValue && arg.type === OptionType.BOOLEAN) {
                        result = true;
                    }
                }
                if (increase) {
                    i++;
                    if (!arg.variadic) {
                        argIndex++;
                    }
                    else if (args[argIndex + 1]) {
                        throw new ArgumentFollowsVariadicArgument(next());
                    }
                }
                if (typeof result !== "undefined" && ((args.length > 1) || arg.variadic)) {
                    if (!flags[propName]) {
                        flags[propName] = [];
                    }
                    flags[propName].push(result);
                    if (hasNext(arg)) {
                        parseNext(option, args);
                    }
                }
                else {
                    flags[propName] = result;
                }
                function hasNext(arg) {
                    return !!(normalized[i + 1] &&
                        (arg.optionalValue || arg.requiredValue || arg.variadic) &&
                        (normalized[i + 1][0] !== "-" ||
                            (arg.type === OptionType.NUMBER &&
                                !isNaN(Number(normalized[i + 1])))) &&
                        arg);
                }
                function parseValue(option, arg, value) {
                    const type = arg.type || OptionType.STRING;
                    const result = opts.parse
                        ? opts.parse({
                            label: "Option",
                            type,
                            name: `--${option.name}`,
                            value,
                        })
                        : parseFlagValue(option, arg, value);
                    if (typeof result !== "undefined") {
                        increase = true;
                    }
                    return result;
                }
            }
        }
        else {
            if (opts.stopEarly) {
                stopEarly = true;
            }
            unknown.push(current);
        }
    }
    if (opts.flags && opts.flags.length) {
        validateFlags(opts.flags, flags, opts.knownFlaks, opts.allowEmpty, optionNames);
    }
    const result = Object.keys(flags)
        .reduce((result, key) => {
        if (~key.indexOf(".")) {
            key.split(".").reduce((result, subKey, index, parts) => {
                if (index === parts.length - 1) {
                    result[subKey] = flags[key];
                }
                else {
                    result[subKey] = result[subKey] ?? {};
                }
                return result[subKey];
            }, result);
        }
        else {
            result[key] = flags[key];
        }
        return result;
    }, {});
    return { flags: result, unknown, literal };
}
function parseFlagValue(option, arg, value) {
    const type = arg.type || OptionType.STRING;
    const parseType = Types[type];
    if (!parseType) {
        throw new UnknownType(type, Object.keys(Types));
    }
    return parseType({
        label: "Option",
        type,
        name: `--${option.name}`,
        value,
    });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmxhZ3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmbGFncy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZUFBZSxFQUFFLFNBQVMsRUFBRSxvQkFBb0IsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsK0JBQStCLEVBQy9CLG1CQUFtQixFQUNuQixrQkFBa0IsRUFDbEIsa0JBQWtCLEVBQ2xCLHVDQUF1QyxFQUN2Qyx3QkFBd0IsRUFDeEIsYUFBYSxFQUNiLHFCQUFxQixFQUNyQixXQUFXLEdBQ1osTUFBTSxjQUFjLENBQUM7QUFDdEIsT0FBTyxFQUFFLFNBQVMsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBUTNDLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDeEMsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQzdDLE9BQU8sRUFBRSxNQUFNLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDM0MsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3BELE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUU3QyxNQUFNLEtBQUssR0FBMEM7SUFDbkQsQ0FBQyxVQUFVLENBQUMsTUFBTSxDQUFDLEVBQUUsTUFBTTtJQUMzQixDQUFDLFVBQVUsQ0FBQyxNQUFNLENBQUMsRUFBRSxNQUFNO0lBQzNCLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFLE9BQU87SUFDN0IsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLEVBQUUsT0FBTztDQUM5QixDQUFDO0FBNEJGLE1BQU0sVUFBVSxVQUFVLENBQ3hCLElBQWMsRUFDZCxPQUFzQixFQUFFO0lBRXhCLENBQUMsSUFBSSxDQUFDLEtBQUssSUFBSSxDQUFDLElBQUksQ0FBQyxLQUFLLEdBQUcsRUFB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