export const number = ({ label, name, value, type }) => {
    if (isNaN(Number(value))) {
        throw new Error(`${label} "${name}" must be of type "${type}", but got "${value}".`);
    }
    return parseFloat(value);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibnVtYmVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibnVtYmVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sQ0FBQyxNQUFNLE1BQU0sR0FBeUIsQ0FDMUMsRUFBRSxLQUFLLEVBQUUsSUFBSSxFQUFFLEtBQUssRUFBRSxJQUFJLEVBQWEsRUFDL0IsRUFBRTtJQUNWLElBQUksS0FBSyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUFFO1FBQ3hCLE1BQU0sSUFBSSxLQUFLLENBQ2IsR0FBRyxLQUFLLEtBQUssSUFBSSxzQkFBc0IsSUFBSSxlQUFlLEtBQUssSUFBSSxDQUNwRSxDQUFDO0tBQ0g7SUFFRCxPQUFPLFVBQVUsQ0FBQyxLQUFLLENBQUMsQ0FBQztBQUMzQixDQUFDLENBQUMifQ==