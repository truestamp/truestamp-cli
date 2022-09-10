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
    let inLiteral = false;
    const flags = {};
    /** Option name mapping: propertyName -> option.name */ const optionNameMap = {};
    let literal = [];
    let unknown = [];
    let stopEarly = null;
    opts.flags?.forEach((opt)=>{
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
        let option;
        let optionArgs;
        let current = args[argsIndex];
        let currentValue;
        let negate = false;
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
                throw new InvalidOption(current, opts.flags ?? []);
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
            option = opts.flags && getOption(opts.flags, current);
            if (!option) {
                if (opts.flags?.length) {
                    const name = current.replace(/^-+/g, "");
                    option = matchWildCardOptions(name, opts.flags);
                    if (!option) {
                        throw new UnknownOption(current, opts.flags);
                    }
                }
                if (!option) {
                    option = {
                        name: current.replace(/^-+/, ""),
                        optionalValue: true,
                        type: OptionType.STRING
                    };
                }
            }
            const positiveName = negate ? option.name.replace(/^no-?/, "") : option.name;
            const propName = paramCaseToCamelCase(positiveName);
            if (typeof flags[propName] !== "undefined") {
                if (!opts.flags?.length) {
                    option.collect = true;
                } else if (!option.collect) {
                    throw new DuplicateOption(current);
                }
            }
            optionArgs = option.args?.length ? option.args : [
                {
                    type: option.type,
                    requiredValue: option.requiredValue,
                    optionalValue: option.optionalValue,
                    variadic: option.variadic,
                    list: option.list,
                    separator: option.separator
                }
            ];
            let optionArgsIndex = 0;
            let inOptionalArg = false;
            const previous = flags[propName];
            parseNext(option, optionArgs);
            if (typeof flags[propName] === "undefined") {
                if (optionArgs[optionArgsIndex].requiredValue) {
                    throw new MissingOptionValue(option.name);
                } else if (typeof option.default !== "undefined") {
                    flags[propName] = getDefaultValue(option);
                } else {
                    flags[propName] = true;
                }
            }
            if (option.value) {
                flags[propName] = option.value(flags[propName], previous);
            } else if (option.collect) {
                const value = typeof previous !== "undefined" ? Array.isArray(previous) ? previous : [
                    previous
                ] : [];
                value.push(flags[propName]);
                flags[propName] = value;
            }
            optionNameMap[propName] = option.name;
            opts.option?.(option, flags[propName]);
            /** Parse next argument for current option. */ // deno-lint-ignore no-inner-declarations
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
                    // make all values required by default
                    if ((typeof arg.optionalValue === "undefined" || arg.optionalValue === false) && typeof arg.requiredValue === "undefined") {
                        arg.requiredValue = true;
                    }
                } else {
                    // make non boolean value required by default
                    if (arg.type !== OptionType.BOOLEAN && (typeof arg.optionalValue === "undefined" || arg.optionalValue === false) && typeof arg.requiredValue === "undefined") {
                        arg.requiredValue = true;
                    }
                }
                if (arg.requiredValue) {
                    if (inOptionalArg) {
                        throw new RequiredArgumentFollowsOptionalArgument(option.name);
                    }
                } else {
                    inOptionalArg = true;
                }
                if (negate) {
                    flags[propName] = false;
                    return;
                }
                let result;
                let increase = false;
                if (arg.list && hasNext(arg)) {
                    const parsed = next().split(arg.separator || ",").map((nextValue)=>{
                        const value = parseValue(option, arg, nextValue);
                        if (typeof value === "undefined") {
                            throw new InvalidOptionValue(option.name, arg.type ?? "?", nextValue);
                        }
                        return value;
                    });
                    if (parsed?.length) {
                        result = parsed;
                    }
                } else {
                    if (hasNext(arg)) {
                        result = parseValue(option, arg, next());
                    } else if (arg.optionalValue && arg.type === OptionType.BOOLEAN) {
                        result = true;
                    }
                }
                if (increase && typeof currentValue === "undefined") {
                    argsIndex++;
                    if (!arg.variadic) {
                        optionArgsIndex++;
                    } else if (optionArgs[optionArgsIndex + 1]) {
                        throw new ArgumentFollowsVariadicArgument(next());
                    }
                }
                if (typeof result !== "undefined" && (optionArgs.length > 1 || arg.variadic)) {
                    if (!flags[propName]) {
                        flags[propName] = [];
                    }
                    flags[propName].push(result);
                    if (hasNext(arg)) {
                        parseNext(option, optionArgs);
                    }
                } else {
                    flags[propName] = result;
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
                    if (option.equalsSign && arg.optionalValue && !arg.variadic && typeof currentValue === "undefined") {
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
    const result = Object.keys(flags).reduce((result, key)=>{
        if (~key.indexOf(".")) {
            key.split(".").reduce((// deno-lint-ignore no-explicit-any
            result, subKey, index, parts)=>{
                if (index === parts.length - 1) {
                    result[subKey] = flags[key];
                } else {
                    result[subKey] = result[subKey] ?? {};
                }
                return result[subKey];
            }, result);
        } else {
            result[key] = flags[key];
        }
        return result;
    }, {});
    return {
        flags: result,
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvZmxhZ3MvZmxhZ3MudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHtcbiAgZ2V0RGVmYXVsdFZhbHVlLFxuICBnZXRPcHRpb24sXG4gIG1hdGNoV2lsZENhcmRPcHRpb25zLFxuICBwYXJhbUNhc2VUb0NhbWVsQ2FzZSxcbn0gZnJvbSBcIi4vX3V0aWxzLnRzXCI7XG5pbXBvcnQge1xuICBBcmd1bWVudEZvbGxvd3NWYXJpYWRpY0FyZ3VtZW50LFxuICBEdXBsaWNhdGVPcHRpb24sXG4gIEludmFsaWRPcHRpb24sXG4gIEludmFsaWRPcHRpb25WYWx1ZSxcbiAgTWlzc2luZ09wdGlvblZhbHVlLFxuICBSZXF1aXJlZEFyZ3VtZW50Rm9sbG93c09wdGlvbmFsQXJndW1lbnQsXG4gIFVua25vd25Db25mbGljdGluZ09wdGlvbixcbiAgVW5rbm93bk9wdGlvbixcbiAgVW5rbm93blJlcXVpcmVkT3B0aW9uLFxuICBVbmtub3duVHlwZSxcbn0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHR5cGUge1xuICBJRmxhZ0FyZ3VtZW50LFxuICBJRmxhZ09wdGlvbnMsXG4gIElGbGFnc1Jlc3VsdCxcbiAgSVBhcnNlT3B0aW9ucyxcbiAgSVR5cGVIYW5kbGVyLFxufSBmcm9tIFwiLi90eXBlcy50c1wiO1xuaW1wb3J0IHsgT3B0aW9uVHlwZSB9IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBib29sZWFuIH0gZnJvbSBcIi4vdHlwZXMvYm9vbGVhbi50c1wiO1xuaW1wb3J0IHsgbnVtYmVyIH0gZnJvbSBcIi4vdHlwZXMvbnVtYmVyLnRzXCI7XG5pbXBvcnQgeyBzdHJpbmcgfSBmcm9tIFwiLi90eXBlcy9zdHJpbmcudHNcIjtcbmltcG9ydCB7IHZhbGlkYXRlRmxhZ3MgfSBmcm9tIFwiLi92YWxpZGF0ZV9mbGFncy50c1wiO1xuaW1wb3J0IHsgaW50ZWdlciB9IGZyb20gXCIuL3R5cGVzL2ludGVnZXIudHNcIjtcblxuY29uc3QgVHlwZXM6IFJlY29yZDxzdHJpbmcsIElUeXBlSGFuZGxlcjx1bmtub3duPj4gPSB7XG4gIFtPcHRpb25UeXBlLlNUUklOR106IHN0cmluZyxcbiAgW09wdGlvblR5cGUuTlVNQkVSXTogbnVtYmVyLFxuICBbT3B0aW9uVHlwZS5JTlRFR0VSXTogaW50ZWdlcixcbiAgW09wdGlvblR5cGUuQk9PTEVBTl06IGJvb2xlYW4sXG59O1xuXG4vKipcbiAqIFBhcnNlIGNvbW1hbmQgbGluZSBhcmd1bWVudHMuXG4gKiBAcGFyYW0gYXJncyAgQ29tbWFuZCBsaW5lIGFyZ3VtZW50cyBlLmc6IGBEZW5vLmFyZ3NgXG4gKiBAcGFyYW0gb3B0cyAgUGFyc2Ugb3B0aW9ucy5cbiAqIGBgYFxuICogLy8gZXhhbXBsZS50cyAteCAzIC15LnogLW41IC1hYmMgLS1iZWVwPWJvb3AgZm9vIGJhciBiYXogLS1kZW5vLmxhbmQgLS0gLS1jbGlmZnlcbiAqIHBhcnNlRmxhZ3MoRGVuby5hcmdzKTtcbiAqIGBgYFxuICogYGBgXG4gKiB7XG4gKiAgIGZsYWdzOiB7XG4gKiAgICAgeDogXCIzXCIsXG4gKiAgICAgeTogeyB6OiB0cnVlIH0sXG4gKiAgICAgbjogXCI1XCIsXG4gKiAgICAgYTogdHJ1ZSxcbiAqICAgICBiOiB0cnVlLFxuICogICAgIGM6IHRydWUsXG4gKiAgICAgYmVlcDogXCJib29wXCIsXG4gKiAgICAgZGVubzogeyBsYW5kOiB0cnVlIH1cbiAqICAgfSxcbiAqICAgdW5rbm93bjogWyBcImZvb1wiLCBcImJhclwiLCBcImJhelwiIF0sXG4gKiAgIGxpdGVyYWw6IFsgXCItLWNsaWZmeVwiIF1cbiAqIH1cbiAqIGBgYFxuICovXG5leHBvcnQgZnVuY3Rpb24gcGFyc2VGbGFnczxcbiAgLy8gZGVuby1saW50LWlnbm9yZSBuby1leHBsaWNpdC1hbnlcbiAgTyBleHRlbmRzIFJlY29yZDxzdHJpbmcsIGFueT4gPSBSZWNvcmQ8c3RyaW5nLCBhbnk+LFxuICBUIGV4dGVuZHMgSUZsYWdPcHRpb25zID0gSUZsYWdPcHRpb25zLFxuPihcbiAgYXJnczogc3RyaW5nW10sXG4gIG9wdHM6IElQYXJzZU9wdGlvbnM8VD4gPSB7fSxcbik6IElGbGFnc1Jlc3VsdDxPPiB7XG4gIGFyZ3MgPSBhcmdzLnNsaWNlKCk7XG5cbiAgbGV0IGluTGl0ZXJhbCA9IGZhbHNlO1xuXG4gIGNvbnN0IGZsYWdzOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPiA9IHt9O1xuICAvKiogT3B0aW9uIG5hbWUgbWFwcGluZzogcHJvcGVydHlOYW1lIC0+IG9wdGlvbi5uYW1lICovXG4gIGNvbnN0IG9wdGlvbk5hbWVNYXA6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4gPSB7fTtcbiAgbGV0IGxpdGVyYWw6IHN0cmluZ1tdID0gW107XG4gIGxldCB1bmtub3duOiBzdHJpbmdbXSA9IFtdO1xuICBsZXQgc3RvcEVhcmx5OiBzdHJpbmcgfCBudWxsID0gbnVsbDtcblxuICBvcHRzLmZsYWdzPy5mb3JFYWNoKChvcHQpID0+IHtcbiAgICBvcHQuZGVwZW5kcz8uZm9yRWFjaCgoZmxhZykgPT4ge1xuICAgICAgaWYgKCFvcHRzLmZsYWdzIHx8ICFnZXRPcHRpb24ob3B0cy5mbGFncywgZmxhZykpIHtcbiAgICAgICAgdGhyb3cgbmV3IFVua25vd25SZXF1aXJlZE9wdGlvbihmbGFnLCBvcHRzLmZsYWdzID8/IFtdKTtcbiAgICAgIH1cbiAgICB9KTtcbiAgICBvcHQuY29uZmxpY3RzPy5mb3JFYWNoKChmbGFnKSA9PiB7XG4gICAgICBpZiAoIW9wdHMuZmxhZ3MgfHwgIWdldE9wdGlvbihvcHRzLmZsYWdzLCBmbGFnKSkge1xuICAgICAgICB0aHJvdyBuZXcgVW5rbm93bkNvbmZsaWN0aW5nT3B0aW9uKGZsYWcsIG9wdHMuZmxhZ3MgPz8gW10pO1xuICAgICAgfVxuICAgIH0pO1xuICB9KTtcblxuICBmb3IgKFxuICAgIGxldCBhcmdzSW5kZXggPSAwO1xuICAgIGFyZ3NJbmRleCA8IGFyZ3MubGVuZ3RoO1xuICAgIGFyZ3NJbmRleCsrXG4gICkge1xuICAgIGxldCBvcHRpb246IElGbGFnT3B0aW9ucyB8IHVuZGVmaW5lZDtcbiAgICBsZXQgb3B0aW9uQXJnczogSUZsYWdBcmd1bWVudFtdIHwgdW5kZWZpbmVkO1xuICAgIGxldCBjdXJyZW50OiBzdHJpbmcgPSBhcmdzW2FyZ3NJbmRleF07XG4gICAgbGV0IGN1cnJlbnRWYWx1ZTogc3RyaW5nIHwgdW5kZWZpbmVkO1xuICAgIGxldCBuZWdhdGUgPSBmYWxzZTtcblxuICAgIC8vIGxpdGVyYWwgYXJncyBhZnRlciAtLVxuICAgIGlmIChpbkxpdGVyYWwpIHtcbiAgICAgIGxpdGVyYWwucHVzaChjdXJyZW50KTtcbiAgICAgIGNvbnRpbnVlO1xuICAgIH1cblxuICAgIGlmIChjdXJyZW50ID09PSBcIi0tXCIpIHtcbiAgICAgIGluTGl0ZXJhbCA9IHRydWU7XG4gICAgICBjb250aW51ZTtcbiAgICB9XG5cbiAgICBjb25zdCBpc0ZsYWcgPSBjdXJyZW50Lmxlbmd0aCA+IDEgJiYgY3VycmVudFswXSA9PT0gXCItXCI7XG4gICAgY29uc3QgbmV4dCA9ICgpID0+IGN1cnJlbnRWYWx1ZSA/PyBhcmdzW2FyZ3NJbmRleCArIDFdO1xuXG4gICAgaWYgKGlzRmxhZykge1xuICAgICAgY29uc3QgaXNTaG9ydCA9IGN1cnJlbnRbMV0gIT09IFwiLVwiO1xuICAgICAgY29uc3QgaXNMb25nID0gaXNTaG9ydCA/IGZhbHNlIDogY3VycmVudC5sZW5ndGggPiAzICYmIGN1cnJlbnRbMl0gIT09IFwiLVwiO1xuXG4gICAgICBpZiAoIWlzU2hvcnQgJiYgIWlzTG9uZykge1xuICAgICAgICB0aHJvdyBuZXcgSW52YWxpZE9wdGlvbihjdXJyZW50LCBvcHRzLmZsYWdzID8/IFtdKTtcbiAgICAgIH1cblxuICAgICAgLy8gc3BsaXQgdmFsdWU6IC0tZm9vPVwiYmFyPWJhelwiID0+IC0tZm9vIGJhcj1iYXpcbiAgICAgIGNvbnN0IGVxdWFsU2lnbkluZGV4ID0gY3VycmVudC5pbmRleE9mKFwiPVwiKTtcbiAgICAgIGlmIChlcXVhbFNpZ25JbmRleCA+IC0xKSB7XG4gICAgICAgIGN1cnJlbnRWYWx1ZSA9IGN1cnJlbnQuc2xpY2UoZXF1YWxTaWduSW5kZXggKyAxKSB8fCB1bmRlZmluZWQ7XG4gICAgICAgIGN1cnJlbnQgPSBjdXJyZW50LnNsaWNlKDAsIGVxdWFsU2lnbkluZGV4KTtcbiAgICAgIH1cblxuICAgICAgLy8gbm9ybWFsaXplIHNob3J0IGZsYWdzOiAtYWJjID0+IC1hIC1iIC1jXG4gICAgICBpZiAoaXNTaG9ydCAmJiBjdXJyZW50Lmxlbmd0aCA+IDIgJiYgY3VycmVudFsyXSAhPT0gXCIuXCIpIHtcbiAgICAgICAgYXJncy5zcGxpY2UoYXJnc0luZGV4LCAxLCAuLi5zcGxpdEZsYWdzKGN1cnJlbnQpKTtcbiAgICAgICAgY3VycmVudCA9IGFyZ3NbYXJnc0luZGV4XTtcbiAgICAgIH0gZWxzZSBpZiAoaXNMb25nICYmIGN1cnJlbnQuc3RhcnRzV2l0aChcIi0tbm8tXCIpKSB7XG4gICAgICAgIG5lZ2F0ZSA9IHRydWU7XG4gICAgICB9XG4gICAgICBvcHRpb24gPSBvcHRzLmZsYWdzICYmIGdldE9wdGlvbihvcHRzLmZsYWdzLCBjdXJyZW50KTtcblxuICAgICAgaWYgKCFvcHRpb24pIHtcbiAgICAgICAgaWYgKG9wdHMuZmxhZ3M/Lmxlbmd0aCkge1xuICAgICAgICAgIGNvbnN0IG5hbWUgPSBjdXJyZW50LnJlcGxhY2UoL14tKy9nLCBcIlwiKTtcbiAgICAgICAgICBvcHRpb24gPSBtYXRjaFdpbGRDYXJkT3B0aW9ucyhuYW1lLCBvcHRzLmZsYWdzKTtcbiAgICAgICAgICBpZiAoIW9wdGlvbikge1xuICAgICAgICAgICAgdGhyb3cgbmV3IFVua25vd25PcHRpb24oY3VycmVudCwgb3B0cy5mbGFncyk7XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICAgIGlmICghb3B0aW9uKSB7XG4gICAgICAgICAgb3B0aW9uID0ge1xuICAgICAgICAgICAgbmFtZTogY3VycmVudC5yZXBsYWNlKC9eLSsvLCBcIlwiKSxcbiAgICAgICAgICAgIG9wdGlvbmFsVmFsdWU6IHRydWUsXG4gICAgICAgICAgICB0eXBlOiBPcHRpb25UeXBlLlNUUklORyxcbiAgICAgICAgICB9O1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGNvbnN0IHBvc2l0aXZlTmFtZTogc3RyaW5nID0gbmVnYXRlXG4gICAgICAgID8gb3B0aW9uLm5hbWUucmVwbGFjZSgvXm5vLT8vLCBcIlwiKVxuICAgICAgICA6IG9wdGlvbi5uYW1lO1xuICAgICAgY29uc3QgcHJvcE5hbWU6IHN0cmluZyA9IHBhcmFtQ2FzZVRvQ2FtZWxDYXNlKHBvc2l0aXZlTmFtZSk7XG5cbiAgICAgIGlmICh0eXBlb2YgZmxhZ3NbcHJvcE5hbWVdICE9PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICAgIGlmICghb3B0cy5mbGFncz8ubGVuZ3RoKSB7XG4gICAgICAgICAgb3B0aW9uLmNvbGxlY3QgPSB0cnVlO1xuICAgICAgICB9IGVsc2UgaWYgKCFvcHRpb24uY29sbGVjdCkge1xuICAgICAgICAgIHRocm93IG5ldyBEdXBsaWNhdGVPcHRpb24oY3VycmVudCk7XG4gICAgICAgIH1cbiAgICAgIH1cblxuICAgICAgb3B0aW9uQXJncyA9IG9wdGlvbi5hcmdzPy5sZW5ndGggPyBvcHRpb24uYXJncyA6IFt7XG4gICAgICAgIHR5cGU6IG9wdGlvbi50eXBlLFxuICAgICAgICByZXF1aXJlZFZhbHVlOiBvcHRpb24ucmVxdWlyZWRWYWx1ZSxcbiAgICAgICAgb3B0aW9uYWxWYWx1ZTogb3B0aW9uLm9wdGlvbmFsVmFsdWUsXG4gICAgICAgIHZhcmlhZGljOiBvcHRpb24udmFyaWFkaWMsXG4gICAgICAgIGxpc3Q6IG9wdGlvbi5saXN0LFxuICAgICAgICBzZXBhcmF0b3I6IG9wdGlvbi5zZXBhcmF0b3IsXG4gICAgICB9XTtcblxuICAgICAgbGV0IG9wdGlvbkFyZ3NJbmRleCA9IDA7XG4gICAgICBsZXQgaW5PcHRpb25hbEFyZyA9IGZhbHNlO1xuICAgICAgY29uc3QgcHJldmlvdXMgPSBmbGFnc1twcm9wTmFtZV07XG5cbiAgICAgIHBhcnNlTmV4dChvcHRpb24sIG9wdGlvbkFyZ3MpO1xuXG4gICAgICBpZiAodHlwZW9mIGZsYWdzW3Byb3BOYW1lXSA9PT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICBpZiAob3B0aW9uQXJnc1tvcHRpb25BcmdzSW5kZXhdLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgTWlzc2luZ09wdGlvblZhbHVlKG9wdGlvbi5uYW1lKTtcbiAgICAgICAgfSBlbHNlIGlmICh0eXBlb2Ygb3B0aW9uLmRlZmF1bHQgIT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICBmbGFnc1twcm9wTmFtZV0gPSBnZXREZWZhdWx0VmFsdWUob3B0aW9uKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBmbGFnc1twcm9wTmFtZV0gPSB0cnVlO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGlmIChvcHRpb24udmFsdWUpIHtcbiAgICAgICAgZmxhZ3NbcHJvcE5hbWVdID0gb3B0aW9uLnZhbHVlKGZsYWdzW3Byb3BOYW1lXSwgcHJldmlvdXMpO1xuICAgICAgfSBlbHNlIGlmIChvcHRpb24uY29sbGVjdCkge1xuICAgICAgICBjb25zdCB2YWx1ZTogdW5rbm93bltdID0gdHlwZW9mIHByZXZpb3VzICE9PSBcInVuZGVmaW5lZFwiXG4gICAgICAgICAgPyAoQXJyYXkuaXNBcnJheShwcmV2aW91cykgPyBwcmV2aW91cyA6IFtwcmV2aW91c10pXG4gICAgICAgICAgOiBbXTtcblxuICAgICAgICB2YWx1ZS5wdXNoKGZsYWdzW3Byb3BOYW1lXSk7XG4gICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IHZhbHVlO1xuICAgICAgfVxuXG4gICAgICBvcHRpb25OYW1lTWFwW3Byb3BOYW1lXSA9IG9wdGlvbi5uYW1lO1xuXG4gICAgICBvcHRzLm9wdGlvbj8uKG9wdGlvbiBhcyBULCBmbGFnc1twcm9wTmFtZV0pO1xuXG4gICAgICAvKiogUGFyc2UgbmV4dCBhcmd1bWVudCBmb3IgY3VycmVudCBvcHRpb24uICovXG4gICAgICAvLyBkZW5vLWxpbnQtaWdub3JlIG5vLWlubmVyLWRlY2xhcmF0aW9uc1xuICAgICAgZnVuY3Rpb24gcGFyc2VOZXh0KFxuICAgICAgICBvcHRpb246IElGbGFnT3B0aW9ucyxcbiAgICAgICAgb3B0aW9uQXJnczogSUZsYWdBcmd1bWVudFtdLFxuICAgICAgKTogdm9pZCB7XG4gICAgICAgIGNvbnN0IGFyZzogSUZsYWdBcmd1bWVudCB8IHVuZGVmaW5lZCA9IG9wdGlvbkFyZ3Nbb3B0aW9uQXJnc0luZGV4XTtcblxuICAgICAgICBpZiAoIWFyZykge1xuICAgICAgICAgIGNvbnN0IGZsYWcgPSBuZXh0KCk7XG4gICAgICAgICAgdGhyb3cgbmV3IFVua25vd25PcHRpb24oZmxhZywgb3B0cy5mbGFncyA/PyBbXSk7XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoIWFyZy50eXBlKSB7XG4gICAgICAgICAgYXJnLnR5cGUgPSBPcHRpb25UeXBlLkJPT0xFQU47XG4gICAgICAgIH1cblxuICAgICAgICBpZiAob3B0aW9uLmFyZ3M/Lmxlbmd0aCkge1xuICAgICAgICAgIC8vIG1ha2UgYWxsIHZhbHVlcyByZXF1aXJlZCBieSBkZWZhdWx0XG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgKHR5cGVvZiBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICAgICAgICBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gZmFsc2UpICYmXG4gICAgICAgICAgICB0eXBlb2YgYXJnLnJlcXVpcmVkVmFsdWUgPT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGFyZy5yZXF1aXJlZFZhbHVlID0gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgLy8gbWFrZSBub24gYm9vbGVhbiB2YWx1ZSByZXF1aXJlZCBieSBkZWZhdWx0XG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgYXJnLnR5cGUgIT09IE9wdGlvblR5cGUuQk9PTEVBTiAmJlxuICAgICAgICAgICAgKHR5cGVvZiBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICAgICAgICBhcmcub3B0aW9uYWxWYWx1ZSA9PT0gZmFsc2UpICYmXG4gICAgICAgICAgICB0eXBlb2YgYXJnLnJlcXVpcmVkVmFsdWUgPT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGFyZy5yZXF1aXJlZFZhbHVlID0gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoYXJnLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICBpZiAoaW5PcHRpb25hbEFyZykge1xuICAgICAgICAgICAgdGhyb3cgbmV3IFJlcXVpcmVkQXJndW1lbnRGb2xsb3dzT3B0aW9uYWxBcmd1bWVudChvcHRpb24ubmFtZSk7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGluT3B0aW9uYWxBcmcgPSB0cnVlO1xuICAgICAgICB9XG5cbiAgICAgICAgaWYgKG5lZ2F0ZSkge1xuICAgICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IGZhbHNlO1xuICAgICAgICAgIHJldHVybjtcbiAgICAgICAgfVxuXG4gICAgICAgIGxldCByZXN1bHQ6IHVua25vd247XG4gICAgICAgIGxldCBpbmNyZWFzZSA9IGZhbHNlO1xuXG4gICAgICAgIGlmIChhcmcubGlzdCAmJiBoYXNOZXh0KGFyZykpIHtcbiAgICAgICAgICBjb25zdCBwYXJzZWQ6IHVua25vd25bXSA9IG5leHQoKVxuICAgICAgICAgICAgLnNwbGl0KGFyZy5zZXBhcmF0b3IgfHwgXCIsXCIpXG4gICAgICAgICAgICAubWFwKChuZXh0VmFsdWU6IHN0cmluZykgPT4ge1xuICAgICAgICAgICAgICBjb25zdCB2YWx1ZSA9IHBhcnNlVmFsdWUob3B0aW9uLCBhcmcsIG5leHRWYWx1ZSk7XG4gICAgICAgICAgICAgIGlmICh0eXBlb2YgdmFsdWUgPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgICAgICAgICAgICB0aHJvdyBuZXcgSW52YWxpZE9wdGlvblZhbHVlKFxuICAgICAgICAgICAgICAgICAgb3B0aW9uLm5hbWUsXG4gICAgICAgICAgICAgICAgICBhcmcudHlwZSA/PyBcIj9cIixcbiAgICAgICAgICAgICAgICAgIG5leHRWYWx1ZSxcbiAgICAgICAgICAgICAgICApO1xuICAgICAgICAgICAgICB9XG4gICAgICAgICAgICAgIHJldHVybiB2YWx1ZTtcbiAgICAgICAgICAgIH0pO1xuXG4gICAgICAgICAgaWYgKHBhcnNlZD8ubGVuZ3RoKSB7XG4gICAgICAgICAgICByZXN1bHQgPSBwYXJzZWQ7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGlmIChoYXNOZXh0KGFyZykpIHtcbiAgICAgICAgICAgIHJlc3VsdCA9IHBhcnNlVmFsdWUob3B0aW9uLCBhcmcsIG5leHQoKSk7XG4gICAgICAgICAgfSBlbHNlIGlmIChhcmcub3B0aW9uYWxWYWx1ZSAmJiBhcmcudHlwZSA9PT0gT3B0aW9uVHlwZS5CT09MRUFOKSB7XG4gICAgICAgICAgICByZXN1bHQgPSB0cnVlO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuXG4gICAgICAgIGlmIChpbmNyZWFzZSAmJiB0eXBlb2YgY3VycmVudFZhbHVlID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICAgICAgYXJnc0luZGV4Kys7XG4gICAgICAgICAgaWYgKCFhcmcudmFyaWFkaWMpIHtcbiAgICAgICAgICAgIG9wdGlvbkFyZ3NJbmRleCsrO1xuICAgICAgICAgIH0gZWxzZSBpZiAob3B0aW9uQXJnc1tvcHRpb25BcmdzSW5kZXggKyAxXSkge1xuICAgICAgICAgICAgdGhyb3cgbmV3IEFyZ3VtZW50Rm9sbG93c1ZhcmlhZGljQXJndW1lbnQobmV4dCgpKTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cblxuICAgICAgICBpZiAoXG4gICAgICAgICAgdHlwZW9mIHJlc3VsdCAhPT0gXCJ1bmRlZmluZWRcIiAmJlxuICAgICAgICAgIChvcHRpb25BcmdzLmxlbmd0aCA+IDEgfHwgYXJnLnZhcmlhZGljKVxuICAgICAgICApIHtcbiAgICAgICAgICBpZiAoIWZsYWdzW3Byb3BOYW1lXSkge1xuICAgICAgICAgICAgZmxhZ3NbcHJvcE5hbWVdID0gW107XG4gICAgICAgICAgfVxuXG4gICAgICAgICAgKGZsYWdzW3Byb3BOYW1lXSBhcyBBcnJheTx1bmtub3duPikucHVzaChyZXN1bHQpO1xuXG4gICAgICAgICAgaWYgKGhhc05leHQoYXJnKSkge1xuICAgICAgICAgICAgcGFyc2VOZXh0KG9wdGlvbiwgb3B0aW9uQXJncyk7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIGZsYWdzW3Byb3BOYW1lXSA9IHJlc3VsdDtcbiAgICAgICAgfVxuXG4gICAgICAgIC8qKiBDaGVjayBpZiBjdXJyZW50IG9wdGlvbiBzaG91bGQgaGF2ZSBhbiBhcmd1bWVudC4gKi9cbiAgICAgICAgZnVuY3Rpb24gaGFzTmV4dChhcmc6IElGbGFnQXJndW1lbnQpOiBib29sZWFuIHtcbiAgICAgICAgICBjb25zdCBuZXh0VmFsdWUgPSBjdXJyZW50VmFsdWUgPz8gYXJnc1thcmdzSW5kZXggKyAxXTtcbiAgICAgICAgICBpZiAoIW5leHRWYWx1ZSkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAob3B0aW9uQXJncy5sZW5ndGggPiAxICYmIG9wdGlvbkFyZ3NJbmRleCA+PSBvcHRpb25BcmdzLmxlbmd0aCkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoYXJnLnJlcXVpcmVkVmFsdWUpIHtcbiAgICAgICAgICAgIHJldHVybiB0cnVlO1xuICAgICAgICAgIH1cbiAgICAgICAgICAvLyByZXF1aXJlIG9wdGlvbmFsIHZhbHVlcyB0byBiZSBjYWxsZWQgd2l0aCBhbiBlcXVhbCBzaWduOiBmb289YmFyXG4gICAgICAgICAgaWYgKFxuICAgICAgICAgICAgb3B0aW9uLmVxdWFsc1NpZ24gJiYgYXJnLm9wdGlvbmFsVmFsdWUgJiYgIWFyZy52YXJpYWRpYyAmJlxuICAgICAgICAgICAgdHlwZW9mIGN1cnJlbnRWYWx1ZSA9PT0gXCJ1bmRlZmluZWRcIlxuICAgICAgICAgICkge1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoYXJnLm9wdGlvbmFsVmFsdWUgfHwgYXJnLnZhcmlhZGljKSB7XG4gICAgICAgICAgICByZXR1cm4gbmV4dFZhbHVlWzBdICE9PSBcIi1cIiB8fFxuICAgICAgICAgICAgICAoYXJnLnR5cGUgPT09IE9wdGlvblR5cGUuTlVNQkVSICYmICFpc05hTihOdW1iZXIobmV4dFZhbHVlKSkpO1xuICAgICAgICAgIH1cblxuICAgICAgICAgIHJldHVybiBmYWxzZTtcbiAgICAgICAgfVxuXG4gICAgICAgIC8qKiBQYXJzZSBhcmd1bWVudCB2YWx1ZS4gICovXG4gICAgICAgIGZ1bmN0aW9uIHBhcnNlVmFsdWUoXG4gICAgICAgICAgb3B0aW9uOiBJRmxhZ09wdGlvbnMsXG4gICAgICAgICAgYXJnOiBJRmxhZ0FyZ3VtZW50LFxuICAgICAgICAgIHZhbHVlOiBzdHJpbmcsXG4gICAgICAgICk6IHVua25vd24ge1xuICAgICAgICAgIGNvbnN0IHR5cGU6IHN0cmluZyA9IGFyZy50eXBlIHx8IE9wdGlvblR5cGUuU1RSSU5HO1xuICAgICAgICAgIGNvbnN0IHJlc3VsdDogdW5rbm93biA9IG9wdHMucGFyc2VcbiAgICAgICAgICAgID8gb3B0cy5wYXJzZSh7XG4gICAgICAgICAgICAgIGxhYmVsOiBcIk9wdGlvblwiLFxuICAgICAgICAgICAgICB0eXBlLFxuICAgICAgICAgICAgICBuYW1lOiBgLS0ke29wdGlvbi5uYW1lfWAsXG4gICAgICAgICAgICAgIHZhbHVlLFxuICAgICAgICAgICAgfSlcbiAgICAgICAgICAgIDogcGFyc2VGbGFnVmFsdWUob3B0aW9uLCBhcmcsIHZhbHVlKTtcblxuICAgICAgICAgIGlmIChcbiAgICAgICAgICAgIHR5cGVvZiByZXN1bHQgIT09IFwidW5kZWZpbmVkXCJcbiAgICAgICAgICApIHtcbiAgICAgICAgICAgIGluY3JlYXNlID0gdHJ1ZTtcbiAgICAgICAgICB9XG5cbiAgICAgICAgICByZXR1cm4gcmVzdWx0O1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfSBlbHNlIHtcbiAgICAgIGlmIChvcHRzLnN0b3BFYXJseSkge1xuICAgICAgICBzdG9wRWFybHkgPSBjdXJyZW50O1xuICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICAgIHVua25vd24ucHVzaChjdXJyZW50KTtcbiAgICB9XG4gIH1cblxuICBpZiAoc3RvcEVhcmx5KSB7XG4gICAgY29uc3Qgc3RvcEVhcmx5QXJnSW5kZXg6IG51bWJlciA9IGFyZ3MuaW5kZXhPZihzdG9wRWFybHkpO1xuICAgIGlmIChzdG9wRWFybHlBcmdJbmRleCAhPT0gLTEpIHtcbiAgICAgIGNvbnN0IGRvdWJsZURhc2hJbmRleDogbnVtYmVyID0gYXJncy5pbmRleE9mKFwiLS1cIik7XG4gICAgICB1bmtub3duID0gYXJncy5zbGljZShcbiAgICAgICAgc3RvcEVhcmx5QXJnSW5kZXgsXG4gICAgICAgIGRvdWJsZURhc2hJbmRleCA9PT0gLTEgPyB1bmRlZmluZWQgOiBkb3VibGVEYXNoSW5kZXgsXG4gICAgICApO1xuICAgICAgaWYgKGRvdWJsZURhc2hJbmRleCAhPT0gLTEpIHtcbiAgICAgICAgbGl0ZXJhbCA9IGFyZ3Muc2xpY2UoZG91YmxlRGFzaEluZGV4ICsgMSk7XG4gICAgICB9XG4gICAgfVxuICB9XG5cbiAgdmFsaWRhdGVGbGFncyhvcHRzLCBmbGFncywgb3B0aW9uTmFtZU1hcCk7XG5cbiAgLy8gY29udmVydCBkb3R0ZWQgb3B0aW9uIGtleXMgaW50byBuZXN0ZWQgb2JqZWN0c1xuICBjb25zdCByZXN1bHQgPSBPYmplY3Qua2V5cyhmbGFncylcbiAgICAucmVkdWNlKChyZXN1bHQ6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LCBrZXk6IHN0cmluZykgPT4ge1xuICAgICAgaWYgKH5rZXkuaW5kZXhPZihcIi5cIikpIHtcbiAgICAgICAga2V5LnNwbGl0KFwiLlwiKS5yZWR1Y2UoXG4gICAgICAgICAgKFxuICAgICAgICAgICAgLy8gZGVuby1saW50LWlnbm9yZSBuby1leHBsaWNpdC1hbnlcbiAgICAgICAgICAgIHJlc3VsdDogUmVjb3JkPHN0cmluZywgYW55PixcbiAgICAgICAgICAgIHN1YktleTogc3RyaW5nLFxuICAgICAgICAgICAgaW5kZXg6IG51bWJlcixcbiAgICAgICAgICAgIHBhcnRzOiBzdHJpbmdbXSxcbiAgICAgICAgICApID0+IHtcbiAgICAgICAgICAgIGlmIChpbmRleCA9PT0gcGFydHMubGVuZ3RoIC0gMSkge1xuICAgICAgICAgICAgICByZXN1bHRbc3ViS2V5XSA9IGZsYWdzW2tleV07XG4gICAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgICByZXN1bHRbc3ViS2V5XSA9IHJlc3VsdFtzdWJLZXldID8/IHt9O1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgcmV0dXJuIHJlc3VsdFtzdWJLZXldO1xuICAgICAgICAgIH0sXG4gICAgICAgICAgcmVzdWx0LFxuICAgICAgICApO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0W2tleV0gPSBmbGFnc1trZXldO1xuICAgICAgfVxuICAgICAgcmV0dXJuIHJlc3VsdDtcbiAgICB9LCB7fSk7XG5cbiAgcmV0dXJuIHsgZmxhZ3M6IHJlc3VsdCBhcyBPLCB1bmtub3duLCBsaXRlcmFsIH07XG59XG5cbmZ1bmN0aW9uIHNwbGl0RmxhZ3MoZmxhZzogc3RyaW5nKTogQXJyYXk8c3RyaW5nPiB7XG4gIGNvbnN0IG5vcm1hbGl6ZWQ6IEFycmF5PHN0cmluZz4gPSBbXTtcbiAgY29uc3QgZmxhZ3MgPSBmbGFnLnNsaWNlKDEpLnNwbGl0KFwiXCIpO1xuXG4gIGlmIChpc05hTihOdW1iZXIoZmxhZ1tmbGFnLmxlbmd0aCAtIDFdKSkpIHtcbiAgICBmbGFncy5mb3JFYWNoKCh2YWwpID0+IG5vcm1hbGl6ZWQucHVzaChgLSR7dmFsfWApKTtcbiAgfSBlbHNlIHtcbiAgICBub3JtYWxpemVkLnB1c2goYC0ke2ZsYWdzLnNoaWZ0KCl9YCk7XG4gICAgaWYgKGZsYWdzLmxlbmd0aCkge1xuICAgICAgbm9ybWFsaXplZC5wdXNoKGZsYWdzLmpvaW4oXCJcIikpO1xuICAgIH1cbiAgfVxuXG4gIHJldHVybiBub3JtYWxpemVkO1xufVxuXG5mdW5jdGlvbiBwYXJzZUZsYWdWYWx1ZShcbiAgb3B0aW9uOiBJRmxhZ09wdGlvbnMsXG4gIGFyZzogSUZsYWdBcmd1bWVudCxcbiAgdmFsdWU6IHN0cmluZyxcbik6IHVua25vd24ge1xuICBjb25zdCB0eXBlOiBzdHJpbmcgPSBhcmcudHlwZSB8fCBPcHRpb25UeXBlLlNUUklORztcbiAgY29uc3QgcGFyc2VUeXBlID0gVHlwZXNbdHlwZV07XG5cbiAgaWYgKCFwYXJzZVR5cGUpIHtcbiAgICB0aHJvdyBuZXcgVW5rbm93blR5cGUodHlwZSwgT2JqZWN0LmtleXMoVHlwZXMpKTtcbiAgfVxuXG4gIHJldHVybiBwYXJzZVR5cGUoe1xuICAgIGxhYmVsOiBcIk9wdGlvblwiLFxuICAgIHR5cGUsXG4gICAgbmFtZTogYC0tJHtvcHRpb24ubmFtZX1gLFxuICAgIHZhbHVlLFxuICB9KTtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUNFLGVBQWUsRUFDZixTQUFTLEVBQ1Qsb0JBQW9CLEVBQ3BCLG9CQUFvQixRQUNmLGFBQWEsQ0FBQztBQUNyQixTQUNFLCtCQUErQixFQUMvQixlQUFlLEVBQ2YsYUFBYSxFQUNi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