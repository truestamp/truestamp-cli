import { getFlag } from "../../flags/_utils.ts";
import { Table } from "../../table/table.ts";
import { parseArgumentsDefinition } from "../_utils.ts";
import { blue, bold, dim, green, italic, magenta, red, yellow, } from "../deps.ts";
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
            ...options,
        };
    }
    generate() {
        return this.generateHeader() +
            this.generateDescription() +
            this.generateOptions() +
            this.generateCommands() +
            this.generateEnvironmentVariables() +
            this.generateExamples() +
            "\n";
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
            rows.push([bold("Version:"), yellow(`v${this.cmd.getVersion()}`)]);
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
        typeof option.default !== "undefined" && hints.push(bold(`Default: `) + inspect(option.default));
        option.depends?.length && hints.push(yellow(bold(`Depends: `)) +
            italic(option.depends.map(getFlag).join(", ")));
        option.conflicts?.length && hints.push(red(bold(`Conflicts: `)) +
            italic(option.conflicts.map(getFlag).join(", ")));
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
function inspect(value) {
    return Deno.inspect(value, { depth: 1, colors: true, trailingComma: false });
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2hlbHBfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2hlbHBfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNoRCxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0MsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXhELE9BQU8sRUFDTCxJQUFJLEVBQ0osSUFBSSxFQUNKLEdBQUcsRUFDSCxLQUFLLEVBQ0wsTUFBTSxFQUNOLE9BQU8sRUFDUCxHQUFHLEVBQ0gsTUFBTSxHQUNQLE1BQU0sWUFBWSxDQUFDO0FBVXBCLE1BQU0sT0FBTyxhQUFhO0lBU0k7SUFScEIsTUFBTSxHQUFHLENBQUMsQ0FBQztJQUNYLE9BQU8sQ0FBYztJQUd0QixNQUFNLENBQUMsUUFBUSxDQUFDLEdBQVksRUFBRSxPQUFxQjtRQUN4RCxPQUFPLElBQUksYUFBYSxDQUFDLEdBQUcsRUFBRSxPQUFPLENBQUMsQ0FBQyxRQUFRLEVBQUUsQ0FBQztJQUNwRCxDQUFDO0lBRUQsWUFBNEIsR0FBWSxFQUFFLFVBQXVCLEVBQUU7UUFBdkMsUUFBRyxHQUFILEdBQUcsQ0FBUztRQUN0QyxJQUFJLENBQUMsT0FBTyxHQUFHO1lBQ2IsS0FBSyxFQUFFLEtBQUs7WUFDWixLQUFLLEVBQUUsSUFBSTtZQUNYLEdBQUcsT0FBTztTQUNYLENBQUM7SUFDSixDQUFDO0lBRU8sUUFBUTtRQUNkLE9BQU8sSUFBSSxDQUFDLGNBQWMsRUFBRTtZQUMxQixJQUFJLENBQUMsbUJBQW1CLEVBQUU7WUFDMUIsSUFBSSxDQUFDLGVBQWUsRUFBRTtZQUN0QixJQUFJLENBQUMsZ0JBQWdCLEVBQUU7WUFDdkIsSUFBSSxDQUFDLDRCQUE0QixFQUFFO1lBQ25DLElBQUksQ0FBQyxnQkFBZ0IsRUFBRTtZQUN2QixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sY0FBYztRQUNwQixNQUFNLElBQUksR0FBRztZQUNYO2dCQUNFLElBQUksQ0FBQyxRQUFRLENBQUM7Z0JBQ2QsT0FBTyxDQUNMLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsR0FDbkIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsRUFBRTtvQkFDMUIsQ0FBQyxDQUFDLEdBQUcsR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLGlCQUFpQixFQUFFO29CQUNwQyxDQUFDLENBQUMsRUFDTixFQUFFLENBQ0g7YUFDRjtTQUNGLENBQUM7UUFDRixNQUFNLE9BQU8sR0FBdUIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsQ0FBQztRQUMxRCxJQUFJLE9BQU8sRUFBRTtZQUNYLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLEVBQUUsTUFBTSxDQUFDLElBQUksSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQyxDQUFDO1NBQ3BFO1FBQ0QsT0FBTyxJQUFJO1lBQ1QsS0FBSyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUM7aUJBQ2IsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUM7aUJBQ25CLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsUUFBUSxFQUFFO1lBQ2IsSUFBSSxDQUFDO0lBQ1QsQ0FBQztJQUVPLG1CQUFtQjtRQUN6QixJQUFJLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsRUFBRTtZQUM5QixPQUFPLEVBQUUsQ0FBQztTQUNYO1FBQ0QsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLGFBQWEsQ0FBQztZQUM5QixLQUFLLENBQUMsSUFBSSxDQUFDO2dCQUNULENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsQ0FBQzthQUM1QixDQUFDO2lCQUNDLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsQ0FBQztpQkFDdkIsV0FBVyxDQUFDLEdBQUcsQ0FBQztpQkFDaEIsT0FBTyxDQUFDLENBQUMsQ0FBQztpQkFDVixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sZUFBZTtRQUNyQixNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsQ0FBQyxLQUFLLENBQUMsQ0FBQztRQUMzQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sRUFBRTtZQUNuQixPQUFPLEVBQUUsQ0FBQztTQUNYO1FBRUQsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLE1BQU0sRUFBRSxFQUFFLENBQ25ELENBQUMsQ0FBQyxNQUFNLENBQUMsY0FBYyxDQUN4QixDQUFDO1FBRUYsSUFBSSxrQkFBa0IsRUFBRTtZQUN0QixPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDO2dCQUMxQixLQUFLLENBQUMsSUFBSSxDQUFDO29CQUNULEdBQUcsT0FBTyxDQUFDLEdBQUcsQ0FBQyxDQUFDLE1BQWUsRUFBRSxFQUFFLENBQUM7d0JBQ2xDLE1BQU0sQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDO3dCQUNqRCxrQkFBa0IsQ0FDaEIsTUFBTSxDQUFDLGNBQWMsSUFBSSxFQUFFLEVBQzNCLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUNuQjt3QkFDRCxHQUFHLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsR0FBRzs0QkFDcEIsTUFBTSxDQUFDLFdBQVcsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO3dCQUNoRCxJQUFJLENBQUMsYUFBYSxDQUFDLE1BQU0sQ0FBQztxQkFDM0IsQ0FBQztpQkFDSCxDQUFDO3FCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7cUJBQ2xCLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsQ0FBQztxQkFDdkIsV0FBVyxDQUFDLENBQUMsRUFBRSxFQUFFLEVBQUUsRUFBRSxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7cUJBQzdCLFFBQVEsRUFBRTtnQkFDYixJQUFJLENBQUM7U0FDUjtRQUVELE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLENBQUM7WUFDMUIsS0FBSyxDQUFDLElBQUksQ0FBQztnQkFDVCxHQUFHLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQyxNQUFlLEVBQUUsRUFBRSxDQUFDO29CQUNsQyxNQUFNLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQUksRUFBRSxFQUFFLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQztvQkFDakQsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7d0JBQ3BCLE1BQU0sQ0FBQyxXQUFXLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxDQUFDLEtBQUssRUFBWTtvQkFDaEQsSUFBSSxDQUFDLGFBQWEsQ0FBQyxNQUFNLENBQUM7aUJBQzNCLENBQUM7YUFDSCxDQUFDO2lCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztpQkFDZixNQUFNLENBQUMsSUFBSSxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUM7aUJBQ3ZCLFdBQVcsQ0FBQyxDQUFDLEVBQUUsRUFBRSxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7aUJBQ3pCLFFBQVEsRUFBRTtZQUNiLElBQUksQ0FBQztJQUNULENBQUM7SUFFTyxnQkFBZ0I7UUFDdEIsTUFBTSxRQUFRLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxXQUFXLENBQUMsS0FBSyxDQUFDLENBQUM7UUFDN0MsSUFBSSxDQUFDLFFBQVEsQ0FBQyxNQUFNLEVBQUU7WUFDcEIsT0FBTyxFQUFFLENBQUM7U0FDWDtRQUVELE1BQU0sa0JBQWtCLEdBQUcsQ0FBQyxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUNyRCxDQUFDLENBQUMsT0FBTyxDQUFDLGlCQUFpQixFQUFFLENBQzlCLENBQUM7UUFFRixJQUFJLGtCQUFrQixFQUFFO1lBQ3RCLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUM7Z0JBQzNCLEtBQUssQ0FBQyxJQUFJLENBQUM7b0JBQ1QsR0FBRyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBZ0IsRUFBRSxFQUFFLENBQUM7d0JBQ3BDLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxFQUFFLEdBQUcsT0FBTyxDQUFDLFVBQVUsRUFBRSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FDeEQsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUNYLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQzt3QkFDWixrQkFBa0IsQ0FDaEIsT0FBTyxDQUFDLGlCQUFpQixFQUFFLElBQUksRUFBRSxFQUNqQyxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FDbkI7d0JBQ0QsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7NEJBQ3BCLE9BQU8sQ0FBQyxjQUFjLEVBQUUsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO3FCQUN2RCxDQUFDO2lCQUNILENBQUM7cUJBQ0MsT0FBTyxDQUFDLENBQUMsQ0FBQyxFQUFFLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztxQkFDbEIsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO3FCQUN2QixRQUFRLEVBQUU7Z0JBQ2IsSUFBSSxDQUFDO1NBQ1I7UUFFRCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDO1lBQzNCLEtBQUssQ0FBQyxJQUFJLENBQUM7Z0JBQ1QsR0FBRyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBZ0IsRUFBRSxFQUFFLENBQUM7b0JBQ3BDLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxFQUFFLEdBQUcsT0FBTyxDQUFDLFVBQVUsRUFBRSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7eUJBQ25FLElBQUksQ0FBQyxJQUFJLENBQUM7b0JBQ2IsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEdBQUc7d0JBQ3BCLE9BQU8sQ0FBQyxjQUFjLEVBQUUsQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsS0FBSyxFQUFZO2lCQUN2RCxDQUFDO2FBQ0gsQ0FBQztpQkFDQyxPQUFPLENBQUMsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7aUJBQ2YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sNEJBQTRCO1FBQ2xDLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxDQUFDLEtBQUssQ0FBQyxDQUFDO1FBQzNDLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxFQUFFO1lBQ25CLE9BQU8sRUFBRSxDQUFDO1NBQ1g7UUFDRCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsdUJBQXVCLENBQUM7WUFDeEMsS0FBSyxDQUFDLElBQUksQ0FBQztnQkFDVCxHQUFHLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQyxNQUFlLEVBQUUsRUFBRSxDQUFDO29CQUNsQyxNQUFNLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQVksRUFBRSxFQUFFLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQztvQkFDekQsd0JBQXdCLENBQ3RCLE1BQU0sQ0FBQyxPQUFPLEVBQ2QsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQ25CO29CQUNELEdBQUcsR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxJQUFJLE1BQU0sQ0FBQyxXQUFXLEVBQUU7aUJBQzFDLENBQUM7YUFDSCxDQUFDO2lCQUNDLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sZ0JBQWdCO1FBQ3RCLE1BQU0sUUFBUSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsV0FBVyxFQUFFLENBQUM7UUFDeEMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxNQUFNLEVBQUU7WUFDcEIsT0FBTyxFQUFFLENBQUM7U0FDWDtRQUNELE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxVQUFVLENBQUM7WUFDM0IsS0FBSyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsT0FBaUIsRUFBRSxFQUFFLENBQUM7Z0JBQzdDLEdBQUcsQ0FBQyxJQUFJLENBQUMsR0FBRyxVQUFVLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQztnQkFDekMsT0FBTyxDQUFDLFdBQVc7YUFDcEIsQ0FBQyxDQUFDO2lCQUNBLE9BQU8sQ0FBQyxDQUFDLENBQUM7aUJBQ1YsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDO2lCQUN2QixXQUFXLENBQUMsR0FBRyxDQUFDO2lCQUNoQixRQUFRLEVBQUU7WUFDYixJQUFJLENBQUM7SUFDVCxDQUFDO0lBRU8sYUFBYSxDQUFDLE1BQWU7UUFDbkMsSUFBSSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxFQUFFO1lBQ3ZCLE9BQU8sRUFBRSxDQUFDO1NBQ1g7UUFDRCxNQUFNLEtBQUssR0FBRyxFQUFFLENBQUM7UUFFakIsTUFBTSxDQUFDLFFBQVEsSUFBSSxLQUFLLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxVQUFVLENBQUMsQ0FBQyxDQUFDO1FBQ2xELE9BQU8sTUFBTSxDQUFDLE9BQU8sS0FBSyxXQUFXLElBQUksS0FBSyxDQUFDLElBQUksQ0FDakQsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLE9BQU8sQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLENBQzVDLENBQUM7UUFDRixNQUFNLENBQUMsT0FBTyxFQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsSUFBSSxDQUNsQyxNQUFNLENBQUMsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDO1lBQ3ZCLE1BQU0sQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxPQUFPLENBQUMsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FDakQsQ0FBQztRQUNGLE1BQU0sQ0FBQyxTQUFTLEVBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQyxJQUFJLENBQ3BDLEdBQUcsQ0FBQyxJQUFJLENBQUMsYUFBYSxDQUFDLENBQUM7WUFDdEIsTUFBTSxDQUFDLE1BQU0sQ0FBQyxTQUFTLENBQUMsR0FBRyxDQUFDLE9BQU8sQ0FBQyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUNuRCxDQUFDO1FBRUYsSUFBSSxLQUFLLENBQUMsTUFBTSxFQUFFO1lBQ2hCLE9BQU8sSUFBSSxLQUFLLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUM7U0FDaEM7UUFFRCxPQUFPLEVBQUUsQ0FBQztJQUNaLENBQUM7SUFFTyxLQUFLLENBQUMsS0FBYTtRQUN6QixPQUFPLElBQUk7WUFDVCxHQUFHLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsR0FBRyxJQUFJLENBQUMsR0FBRyxLQUFLLEdBQUcsQ0FBQztZQUMzQyxNQUFNLENBQUM7SUFDWCxDQUFDO0NBQ0Y7QUFFRCxTQUFTLFVBQVUsQ0FBQyxNQUFjO0lBQ2hDLE9BQU8sTUFBTSxFQUFFLE1BQU0sQ0FBQyxDQUFDLENBQUMsQ0FBQyxXQUFXLEVBQUUsR0FBRyxNQUFNLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxJQUFJLEVBQUUsQ0FBQztBQUNqRSxDQUFDO0FBRUQsU0FBUyxPQUFPLENBQUMsS0FBYztJQUM3QixPQUFPLElBQUksQ0FBQyxPQUFPLENBQ2pCLEtBQUssRUFFTCxFQUFFLEtBQUssRUFBRSxDQUFDLEVBQUUsTUFBTSxFQUFFLElBQUksRUFBRSxhQUFhLEVBQUUsS0FBSyxFQUF5QixDQUN4RSxDQUFDO0FBQ0osQ0FBQztBQU9ELFNBQVMsa0JBQWtCLENBQUMsY0FBc0IsRUFBRSxLQUFLLEdBQUcsSUFBSTtJQUM5RCxJQUFJLENBQUMsY0FBYyxFQUFFO1FBQ25CLE9BQU8sRUFBRSxDQUFDO0tBQ1g7SUFFRCxPQUFPLHdCQUF3QixDQUFDLGNBQWMsQ0FBQztTQUM1QyxHQUFHLENBQUMsQ0FBQyxHQUFjLEVBQUUsRUFBRSxDQUFDLHdCQUF3QixDQUFDLEdBQUcsRUFBRSxLQUFLLENBQUMsQ0FBQyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQztBQUM3RSxDQUFDO0FBT0QsU0FBUyx3QkFBd0IsQ0FDL0IsR0FBYyxFQUNkLEtBQUssR0FBRyxJQUFJO0lBRVosSUFBSSxHQUFHLEdBQUcsRUFBRSxDQUFDO0lBRWIsR0FBRyxJQUFJLE1BQU0sQ0FBQyxHQUFHLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBRTdDLElBQUksSUFBSSxHQUFHLEVBQUUsQ0FBQztJQUNkLElBQUksSUFBSSxHQUFHLENBQUMsSUFBSSxDQUFDO0lBQ2pCLElBQUksR0FBRyxDQUFDLFFBQVEsRUFBRTtRQUNoQixJQUFJLElBQUksS0FBSyxDQUFDO0tBQ2Y7SUFDRCxJQUFJLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDO0lBRXJCLEdBQUcsSUFBSSxJQUFJLENBQUM7SUFFWixJQUFJLEtBQUssRUFBRTtRQUNULEdBQUcsSUFBSSxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUM7UUFDbkIsR0FBRyxJQUFJLEdBQUcsQ0FBQyxHQUFHLENBQUMsSUFBSSxDQUFDLENBQUM7S0FDdEI7SUFFRCxJQUFJLEdBQUcsQ0FBQyxJQUFJLEVBQUU7UUFDWixHQUFHLElBQUksS0FBSyxDQUFDLElBQUksQ0FBQyxDQUFDO0tBQ3BCO0lBRUQsR0FBRyxJQUFJLE1BQU0sQ0FBQyxHQUFHLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBRTdDLE9BQU8sR0FBRyxDQUFDO0FBQ2IsQ0FBQyJ9