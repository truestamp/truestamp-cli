import { getDefaultValue, getOption, paramCaseToCamelCase } from "./_utils.ts";
import { ArgumentFollowsVariadicArgument, DuplicateOption, InvalidOption, InvalidOptionValue, MissingOptionValue, RequiredArgumentFollowsOptionalArgument, UnknownConflictingOption, UnknownOption, UnknownRequiredOption, UnknownType, } from "./_errors.ts";
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
    args = args.slice();
    !opts.flags && (opts.flags = []);
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
    for (let argsIndex = 0; argsIndex < args.length; argsIndex++) {
        let option;
        let optionArgs;
        let current = args[argsIndex];
        let currentValue;
        if (inLiteral) {
            literal.push(current);
            continue;
        }
        if (current === "--") {
            inLiteral = true;
            continue;
        }
        const isFlag = current.length > 1 && current[0] === "-";
        const next = () => currentValue ?? args[argsIndex + 1];
        if (isFlag) {
            const isShort = current[1] !== "-";
            const isLong = isShort ? false : current.length > 3 && current[2] !== "-";
            if (!isShort && !isLong) {
                throw new InvalidOption(current, opts.flags);
            }
            const equalSignIndex = current.indexOf("=");
            if (equalSignIndex > -1) {
                currentValue = current.slice(equalSignIndex + 1) || undefined;
                current = current.slice(0, equalSignIndex);
            }
            if (isShort && current.length > 2 && current[2] !== ".") {
                args.splice(argsIndex, 1, ...splitFlags(current));
                current = args[argsIndex];
            }
            else if (isLong && current.startsWith("--no-")) {
                negate = true;
            }
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
            const positiveName = negate
                ? option.name.replace(/^no-?/, "")
                : option.name;
            const propName = paramCaseToCamelCase(positiveName);
            if (typeof flags[propName] !== "undefined") {
                if (!opts.flags.length) {
                    option.collect = true;
                }
                else if (!option.collect) {
                    throw new DuplicateOption(current);
                }
            }
            optionArgs = option.args?.length ? option.args : [{
                    type: option.type,
                    requiredValue: option.requiredValue,
                    optionalValue: option.optionalValue,
                    variadic: option.variadic,
                    list: option.list,
                    separator: option.separator,
                }];
            let optionArgsIndex = 0;
            let inOptionalArg = false;
            const previous = flags[propName];
            parseNext(option, optionArgs);
            if (typeof flags[propName] === "undefined") {
                if (optionArgs[optionArgsIndex].requiredValue) {
                    throw new MissingOptionValue(option.name);
                }
                else if (typeof option.default !== "undefined") {
                    flags[propName] = getDefaultValue(option);
                }
                else {
                    flags[propName] = true;
                }
            }
            if (option.value) {
                flags[propName] = option.value(flags[propName], previous);
            }
            else if (option.collect) {
                const value = typeof previous !== "undefined"
                    ? (Array.isArray(previous) ? previous : [previous])
                    : [];
                value.push(flags[propName]);
                flags[propName] = value;
            }
            optionNames[propName] = option.name;
            opts.option?.(option, flags[propName]);
            function parseNext(option, optionArgs) {
                const arg = optionArgs[optionArgsIndex];
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
                if (increase && typeof currentValue === "undefined") {
                    argsIndex++;
                    if (!arg.variadic) {
                        optionArgsIndex++;
                    }
                    else if (optionArgs[optionArgsIndex + 1]) {
                        throw new ArgumentFollowsVariadicArgument(next());
                    }
                }
                if (typeof result !== "undefined" &&
                    (optionArgs.length > 1 || arg.variadic)) {
                    if (!flags[propName]) {
                        flags[propName] = [];
                    }
                    flags[propName].push(result);
                    if (hasNext(arg)) {
                        parseNext(option, optionArgs);
                    }
                }
                else {
                    flags[propName] = result;
                }
                function hasNext(arg) {
                    const nextValue = currentValue ?? args[argsIndex + 1];
                    if (!currentValue && !nextValue) {
                        return false;
                    }
                    if (arg.requiredValue) {
                        return true;
                    }
                    if (arg.optionalValue || arg.variadic) {
                        return nextValue[0] !== "-" ||
                            (arg.type === OptionType.NUMBER && !isNaN(Number(nextValue)));
                    }
                    return false;
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
function splitFlags(flag) {
    const normalized = [];
    const flags = flag.slice(1).split("");
    if (isNaN(Number(flag[flag.length - 1]))) {
        flags.forEach((val) => normalized.push(`-${val}`));
    }
    else {
        normalized.push(`-${flags.shift()}`);
        if (flags.length) {
            normalized.push(flags.join(""));
        }
    }
    return normalized;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmxhZ3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmbGFncy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZUFBZSxFQUFFLFNBQVMsRUFBRSxvQkFBb0IsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsK0JBQStCLEVBQy9CLGVBQWUsRUFDZixhQUFhLEVBQ2Isa0JBQWtCLEVBQ2xCLGtCQUFrQixFQUNsQix1Q0FBdUMsRUFDdkMsd0JBQXdCLEVBQ3hCLGFBQWEsRUFDYixxQkFBcUIsRUFDckIsV0FBVyxHQUNaLE1BQU0sY0FBYyxDQUFDO0FBUXRCLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDeEMsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQzdDLE9BQU8sRUFBRSxNQUFNLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDM0MsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3BELE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUU3QyxNQUFNLEtBQUssR0FBMEM7SUFDbkQsQ0FBQyxVQUFVLENBQUMsTUFBTSxDQUFDLEVBQUUsTUFBTTtJQUMzQixDQUFDLFVBQVUsQ0FBQyxNQUFNLENBQUMsRUFBRSxNQUFNO0lBQzNCLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFLE9BQU87SUFDN0IsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLEVBQUUsT0FBTztDQUM5QixDQUFDO0FBNEJGLE1BQU0sVUFBVSxVQUFVLENBQ3hCLElBQWMsRUFDZCxPQUFzQixFQUFFO0lBRXhCLElBQUksR0FBRyxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7SUFDcEIsQ0FBQyxJQUFJLENBQUMsS0FBSyxJQUFJLENBQUMsSUFBSSxDQUFDLEtBQUssR0FB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