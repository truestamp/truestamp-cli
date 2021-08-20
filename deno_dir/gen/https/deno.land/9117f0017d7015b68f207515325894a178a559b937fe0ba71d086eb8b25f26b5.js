import { getFlag } from "../../flags/_utils.ts";
import { Table } from "../../table/table.ts";
import { parseArgumentsDefinition } from "../_utils.ts";
import { blue, bold, dim, getColorEnabled, green, italic, magenta, red, setColorEnabled, yellow, } from "../deps.ts";
import { Type } from "../type.ts";
export class HelpGenerator {
    cmd;
    indent = 2;
    options;
    static generate(cmd, options) {
        return new HelpGenerator(cmd, options).generate();
    }
    constructor(cmd, options = {}) {
        this.cmd = cmd;
        this.options = {
            types: false,
            hints: true,
            colors: true,
            ...options,
        };
    }
    generate() {
        const areColorsEnabled = getColorEnabled();
        setColorEnabled(this.options.colors);
        const result = this.generateHeader() +
            this.generateDescription() +
            this.generateOptions() +
            this.generateCommands() +
            this.generateEnvironmentVariables() +
            this.generateExamples();
        setColorEnabled(areColorsEnabled);
        return result;
    }
    generateHeader() {
        const rows = [
            [
                bold("Usage:"),
                magenta(`${this.cmd.getPath()}${this.cmd.getArgsDefinition()
                    ? " " + this.cmd.getArgsDefinition()
                    : ""}`),
            ],
        ];
        const version = this.cmd.getVersion();
        if (version) {
            rows.push([bold("Version:"), yellow(`${this.cmd.getVersion()}`)]);
        }
        return "\n" +
            Table.from(rows)
                .indent(this.indent)
                .padding(1)
                .toString() +
            "\n";
    }
    generateDescription() {
        if (!this.cmd.getDescription()) {
            return "";
        }
        return this.label("Description") +
            Table.from([
                [this.cmd.getDescription()],
            ])
                .indent(this.indent * 2)
                .maxColWidth(140)
                .padding(1)
                .toString() +
            "\n";
    }
    generateOptions() {
        const options = this.cmd.getOptions(false);
        if (!options.length) {
            return "";
        }
        const hasTypeDefinitions = !!options.find((option) => !!option.typeDefinition);
        if (hasTypeDefinitions) {
            return this.label("Options") +
                Table.from([
                    ...options.map((option) => [
                        option.flags.map((flag) => blue(flag)).join(", "),
                        highlightArguments(option.typeDefinition || "", this.options.types),
                        red(bold("-")) + " " +
                            option.description.split("\n").shift(),
                        this.generateHints(option),
                    ]),
                ])
                    .padding([2, 2, 2])
                    .indent(this.indent * 2)
                    .maxColWidth([60, 60, 80, 60])
                    .toString() +
                "\n";
        }
        return this.label("Options") +
            Table.from([
                ...options.map((option) => [
                    option.flags.map((flag) => blue(flag)).join(", "),
                    red(bold("-")) + " " +
                        option.description.split("\n").shift(),
                    this.generateHints(option),
                ]),
            ])
                .padding([2, 2])
                .indent(this.indent * 2)
                .maxColWidth([60, 80, 60])
                .toString() +
            "\n";
    }
    generateCommands() {
        const commands = this.cmd.getCommands(false);
        if (!commands.length) {
            return "";
        }
        const hasTypeDefinitions = !!commands.find((command) => !!command.getArgsDefinition());
        if (hasTypeDefinitions) {
            return this.label("Commands") +
                Table.from([
                    ...commands.map((command) => [
                        [command.getName(), ...command.getAliases()].map((name) => blue(name)).join(", "),
                        highlightArguments(command.getArgsDefinition() || "", this.options.types),
                        red(bold("-")) + " " +
                            command.getDescription().split("\n").shift(),
                    ]),
                ])
                    .padding([2, 2, 2])
                    .indent(this.indent * 2)
                    .toString() +
                "\n";
        }
        return this.label("Commands") +
            Table.from([
                ...commands.map((command) => [
                    [command.getName(), ...command.getAliases()].map((name) => blue(name))
                        .join(", "),
                    red(bold("-")) + " " +
                        command.getDescription().split("\n").shift(),
                ]),
            ])
                .padding([2, 2])
                .indent(this.indent * 2)
                .toString() +
            "\n";
    }
    generateEnvironmentVariables() {
        const envVars = this.cmd.getEnvVars(false);
        if (!envVars.length) {
            return "";
        }
        return this.label("Environment variables") +
            Table.from([
                ...envVars.map((envVar) => [
                    envVar.names.map((name) => blue(name)).join(", "),
                    highlightArgumentDetails(envVar.details, this.options.types),
                    `${red(bold("-"))} ${envVar.description}`,
                ]),
            ])
                .padding(2)
                .indent(this.indent * 2)
                .toString() +
            "\n";
    }
    generateExamples() {
        const examples = this.cmd.getExamples();
        if (!examples.length) {
            return "";
        }
        return this.label("Examples") +
            Table.from(examples.map((example) => [
                dim(bold(`${capitalize(example.name)}:`)),
                example.description,
            ]))
                .padding(1)
                .indent(this.indent * 2)
                .maxColWidth(150)
                .toString() +
            "\n";
    }
    generateHints(option) {
        if (!this.options.hints) {
            return "";
        }
        const hints = [];
        option.required && hints.push(yellow(`required`));
        typeof option.default !== "undefined" && hints.push(bold(`Default: `) + inspect(option.default, this.options.colors));
        option.depends?.length && hints.push(yellow(bold(`Depends: `)) +
            italic(option.depends.map(getFlag).join(", ")));
        option.conflicts?.length && hints.push(red(bold(`Conflicts: `)) +
            italic(option.conflicts.map(getFlag).join(", ")));
        const type = this.cmd.getType(option.args[0]?.type)?.handler;
        if (type instanceof Type) {
            const possibleValues = type.values?.(this.cmd, this.cmd.getParent());
            if (possibleValues?.length) {
                hints.push(bold(`Values: `) +
                    possibleValues.map((value) => inspect(value, this.options.colors)).join(", "));
            }
        }
        if (hints.length) {
            return `(${hints.join(", ")})`;
        }
        return "";
    }
    label(label) {
        return "\n" +
            " ".repeat(this.indent) + bold(`${label}:`) +
            "\n\n";
    }
}
function capitalize(string) {
    return string?.charAt(0).toUpperCase() + string.slice(1) ?? "";
}
function inspect(value, colors) {
    return Deno.inspect(value, { depth: 1, colors, trailingComma: false });
}
function highlightArguments(argsDefinition, types = true) {
    if (!argsDefinition) {
        return "";
    }
    return parseArgumentsDefinition(argsDefinition)
        .map((arg) => highlightArgumentDetails(arg, types)).join(" ");
}
function highlightArgumentDetails(arg, types = true) {
    let str = "";
    str += yellow(arg.optionalValue ? "[" : "<");
    let name = "";
    name += arg.name;
    if (arg.variadic) {
        name += "...";
    }
    name = magenta(name);
    str += name;
    if (types) {
        str += yellow(":");
        str += red(arg.type);
    }
    if (arg.list) {
        str += green("[]");
    }
    str += yellow(arg.optionalValue ? "]" : ">");
    return str;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2hlbHBfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2hlbHBfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNoRCxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0MsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXhELE9BQU8sRUFDTCxJQUFJLEVBQ0osSUFBSSxFQUNKLEdBQUcsRUFDSCxlQUFlLEVBQ2YsS0FBSyxFQUNMLE1BQU0sRUFDTixPQUFPLEVBQ1AsR0FBRyxFQUNILGVBQWUsRUFDZixNQUFNLEdBQ1AsTUFBTSxZQUFZLENBQUM7QUFHcEIsT0FBTyxFQUFFLElBQUksRUFBRSxNQUFNLFlBQVksQ0FBQztBQVNsQyxNQUFNLE9BQU8sYUFBYTtJQVNJO0lBUnBCLE1BQU0sR0FBRyxDQUFDLENBQUM7SUFDWCxPQUFPLENBQXdCO0lBR2hDLE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWSxFQUFFLE9BQXFCO1FBQ3hELE9BQU8sSUFBSSxhQUFhLENBQUMsR0FBRyxFQUFFLE9BQU8sQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3BELENBQUM7SUFFRCxZQUE0QixHQUFZLEVBQUUsVUFBdUIsRUFBRTtRQUF2QyxRQUFHLEdBQUgsR0FBRyxDQUFTO1FBQ3RDLElBQUksQ0FBQyxPQUFPLEdBQUc7WUFDYixLQUFLLEVBQUUsS0FBSztZQUNaLEtBQUssRUFBRSxJQUFJO1lBQ1gsTUFBTSxFQUFFLElBQUk7WUFDWixHQUFHLE9BQU87U0FDWCxDQUFDO0lBQ0osQ0FBQztJQUVPLFFBQVE7UUFDZCxNQUFNLGdCQUFnQixHQUFHLGVBQWUsRUFBRSxDQUFDO1FBQzNDLGVBQWUsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDO1FBQ3JDLE1BQU0sTUFBTSxHQUFHLElBQUksQ0FBQyxjQUFjLEVBQUU7WUFDbEMsSUFBSSxDQUFDLG1CQUFtQixFQUFFO1lBQzFCLElBQUksQ0FBQyxlQUFlLEVBQUU7WUFDdEIsSUFBSSxDQUFDLGdCQUFnQixFQUFFO1lBQ3ZCLElBQUksQ0FBQyw0QkFBNEIsRUFBRTtZQUNuQyxJQUFJLENBQUMsZ0JBQWdCLEVBQUUsQ0FBQztRQUMxQixlQUFlLENBQUMsZ0JBQWdCLENBQUMsQ0FBQztRQUNsQyxPQUFPLE1BQU0sQ0FBQztJQUNoQixDQUFDO0lBRU8sY0FBYztRQUNwQixNQUFNLElBQUksR0FBRztZQUNYO2dCQUNFLElBQUksQ0FBQyxRQUFRLENBQUM7Z0JBQ2QsT0FBTyxDQUNMLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsR0FDbkIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsRUFBRTtvQkFDMUIsQ0FBQyxDQUFDLEdBQUcsR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLGlCQUFpQixFQUFFO29CQUNwQyxDQUFDLENBQUMsRUFDTixFQUFFLENBQ0g7YUFDRjtTQUNGLENBQUM7UUFDRixNQUFNLE9BQU8sR0FBdUIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsQ0FBQztRQUMxRCxJQUFJLE9BQU8sRUFBRTtZQUNYLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLEVBQUUsTUFBTSxDQUFDLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQyxDQUFDO1NBQ25FO1FBQ0QsT0FBTyxJQUFJO1lBQ1QsS0FBSyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUM7aUJBQ2IsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUM7aUJBQ25CLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsUUFBUSxFQUFFO1lBQ2IsSUFBSSxDQUFDO0lBQ1QsQ0FBQztJQUVPLG1CQUFtQjtRQUN6QixJQUFJLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsRUFBRTtZQUM5QixPQUFPLEVBQUUsQ0FBQztTQUNYO1FBQ0QsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLGFBQWEsQ0FBQztZQUM5QixLQUFLLENBQUMsSUFBSSxDQUFDO2dCQUNULENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsQ0FBQzthQUM1QixDQUFDO2lCQUNDLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsQ0FBQztpQkFDdkIsV0FBVyxDQUFDLEdBQUcsQ0FBQztpQkFDaEIsT0FBTyxDQUFDLENBQUMsQ0FBQztpQkFDVixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sZUFBZTtRQUNyQixNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsQ0FBQyxLQUFLLENBQUMsQ0FBQztRQUMzQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sRUFBRTtZQUNuQixPQUFPLEVBQUUsQ0FBQztTQUNYO1FBRUQsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLE1BQU0sRUFBRSxFQUFFLENBQ25ELENBQUMsQ0FBQyxNQUFNLENBQUMsY0FBYyxDQUN4QixDQUFDO1FBRUYsSUFBSSxrQkFBa0IsRUFBRTtZQUN0QixPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDO2dCQUMxQixLQUFLLENBQUMsSUFBSSxDQUFDO29CQUNULEdBQUcsT0FBTyxDQUFDLEdBQUcsQ0FBQyxDQUFDLE1BQWUsRUFBRSxFQUFFLENBQUM7d0JBQ2xDLE1BQU0sQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDO3dCQUNqRCxrQkFBa0IsQ0FDaEIsTUFBTSxDQUFDLGNBQWMsSUFBSSxFQUFFLEVBQzNCLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUNuQjt3QkFDRCxHQUFHLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsR0FBRzs0QkFDcEIsTUFBTSxDQUFDLFdBQVcsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO3dCQUNoRCxJQUFJLENBQUMsYUFBYSxDQUFDLE1BQU0sQ0FBQztxQkFDM0IsQ0FBQztpQkFDSCxDQUFDO3FCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7cUJBQ2xCLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsQ0FBQztxQkFDdkIsV0FBVyxDQUFDLENBQUMsRUFBRSxFQUFFLEVBQUUsRUFBRSxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7cUJBQzdCLFFBQVEsRUFBRTtnQkFDYixJQUFJLENBQUM7U0FDUjtRQUVELE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLENBQUM7WUFDMUIsS0FBSyxDQUFDLElBQUksQ0FBQztnQkFDVCxHQUFHLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQyxNQUFlLEVBQUUsRUFBRSxDQUFDO29CQUNsQyxNQUFNLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQUksRUFBRSxFQUFFLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQztvQkFDakQsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7d0JBQ3BCLE1BQU0sQ0FBQyxXQUFXLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxDQUFDLEtBQUssRUFBWTtvQkFDaEQsSUFBSSxDQUFDLGFBQWEsQ0FBQyxNQUFNLENBQUM7aUJBQzNCLENBQUM7YUFDSCxDQUFDO2lCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztpQkFDZixNQUFNLENBQUMsSUFBSSxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUM7aUJBQ3ZCLFdBQVcsQ0FBQyxDQUFDLEVBQUUsRUFBRSxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7aUJBQ3pCLFFBQVEsRUFBRTtZQUNiLElBQUksQ0FBQztJQUNULENBQUM7SUFFTyxnQkFBZ0I7UUFDdEIsTUFBTSxRQUFRLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxXQUFXLENBQUMsS0FBSyxDQUFDLENBQUM7UUFDN0MsSUFBSSxDQUFDLFFBQVEsQ0FBQyxNQUFNLEVBQUU7WUFDcEIsT0FBTyxFQUFFLENBQUM7U0FDWDtRQUVELE1BQU0sa0JBQWtCLEdBQUcsQ0FBQyxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUNyRCxDQUFDLENBQUMsT0FBTyxDQUFDLGlCQUFpQixFQUFFLENBQzlCLENBQUM7UUFFRixJQUFJLGtCQUFrQixFQUFFO1lBQ3RCLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUM7Z0JBQzNCLEtBQUssQ0FBQyxJQUFJLENBQUM7b0JBQ1QsR0FBRyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBZ0IsRUFBRSxFQUFFLENBQUM7d0JBQ3BDLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxFQUFFLEdBQUcsT0FBTyxDQUFDLFVBQVUsRUFBRSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FDeEQsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUNYLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQzt3QkFDWixrQkFBa0IsQ0FDaEIsT0FBTyxDQUFDLGlCQUFpQixFQUFFLElBQUksRUFBRSxFQUNqQyxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FDbkI7d0JBQ0QsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7NEJBQ3BCLE9BQU8sQ0FBQyxjQUFjLEVBQUUsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO3FCQUN2RCxDQUFDO2lCQUNILENBQUM7cUJBQ0MsT0FBTyxDQUFDLENBQUMsQ0FBQyxFQUFFLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztxQkFDbEIsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO3FCQUN2QixRQUFRLEVBQUU7Z0JBQ2IsSUFBSSxDQUFDO1NBQ1I7UUFFRCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDO1lBQzNCLEtBQUssQ0FBQyxJQUFJLENBQUM7Z0JBQ1QsR0FBRyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBZ0IsRUFBRSxFQUFFLENBQUM7b0JBQ3BDLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxFQUFFLEdBQUcsT0FBTyxDQUFDLFVBQVUsRUFBRSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7eUJBQ25FLElBQUksQ0FBQyxJQUFJLENBQUM7b0JBQ2IsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7d0JBQ3BCLE9BQU8sQ0FBQyxjQUFjLEVBQUUsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO2lCQUN2RCxDQUFDO2FBQ0gsQ0FBQztpQkFDQyxPQUFPLENBQUMsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7aUJBQ2YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sNEJBQTRCO1FBQ2xDLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxDQUFDLEtBQUssQ0FBQyxDQUFDO1FBQzNDLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxFQUFFO1lBQ25CLE9BQU8sRUFBRSxDQUFDO1NBQ1g7UUFDRCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsdUJBQXVCLENBQUM7WUFDeEMsS0FBSyxDQUFDLElBQUksQ0FBQztnQkFDVCxHQUFHLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQyxNQUFlLEVBQUUsRUFBRSxDQUFDO29CQUNsQyxNQUFNLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQVksRUFBRSxFQUFFLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQztvQkFDekQsd0JBQXdCLENBQ3RCLE1BQU0sQ0FBQyxPQUFPLEVBQ2QsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQ25CO29CQUNELEdBQUcsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxJQUFJLE1BQU0sQ0FBQyxXQUFXLEVBQUU7aUJBQzFDLENBQUM7YUFDSCxDQUFDO2lCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sZ0JBQWdCO1FBQ3RCLE1BQU0sUUFBUSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsV0FBVyxFQUFFLENBQUM7UUFDeEMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxNQUFNLEVBQUU7WUFDcEIsT0FBTyxFQUFFLENBQUM7U0FDWDtRQUNELE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUM7WUFDM0IsS0FBSyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBaUIsRUFBRSxFQUFFLENBQUM7Z0JBQzdDLEdBQUcsQ0FBQyxJQUFJLENBQUMsR0FBRyxVQUFVLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQztnQkFDekMsT0FBTyxDQUFDLFdBQVc7YUFDcEIsQ0FBQyxDQUFDO2lCQUNBLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixXQUFXLENBQUMsR0FBRyxDQUFDO2lCQUNoQixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sYUFBYSxDQUFDLE1BQWU7UUFDbkMsSUFBSSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxFQUFFO1lBQ3ZCLE9BQU8sRUFBRSxDQUFDO1NBQ1g7UUFDRCxNQUFNLEtBQUssR0FBRyxFQUFFLENBQUM7UUFFakIsTUFBTSxDQUFDLFFBQVEsSUFBSSxLQUFLLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxVQUFVLENBQUMsQ0FBQyxDQUFDO1FBQ2xELE9BQU8sTUFBTSxDQUFDLE9BQU8sS0FBSyxXQUFXLElBQUksS0FBSyxDQUFDLElBQUksQ0FDakQsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLE9BQU8sQ0FBQyxNQUFNLENBQUMsT0FBTyxFQUFFLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQ2pFLENBQUM7UUFDRixNQUFNLENBQUMsT0FBTyxFQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsSUFBSSxDQUNsQyxNQUFNLENBQUMsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDO1lBQ3ZCLE1BQU0sQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxPQUFPLENBQUMsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FDakQsQ0FBQztRQUNGLE1BQU0sQ0FBQyxTQUFTLEVBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQyxJQUFJLENBQ3BDLEdBQUcsQ0FBQyxJQUFJLENBQUMsYUFBYSxDQUFDLENBQUM7WUFDdEIsTUFBTSxDQUFDLE1BQU0sQ0FBQyxTQUFTLENBQUMsR0FBRyxDQUFDLE9BQU8sQ0FBQyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUNuRCxDQUFDO1FBRUYsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsRUFBRSxJQUFJLENBQUMsRUFBRSxPQUFPLENBQUM7UUFDN0QsSUFBSSxJQUFJLFlBQVksSUFBSSxFQUFFO1lBQ3hCLE1BQU0sY0FBYyxHQUFHLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLElBQUksQ0FBQyxHQUFHLENBQUMsU0FBUyxFQUFFLENBQUMsQ0FBQztZQUNyRSxJQUFJLGNBQWMsRUFBRSxNQUFNLEVBQUU7Z0JBQzFCLEtBQUssQ0FBQyxJQUFJLENBQ1IsSUFBSSxDQUFDLFVBQVUsQ0FBQztvQkFDZCxjQUFjLENBQUMsR0FBRyxDQUFDLENBQUMsS0FBYyxFQUFFLEVBQUUsQ0FDcEMsT0FBTyxDQUFDLEtBQUssRUFBRSxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUNwQyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FDZixDQUFDO2FBQ0g7U0FDRjtRQUVELElBQUksS0FBSyxDQUFDLE1BQU0sRUFBRTtZQUNoQixPQUFPLElBQUksS0FBSyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDO1NBQ2hDO1FBRUQsT0FBTyxFQUFFLENBQUM7SUFDWixDQUFDO0lBRU8sS0FBSyxDQUFDLEtBQWE7UUFDekIsT0FBTyxJQUFJO1lBQ1QsR0FBRyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEdBQUcsSUFBSSxDQUFDLEdBQUcsS0FBSyxHQUFHLENBQUM7WUFDM0MsTUFBTSxDQUFDO0lBQ1gsQ0FBQztDQUNGO0FBRUQsU0FBUyxVQUFVLENBQUMsTUFBYztJQUNoQyxPQUFPLE1BQU0sRUFBRSxNQUFNLENBQUMsQ0FBQyxDQUFDLENBQUMsV0FBVyxFQUFFLEdBQUcsTUFBTSxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsSUFBSSxFQUFFLENBQUM7QUFDakUsQ0FBQztBQUVELFNBQVMsT0FBTyxDQUFDLEtBQWMsRUFBRSxNQUFlO0lBQzlDLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FDakIsS0FBSyxFQUVMLEVBQUUsS0FBSyxFQUFFLENBQUMsRUFBRSxNQUFNLEVBQUUsYUFBYSxFQUFFLEtBQUssRUFBeUIsQ0FDbEUsQ0FBQztBQUNKLENBQUM7QUFPRCxTQUFTLGtCQUFrQixDQUFDLGNBQXNCLEVBQUUsS0FBSyxHQUFHLElBQUk7SUFDOUQsSUFBSSxDQUFDLGNBQWMsRUFBRTtRQUNuQixPQUFPLEVBQUUsQ0FBQztLQUNYO0lBRUQsT0FBTyx3QkFBd0IsQ0FBQyxjQUFjLENBQUM7U0FDNUMsR0FBRyxDQUFDLENBQUMsR0FBYyxFQUFFLEVBQUUsQ0FBQyx3QkFBd0IsQ0FBQyxHQUFHLEVBQUUsS0FBSyxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7QUFDN0UsQ0FBQztBQU9ELFNBQVMsd0JBQXdCLENBQy9CLEdBQWMsRUFDZCxLQUFLLEdBQUcsSUFBSTtJQUVaLElBQUksR0FBRyxHQUFHLEVBQUUsQ0FBQztJQUViLEdBQUcsSUFBSSxNQUFNLENBQUMsR0FBRyxDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQztJQUU3QyxJQUFJLElBQUksR0FBRyxFQUFFLENBQUM7SUFDZCxJQUFJLElBQUksR0FBRyxDQUFDLElBQUksQ0FBQztJQUNqQixJQUFJLEdBQUcsQ0FBQyxRQUFRLEVBQUU7UUFDaEIsSUFBSSxJQUFJLEtBQUssQ0FBQztLQUNmO0lBQ0QsSUFBSSxHQUFHLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUVyQixHQUFHLElBQUksSUFBSSxDQUFDO0lBRVosSUFBSSxLQUFLLEVBQUU7UUFDVCxHQUFHLElBQUksTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ25CLEdBQUcsSUFBSSxHQUFHLENBQUMsR0FBRyxDQUFDLElBQUksQ0FBQyxDQUFDO0tBQ3RCO0lBRUQsSUFBSSxHQUFHLENBQUMsSUFBSSxFQUFFO1FBQ1osR0FBRyxJQUFJLEtBQUssQ0FBQyxJQUFJLENBQUMsQ0FBQztLQUNwQjtJQUVELEdBQUcsSUFBSSxNQUFNLENBQUMsR0FBRyxDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQztJQUU3QyxPQUFPLEdBQUcsQ0FBQztBQUNiLENBQUMifQ==