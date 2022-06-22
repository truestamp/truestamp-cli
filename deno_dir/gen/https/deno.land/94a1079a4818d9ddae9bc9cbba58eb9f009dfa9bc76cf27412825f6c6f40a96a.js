import { getDefaultValue, getOption, matchWildCardOptions, paramCaseToCamelCase } from "./_utils.ts";
import { ArgumentFollowsVariadicArgument, DuplicateOption, InvalidOption, InvalidOptionValue, MissingOptionValue, RequiredArgumentFollowsOptionalArgument, UnknownConflictingOption, UnknownOption, UnknownRequiredOption, UnknownType } from "./_errors.ts";
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
    [OptionType.BOOLEAN]: boolean
};
/**
 * Parse command line arguments.
 * @param args  Command line arguments e.g: `Deno.args`
 * @param opts  Parse options.
 * ```
 * // example.ts -x 3 -y.z -n5 -abc --beep=boop foo bar baz --deno.land -- --cliffy
 * parseFlags(Deno.args);
 * ```
 * ```
 * {
 *   flags: {
 *     x: "3",
 *     y: { z: true },
 *     n: "5",
 *     a: true,
 *     b: true,
 *     c: true,
 *     beep: "boop",
 *     deno: { land: true }
 *   },
 *   unknown: [ "foo", "bar", "baz" ],
 *   literal: [ "--cliffy" ]
 * }
 * ```
 */ export function parseFlags(args, opts = {}) {
    args = args.slice();
    !opts.flags && (opts.flags = []);
    let inLiteral = false;
    let negate = false;
    const flags = {};
    /** Option name mapping: propertyName -> option.name */ const optionNameMap = {};
    let literal = [];
    let unknown = [];
    let stopEarly = null;
    opts.flags.forEach((opt)=>{
        opt.depends?.forEach((flag)=>{
            if (!opts.flags || !getOption(opts.flags, flag)) {
                throw new UnknownRequiredOption(flag, opts.flags ?? []);
            }
        });
        opt.conflicts?.forEach((flag)=>{
            if (!opts.flags || !getOption(opts.flags, flag)) {
                throw new UnknownConflictingOption(flag, opts.flags ?? []);
            }
        });
    });
    for(let argsIndex = 0; argsIndex < args.length; argsIndex++){
        let option1;
        let optionArgs1;
        let current = args[argsIndex];
        let currentValue;
        // literal args after --
        if (inLiteral) {
            literal.push(current);
            continue;
        }
        if (current === "--") {
            inLiteral = true;
            continue;
        }
        const isFlag = current.length > 1 && current[0] === "-";
        const next = ()=>currentValue ?? args[argsIndex + 1];
        if (isFlag) {
            const isShort = current[1] !== "-";
            const isLong = isShort ? false : current.length > 3 && current[2] !== "-";
            if (!isShort && !isLong) {
                throw new InvalidOption(current, opts.flags);
            }
            // split value: --foo="bar=baz" => --foo bar=baz
            const equalSignIndex = current.indexOf("=");
            if (equalSignIndex > -1) {
                currentValue = current.slice(equalSignIndex + 1) || undefined;
                current = current.slice(0, equalSignIndex);
            }
            // normalize short flags: -abc => -a -b -c
            if (isShort && current.length > 2 && current[2] !== ".") {
                args.splice(argsIndex, 1, ...splitFlags(current));
                current = args[argsIndex];
            } else if (isLong && current.startsWith("--no-")) {
                negate = true;
            }
            option1 = getOption(opts.flags, current);
            if (!option1) {
                if (opts.flags.length) {
                    const name = current.replace(/^-+/g, "");
                    option1 = matchWildCardOptions(name, opts.flags);
                    if (!option1) {
                        throw new UnknownOption(current, opts.flags);
                    }
                }
                if (!option1) {
                    option1 = {
                        name: current.replace(/^-+/, ""),
                        optionalValue: true,
                        type: OptionType.STRING
                    };
                }
            }
            const positiveName = negate ? option1.name.replace(/^no-?/, "") : option1.name;
            const propName = paramCaseToCamelCase(positiveName);
            if (typeof flags[propName] !== "undefined") {
                if (!opts.flags.length) {
                    option1.collect = true;
                } else if (!option1.collect) {
                    throw new DuplicateOption(current);
                }
            }
            optionArgs1 = option1.args?.length ? option1.args : [
                {
                    type: option1.type,
                    requiredValue: option1.requiredValue,
                    optionalValue: option1.optionalValue,
                    variadic: option1.variadic,
                    list: option1.list,
                    separator: option1.separator
                }
            ];
            let optionArgsIndex = 0;
            let inOptionalArg = false;
            const previous = flags[propName];
            parseNext(option1, optionArgs1);
            if (typeof flags[propName] === "undefined") {
                if (optionArgs1[optionArgsIndex].requiredValue) {
                    throw new MissingOptionValue(option1.name);
                } else if (typeof option1.default !== "undefined") {
                    flags[propName] = getDefaultValue(option1);
                } else {
                    flags[propName] = true;
                }
            }
            if (option1.value) {
                flags[propName] = option1.value(flags[propName], previous);
            } else if (option1.collect) {
                const value = typeof previous !== "undefined" ? Array.isArray(previous) ? previous : [
                    previous
                ] : [];
                value.push(flags[propName]);
                flags[propName] = value;
            }
            optionNameMap[propName] = option1.name;
            opts.option?.(option1, flags[propName]);
            /** Parse next argument for current option. */ // deno-lint-ignore no-inner-declarations
            function parseNext(option3, optionArgs) {
                const arg1 = optionArgs[optionArgsIndex];
                if (!arg1) {
                    const flag = next();
                    throw new UnknownOption(flag, opts.flags ?? []);
                }
                if (!arg1.type) {
                    arg1.type = OptionType.BOOLEAN;
                }
                if (option3.args?.length) {
                    // make all values required by default
                    if ((typeof arg1.optionalValue === "undefined" || arg1.optionalValue === false) && typeof arg1.requiredValue === "undefined") {
                        arg1.requiredValue = true;
                    }
                } else {
                    // make non boolean value required by default
                    if (arg1.type !== OptionType.BOOLEAN && (typeof arg1.optionalValue === "undefined" || arg1.optionalValue === false) && typeof arg1.requiredValue === "undefined") {
                        arg1.requiredValue = true;
                    }
                }
                if (arg1.requiredValue) {
                    if (inOptionalArg) {
                        throw new RequiredArgumentFollowsOptionalArgument(option3.name);
                    }
                } else {
                    inOptionalArg = true;
                }
                if (negate) {
                    flags[propName] = false;
                    return;
                }
                let result2;
                let increase = false;
                if (arg1.list && hasNext(arg1)) {
                    const parsed = next().split(arg1.separator || ",").map((nextValue)=>{
                        const value = parseValue(option3, arg1, nextValue);
                        if (typeof value === "undefined") {
                            throw new InvalidOptionValue(option3.name, arg1.type ?? "?", nextValue);
                        }
                        return value;
                    });
                    if (parsed?.length) {
                        result2 = parsed;
                    }
                } else {
                    if (hasNext(arg1)) {
                        result2 = parseValue(option3, arg1, next());
                    } else if (arg1.optionalValue && arg1.type === OptionType.BOOLEAN) {
                        result2 = true;
                    }
                }
                if (increase && typeof currentValue === "undefined") {
                    argsIndex++;
                    if (!arg1.variadic) {
                        optionArgsIndex++;
                    } else if (optionArgs[optionArgsIndex + 1]) {
                        throw new ArgumentFollowsVariadicArgument(next());
                    }
                }
                if (typeof result2 !== "undefined" && (optionArgs.length > 1 || arg1.variadic)) {
                    if (!flags[propName]) {
                        flags[propName] = [];
                    }
                    flags[propName].push(result2);
                    if (hasNext(arg1)) {
                        parseNext(option3, optionArgs);
                    }
                } else {
                    flags[propName] = result2;
                }
                /** Check if current option should have an argument. */ function hasNext(arg) {
                    const nextValue = currentValue ?? args[argsIndex + 1];
                    if (!nextValue) {
                        return false;
                    }
                    if (optionArgs.length > 1 && optionArgsIndex >= optionArgs.length) {
                        return false;
                    }
                    if (arg.requiredValue) {
                        return true;
                    }
                    // require optional values to be called with an equal sign: foo=bar
                    if (option3.equalsSign && arg.optionalValue && !arg.variadic && typeof currentValue === "undefined") {
                        return false;
                    }
                    if (arg.optionalValue || arg.variadic) {
                        return nextValue[0] !== "-" || arg.type === OptionType.NUMBER && !isNaN(Number(nextValue));
                    }
                    return false;
                }
                /** Parse argument value.  */ function parseValue(option, arg, value) {
                    const type = arg.type || OptionType.STRING;
                    const result = opts.parse ? opts.parse({
                        label: "Option",
                        type,
                        name: `--${option.name}`,
                        value
                    }) : parseFlagValue(option, arg, value);
                    if (typeof result !== "undefined") {
                        increase = true;
                    }
                    return result;
                }
            }
        } else {
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
    validateFlags(opts, flags, optionNameMap);
    // convert dotted option keys into nested objects
    const result1 = Object.keys(flags).reduce((result3, key)=>{
        if (~key.indexOf(".")) {
            key.split(".").reduce((// deno-lint-ignore no-explicit-any
            result, subKey, index, parts)=>{
                if (index === parts.length - 1) {
                    result[subKey] = flags[key];
                } else {
                    result[subKey] = result[subKey] ?? {};
                }
                return result[subKey];
            }, result3);
        } else {
            result3[key] = flags[key];
        }
        return result3;
    }, {});
    return {
        flags: result1,
        unknown,
        literal
    };
}
function splitFlags(flag) {
    const normalized = [];
    const flags = flag.slice(1).split("");
    if (isNaN(Number(flag[flag.length - 1]))) {
        flags.forEach((val)=>normalized.push(`-${val}`));
    } else {
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
        value
    });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvZmxhZ3MvZmxhZ3MudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHtcbiAgZ2V0RGVmYXVsdFZhbHVlLFxuICBnZXRPcHRpb24sXG4gIG1hdGNoV2lsZENhcmRPcHRpb25zLFxuICBwYXJhbUNhc2VUb0NhbWVsQ2FzZSxcbn0gZnJvbSBcIi4vX3V0aWxzLnRzXCI7XG5pbXBvcnQge1xuICBBcmd1bWVudEZvbGxvd3NWYXJpYWRpY0FyZ3VtZW50LFxuICBEdXBsaWNhdGVPcHRpb24sXG4gIEludmFsaWRPcHRpb24sXG4gIEludmFsaWRPcHRpb25WYWx1ZSxcbiAgTWlzc2luZ09wdGlvblZhbHVlLFxuICBSZXF1aXJlZEFyZ3VtZW50Rm9sbG93c09wdGlvbmFsQXJndW1lbnQsXG4gIFVua25vd25Db25mbGljdGluZ09wdGlvbixcbiAgVW5rbm93bk9wdGlvbixcbiAgVW5rbm93blJlcXVpcmVkT3B0aW9uLFxuICBVbmtub3duVHlwZSxcbn0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHR5cGUge1xuICBJRmxhZ0FyZ3VtZW50LFxuICBJRmxhZ09wdGlvbnMsXG4gIElGbGFnc1Jlc3VsdCxcbiAgSVBhcnNlT3B0aW9ucyxcbiAgSVR5cGVIYW5kbGVyLFxufSBmcm9tIFwiLi90eXBlcy50c1wiO1xuaW1wb3J0IHsgT3B0aW9uVHlwZSB9IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBib29sZWFuIH0gZnJvbSBcIi4vdHlwZXMvYm9vbGVhbi50c1wiO1xuaW1wb3J0IHsgbnVtYmVyIH0gZnJvbSBcIi4vdHlwZXMvbnVtYmVyLnRzXCI7XG5pbXBvcnQgeyBzdHJpbmcgfSBmcm9tIFwiLi90eXBlcy9zdHJpbmcudHNcIjtcbmltcG9ydCB7IHZhbGlkYXRlRmxhZ3MgfSBmcm9tIFwiLi92YWxpZGF0ZV9mbGFncy50c1wiO1xuaW1wb3J0IHsgaW50ZWdlciB9IGZyb20gXCIuL3R5cGVzL2ludGVnZXIudHNcIjtcblxuY29uc3QgVHlwZXM6IFJlY29yZDxzdHJpbmcsIElUeXBlSGFuZGxlcjx1bmtub3duPj4gPSB7XG4gIFtPcHRpb25UeXBlLlNUUklOR106IHN0cmluZyxcbiAgW09wdGlvblR5cGUuTlVNQkVSXTogbnVtYmVyLFxuICBbT3B0aW9uVHlwZS5JTlRFR0VSXTogaW50ZWdlcixcbiAgW09wdGlvblR5cGUuQk9PTEVBTl06IGJvb2xlYW4sXG59O1xuXG4vKipcbiAqIFBhcnNlIGNvbW1hbmQgbGluZSBhcmd1bWVudHMuXG4gKiBAcGFyYW0gYXJncyAgQ29tbWFuZCBsaW5lIGFyZ3VtZW50cyBlLmc6IGBEZW5vLmFyZ3NgXG4gKiBAcGFyYW0gb3B0cyAgUGFyc2Ugb3B0aW9ucy5cbiAqIGBgYFxuICogLy8gZXhhbXBsZS50cyAteCAzIC15LnogLW41IC1hYmMgLS1iZWVwPWJvb3AgZm9vIGJhciBiYXogLS1kZW5vLmxhbmQgLS0gLS1jbGlmZnlcbiAqIHBhcnNlRmxhZ3MoRGVuby5hcmdzKTtcbiAqIGBgYFxuICogYGBgXG4gKiB7XG4gKiAgIGZsYWdzOiB7XG4gKiAgICAgeDogXCIzXCIsXG4gKiAgICAgeTogeyB6OiB0cnVlIH0sXG4gKiAgICAgbjogXCI1XCIsXG4gKiAgICAgYTogdHJ1ZSxcbiAqICAgICBiOiB0cnVlLFxuICogICAgIGM6IHRydWUsXG4gKiAgICAgYmVlcDogXCJib29wXCIsXG4gKiAgICAgZGVubzogeyBsYW5kOiB0cnVlIH1cbiAqICAgfSxcbiAqICAgdW5rbm93bjogWyBcImZvb1wiLCBcImJhclwiLCBcImJhelwiIF0sXG4gKiAgIGxpdGVyYWw6IFsgXCItLWNsaWZmeVwiIF1cbiAqIH1cbiAqIGBgYFxuICovXG5leHBvcnQgZnVuY3Rpb24gcGFyc2VGbGFnczxcbiAgLy8gZGVuby1saW50LWlnbm9yZSBuby1leHBsaWNpdC1hbnlcbiAgTyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gPSBSZWNvcmQ8c3RyaW5nLCBhbnk+LFxuICBUIGV4dGVuZHMgSUZsYWdPcHRpb25zID0gSUZsYWdPcHRpb25zLFxuPihcbiAgYXJnczogc3RyaW5nW10sXG4gIG9wdHM6IElQYXJzZU9wdGlvbnM8VD4gPSB7fSxcbik6IElGbGFnc1Jlc3VsdDxPPiB7XG4gIGFyZ3MgPSBhcmdzLnNsaWNlKCk7XG4gICFvcHRzLmZsYWdzICYmIChvcHRzLmZsYWdzID0gW10pO1xuXG4gIGxldCBpbkxpdGVyYWwgPSBmYWxzZTtcbiAgbGV0IG5lZ2F0ZSA9IGZhbHNlO1xuXG4gIGNvbnN0IGZsYWdzOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiA9IHt9O1xuICAvKiogT3B0aW9uIG5hbWUgbWFwcGluZzogcHJvcGVydHlOYW1lIC0+IG9wdGlvbi5uYW1lICovXG4gIGNvbnN0IG9wdGlvbk5hbWVNYXA6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4gPSB7fTtcbiAgbGV0IGxpdGVyYWw6IHN0cmluZ1tdID0gW107XG4gIGxldCB1bmtub3duOiBzdHJpbmdbXSA9IFtdO1xuICBsZXQgc3RvcEVhcmx5OiBzdHJpbmcgfCBudWxsID0gbnVsbDtcblxuICBvcHRzLmZsYWdzLmZvckVhY2goKG9wdCkgPT4ge1xuICAgIG9wdC5kZXBlbmRzPy5mb3JFYWNoKChmbGFnKSA9PiB7XG4gICAgICBpZiAoIW9wdHMuZmxhZ3MgfHwgIWdldE9wdGlvbihvcHRzLmZsYWdzLCBmbGFnKSkge1xuICAgICAgICB0aHJvdyBuZXcgVW5rbm93blJlcXVpcmVkT3B0aW9uKGZsYWcsIG9wdHMuZmxhZ3MgPz8gW10pO1xuICAgICAgfVxuICAgIH0pO1xuICAgIG9wdC5jb25mbGljdHM/LmZvckVhY2goKGZsYWcpID0+IHtcbiAgICAgIGlmICghb3B0cy5mbGFncyB8fCAhZ2V0T3B0aW9uKG9wdHMuZmxhZ3MsIGZsYWcpKSB7XG4gICAgICAgIHRocm93IG5ldyBVbmtub3duQ29uZmxpY3RpbmdPcHRpb24oZmxhZywgb3B0cy5mbGFncyA/PyBbXSk7XG4gICAgICB9XG4gICAgfSk7XG4gIH0pO1xuXG4gIGZvciAoXG4gICAgbGV0IGFyZ3NJbmRleCA9IDA7XG4gICAgYXJnc0luZGV4IDwgYXJncy5sZW5ndGg7XG4gICAgYXJnc0luZGV4KytcbiAgKSB7XG4gICAgbGV0IG9wdGlvbjogSUZsYWdPcHRpb25zIHwgdW5kZWZpbmVkO1xuICAgIGxldCBvcHRpb25BcmdzOiBJRmxhZ0FyZ3VtZW50W10gfCB1bmRlZmluZWQ7XG4gICAgbGV0IGN1cnJlbnQ6IHN0cmluZyA9IGFyZ3NbYXJnc0luZGV4XTtcbiAgICBsZXQgY3VycmVudFZhbHVlOiBzdHJpbmcgfCB1bmRlZmluZWQ7XG5cbiAgICAvLyBsaXRlcmFsIGFyZ3MgYWZ0ZXIgLS1cbiAgICBpZiAoaW5MaXRlcmFsKSB7XG4gICAgICBsaXRlcmFsLnB1c2goY3VycmVudCk7XG4gICAgICBjb250aW51ZTtcbiAgICB9XG5cbiAgICBpZiAoY3VycmVudCA9PT0gXCItLVwiKSB7XG4gICAgICBpbkxpdGVyYWwgPSB0cnVlO1xuICAgICAgY29udGludWU7XG4gICAgfVxuXG4gICAgY29uc3QgaXNGbGFnID0gY3VycmVudC5sZW5ndGggPiAxICYmIGN1cnJlbnRbMF0gPT09IFwiLVwiO1xuICAgIGNvbnN0IG5leHQgPSAoKSA9PiBjdXJyZW50VmFsdWUgPz8gYXJnc1thcmdzSW5kZXggKyAxXTtcblxuICAgIGlmIChpc0ZsYWcpIHtcbiAgICAgIGNvbnN0IGlzU2hvcnQgPSBjdXJyZW50WzFdICE9PSBcIi1cIjtcbiAgICAgIGNvbnN0IGlzTG9uZyA9IGlzU2hvcnQgPyBmYWxzZSA6IGN1cnJlbnQubGVuZ3RoID4gMyAmJiBjdXJyZW50WzJdICE9PSBcIi1cIjtcblxuICAgICAgaWYgKCFpc1Nob3J0ICYmICFpc0xvbmcpIHtcbiAgICAgICAgdGhyb3cgbmV3IEludmFsaWRPcHRpb24oY3VycmVudCwgb3B0cy5mbGFncyk7XG4gICAgICB9XG5cbiAgICAgIC8vIHNwbGl0IHZhbHVlOiAtLWZvbz1cImJhcj1iYXpcIiA9PiAtLWZvbyBiYXI9YmF6XG4gICAgICBjb25zdCBlcXVhbFNpZ25JbmRleCA9IGN1cnJlbnQuaW5kZXhPZihcIj1cIik7XG4gICAgICBpZiAoZXF1YWxTaWduSW5kZXggPiAtMSkge1xuICAgICAgICBjdXJyZW50VmFsdWUgPSBjdXJyZW50LnNsaWNlKGVxdWFsU2lnbkluZGV4ICsgMSkgfHwgdW5kZWZpbmVkO1xuICAgICAgICBjdXJyZW50ID0gY3VycmVudC5zbGljZSgwLCBlcXVhbFNpZ25JbmRleCk7XG4gICAgICB9XG5cbiAgICAgIC8vIG5vcm1hbGl6ZSBzaG9ydCBmbGFnczogLWFiYyA9PiAtYSAtYiAtY1xuICAgICAgaWYgKGlzU2hvcnQgJiYgY3VycmVudC5sZW5ndGggPiAyICYmIGN1cnJlbnRbMl0gIT09IFwiLlwiKSB7XG4gICAgICAgIGFyZ3Muc3BsaWNlKGFyZ3NJbmRleCwgMSwgLi4uc3BsaXRGbGFncyhjdXJyZW50KSk7XG4gICAgICAgIGN1cnJlbnQgPSBhcmdzW2FyZ3NJbmRleF07XG4gICAgICB9IGVsc2UgaWYgKGlzTG9uZyAmJiBjdXJyZW50LnN0YXJ0c1dpdGgoXCItLW5vLVwiKSkge1xuICAgICAgICBuZWdhdGUgPSB0cnVlO1xuICAgICAgfVxuICAgICAgb3B0aW9uID0gZ2V0T3B0aW9uKG9wdHMuZmxhZ3MsIGN1cnJlbnQpO1xuXG4gICAgICBpZiAoIW9wdGlvbikge1xuICAgICAgICBpZiAob3B0cy5mbGFncy5sZW5ndGgpIHtcbiAgICAgICAgICBjb25zdCBuYW1lID0gY3VycmVudC5yZXBsYWNlKC9eLSsvZywgXCJcIik7XG4gICAgICAgICAgb3B0aW9uID0gbWF0Y2hXaWxkQ2FyZE9wdGlvbnMobmFtZSwgb3B0cy5mbGFncyk7XG4gICAgICAgICAgaWYgKCFvcHRpb24pIHtcbiAgICAgICAgICAgIHRocm93IG5ldyBVbmtub3duT3B0aW9uKGN1cnJlbnQsIG9wdHMuZmxhZ3MpO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuICAgICAgICBpZiAoIW9wdGlvbikge1xuICAgICAgICAgIG9wdGlvbiA9IHtcbiAgICAgICAgICAgIG5hbWU6IGN1cnJlbnQucmVwbGFjZSgvXi0rLywgXCJcIiksXG4gICAgICAgICAgICBvcHRpb25hbFZhbHVlOiB0cnVlLFxuICAgICAgICAgICAgdHlwZTogT3B0aW9uVHlwZS5TVFJJTkcsXG4gICAgICAgICAgfTtcbiAgICAgICAgfVxuICAgICAgfVxuXG4gICAgICBjb25zdCBwb3NpdGl2ZU5hbWU6IHN0cmluZyA9IG5lZ2F0ZVxuICAgICAgICA/IG9wdGlvbi5uYW1lLnJlcGxhY2UoL15uby0/LywgXCJcIilcbiAgICAgICAgOiBvcHRpb24ubmFtZTtcbiAgICAgIGNvbnN0IHByb3BOYW1lOiBzdHJpbmcgPSBwYXJhbUNhc2VUb0NhbWVsQ2FzZShwb3NpdGl2ZU5hbWUpO1xuXG4gICAgICBpZiAodHlwZW9mIGZsYWdzW3Byb3BOYW1lXSAhPT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICBpZiAoIW9wdHMuZmxhZ3MubGVuZ3RoKSB7XG4gICAgICAgICAgb3B0aW9uLmNvbGxlY3QgPSB0cnVlO1xuICAgICAgICB9IGVsc2UgaWYgKCFvcHRpb24uY29sbGVjdCkge1xuICAgICAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVPcHRpb24oY3VycmVudCk7XG4gICAgICAgIH1cbiAgICAgIH1cblxuICAgICAgb3B0aW9uQXJncyA9IG9wdGlvbi5hcmdzPy5sZW5ndGggPyBvcHRpb24uYXJncyA6IFt7XG4gICAgICAgIHR5cGU6IG9wdGlvbi50eXBlLFxuICAgICAgICByZXF1aXJlZFZhbHVlOiBvcHRpb24ucmVxdWlyZWRWYWx1ZSxcbiAgICAgICAgb3B0aW9uYWxWYWx1ZTogb3B0aW9uLm9wdGlvbmFsVmFsdWUsXG4gICAgICAgIHZhcmlhZGljOiBvcHRpb24udmFyaWFkaWMsXG4gICAgICAgIGxpc3Q6IG9wdGlvbi5saXN0LFxuICAgICAgICBzZXBhcmF0b3I6IG9wdGlvbi5zZXBhcmF0b3IsXG4gICAgICB9XTtcblxuICAgICAgbGV0IG9wdGlvbkFyZ3NJbmRleCA9IDA7XG4gICAgICBsZXQgaW5PcHRpb25hbEFyZyA9IGZhbHNlO1xuICAgICAgY29uc3QgcHJldmlvdXMgPSBmbGFnc1twcm9wTmFtZV07XG5cbiAgICAgIHBhcnNlTmV4dChvcHRpb24sIG9wdGlvbkFyZ3MpO1xuXG4gICAgICBpZiAodHlwZW9mIGZsYWdzW3Byb3BOYW1lXSA9PT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICBpZiAob3B0aW9uQXJnc1tvcHRpb25BcmdzSW5kZXhdLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgTWlzc2luZ09wdGlvblZhbHVlKG9wdGlvbi5uYW1lKTtcbiAgICAgICAgfSBlbHNlIGlmICh0eXBlb2Ygb3B0aW9uLmRlZmF1bHQgIT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICBmbGFnc1twcm9wTmFtZV0gPSBnZXREZWZhdWx0VmFsdWUob3B0aW9uKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBmbGFnc1twcm9wTmFtZV0gPSB0cnVlO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGlmIChvcHRpb24udmFsdWUpIHtcbiAgICAgICAgZmxhZ3NbcHJvcE5hbWVdID0gb3B0aW9uLnZhbHVlKGZsYWdzW3Byb3BOYW1lXSwgcHJldmlvdXMpO1xuICAgICAgfSBlbHNlIGlmIChvcHRpb24uY29sbGVjdCkge1xuICAgICAgICBjb25zdCB2YWx1ZTogdW5rbm93bltdID0gdHlwZW9mIHByZXZpb3VzICE9PSBcInVuZGVmaW5lZFwiXG4gICAgICAgICAgPyAoQXJyYXkuaXNBcnJheShwcmV2aW91cykgPyBwcmV2aW91cyA6IFtwcmV2aW91c10pXG4gICAgICAgICAgOiBbXTtcblxuICAgICAgICB2YWx1ZS5wdXNoKGZsYWdzW3Byb3BOYW1lXSk7XG4gICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IHZhbHVlO1xuICAgICAgfVxuXG4gICAgICBvcHRpb25OYW1lTWFwW3Byb3BOYW1lXSA9IG9wdGlvbi5uYW1lO1xuXG4gICAgICBvcHRzLm9wdGlvbj8uKG9wdGlvbiBhcyBULCBmbGFnc1twcm9wTmFtZV0pO1xuXG4gICAgICAvKiogUGFyc2UgbmV4dCBhcmd1bWVudCBmb3IgY3VycmVudCBvcHRpb24uICovXG4gICAgICAvLyBkZW5vLWxpbnQtaWdub3JlIG5vLWlubmVyLWRlY2xhcmF0aW9uc1xuICAgICAgZnVuY3Rpb24gcGFyc2VOZXh0KFxuICAgICAgICBvcHRpb246IElGbGFnT3B0aW9ucyxcbiAgICAgICAgb3B0aW9uQXJnczogSUZsYWdBcmd1bWVudFtdLFxuICAgICAgKTogdm9pZCB7XG4gICAgICAgIGNvbnN0IGFyZzogSUZsYWdBcmd1bWVudCB8IHVuZGVmaW5lZCA9IG9wdGlvbkFyZ3Nbb3B0aW9uQXJnc0luZGV4XTtcblxuICAgICAgICBpZiAoIWFyZykge1xuICAgICAgICAgIGNvbnN0IGZsYWcgPSBuZXh0KCk7XG4gICAgICAgICAgdGhyb3cgbmV3IFVua25vd25PcHRpb24oZmxhZywgb3B0cy5mbGFncyA/PyBbXSk7XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoIWFyZy50eXBlKSB7XG4gICAgICAgICAgYXJnLnR5cGUgPSBPcHRpb25UeXBlLkJPT0xFQU47XG4gICAgICAgIH1cblxuICAgICAgICBpZiAob3B0aW9uLmFyZ3M/Lmxlbmd0aCkge1xuICAgICAgICAgIC8vIG1ha2UgYWxsIHZhbHVlcyByZXF1aXJlZCBieSBkZWZhdWx0XG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgKHR5cGVvZiBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICAgICAgICBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gZmFsc2UpICYmXG4gICAgICAgICAgICB0eXBlb2YgYXJnLnJlcXVpcmVkVmFsdWUgPT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGFyZy5yZXF1aXJlZFZhbHVlID0gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgLy8gbWFrZSBub24gYm9vbGVhbiB2YWx1ZSByZXF1aXJlZCBieSBkZWZhdWx0XG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgYXJnLnR5cGUgIT09IE9wdGlvblR5cGUuQk9PTEVBTiAmJlxuICAgICAgICAgICAgKHR5cGVvZiBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICAgICAgICBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gZmFsc2UpICYmXG4gICAgICAgICAgICB0eXBlb2YgYXJnLnJlcXVpcmVkVmFsdWUgPT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGFyZy5yZXF1aXJlZFZhbHVlID0gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoYXJnLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICBpZiAoaW5PcHRpb25hbEFyZykge1xuICAgICAgICAgICAgdGhyb3cgbmV3IFJlcXVpcmVkQXJndW1lbnRGb2xsb3dzT3B0aW9uYWxBcmd1bWVudChvcHRpb24ubmFtZSk7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGluT3B0aW9uYWxBcmcgPSB0cnVlO1xuICAgICAgICB9XG5cbiAgICAgICAgaWYgKG5lZ2F0ZSkge1xuICAgICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IGZhbHNlO1xuICAgICAgICAgIHJldHVybjtcbiAgICAgICAgfVxuXG4gICAgICAgIGxldCByZXN1bHQ6IHVua25vd247XG4gICAgICAgIGxldCBpbmNyZWFzZSA9IGZhbHNlO1xuXG4gICAgICAgIGlmIChhcmcubGlzdCAmJiBoYXNOZXh0KGFyZykpIHtcbiAgICAgICAgICBjb25zdCBwYXJzZWQ6IHVua25vd25bXSA9IG5leHQoKVxuICAgICAgICAgICAgLnNwbGl0KGFyZy5zZXBhcmF0b3IgfHwgXCIsXCIpXG4gICAgICAgICAgICAubWFwKChuZXh0VmFsdWU6IHN0cmluZykgPT4ge1xuICAgICAgICAgICAgICBjb25zdCB2YWx1ZSA9IHBhcnNlVmFsdWUob3B0aW9uLCBhcmcsIG5leHRWYWx1ZSk7XG4gICAgICAgICAgICAgIGlmICh0eXBlb2YgdmFsdWUgPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICAgICAgICB0aHJvdyBuZXcgSW52YWxpZE9wdGlvblZhbHVlKFxuICAgICAgICAgICAgICAgICAgb3B0aW9uLm5hbWUsXG4gICAgICAgICAgICAgICAgICBhcmcudHlwZSA/PyBcIj9cIixcbiAgICAgICAgICAgICAgICAgIG5leHRWYWx1ZSxcbiAgICAgICAgICAgICAgICApO1xuICAgICAgICAgICAgICB9XG4gICAgICAgICAgICAgIHJldHVybiB2YWx1ZTtcbiAgICAgICAgICAgIH0pO1xuXG4gICAgICAgICAgaWYgKHBhcnNlZD8ubGVuZ3RoKSB7XG4gICAgICAgICAgICByZXN1bHQgPSBwYXJzZWQ7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGlmIChoYXNOZXh0KGFyZykpIHtcbiAgICAgICAgICAgIHJlc3VsdCA9IHBhcnNlVmFsdWUob3B0aW9uLCBhcmcsIG5leHQoKSk7XG4gICAgICAgICAgfSBlbHNlIGlmIChhcmcub3B0aW9uYWxWYWx1ZSAmJiBhcmcudHlwZSA9PT0gT3B0aW9uVHlwZS5CT09MRUFOKSB7XG4gICAgICAgICAgICByZXN1bHQgPSB0cnVlO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuXG4gICAgICAgIGlmIChpbmNyZWFzZSAmJiB0eXBlb2YgY3VycmVudFZhbHVlID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICAgICAgYXJnc0luZGV4Kys7XG4gICAgICAgICAgaWYgKCFhcmcudmFyaWFkaWMpIHtcbiAgICAgICAgICAgIG9wdGlvbkFyZ3NJbmRleCsrO1xuICAgICAgICAgIH0gZWxzZSBpZiAob3B0aW9uQXJnc1tvcHRpb25BcmdzSW5kZXggKyAxXSkge1xuICAgICAgICAgICAgdGhyb3cgbmV3IEFyZ3VtZW50Rm9sbG93c1ZhcmlhZGljQXJndW1lbnQobmV4dCgpKTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoXG4gICAgICAgICAgdHlwZW9mIHJlc3VsdCAhPT0gXCJ1bmRlZmluZWRcIiAmJlxuICAgICAgICAgIChvcHRpb25BcmdzLmxlbmd0aCA+IDEgfHwgYXJnLnZhcmlhZGljKVxuICAgICAgICApIHtcbiAgICAgICAgICBpZiAoIWZsYWdzW3Byb3BOYW1lXSkge1xuICAgICAgICAgICAgZmxhZ3NbcHJvcE5hbWVdID0gW107XG4gICAgICAgICAgfVxuXG4gICAgICAgICAgKGZsYWdzW3Byb3BOYW1lXSBhcyBBcnJheTx1bmtub3duPikucHVzaChyZXN1bHQpO1xuXG4gICAgICAgICAgaWYgKGhhc05leHQoYXJnKSkge1xuICAgICAgICAgICAgcGFyc2VOZXh0KG9wdGlvbiwgb3B0aW9uQXJncyk7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IHJlc3VsdDtcbiAgICAgICAgfVxuXG4gICAgICAgIC8qKiBDaGVjayBpZiBjdXJyZW50IG9wdGlvbiBzaG91bGQgaGF2ZSBhbiBhcmd1bWVudC4gKi9cbiAgICAgICAgZnVuY3Rpb24gaGFzTmV4dChhcmc6IElGbGFnQXJndW1lbnQpOiBib29sZWFuIHtcbiAgICAgICAgICBjb25zdCBuZXh0VmFsdWUgPSBjdXJyZW50VmFsdWUgPz8gYXJnc1thcmdzSW5kZXggKyAxXTtcbiAgICAgICAgICBpZiAoIW5leHRWYWx1ZSkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAob3B0aW9uQXJncy5sZW5ndGggPiAxICYmIG9wdGlvbkFyZ3NJbmRleCA+PSBvcHRpb25BcmdzLmxlbmd0aCkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoYXJnLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICAgIHJldHVybiB0cnVlO1xuICAgICAgICAgIH1cbiAgICAgICAgICAvLyByZXF1aXJlIG9wdGlvbmFsIHZhbHVlcyB0byBiZSBjYWxsZWQgd2l0aCBhbiBlcXVhbCBzaWduOiBmb289YmFyXG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgb3B0aW9uLmVxdWFsc1NpZ24gJiYgYXJnLm9wdGlvbmFsVmFsdWUgJiYgIWFyZy52YXJpYWRpYyAmJlxuICAgICAgICAgICAgdHlwZW9mIGN1cnJlbnRWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIlxuICAgICAgICAgICkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoYXJnLm9wdGlvbmFsVmFsdWUgfHwgYXJnLnZhcmlhZGljKSB7XG4gICAgICAgICAgICByZXR1cm4gbmV4dFZhbHVlWzBdICE9PSBcIi1cIiB8fFxuICAgICAgICAgICAgICAoYXJnLnR5cGUgPT09IE9wdGlvblR5cGUuTlVNQkVSICYmICFpc05hTihOdW1iZXIobmV4dFZhbHVlKSkpO1xuICAgICAgICAgIH1cblxuICAgICAgICAgIHJldHVybiBmYWxzZTtcbiAgICAgICAgfVxuXG4gICAgICAgIC8qKiBQYXJzZSBhcmd1bWVudCB2YWx1ZS4gICovXG4gICAgICAgIGZ1bmN0aW9uIHBhcnNlVmFsdWUoXG4gICAgICAgICAgb3B0aW9uOiBJRmxhZ09wdGlvbnMsXG4gICAgICAgICAgYXJnOiBJRmxhZ0FyZ3VtZW50LFxuICAgICAgICAgIHZhbHVlOiBzdHJpbmcsXG4gICAgICAgICk6IHVua25vd24ge1xuICAgICAgICAgIGNvbnN0IHR5cGU6IHN0cmluZyA9IGFyZy50eXBlIHx8IE9wdGlvblR5cGUuU1RSSU5HO1xuICAgICAgICAgIGNvbnN0IHJlc3VsdDogdW5rbm93biA9IG9wdHMucGFyc2VcbiAgICAgICAgICAgID8gb3B0cy5wYXJzZSh7XG4gICAgICAgICAgICAgIGxhYmVsOiBcIk9wdGlvblwiLFxuICAgICAgICAgICAgICB0eXBlLFxuICAgICAgICAgICAgICBuYW1lOiBgLS0ke29wdGlvbi5uYW1lfWAsXG4gICAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgICAgfSlcbiAgICAgICAgICAgIDogcGFyc2VGbGFnVmFsdWUob3B0aW9uLCBhcmcsIHZhbHVlKTtcblxuICAgICAgICAgIGlmIChcbiAgICAgICAgICAgIHR5cGVvZiByZXN1bHQgIT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGluY3JlYXNlID0gdHJ1ZTtcbiAgICAgICAgICB9XG5cbiAgICAgICAgICByZXR1cm4gcmVzdWx0O1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfSBlbHNlIHtcbiAgICAgIGlmIChvcHRzLnN0b3BFYXJseSkge1xuICAgICAgICBzdG9wRWFybHkgPSBjdXJyZW50O1xuICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICAgIHVua25vd24ucHVzaChjdXJyZW50KTtcbiAgICB9XG4gIH1cblxuICBpZiAoc3RvcEVhcmx5KSB7XG4gICAgY29uc3Qgc3RvcEVhcmx5QXJnSW5kZXg6IG51bWJlciA9IGFyZ3MuaW5kZXhPZihzdG9wRWFybHkpO1xuICAgIGlmIChzdG9wRWFybHlBcmdJbmRleCAhPT0gLTEpIHtcbiAgICAgIGNvbnN0IGRvdWJsZURhc2hJbmRleDogbnVtYmVyID0gYXJncy5pbmRleE9mKFwiLS1cIik7XG4gICAgICB1bmtub3duID0gYXJncy5zbGljZShcbiAgICAgICAgc3RvcEVhcmx5QXJnSW5kZXgsXG4gICAgICAgIGRvdWJsZURhc2hJbmRleCA9PT0gLTEgPyB1bmRlZmluZWQgOiBkb3VibGVEYXNoSW5kZXgsXG4gICAgICApO1xuICAgICAgaWYgKGRvdWJsZURhc2hJbmRleCAhPT0gLTEpIHtcbiAgICAgICAgbGl0ZXJhbCA9IGFyZ3Muc2xpY2UoZG91YmxlRGFzaEluZGV4ICsgMSk7XG4gICAgICB9XG4gICAgfVxuICB9XG5cbiAgdmFsaWRhdGVGbGFncyhvcHRzLCBmbGFncywgb3B0aW9uTmFtZU1hcCk7XG5cbiAgLy8gY29udmVydCBkb3R0ZWQgb3B0aW9uIGtleXMgaW50byBuZXN0ZWQgb2JqZWN0c1xuICBjb25zdCByZXN1bHQgPSBPYmplY3Qua2V5cyhmbGFncylcbiAgICAucmVkdWNlKChyZXN1bHQ6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LCBrZXk6IHN0cmluZykgPT4ge1xuICAgICAgaWYgKH5rZXkuaW5kZXhPZihcIi5cIikpIHtcbiAgICAgICAga2V5LnNwbGl0KFwiLlwiKS5yZWR1Y2UoXG4gICAgICAgICAgKFxuICAgICAgICAgICAgLy8gZGVuby1saW50LWlnbm9yZSBuby1leHBsaWNpdC1hbnlcbiAgICAgICAgICAgIHJlc3VsdDogUmVjb3JkPHN0cmluZywgYW55PixcbiAgICAgICAgICAgIHN1YktleTogc3RyaW5nLFxuICAgICAgICAgICAgaW5kZXg6IG51bWJlcixcbiAgICAgICAgICAgIHBhcnRzOiBzdHJpbmdbXSxcbiAgICAgICAgICApID0+IHtcbiAgICAgICAgICAgIGlmIChpbmRleCA9PT0gcGFydHMubGVuZ3RoIC0gMSkge1xuICAgICAgICAgICAgICByZXN1bHRbc3ViS2V5XSA9IGZsYWdzW2tleV07XG4gICAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgICByZXN1bHRbc3ViS2V5XSA9IHJlc3VsdFtzdWJLZXldID8/IHt9O1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgcmV0dXJuIHJlc3VsdFtzdWJLZXldO1xuICAgICAgICAgIH0sXG4gICAgICAgICAgcmVzdWx0LFxuICAgICAgICApO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0W2tleV0gPSBmbGFnc1trZXldO1xuICAgICAgfVxuICAgICAgcmV0dXJuIHJlc3VsdDtcbiAgICB9LCB7fSk7XG5cbiAgcmV0dXJuIHsgZmxhZ3M6IHJlc3VsdCBhcyBPLCB1bmtub3duLCBsaXRlcmFsIH07XG59XG5cbmZ1bmN0aW9uIHNwbGl0RmxhZ3MoZmxhZzogc3RyaW5nKTogQXJyYXk8c3RyaW5nPiB7XG4gIGNvbnN0IG5vcm1hbGl6ZWQ6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgY29uc3QgZmxhZ3MgPSBmbGFnLnNsaWNlKDEpLnNwbGl0KFwiXCIpO1xuXG4gIGlmIChpc05hTihOdW1iZXIoZmxhZ1tmbGFnLmxlbmd0aCAtIDFdKSkpIHtcbiAgICBmbGFncy5mb3JFYWNoKCh2YWwpID0+IG5vcm1hbGl6ZWQucHVzaChgLSR7dmFsfWApKTtcbiAgfSBlbHNlIHtcbiAgICBub3JtYWxpemVkLnB1c2goYC0ke2ZsYWdzLnNoaWZ0KCl9YCk7XG4gICAgaWYgKGZsYWdzLmxlbmd0aCkge1xuICAgICAgbm9ybWFsaXplZC5wdXNoKGZsYWdzLmpvaW4oXCJcIikpO1xuICAgIH1cbiAgfVxuXG4gIHJldHVybiBub3JtYWxpemVkO1xufVxuXG5mdW5jdGlvbiBwYXJzZUZsYWdWYWx1ZShcbiAgb3B0aW9uOiBJRmxhZ09wdGlvbnMsXG4gIGFyZzogSUZsYWdBcmd1bWVudCxcbiAgdmFsdWU6IHN0cmluZyxcbik6IHVua25vd24ge1xuICBjb25zdCB0eXBlOiBzdHJpbmcgPSBhcmcudHlwZSB8fCBPcHRpb25UeXBlLlNUUklORztcbiAgY29uc3QgcGFyc2VUeXBlID0gVHlwZXNbdHlwZV07XG5cbiAgaWYgKCFwYXJzZVR5cGUpIHtcbiAgICB0aHJvdyBuZXcgVW5rbm93blR5cGUodHlwZSwgT2JqZWN0LmtleXMoVHlwZXMpKTtcbiAgfVxuXG4gIHJldHVybiBwYXJzZVR5cGUoe1xuICAgIGxhYmVsOiBcIk9wdGlvblwiLFxuICAgIHR5cGUsXG4gICAgbmFtZTogYC0tJHtvcHRpb24ubmFtZX1gLFxuICAgIHZhbHVlLFxuICB9KTtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUNFLGVBQWUsRUFDZixTQUFTLEVBQ1Qsb0JBQW9CLEVBQ3BCLG9CQUFvQixRQUNmLGFBQWEsQ0FBQztBQUNyQixTQUNFLCtCQUErQixFQUMvQixlQUFlLEVB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