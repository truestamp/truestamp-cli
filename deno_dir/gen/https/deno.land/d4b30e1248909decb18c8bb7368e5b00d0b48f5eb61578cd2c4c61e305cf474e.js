import { didYouMean } from "../flags/_utils.ts";
import { ArgumentFollowsVariadicArgument, RequiredArgumentFollowsOptionalArgument, } from "../flags/_errors.ts";
import { OptionType } from "../flags/types.ts";
export function didYouMeanCommand(command, commands, excludes = []) {
    const commandNames = commands
        .map((command) => command.getName())
        .filter((command) => !excludes.includes(command));
    return didYouMean(" Did you mean command", command, commandNames);
}
const ARGUMENT_REGEX = /^[<\[].+[\]>]$/;
const ARGUMENT_DETAILS_REGEX = /[<\[:>\]]/;
export function splitArguments(args) {
    const parts = args.trim().split(/[, =] */g);
    const typeParts = [];
    while (parts[parts.length - 1] &&
        ARGUMENT_REGEX.test(parts[parts.length - 1])) {
        typeParts.unshift(parts.pop());
    }
    const typeDefinition = typeParts.join(" ");
    return { flags: parts, typeDefinition };
}
export function parseArgumentsDefinition(argsDefinition, validate = true, all) {
    const argumentDetails = [];
    let hasOptional = false;
    let hasVariadic = false;
    const parts = argsDefinition.split(/ +/);
    for (const arg of parts) {
        if (validate && hasVariadic) {
            throw new ArgumentFollowsVariadicArgument(arg);
        }
        const parts = arg.split(ARGUMENT_DETAILS_REGEX);
        if (!parts[1]) {
            if (all) {
                argumentDetails.push(parts[0]);
            }
            continue;
        }
        const type = parts[2] || OptionType.STRING;
        const details = {
            optionalValue: arg[0] !== "<",
            name: parts[1],
            action: parts[3] || type,
            variadic: false,
            list: type ? arg.indexOf(type + "[]") !== -1 : false,
            type,
        };
        if (validate && !details.optionalValue && hasOptional) {
            throw new RequiredArgumentFollowsOptionalArgument(details.name);
        }
        if (arg[0] === "[") {
            hasOptional = true;
        }
        if (details.name.length > 3) {
            const istVariadicLeft = details.name.slice(0, 3) === "...";
            const istVariadicRight = details.name.slice(-3) === "...";
            hasVariadic = details.variadic = istVariadicLeft || istVariadicRight;
            if (istVariadicLeft) {
                details.name = details.name.slice(3);
            }
            else if (istVariadicRight) {
                details.name = details.name.slice(0, -3);
            }
        }
        argumentDetails.push(details);
    }
    return argumentDetails;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUNoRCxPQUFPLEVBQ0wsK0JBQStCLEVBQy9CLHVDQUF1QyxHQUN4QyxNQUFNLHFCQUFxQixDQUFDO0FBQzdCLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUkvQyxNQUFNLFVBQVUsaUJBQWlCLENBQy9CLE9BQWUsRUFDZixRQUF3QixFQUN4QixXQUEwQixFQUFFO0lBRTVCLE1BQU0sWUFBWSxHQUFHLFFBQVE7U0FDMUIsR0FBRyxDQUFDLENBQUMsT0FBTyxFQUFFLEVBQUUsQ0FBQyxPQUFPLENBQUMsT0FBTyxFQUFFLENBQUM7U0FDbkMsTUFBTSxDQUFDLENBQUMsT0FBTyxFQUFFLEVBQUUsQ0FBQyxDQUFDLFFBQVEsQ0FBQyxRQUFRLENBQUMsT0FBTyxDQUFDLENBQUMsQ0FBQztJQUNwRCxPQUFPLFVBQVUsQ0FBQyx1QkFBdUIsRUFBRSxPQUFPLEVBQUUsWUFBWSxDQUFDLENBQUM7QUFDcEUsQ0FBQztBQUVELE1BQU0sY0FBYyxHQUFHLGdCQUFnQixDQUFDO0FBQ3hDLE1BQU0sc0JBQXNCLEdBQUcsV0FBVyxDQUFDO0FBZ0IzQyxNQUFNLFVBQVUsY0FBYyxDQUM1QixJQUFZO0lBRVosTUFBTSxLQUFLLEdBQUcsSUFBSSxDQUFDLElBQUksRUFBRSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUMsQ0FBQztJQUM1QyxNQUFNLFNBQVMsR0FBRyxFQUFFLENBQUM7SUFFckIsT0FDRSxLQUFLLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUM7UUFDdkIsY0FBYyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUMsQ0FBQyxFQUM1QztRQUNBLFNBQVMsQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsRUFBRSxDQUFDLENBQUM7S0FDaEM7SUFFRCxNQUFNLGNBQWMsR0FBVyxTQUFTLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBRW5ELE9BQU8sRUFBRSxLQUFLLEVBQUUsS0FBSyxFQUFFLGNBQWMsRUFBRSxDQUFDO0FBQzFDLENBQUM7QUFnQkQsTUFBTSxVQUFVLHdCQUF3QixDQUN0QyxjQUFzQixFQUN0QixRQUFRLEdBQUcsSUFBSSxFQUNmLEdBQU87SUFFUCxNQUFNLGVBQWUsR0FBOEIsRUFBRSxDQUFDO0lBRXRELElBQUksV0FBVyxHQUFHLEtBQUssQ0FBQztJQUN4QixJQUFJLFdBQVcsR0FBRyxLQUFLLENBQUM7SUFDeEIsTUFBTSxLQUFLLEdBQWEsY0FBYyxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUVuRCxLQUFLLE1BQU0sR0FBRyxJQUFJLEtBQUssRUFBRTtRQUN2QixJQUFJLFFBQVEsSUFBSSxXQUFXLEVBQUU7WUFDM0IsTUFBTSxJQUFJLCtCQUErQixDQUFDLEdBQUcsQ0FBQyxDQUFDO1NBQ2hEO1FBQ0QsTUFBTSxLQUFLLEdBQWEsR0FBRyxDQUFDLEtBQUssQ0FBQyxzQkFBc0IsQ0FBQyxDQUFDO1FBRTFELElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLEVBQUU7WUFDYixJQUFJLEdBQUcsRUFBRTtnQkFDUCxlQUFlLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO2FBQ2hDO1lBQ0QsU0FBUztTQUNWO1FBQ0QsTUFBTSxJQUFJLEdBQXVCLEtBQUssQ0FBQyxDQUFDLENBQUMsSUFBSSxVQUFVLENBQUMsTUFBTSxDQUFDO1FBRS9ELE1BQU0sT0FBTyxHQUFjO1lBQ3pCLGFBQWEsRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRztZQUM3QixJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUMsQ0FBQztZQUNkLE1BQU0sRUFBRSxLQUFLLENBQUMsQ0FBQyxDQUFDLElBQUksSUFBSTtZQUN4QixRQUFRLEVBQUUsS0FBSztZQUNmLElBQUksRUFBRSxJQUFJLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxPQUFPLENBQUMsSUFBSSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxLQUFLO1lBQ3BELElBQUk7U0FDTCxDQUFDO1FBRUYsSUFBSSxRQUFRLElBQUksQ0FBQyxPQUFPLENBQUMsYUFBYSxJQUFJLFdBQVcsRUFBRTtZQUNyRCxNQUFNLElBQUksdUNBQXVDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ2pFO1FBRUQsSUFBSSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRyxFQUFFO1lBQ2xCLFdBQVcsR0FBRyxJQUFJLENBQUM7U0FDcEI7UUFFRCxJQUFJLE9BQU8sQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsRUFBRTtZQUMzQixNQUFNLGVBQWUsR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLEtBQUssS0FBSyxDQUFDO1lBQzNELE1BQU0sZ0JBQWdCLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUMsS0FBSyxLQUFLLENBQUM7WUFFMUQsV0FBVyxHQUFHLE9BQU8sQ0FBQyxRQUFRLEdBQUcsZUFBZSxJQUFJLGdCQUFnQixDQUFDO1lBRXJFLElBQUksZUFBZSxFQUFFO2dCQUNuQixPQUFPLENBQUMsSUFBSSxHQUFHLE9BQU8sQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDO2FBQ3RDO2lCQUFNLElBQUksZ0JBQWdCLEVBQUU7Z0JBQzNCLE9BQU8sQ0FBQyxJQUFJLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUFFLENBQUMsQ0FBQyxDQUFDLENBQUM7YUFDMUM7U0FDRjtRQUVELGVBQWUsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLENBQUM7S0FDL0I7SUFFRCxPQUFPLGVBRU4sQ0FBQztBQUNKLENBQUMiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBkaWRZb3VNZWFuIH0gZnJvbSBcIi4uL2ZsYWdzL191dGlscy50c1wiO1xuaW1wb3J0IHtcbiAgQXJndW1lbnRGb2xsb3dzVmFyaWFkaWNBcmd1bWVudCxcbiAgUmVxdWlyZWRBcmd1bWVudEZvbGxvd3NPcHRpb25hbEFyZ3VtZW50LFxufSBmcm9tIFwiLi4vZmxhZ3MvX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgT3B0aW9uVHlwZSB9IGZyb20gXCIuLi9mbGFncy90eXBlcy50c1wiO1xuaW1wb3J0IHR5cGUgeyBDb21tYW5kIH0gZnJvbSBcIi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHR5cGUgeyBJQXJndW1lbnQgfSBmcm9tIFwiLi90eXBlcy50c1wiO1xuXG5leHBvcnQgZnVuY3Rpb24gZGlkWW91TWVhbkNvbW1hbmQoXG4gIGNvbW1hbmQ6IHN0cmluZyxcbiAgY29tbWFuZHM6IEFycmF5PENvbW1hbmQ+LFxuICBleGNsdWRlczogQXJyYXk8c3RyaW5nPiA9IFtdLFxuKTogc3RyaW5nIHtcbiAgY29uc3QgY29tbWFuZE5hbWVzID0gY29tbWFuZHNcbiAgICAubWFwKChjb21tYW5kKSA9PiBjb21tYW5kLmdldE5hbWUoKSlcbiAgICAuZmlsdGVyKChjb21tYW5kKSA9PiAhZXhjbHVkZXMuaW5jbHVkZXMoY29tbWFuZCkpO1xuICByZXR1cm4gZGlkWW91TWVhbihcIiBEaWQgeW91IG1lYW4gY29tbWFuZFwiLCBjb21tYW5kLCBjb21tYW5kTmFtZXMpO1xufVxuXG5jb25zdCBBUkdVTUVOVF9SRUdFWCA9IC9eWzxcXFtdLitbXFxdPl0kLztcbmNvbnN0IEFSR1VNRU5UX0RFVEFJTFNfUkVHRVggPSAvWzxcXFs6PlxcXV0vO1xuXG4vKipcbiAqIFNwbGl0IG9wdGlvbnMgYW5kIGFyZ3VtZW50cy5cbiAqIEBwYXJhbSBhcmdzIEFyZ3VtZW50cyBkZWZpbml0aW9uOiBgLS1jb2xvciwgLWMgPGNvbG9yMTpzdHJpbmc+IDxjb2xvcjI6c3RyaW5nPmBcbiAqXG4gKiBGb3IgZXhhbXBsZTogYC1jLCAtLWNvbG9yIDxjb2xvcjE6c3RyaW5nPiA8Y29sb3IyOnN0cmluZz5gXG4gKlxuICogV2lsbCByZXN1bHQgaW46XG4gKiBgYGBcbiAqIHtcbiAqICAgZmxhZ3M6IFsgXCItY1wiLCBcIi0tY29sb3JcIiBdLFxuICogICB0eXBlRGVmaW5pdGlvbjogXCI8Y29sb3IxOnN0cmluZz4gPGNvbG9yMjpzdHJpbmc+XCJcbiAqIH1cbiAqIGBgYFxuICovXG5leHBvcnQgZnVuY3Rpb24gc3BsaXRBcmd1bWVudHMoXG4gIGFyZ3M6IHN0cmluZyxcbik6IHsgZmxhZ3M6IHN0cmluZ1tdOyB0eXBlRGVmaW5pdGlvbjogc3RyaW5nIH0ge1xuICBjb25zdCBwYXJ0cyA9IGFyZ3MudHJpbSgpLnNwbGl0KC9bLCA9XSAqL2cpO1xuICBjb25zdCB0eXBlUGFydHMgPSBbXTtcblxuICB3aGlsZSAoXG4gICAgcGFydHNbcGFydHMubGVuZ3RoIC0gMV0gJiZcbiAgICBBUkdVTUVOVF9SRUdFWC50ZXN0KHBhcnRzW3BhcnRzLmxlbmd0aCAtIDFdKVxuICApIHtcbiAgICB0eXBlUGFydHMudW5zaGlmdChwYXJ0cy5wb3AoKSk7XG4gIH1cblxuICBjb25zdCB0eXBlRGVmaW5pdGlvbjogc3RyaW5nID0gdHlwZVBhcnRzLmpvaW4oXCIgXCIpO1xuXG4gIHJldHVybiB7IGZsYWdzOiBwYXJ0cywgdHlwZURlZmluaXRpb24gfTtcbn1cblxuLyoqXG4gKiBQYXJzZSBhcmd1bWVudHMgc3RyaW5nLlxuICogQHBhcmFtIGFyZ3NEZWZpbml0aW9uIEFyZ3VtZW50cyBkZWZpbml0aW9uOiBgPGNvbG9yMTpzdHJpbmc+IDxjb2xvcjI6c3RyaW5nPmBcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbjxUIGV4dGVuZHMgYm9vbGVhbj4oXG4gIGFyZ3NEZWZpbml0aW9uOiBzdHJpbmcsXG4gIHZhbGlkYXRlOiBib29sZWFuLFxuICBhbGw6IHRydWUsXG4pOiBBcnJheTxJQXJndW1lbnQgfCBzdHJpbmc+O1xuZXhwb3J0IGZ1bmN0aW9uIHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbjxUIGV4dGVuZHMgYm9vbGVhbj4oXG4gIGFyZ3NEZWZpbml0aW9uOiBzdHJpbmcsXG4gIHZhbGlkYXRlPzogYm9vbGVhbixcbiAgYWxsPzogZmFsc2UsXG4pOiBBcnJheTxJQXJndW1lbnQ+O1xuZXhwb3J0IGZ1bmN0aW9uIHBhcnNlQXJndW1lbnRzRGVmaW5pdGlvbjxUIGV4dGVuZHMgYm9vbGVhbj4oXG4gIGFyZ3NEZWZpbml0aW9uOiBzdHJpbmcsXG4gIHZhbGlkYXRlID0gdHJ1ZSxcbiAgYWxsPzogVCxcbik6IFQgZXh0ZW5kcyB0cnVlID8gQXJyYXk8SUFyZ3VtZW50IHwgc3RyaW5nPiA6IEFycmF5PElBcmd1bWVudD4ge1xuICBjb25zdCBhcmd1bWVudERldGFpbHM6IEFycmF5PElBcmd1bWVudCB8IHN0cmluZz4gPSBbXTtcblxuICBsZXQgaGFzT3B0aW9uYWwgPSBmYWxzZTtcbiAgbGV0IGhhc1ZhcmlhZGljID0gZmFsc2U7XG4gIGNvbnN0IHBhcnRzOiBzdHJpbmdbXSA9IGFyZ3NEZWZpbml0aW9uLnNwbGl0KC8gKy8pO1xuXG4gIGZvciAoY29uc3QgYXJnIG9mIHBhcnRzKSB7XG4gICAgaWYgKHZhbGlkYXRlICYmIGhhc1ZhcmlhZGljKSB7XG4gICAgICB0aHJvdyBuZXcgQXJndW1lbnRGb2xsb3dzVmFyaWFkaWNBcmd1bWVudChhcmcpO1xuICAgIH1cbiAgICBjb25zdCBwYXJ0czogc3RyaW5nW10gPSBhcmcuc3BsaXQoQVJHVU1FTlRfREVUQUlMU19SRUdFWCk7XG5cbiAgICBpZiAoIXBhcnRzWzFdKSB7XG4gICAgICBpZiAoYWxsKSB7XG4gICAgICAgIGFyZ3VtZW50RGV0YWlscy5wdXNoKHBhcnRzWzBdKTtcbiAgICAgIH1cbiAgICAgIGNvbnRpbnVlO1xuICAgIH1cbiAgICBjb25zdCB0eXBlOiBzdHJpbmcgfCB1bmRlZmluZWQgPSBwYXJ0c1syXSB8fCBPcHRpb25UeXBlLlNUUklORztcblxuICAgIGNvbnN0IGRldGFpbHM6IElBcmd1bWVudCA9IHtcbiAgICAgIG9wdGlvbmFsVmFsdWU6IGFyZ1swXSAhPT0gXCI8XCIsXG4gICAgICBuYW1lOiBwYXJ0c1sxXSxcbiAgICAgIGFjdGlvbjogcGFydHNbM10gfHwgdHlwZSxcbiAgICAgIHZhcmlhZGljOiBmYWxzZSxcbiAgICAgIGxpc3Q6IHR5cGUgPyBhcmcuaW5kZXhPZih0eXBlICsgXCJbXVwiKSAhPT0gLTEgOiBmYWxzZSxcbiAgICAgIHR5cGUsXG4gICAgfTtcblxuICAgIGlmICh2YWxpZGF0ZSAmJiAhZGV0YWlscy5vcHRpb25hbFZhbHVlICYmIGhhc09wdGlvbmFsKSB7XG4gICAgICB0aHJvdyBuZXcgUmVxdWlyZWRBcmd1bWVudEZvbGxvd3NPcHRpb25hbEFyZ3VtZW50KGRldGFpbHMubmFtZSk7XG4gICAgfVxuXG4gICAgaWYgKGFyZ1swXSA9PT0gXCJbXCIpIHtcbiAgICAgIGhhc09wdGlvbmFsID0gdHJ1ZTtcbiAgICB9XG5cbiAgICBpZiAoZGV0YWlscy5uYW1lLmxlbmd0aCA+IDMpIHtcbiAgICAgIGNvbnN0IGlzdFZhcmlhZGljTGVmdCA9IGRldGFpbHMubmFtZS5zbGljZSgwLCAzKSA9PT0gXCIuLi5cIjtcbiAgICAgIGNvbnN0IGlzdFZhcmlhZGljUmlnaHQgPSBkZXRhaWxzLm5hbWUuc2xpY2UoLTMpID09PSBcIi4uLlwiO1xuXG4gICAgICBoYXNWYXJpYWRpYyA9IGRldGFpbHMudmFyaWFkaWMgPSBpc3RWYXJpYWRpY0xlZnQgfHwgaXN0VmFyaWFkaWNSaWdodDtcblxuICAgICAgaWYgKGlzdFZhcmlhZGljTGVmdCkge1xuICAgICAgICBkZXRhaWxzLm5hbWUgPSBkZXRhaWxzLm5hbWUuc2xpY2UoMyk7XG4gICAgICB9IGVsc2UgaWYgKGlzdFZhcmlhZGljUmlnaHQpIHtcbiAgICAgICAgZGV0YWlscy5uYW1lID0gZGV0YWlscy5uYW1lLnNsaWNlKDAsIC0zKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBhcmd1bWVudERldGFpbHMucHVzaChkZXRhaWxzKTtcbiAgfVxuXG4gIHJldHVybiBhcmd1bWVudERldGFpbHMgYXMgKFxuICAgIFQgZXh0ZW5kcyB0cnVlID8gQXJyYXk8SUFyZ3VtZW50IHwgc3RyaW5nPiA6IEFycmF5PElBcmd1bWVudD5cbiAgKTtcbn1cbiJdfQ==