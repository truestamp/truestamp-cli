export const parseBoolean = (value) => {
    switch (value) {
        case "true":
            return true;
        case "false":
            return false;
        default:
            throw new Error(`Unable to parse boolean value "${value}"`);
    }
};
export const expectBoolean = (value) => {
    if (value === null || value === undefined) {
        return undefined;
    }
    if (typeof value === "boolean") {
        return value;
    }
    throw new TypeError(`Expected boolean, got ${typeof value}`);
};
export const expectNumber = (value) => {
    if (value === null || value === undefined) {
        return undefined;
    }
    if (typeof value === "number") {
        return value;
    }
    throw new TypeError(`Expected number, got ${typeof value}`);
};
export const expectString = (value) => {
    if (value === null || value === undefined) {
        return undefined;
    }
    if (typeof value === "string") {
        return value;
    }
    throw new TypeError(`Expected string, got ${typeof value}`);
};
export const handleFloat = (value) => {
    if (typeof value == "string") {
        switch (value) {
            case "NaN":
                return NaN;
            case "Infinity":
                return Infinity;
            case "-Infinity":
                return -Infinity;
            default:
                throw new Error(`Unable to parse float value: ${value}`);
        }
    }
    return expectNumber(value);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGFyc2UtdXRpbHMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJwYXJzZS11dGlscy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFNQSxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQUcsQ0FBQyxLQUFhLEVBQVcsRUFBRTtJQUNyRCxRQUFRLEtBQUssRUFBRTtRQUNiLEtBQUssTUFBTTtZQUNULE9BQU8sSUFBSSxDQUFDO1FBQ2QsS0FBSyxPQUFPO1lBQ1YsT0FBTyxLQUFLLENBQUM7UUFDZjtZQUNFLE1BQU0sSUFBSSxLQUFLLENBQUMsa0NBQWtDLEtBQUssR0FBRyxDQUFDLENBQUM7S0FDL0Q7QUFDSCxDQUFDLENBQUM7QUFTRixNQUFNLENBQUMsTUFBTSxhQUFhLEdBQUcsQ0FBQyxLQUFVLEVBQXVCLEVBQUU7SUFDL0QsSUFBSSxLQUFLLEtBQUssSUFBSSxJQUFJLEtBQUssS0FBSyxTQUFTLEVBQUU7UUFDekMsT0FBTyxTQUFTLENBQUM7S0FDbEI7SUFDRCxJQUFJLE9BQU8sS0FBSyxLQUFLLFNBQVMsRUFBRTtRQUM5QixPQUFPLEtBQUssQ0FBQztLQUNkO0lBQ0QsTUFBTSxJQUFJLFNBQVMsQ0FBQyx5QkFBeUIsT0FBTyxLQUFLLEVBQUUsQ0FBQyxDQUFDO0FBQy9ELENBQUMsQ0FBQztBQVNGLE1BQU0sQ0FBQyxNQUFNLFlBQVksR0FBRyxDQUFDLEtBQVUsRUFBc0IsRUFBRTtJQUM3RCxJQUFJLEtBQUssS0FBSyxJQUFJLElBQUksS0FBSyxLQUFLLFNBQVMsRUFBRTtRQUN6QyxPQUFPLFNBQVMsQ0FBQztLQUNsQjtJQUNELElBQUksT0FBTyxLQUFLLEtBQUssUUFBUSxFQUFFO1FBQzdCLE9BQU8sS0FBSyxDQUFDO0tBQ2Q7SUFDRCxNQUFNLElBQUksU0FBUyxDQUFDLHdCQUF3QixPQUFPLEtBQUssRUFBRSxDQUFDLENBQUM7QUFDOUQsQ0FBQyxDQUFDO0FBU0YsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLENBQUMsS0FBVSxFQUFzQixFQUFFO0lBQzdELElBQUksS0FBSyxLQUFLLElBQUksSUFBSSxLQUFLLEtBQUssU0FBUyxFQUFFO1FBQ3pDLE9BQU8sU0FBUyxDQUFDO0tBQ2xCO0lBQ0QsSUFBSSxPQUFPLEtBQUssS0FBSyxRQUFRLEVBQUU7UUFDN0IsT0FBTyxLQUFLLENBQUM7S0FDZDtJQUNELE1BQU0sSUFBSSxTQUFTLENBQUMsd0JBQXdCLE9BQU8sS0FBSyxFQUFFLENBQUMsQ0FBQztBQUM5RCxDQUFDLENBQUM7QUFVRixNQUFNLENBQUMsTUFBTSxXQUFXLEdBQUcsQ0FBQyxLQUFzQixFQUFzQixFQUFFO0lBQ3hFLElBQUksT0FBTyxLQUFLLElBQUksUUFBUSxFQUFFO1FBQzVCLFFBQVEsS0FBSyxFQUFFO1lBQ2IsS0FBSyxLQUFLO2dCQUNSLE9BQU8sR0FBRyxDQUFDO1lBQ2IsS0FBSyxVQUFVO2dCQUNiLE9BQU8sUUFBUSxDQUFDO1lBQ2xCLEtBQUssV0FBVztnQkFDZCxPQUFPLENBQUMsUUFBUSxDQUFDO1lBQ25CO2dCQUNFLE1BQU0sSUFBSSxLQUFLLENBQUMsZ0NBQWdDLEtBQUssRUFBRSxDQUFDLENBQUM7U0FDNUQ7S0FDRjtJQUNELE9BQU8sWUFBWSxDQUFDLEtBQUssQ0FBQyxDQUFDO0FBQzdCLENBQUMsQ0FBQyJ9