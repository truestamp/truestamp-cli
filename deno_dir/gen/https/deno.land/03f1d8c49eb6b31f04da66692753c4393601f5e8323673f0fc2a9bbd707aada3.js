import { didYouMean } from "../flags/_utils.ts";
import { ArgumentFollowsVariadicArgument, RequiredArgumentFollowsOptionalArgument, } from "../flags/_errors.ts";
import { OptionType } from "../flags/types.ts";
export function getPermissions() {
    return hasPermissions([
        "env",
        "hrtime",
        "net",
        "plugin",
        "read",
        "run",
        "write",
    ]);
}
export function isUnstable() {
    return !!Deno.permissions;
}
export function didYouMeanCommand(command, commands, excludes = []) {
    const commandNames = commands
        .map((command) => command.getName())
        .filter((command) => !excludes.includes(command));
    return didYouMean(" Did you mean command", command, commandNames);
}
export async function hasPermission(permission) {
    try {
        return (await Deno.permissions?.query?.({ name: permission }))
            ?.state === "granted";
    }
    catch {
        return false;
    }
}
async function hasPermissions(names) {
    const permissions = {};
    await Promise.all(names.map((name) => hasPermission(name).then((hasPermission) => permissions[name] = hasPermission)));
    return permissions;
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
export function parseArgumentsDefinition(argsDefinition) {
    const argumentDetails = [];
    let hasOptional = false;
    let hasVariadic = false;
    const parts = argsDefinition.split(/ +/);
    for (const arg of parts) {
        if (hasVariadic) {
            throw new ArgumentFollowsVariadicArgument(arg);
        }
        const parts = arg.split(ARGUMENT_DETAILS_REGEX);
        const type = parts[2] || OptionType.STRING;
        const details = {
            optionalValue: arg[0] !== "<",
            name: parts[1],
            action: parts[3] || type,
            variadic: false,
            list: type ? arg.indexOf(type + "[]") !== -1 : false,
            type,
        };
        if (!details.optionalValue && hasOptional) {
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
        if (details.name) {
            argumentDetails.push(details);
        }
    }
    return argumentDetails;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUNoRCxPQUFPLEVBQ0wsK0JBQStCLEVBQy9CLHVDQUF1QyxHQUN4QyxNQUFNLHFCQUFxQixDQUFDO0FBQzdCLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQWEvQyxNQUFNLFVBQVUsY0FBYztJQUM1QixPQUFPLGNBQWMsQ0FBQztRQUNwQixLQUFLO1FBQ0wsUUFBUTtRQUNSLEtBQUs7UUFDTCxRQUFRO1FBQ1IsTUFBTTtRQUNOLEtBQUs7UUFDTCxPQUFPO0tBQ1IsQ0FBQyxDQUFDO0FBQ0wsQ0FBQztBQUVELE1BQU0sVUFBVSxVQUFVO0lBRXhCLE9BQU8sQ0FBQyxDQUFFLElBQVksQ0FBQyxXQUFXLENBQUM7QUFDckMsQ0FBQztBQUVELE1BQU0sVUFBVSxpQkFBaUIsQ0FDL0IsT0FBZSxFQUNmLFFBQXdCLEVBQ3hCLFdBQTBCLEVBQUU7SUFFNUIsTUFBTSxZQUFZLEdBQUcsUUFBUTtTQUMxQixHQUFHLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUFDLE9BQU8sQ0FBQyxPQUFPLEVBQUUsQ0FBQztTQUNuQyxNQUFNLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUFDLENBQUMsUUFBUSxDQUFDLFFBQVEsQ0FBQyxPQUFPLENBQUMsQ0FBQyxDQUFDO0lBQ3BELE9BQU8sVUFBVSxDQUFDLHVCQUF1QixFQUFFLE9BQU8sRUFBRSxZQUFZLENBQUMsQ0FBQztBQUNwRSxDQUFDO0FBRUQsTUFBTSxDQUFDLEtBQUssVUFBVSxhQUFhLENBQ2pDLFVBQTBCO0lBRTFCLElBQUk7UUFFRixPQUFPLENBQUMsTUFBTyxJQUFZLENBQUMsV0FBVyxFQUFFLEtBQUssRUFBRSxDQUFDLEVBQUUsSUFBSSxFQUFFLFVBQVUsRUFBRSxDQUFDLENBQUM7WUFDckUsRUFBRSxLQUFLLEtBQUssU0FBUyxDQUFDO0tBQ3pCO0lBQUMsTUFBTTtRQUNOLE9BQU8sS0FBSyxDQUFDO0tBQ2Q7QUFDSCxDQUFDO0FBRUQsS0FBSyxVQUFVLGNBQWMsQ0FDM0IsS0FBVTtJQUVWLE1BQU0sV0FBVyxHQUE0QixFQUFFLENBQUM7SUFDaEQsTUFBTSxPQUFPLENBQUMsR0FBRyxDQUNmLEtBQUssQ0FBQyxHQUFHLENBQUMsQ0FBQyxJQUFPLEVBQUUsRUFBRSxDQUNwQixhQUFhLENBQUMsSUFBSSxDQUFDLENBQUMsSUFBSSxDQUFDLENBQUMsYUFBYSxFQUFFLEVBQUUsQ0FDekMsV0FBVyxDQUFDLElBQUksQ0FBQyxHQUFHLGFBQWEsQ0FDbEMsQ0FDRixDQUNGLENBQUM7SUFDRixPQUFPLFdBQWlDLENBQUM7QUFDM0MsQ0FBQztBQUVELE1BQU0sY0FBYyxHQUFHLGdCQUFnQixDQUFDO0FBQ3hDLE1BQU0sc0JBQXNCLEdBQUcsV0FBVyxDQUFDO0FBZ0IzQyxNQUFNLFVBQVUsY0FBYyxDQUM1QixJQUFZO0lBRVosTUFBTSxLQUFLLEdBQUcsSUFBSSxDQUFDLElBQUksRUFBRSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUMsQ0FBQztJQUM1QyxNQUFNLFNBQVMsR0FBRyxFQUFFLENBQUM7SUFFckIsT0FDRSxLQUFLLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUM7UUFDdkIsY0FBYyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUMsQ0FBQyxFQUM1QztRQUNBLFNBQVMsQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsRUFBRSxDQUFDLENBQUM7S0FDaEM7SUFFRCxNQUFNLGNBQWMsR0FBVyxTQUFTLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBRW5ELE9BQU8sRUFBRSxLQUFLLEVBQUUsS0FBSyxFQUFFLGNBQWMsRUFBRSxDQUFDO0FBQzFDLENBQUM7QUFNRCxNQUFNLFVBQVUsd0JBQXdCLENBQUMsY0FBc0I7SUFDN0QsTUFBTSxlQUFlLEdBQWdCLEVBQUUsQ0FBQztJQUV4QyxJQUFJLFdBQVcsR0FBRyxLQUFLLENBQUM7SUFDeEIsSUFBSSxXQUFXLEdBQUcsS0FBSyxDQUFDO0lBQ3hCLE1BQU0sS0FBSyxHQUFhLGNBQWMsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUM7SUFFbkQsS0FBSyxNQUFNLEdBQUcsSUFBSSxLQUFLLEVBQUU7UUFDdkIsSUFBSSxXQUFXLEVBQUU7WUFDZixNQUFNLElBQUksK0JBQStCLENBQUMsR0FBRyxDQUFDLENBQUM7U0FDaEQ7UUFFRCxNQUFNLEtBQUssR0FBYSxHQUFHLENBQUMsS0FBSyxDQUFDLHNCQUFzQixDQUFDLENBQUM7UUFDMUQsTUFBTSxJQUFJLEdBQXVCLEtBQUssQ0FBQyxDQUFDLENBQUMsSUFBSSxVQUFVLENBQUMsTUFBTSxDQUFDO1FBRS9ELE1BQU0sT0FBTyxHQUFjO1lBQ3pCLGFBQWEsRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRztZQUM3QixJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUMsQ0FBQztZQUNkLE1BQU0sRUFBRSxLQUFLLENBQUMsQ0FBQyxDQUFDLElBQUksSUFBSTtZQUN4QixRQUFRLEVBQUUsS0FBSztZQUNmLElBQUksRUFBRSxJQUFJLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxPQUFPLENBQUMsSUFBSSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxLQUFLO1lBQ3BELElBQUk7U0FDTCxDQUFDO1FBRUYsSUFBSSxDQUFDLE9BQU8sQ0FBQyxhQUFhLElBQUksV0FBVyxFQUFFO1lBQ3pDLE1BQU0sSUFBSSx1Q0FBdUMsQ0FBQyxPQUFPLENBQUMsSUFBSSxDQUFDLENBQUM7U0FDakU7UUFFRCxJQUFJLEdBQUcsQ0FBQyxDQUFDLENBQUMsS0FBSyxHQUFHLEVBQUU7WUFDbEIsV0FBVyxHQUFHLElBQUksQ0FBQztTQUNwQjtRQUVELElBQUksT0FBTyxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxFQUFFO1lBQzNCLE1BQU0sZUFBZSxHQUFHLE9BQU8sQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsS0FBSyxLQUFLLENBQUM7WUFDM0QsTUFBTSxnQkFBZ0IsR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQyxLQUFLLEtBQUssQ0FBQztZQUUxRCxXQUFXLEdBQUcsT0FBTyxDQUFDLFFBQVEsR0FBRyxlQUFlLElBQUksZ0JBQWdCLENBQUM7WUFFckUsSUFBSSxlQUFlLEVBQUU7Z0JBQ25CLE9BQU8sQ0FBQyxJQUFJLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUM7YUFDdEM7aUJBQU0sSUFBSSxnQkFBZ0IsRUFBRTtnQkFDM0IsT0FBTyxDQUFDLElBQUksR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUMsQ0FBQzthQUMxQztTQUNGO1FBRUQsSUFBSSxPQUFPLENBQUMsSUFBSSxFQUFFO1lBQ2hCLGVBQWUsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLENBQUM7U0FDL0I7S0FDRjtJQUVELE9BQU8sZUFBZSxDQUFDO0FBQ3pCLENBQUMifQ==