import { getDefaultValue, getOption, paramCaseToCamelCase } from "./_utils.ts";
import { ArgumentFollowsVariadicArgument, DuplicateOption, InvalidOptionValue, MissingOptionValue, RequiredArgumentFollowsOptionalArgument, UnknownConflictingOption, UnknownOption, UnknownRequiredOption, UnknownType, } from "./_errors.ts";
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
    let literal = [];
    let unknown = [];
    let stopEarly = null;
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
        if (isFlag) {
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
                throw new DuplicateOption(current);
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
                stopEarly = current;
                break;
            }
            unknown.push(current);
        }
    }
    if (stopEarly) {
        const stopEarlyArgIndex = args.indexOf(stopEarly);
        if (stopEarlyArgIndex !== -1) {
            const doubleDashIndex = args.indexOf("--");
            unknown = args.slice(stopEarlyArgIndex, doubleDashIndex === -1 ? undefined : doubleDashIndex);
            if (doubleDashIndex !== -1) {
                literal = args.slice(doubleDashIndex + 1);
            }
        }
    }
    if (opts.flags?.length) {
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmxhZ3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmbGFncy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZUFBZSxFQUFFLFNBQVMsRUFBRSxvQkFBb0IsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsK0JBQStCLEVBQy9CLGVBQWUsRUFDZixrQkFBa0IsRUFDbEIsa0JBQWtCLEVBQ2xCLHVDQUF1QyxFQUN2Qyx3QkFBd0IsRUFDeEIsYUFBYSxFQUNiLHFCQUFxQixFQUNyQixXQUFXLEdBQ1osTUFBTSxjQUFjLENBQUM7QUFDdEIsT0FBTyxFQUFFLFNBQVMsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBUTNDLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDeEMsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQzdDLE9BQU8sRUFBRSxNQUFNLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDM0MsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3BELE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUU3QyxNQUFNLEtBQUssR0FBMEM7SUFDbkQsQ0FBQyxVQUFVLENBQUMsTUFBTSxDQUFDLEVBQUUsTUFBTTtJQUMzQixDQUFDLFVBQVUsQ0FBQyxNQUFNLENBQUMsRUFBRSxNQUFNO0lBQzNCLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFLE9BQU87SUFDN0IsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLEVBQUUsT0FBTztDQUM5QixDQUFDO0FBNEJGLE1BQU0sVUFBVSxVQUFVLENBQ3hCLElBQWMsRUFDZCxPQUFzQixFQUFFO0lBRXhCLENBQUMsSUFBSSxDQUFDLEtBQUssSUFBSSxDQUFDLElBQUksQ0FBQyxLQUFLLEdBQUcsRUFBRSxD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