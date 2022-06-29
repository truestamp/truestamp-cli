import { getDefaultValue, getOption, paramCaseToCamelCase } from "./_utils.ts";
import { ConflictingOption, DependingOption, MissingOptionValue, MissingRequiredOption, OptionNotCombinable, UnknownOption } from "./_errors.ts";
/**
 * Flags post validation. Validations that are not already done by the parser.
 *
 * @param opts            Parse options.
 * @param values          Flag values.
 * @param optionNameMap   Option name mappings: propertyName -> option.name
 */ export function validateFlags(opts, values, optionNameMap = {}) {
    if (!opts.flags?.length) {
        return;
    }
    const defaultValues = setDefaultValues(opts, values, optionNameMap);
    const optionNames = Object.keys(values);
    if (!optionNames.length && opts.allowEmpty) {
        return;
    }
    const options = optionNames.map((name)=>({
            name,
            option: getOption(opts.flags, optionNameMap[name])
        }));
    for (const { name , option  } of options){
        if (!option) {
            throw new UnknownOption(name, opts.flags);
        }
        if (validateStandaloneOption(option, options, optionNames, defaultValues)) {
            return;
        }
        validateConflictingOptions(option, values);
        validateDependingOptions(option, values, defaultValues);
        validateRequiredValues(option, values, name);
    }
    validateRequiredOptions(options, values, opts);
}
/**
 * Adds all default values on the values object and returns a new object with
 * only the default values.
 *
 * @param opts
 * @param values
 * @param optionNameMap
 */ function setDefaultValues(opts, values, optionNameMap = {}) {
    const defaultValues = {};
    if (!opts.flags?.length) {
        return defaultValues;
    }
    // Set default values
    for (const option of opts.flags){
        let name;
        let defaultValue = undefined;
        // if --no-[flag] is present set --[flag] default value to true
        if (option.name.startsWith("no-")) {
            const propName = option.name.replace(/^no-/, "");
            if (propName in values) {
                continue;
            }
            const positiveOption = getOption(opts.flags, propName);
            if (positiveOption) {
                continue;
            }
            name = paramCaseToCamelCase(propName);
            defaultValue = true;
        }
        if (!name) {
            name = paramCaseToCamelCase(option.name);
        }
        if (!(name in optionNameMap)) {
            optionNameMap[name] = option.name;
        }
        const hasDefaultValue = (!opts.ignoreDefaults || typeof opts.ignoreDefaults[name] === "undefined") && typeof values[name] === "undefined" && (typeof option.default !== "undefined" || typeof defaultValue !== "undefined");
        if (hasDefaultValue) {
            values[name] = getDefaultValue(option) ?? defaultValue;
            defaultValues[option.name] = true;
            if (typeof option.value === "function") {
                values[name] = option.value(values[name]);
            }
        }
    }
    return defaultValues;
}
function validateStandaloneOption(option, options, optionNames, defaultValues) {
    if (!option.standalone) {
        return false;
    }
    if (optionNames.length === 1) {
        return true;
    }
    // don't throw an error if all values are coming from the default option.
    if (options.every((opt)=>opt.option && (option === opt.option || defaultValues[opt.option.name]))) {
        return true;
    }
    throw new OptionNotCombinable(option.name);
}
function validateConflictingOptions(option, values) {
    option.conflicts?.forEach((flag)=>{
        if (isset(flag, values)) {
            throw new ConflictingOption(option.name, flag);
        }
    });
}
function validateDependingOptions(option, values, defaultValues) {
    option.depends?.forEach((flag)=>{
        // don't throw an error if the value is coming from the default option.
        if (!isset(flag, values) && !defaultValues[option.name]) {
            throw new DependingOption(option.name, flag);
        }
    });
}
function validateRequiredValues(option, values, name) {
    const isArray = (option.args?.length || 0) > 1;
    option.args?.forEach((arg, i)=>{
        if (arg.requiredValue && (typeof values[name] === "undefined" || isArray && typeof values[name][i] === "undefined")) {
            throw new MissingOptionValue(option.name);
        }
    });
}
function validateRequiredOptions(options, values, opts) {
    if (!opts.flags?.length) {
        return;
    }
    for (const option of opts.flags){
        if (option.required && !(paramCaseToCamelCase(option.name) in values)) {
            if ((!option.conflicts || !option.conflicts.find((flag)=>!!values[flag])) && !options.find((opt)=>opt.option?.conflicts?.find((flag)=>flag === option.name))) {
                throw new MissingRequiredOption(option.name);
            }
        }
    }
}
/**
 * Check if value exists for flag.
 * @param flag    Flag name.
 * @param values  Parsed values.
 */ function isset(flag, values) {
    const name = paramCaseToCamelCase(flag);
    // return typeof values[ name ] !== 'undefined' && values[ name ] !== false;
    return typeof values[name] !== "undefined";
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvZmxhZ3MvdmFsaWRhdGVfZmxhZ3MudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgZ2V0RGVmYXVsdFZhbHVlLCBnZXRPcHRpb24sIHBhcmFtQ2FzZVRvQ2FtZWxDYXNlIH0gZnJvbSBcIi4vX3V0aWxzLnRzXCI7XG5pbXBvcnQge1xuICBDb25mbGljdGluZ09wdGlvbixcbiAgRGVwZW5kaW5nT3B0aW9uLFxuICBNaXNzaW5nT3B0aW9uVmFsdWUsXG4gIE1pc3NpbmdSZXF1aXJlZE9wdGlvbixcbiAgT3B0aW9uTm90Q29tYmluYWJsZSxcbiAgVW5rbm93bk9wdGlvbixcbn0gZnJvbSBcIi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgSVBhcnNlT3B0aW9ucyB9IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5pbXBvcnQgdHlwZSB7IElGbGFnQXJndW1lbnQsIElGbGFnT3B0aW9ucyB9IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5cbi8qKiBGbGFnIG9wdGlvbiBtYXAuICovXG5pbnRlcmZhY2UgSUZsYWdPcHRpb25zTWFwIHtcbiAgbmFtZTogc3RyaW5nO1xuICBvcHRpb24/OiBJRmxhZ09wdGlvbnM7XG59XG5cbi8qKlxuICogRmxhZ3MgcG9zdCB2YWxpZGF0aW9uLiBWYWxpZGF0aW9ucyB0aGF0IGFyZSBub3QgYWxyZWFkeSBkb25lIGJ5IHRoZSBwYXJzZXIuXG4gKlxuICogQHBhcmFtIG9wdHMgICAgICAgICAgICBQYXJzZSBvcHRpb25zLlxuICogQHBhcmFtIHZhbHVlcyAgICAgICAgICBGbGFnIHZhbHVlcy5cbiAqIEBwYXJhbSBvcHRpb25OYW1lTWFwICAgT3B0aW9uIG5hbWUgbWFwcGluZ3M6IHByb3BlcnR5TmFtZSAtPiBvcHRpb24ubmFtZVxuICovXG5leHBvcnQgZnVuY3Rpb24gdmFsaWRhdGVGbGFnczxUIGV4dGVuZHMgSUZsYWdPcHRpb25zID0gSUZsYWdPcHRpb25zPihcbiAgb3B0czogSVBhcnNlT3B0aW9uczxUPixcbiAgdmFsdWVzOiBSZWNvcmQ8c3RyaW5nLCB1bmtub3duPixcbiAgb3B0aW9uTmFtZU1hcDogUmVjb3JkPHN0cmluZywgc3RyaW5nPiA9IHt9LFxuKTogdm9pZCB7XG4gIGlmICghb3B0cy5mbGFncz8ubGVuZ3RoKSB7XG4gICAgcmV0dXJuO1xuICB9XG4gIGNvbnN0IGRlZmF1bHRWYWx1ZXMgPSBzZXREZWZhdWx0VmFsdWVzKG9wdHMsIHZhbHVlcywgb3B0aW9uTmFtZU1hcCk7XG5cbiAgY29uc3Qgb3B0aW9uTmFtZXMgPSBPYmplY3Qua2V5cyh2YWx1ZXMpO1xuICBpZiAoIW9wdGlvbk5hbWVzLmxlbmd0aCAmJiBvcHRzLmFsbG93RW1wdHkpIHtcbiAgICByZXR1cm47XG4gIH1cblxuICBjb25zdCBvcHRpb25zOiBBcnJheTxJRmxhZ09wdGlvbnNNYXA+ID0gb3B0aW9uTmFtZXMubWFwKChuYW1lKSA9PiAoe1xuICAgIG5hbWUsXG4gICAgb3B0aW9uOiBnZXRPcHRpb24ob3B0cy5mbGFncyEsIG9wdGlvbk5hbWVNYXBbbmFtZV0pLFxuICB9KSk7XG5cbiAgZm9yIChjb25zdCB7IG5hbWUsIG9wdGlvbiB9IG9mIG9wdGlvbnMpIHtcbiAgICBpZiAoIW9wdGlvbikge1xuICAgICAgdGhyb3cgbmV3IFVua25vd25PcHRpb24obmFtZSwgb3B0cy5mbGFncyk7XG4gICAgfVxuICAgIGlmICh2YWxpZGF0ZVN0YW5kYWxvbmVPcHRpb24ob3B0aW9uLCBvcHRpb25zLCBvcHRpb25OYW1lcywgZGVmYXVsdFZhbHVlcykpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG4gICAgdmFsaWRhdGVDb25mbGljdGluZ09wdGlvbnMob3B0aW9uLCB2YWx1ZXMpO1xuICAgIHZhbGlkYXRlRGVwZW5kaW5nT3B0aW9ucyhvcHRpb24sIHZhbHVlcywgZGVmYXVsdFZhbHVlcyk7XG4gICAgdmFsaWRhdGVSZXF1aXJlZFZhbHVlcyhvcHRpb24sIHZhbHVlcywgbmFtZSk7XG4gIH1cbiAgdmFsaWRhdGVSZXF1aXJlZE9wdGlvbnMob3B0aW9ucywgdmFsdWVzLCBvcHRzKTtcbn1cblxuLyoqXG4gKiBBZGRzIGFsbCBkZWZhdWx0IHZhbHVlcyBvbiB0aGUgdmFsdWVzIG9iamVjdCBhbmQgcmV0dXJucyBhIG5ldyBvYmplY3Qgd2l0aFxuICogb25seSB0aGUgZGVmYXVsdCB2YWx1ZXMuXG4gKlxuICogQHBhcmFtIG9wdHNcbiAqIEBwYXJhbSB2YWx1ZXNcbiAqIEBwYXJhbSBvcHRpb25OYW1lTWFwXG4gKi9cbmZ1bmN0aW9uIHNldERlZmF1bHRWYWx1ZXM8VCBleHRlbmRzIElGbGFnT3B0aW9ucyA9IElGbGFnT3B0aW9ucz4oXG4gIG9wdHM6IElQYXJzZU9wdGlvbnM8VD4sXG4gIHZhbHVlczogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gIG9wdGlvbk5hbWVNYXA6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4gPSB7fSxcbikge1xuICBjb25zdCBkZWZhdWx0VmFsdWVzOiBSZWNvcmQ8c3RyaW5nLCBib29sZWFuPiA9IHt9O1xuICBpZiAoIW9wdHMuZmxhZ3M/Lmxlbmd0aCkge1xuICAgIHJldHVybiBkZWZhdWx0VmFsdWVzO1xuICB9XG5cbiAgLy8gU2V0IGRlZmF1bHQgdmFsdWVzXG4gIGZvciAoY29uc3Qgb3B0aW9uIG9mIG9wdHMuZmxhZ3MpIHtcbiAgICBsZXQgbmFtZTogc3RyaW5nIHwgdW5kZWZpbmVkO1xuICAgIGxldCBkZWZhdWx0VmFsdWU6IHVua25vd24gPSB1bmRlZmluZWQ7XG5cbiAgICAvLyBpZiAtLW5vLVtmbGFnXSBpcyBwcmVzZW50IHNldCAtLVtmbGFnXSBkZWZhdWx0IHZhbHVlIHRvIHRydWVcbiAgICBpZiAob3B0aW9uLm5hbWUuc3RhcnRzV2l0aChcIm5vLVwiKSkge1xuICAgICAgY29uc3QgcHJvcE5hbWUgPSBvcHRpb24ubmFtZS5yZXBsYWNlKC9ebm8tLywgXCJcIik7XG4gICAgICBpZiAocHJvcE5hbWUgaW4gdmFsdWVzKSB7XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuICAgICAgY29uc3QgcG9zaXRpdmVPcHRpb24gPSBnZXRPcHRpb24ob3B0cy5mbGFncywgcHJvcE5hbWUpO1xuICAgICAgaWYgKHBvc2l0aXZlT3B0aW9uKSB7XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuICAgICAgbmFtZSA9IHBhcmFtQ2FzZVRvQ2FtZWxDYXNlKHByb3BOYW1lKTtcbiAgICAgIGRlZmF1bHRWYWx1ZSA9IHRydWU7XG4gICAgfVxuXG4gICAgaWYgKCFuYW1lKSB7XG4gICAgICBuYW1lID0gcGFyYW1DYXNlVG9DYW1lbENhc2Uob3B0aW9uLm5hbWUpO1xuICAgIH1cblxuICAgIGlmICghKG5hbWUgaW4gb3B0aW9uTmFtZU1hcCkpIHtcbiAgICAgIG9wdGlvbk5hbWVNYXBbbmFtZV0gPSBvcHRpb24ubmFtZTtcbiAgICB9XG5cbiAgICBjb25zdCBoYXNEZWZhdWx0VmFsdWU6IGJvb2xlYW4gPSAoIW9wdHMuaWdub3JlRGVmYXVsdHMgfHxcbiAgICAgIHR5cGVvZiBvcHRzLmlnbm9yZURlZmF1bHRzW25hbWVdID09PSBcInVuZGVmaW5lZFwiKSAmJlxuICAgICAgdHlwZW9mIHZhbHVlc1tuYW1lXSA9PT0gXCJ1bmRlZmluZWRcIiAmJiAoXG4gICAgICAgIHR5cGVvZiBvcHRpb24uZGVmYXVsdCAhPT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICB0eXBlb2YgZGVmYXVsdFZhbHVlICE9PSBcInVuZGVmaW5lZFwiXG4gICAgICApO1xuXG4gICAgaWYgKGhhc0RlZmF1bHRWYWx1ZSkge1xuICAgICAgdmFsdWVzW25hbWVdID0gZ2V0RGVmYXVsdFZhbHVlKG9wdGlvbikgPz8gZGVmYXVsdFZhbHVlO1xuICAgICAgZGVmYXVsdFZhbHVlc1tvcHRpb24ubmFtZV0gPSB0cnVlO1xuICAgICAgaWYgKHR5cGVvZiBvcHRpb24udmFsdWUgPT09IFwiZnVuY3Rpb25cIikge1xuICAgICAgICB2YWx1ZXNbbmFtZV0gPSBvcHRpb24udmFsdWUodmFsdWVzW25hbWVdKTtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICByZXR1cm4gZGVmYXVsdFZhbHVlcztcbn1cblxuZnVuY3Rpb24gdmFsaWRhdGVTdGFuZGFsb25lT3B0aW9uKFxuICBvcHRpb246IElGbGFnT3B0aW9ucyxcbiAgb3B0aW9uczogQXJyYXk8SUZsYWdPcHRpb25zTWFwPixcbiAgb3B0aW9uTmFtZXM6IEFycmF5PHN0cmluZz4sXG4gIGRlZmF1bHRWYWx1ZXM6IFJlY29yZDxzdHJpbmcsIGJvb2xlYW4+LFxuKTogYm9vbGVhbiB7XG4gIGlmICghb3B0aW9uLnN0YW5kYWxvbmUpIHtcbiAgICByZXR1cm4gZmFsc2U7XG4gIH1cbiAgaWYgKG9wdGlvbk5hbWVzLmxlbmd0aCA9PT0gMSkge1xuICAgIHJldHVybiB0cnVlO1xuICB9XG5cbiAgLy8gZG9uJ3QgdGhyb3cgYW4gZXJyb3IgaWYgYWxsIHZhbHVlcyBhcmUgY29taW5nIGZyb20gdGhlIGRlZmF1bHQgb3B0aW9uLlxuICBpZiAoXG4gICAgb3B0aW9ucy5ldmVyeSgob3B0KSA9PlxuICAgICAgb3B0Lm9wdGlvbiAmJlxuICAgICAgKG9wdGlvbiA9PT0gb3B0Lm9wdGlvbiB8fCBkZWZhdWx0VmFsdWVzW29wdC5vcHRpb24ubmFtZV0pXG4gICAgKVxuICApIHtcbiAgICByZXR1cm4gdHJ1ZTtcbiAgfVxuXG4gIHRocm93IG5ldyBPcHRpb25Ob3RDb21iaW5hYmxlKG9wdGlvbi5uYW1lKTtcbn1cblxuZnVuY3Rpb24gdmFsaWRhdGVDb25mbGljdGluZ09wdGlvbnMoXG4gIG9wdGlvbjogSUZsYWdPcHRpb25zLFxuICB2YWx1ZXM6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuKTogdm9pZCB7XG4gIG9wdGlvbi5jb25mbGljdHM/LmZvckVhY2goKGZsYWc6IHN0cmluZykgPT4ge1xuICAgIGlmIChpc3NldChmbGFnLCB2YWx1ZXMpKSB7XG4gICAgICB0aHJvdyBuZXcgQ29uZmxpY3RpbmdPcHRpb24ob3B0aW9uLm5hbWUsIGZsYWcpO1xuICAgIH1cbiAgfSk7XG59XG5cbmZ1bmN0aW9uIHZhbGlkYXRlRGVwZW5kaW5nT3B0aW9ucyhcbiAgb3B0aW9uOiBJRmxhZ09wdGlvbnMsXG4gIHZhbHVlczogUmVjb3JkPHN0cmluZywgdW5rbm93bj4sXG4gIGRlZmF1bHRWYWx1ZXM6IFJlY29yZDxzdHJpbmcsIGJvb2xlYW4+LFxuKTogdm9pZCB7XG4gIG9wdGlvbi5kZXBlbmRzPy5mb3JFYWNoKChmbGFnOiBzdHJpbmcpID0+IHtcbiAgICAvLyBkb24ndCB0aHJvdyBhbiBlcnJvciBpZiB0aGUgdmFsdWUgaXMgY29taW5nIGZyb20gdGhlIGRlZmF1bHQgb3B0aW9uLlxuICAgIGlmICghaXNzZXQoZmxhZywgdmFsdWVzKSAmJiAhZGVmYXVsdFZhbHVlc1tvcHRpb24ubmFtZV0pIHtcbiAgICAgIHRocm93IG5ldyBEZXBlbmRpbmdPcHRpb24ob3B0aW9uLm5hbWUsIGZsYWcpO1xuICAgIH1cbiAgfSk7XG59XG5cbmZ1bmN0aW9uIHZhbGlkYXRlUmVxdWlyZWRWYWx1ZXMoXG4gIG9wdGlvbjogSUZsYWdPcHRpb25zLFxuICB2YWx1ZXM6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICBuYW1lOiBzdHJpbmcsXG4pOiB2b2lkIHtcbiAgY29uc3QgaXNBcnJheSA9IChvcHRpb24uYXJncz8ubGVuZ3RoIHx8IDApID4gMTtcbiAgb3B0aW9uLmFyZ3M/LmZvckVhY2goKGFyZzogSUZsYWdBcmd1bWVudCwgaTogbnVtYmVyKSA9PiB7XG4gICAgaWYgKFxuICAgICAgYXJnLnJlcXVpcmVkVmFsdWUgJiZcbiAgICAgIChcbiAgICAgICAgdHlwZW9mIHZhbHVlc1tuYW1lXSA9PT0gXCJ1bmRlZmluZWRcIiB8fFxuICAgICAgICAoaXNBcnJheSAmJlxuICAgICAgICAgIHR5cGVvZiAodmFsdWVzW25hbWVdIGFzIEFycmF5PHVua25vd24+KVtpXSA9PT0gXCJ1bmRlZmluZWRcIilcbiAgICAgIClcbiAgICApIHtcbiAgICAgIHRocm93IG5ldyBNaXNzaW5nT3B0aW9uVmFsdWUob3B0aW9uLm5hbWUpO1xuICAgIH1cbiAgfSk7XG59XG5cbmZ1bmN0aW9uIHZhbGlkYXRlUmVxdWlyZWRPcHRpb25zPFQgZXh0ZW5kcyBJRmxhZ09wdGlvbnMgPSBJRmxhZ09wdGlvbnM+KFxuICBvcHRpb25zOiBBcnJheTxJRmxhZ09wdGlvbnNNYXA+LFxuICB2YWx1ZXM6IFJlY29yZDxzdHJpbmcsIHVua25vd24+LFxuICBvcHRzOiBJUGFyc2VPcHRpb25zPFQ+LFxuKTogdm9pZCB7XG4gIGlmICghb3B0cy5mbGFncz8ubGVuZ3RoKSB7XG4gICAgcmV0dXJuO1xuICB9XG4gIGZvciAoY29uc3Qgb3B0aW9uIG9mIG9wdHMuZmxhZ3MpIHtcbiAgICBpZiAob3B0aW9uLnJlcXVpcmVkICYmICEocGFyYW1DYXNlVG9DYW1lbENhc2Uob3B0aW9uLm5hbWUpIGluIHZhbHVlcykpIHtcbiAgICAgIGlmIChcbiAgICAgICAgKFxuICAgICAgICAgICFvcHRpb24uY29uZmxpY3RzIHx8XG4gICAgICAgICAgIW9wdGlvbi5jb25mbGljdHMuZmluZCgoZmxhZzogc3RyaW5nKSA9PiAhIXZhbHVlc1tmbGFnXSlcbiAgICAgICAgKSAmJlxuICAgICAgICAhb3B0aW9ucy5maW5kKChvcHQpID0+XG4gICAgICAgICAgb3B0Lm9wdGlvbj8uY29uZmxpY3RzPy5maW5kKChmbGFnOiBzdHJpbmcpID0+IGZsYWcgPT09IG9wdGlvbi5uYW1lKVxuICAgICAgICApXG4gICAgICApIHtcbiAgICAgICAgdGhyb3cgbmV3IE1pc3NpbmdSZXF1aXJlZE9wdGlvbihvcHRpb24ubmFtZSk7XG4gICAgICB9XG4gICAgfVxuICB9XG59XG5cbi8qKlxuICogQ2hlY2sgaWYgdmFsdWUgZXhpc3RzIGZvciBmbGFnLlxuICogQHBhcmFtIGZsYWcgICAgRmxhZyBuYW1lLlxuICogQHBhcmFtIHZhbHVlcyAgUGFyc2VkIHZhbHVlcy5cbiAqL1xuZnVuY3Rpb24gaXNzZXQoZmxhZzogc3RyaW5nLCB2YWx1ZXM6IFJlY29yZDxzdHJpbmcsIHVua25vd24+KTogYm9vbGVhbiB7XG4gIGNvbnN0IG5hbWUgPSBwYXJhbUNhc2VUb0NhbWVsQ2FzZShmbGFnKTtcbiAgLy8gcmV0dXJuIHR5cGVvZiB2YWx1ZXNbIG5hbWUgXSAhPT0gJ3VuZGVmaW5lZCcgJiYgdmFsdWVzWyBuYW1lIF0gIT09IGZhbHNlO1xuICByZXR1cm4gdHlwZW9mIHZhbHVlc1tuYW1lXSAhPT0gXCJ1bmRlZmluZWRcIjtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLGVBQWUsRUFBRSxTQUFTLEVBQUUsb0JBQW9CLFFBQVEsYUFBYSxDQUFDO0FBQy9FLFNBQ0UsaUJBQWlCLEVBQ2pCLGVBQWUsRUFDZixrQkFBa0IsRUFDbEIscUJBQXFCLEVBQ3JCLG1CQUFtQixFQUNuQixhQUFhLFFBQ1IsY0FBYyxDQUFDO0FBVXRCOzs7Ozs7R0FNRyxDQUNILE9BQU8sU0FBUyxhQUFhLENBQzNCLElBQXNCLEVBQ3RCLE1BQStCLEVBQy9CLGFBQXFDLEdBQUcsRUFBRSxFQUNwQztJQUNOLElBQUksQ0FBQyxJQUFJLENBQUMsS0FBSyxFQUFFLE1BQU0sRUFBRTtRQUN2QixPQUFPO0tBQ1I7SUFDRCxNQUFNLGFBQWEsR0FBRyxnQkFBZ0IsQ0FBQyxJQUFJLEVBQUUsTUFBTSxFQUFFLGFBQWEsQ0FBQyxBQUFDO0lBRXBFLE1BQU0sV0FBVyxHQUFHLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEFBQUM7SUFDeEMsSUFBSSxDQUFDLFdBQVcsQ0FBQyxNQUFNLElBQUksSUFBSSxDQUFDLFVBQVUsRUFBRTtRQUMxQyxPQUFPO0tBQ1I7SUFFRCxNQUFNLE9BQU8sR0FBMkIsV0FBVyxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQUksR0FBSyxDQUFDO1lBQ2pFLElBQUk7WUFDSixNQUFNLEVBQUUsU0FBUyxDQUFDLElBQUksQ0FBQyxLQUFLLEVBQUcsYUFBYSxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ3BELENBQUMsQ0FBQyxBQUFDO0lBRUosS0FBSyxNQUFNLEVBQUUsSUFBSSxDQUFBLEVB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