import { getDefaultValue, getOption, paramCaseToCamelCase } from "./_utils.ts";
import { ConflictingOption, DependingOption, MissingOptionValue, MissingRequiredOption, NoArguments, OptionNotCombinable, UnknownOption, } from "./_errors.ts";
export function validateFlags(flags, values, _knownFlaks, allowEmpty, optionNames = {}) {
    const defaultValues = {};
    for (const option of flags) {
        let name;
        let defaultValue = undefined;
        if (option.name.startsWith("no-")) {
            const propName = option.name.replace(/^no-/, "");
            if (propName in values) {
                continue;
            }
            const positiveOption = getOption(flags, propName);
            if (positiveOption) {
                continue;
            }
            name = paramCaseToCamelCase(propName);
            defaultValue = true;
        }
        if (!name) {
            name = paramCaseToCamelCase(option.name);
        }
        if (!(name in optionNames)) {
            optionNames[name] = option.name;
        }
        const hasDefaultValue = typeof values[name] === "undefined" && (typeof option.default !== "undefined" ||
            typeof defaultValue !== "undefined");
        if (hasDefaultValue) {
            values[name] = getDefaultValue(option) ?? defaultValue;
            defaultValues[option.name] = true;
            if (typeof option.value === "function") {
                values[name] = option.value(values[name]);
            }
        }
    }
    const keys = Object.keys(values);
    if (keys.length === 0 && allowEmpty) {
        return;
    }
    const options = keys.map((name) => ({
        name,
        option: getOption(flags, optionNames[name]),
    }));
    for (const { name, option } of options) {
        if (!option) {
            throw new UnknownOption(name, flags);
        }
        if (option.standalone) {
            if (keys.length > 1) {
                if (options.every(({ option: opt }) => opt &&
                    (option === opt || defaultValues[opt.name]))) {
                    return;
                }
                throw new OptionNotCombinable(option.name);
            }
            return;
        }
        option.conflicts?.forEach((flag) => {
            if (isset(flag, values)) {
                throw new ConflictingOption(option.name, flag);
            }
        });
        option.depends?.forEach((flag) => {
            if (!isset(flag, values) && !defaultValues[option.name]) {
                throw new DependingOption(option.name, flag);
            }
        });
        const isArray = (option.args?.length || 0) > 1;
        option.args?.forEach((arg, i) => {
            if (arg.requiredValue &&
                (typeof values[name] === "undefined" ||
                    (isArray &&
                        typeof values[name][i] === "undefined"))) {
                throw new MissingOptionValue(option.name);
            }
        });
    }
    for (const option of flags) {
        if (option.required && !(paramCaseToCamelCase(option.name) in values)) {
            if ((!option.conflicts ||
                !option.conflicts.find((flag) => !!values[flag])) &&
                !options.find((opt) => opt.option?.conflicts?.find((flag) => flag === option.name))) {
                throw new MissingRequiredOption(option.name);
            }
        }
    }
    if (keys.length === 0 && !allowEmpty) {
        throw new NoArguments();
    }
}
function isset(flag, values) {
    const name = paramCaseToCamelCase(flag);
    return typeof values[name] !== "undefined";
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidmFsaWRhdGVfZmxhZ3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJ2YWxpZGF0ZV9mbGFncy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZUFBZSxFQUFFLFNBQVMsRUFBRSxvQkFBb0IsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsaUJBQWlCLEVBQ2pCLGVBQWUsRUFDZixrQkFBa0IsRUFDbEIscUJBQXFCLEVBQ3JCLFdBQVcsRUFDWCxtQkFBbUIsRUFDbkIsYUFBYSxHQUNkLE1BQU0sY0FBYyxDQUFDO0FBb0J0QixNQUFNLFVBQVUsYUFBYSxDQUMzQixLQUFxQixFQUNyQixNQUErQixFQUMvQixXQUFxQyxFQUNyQyxVQUFvQixFQUNwQixjQUFzQyxFQUFFO0lBRXhDLE1BQU0sYUFBYSxHQUE0QixFQUFFLENBQUM7SUFHbEQsS0FBSyxNQUFNLE1BQU0sSUFBSSxLQUFLLEVBQUU7UUFDMUIsSUFBSSxJQUF3QixDQUFDO1FBQzdCLElBQUksWUFBWSxHQUFZLFNBQVMsQ0FBQztRQUd0QyxJQUFJLE1BQU0sQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLEtBQUssQ0FBQyxFQUFFO1lBQ2pDLE1BQU0sUUFBUSxHQUFHLE1BQU0sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sRUFBRSxFQUFFLENBQUMsQ0FBQztZQUNqRCxJQUFJLFFBQVEsSUFBSSxNQUFNLEVBQUU7Z0JBQ3RCLFNBQVM7YUFDVjtZQUNELE1BQU0sY0FBYyxHQUFHLFNBQVMsQ0FBQyxLQUFLLEVBQUUsUUFBUSxDQUFDLENBQUM7WUFDbEQsSUFBSSxjQUFjLEVBQUU7Z0JBQ2xCLFNBQVM7YUFDVjtZQUNELElBQUksR0FBRyxvQkFBb0IsQ0FBQyxRQUFRLENBQUMsQ0FBQztZQUN0QyxZQUFZLEdBQUcsSUFBSSxDQUFDO1NBQ3JCO1FBRUQsSUFBSSxDQUFDLElBQUksRUFBRTtZQUNULElBQUksR0FBRyxvQkFBb0IsQ0FBQyxNQUFNLENB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