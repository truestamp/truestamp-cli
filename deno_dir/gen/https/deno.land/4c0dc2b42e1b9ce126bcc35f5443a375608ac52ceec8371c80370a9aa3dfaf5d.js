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
            this.generateExamples() +
            "\n";
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2hlbHBfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2hlbHBfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNoRCxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0MsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXhELE9BQU8sRUFDTCxJQUFJLEVBQ0osSUFBSSxFQUNKLEdBQUcsRUFDSCxlQUFlLEVBQ2YsS0FBSyxFQUNMLE1BQU0sRUFDTixPQUFPLEVBQ1AsR0FBRyxFQUNILGVBQWUsRUFDZixNQUFNLEdBQ1AsTUFBTSxZQUFZLENBQUM7QUFHcEIsT0FBTyxFQUFFLElBQUksRUFBRSxNQUFNLFlBQVksQ0FBQztBQVNsQyxNQUFNLE9BQU8sYUFBYTtJQVNJO0lBUnBCLE1BQU0sR0FBRyxDQUFDLENBQUM7SUFDWCxPQUFPLENBQXdCO0lBR2hDLE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWSxFQUFFLE9BQXFCO1FBQ3hELE9BQU8sSUFBSSxhQUFhLENBQUMsR0FBRyxFQUFFLE9BQU8sQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3BELENBQUM7SUFFRCxZQUE0QixHQUFZLEVBQUUsVUFBdUIsRUFBRTtRQUF2QyxRQUFHLEdBQUgsR0FBRyxDQUFTO1FBQ3RDLElBQUksQ0FBQyxPQUFPLEdBQUc7WUFDYixLQUFLLEVBQUUsS0FBSztZQUNaLEtBQUssRUFBRSxJQUFJO1lBQ1gsTUFBTSxFQUFFLElBQUk7WUFDWixHQUFHLE9BQU87U0FDWCxDQUFDO0lBQ0osQ0FBQztJQUVPLFFBQVE7UUFDZCxNQUFNLGdCQUFnQixHQUFHLGVBQWUsRUFBRSxDQUFDO1FBQzNDLGVBQWUsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDO1FBQ3JDLE1BQU0sTUFBTSxHQUFHLElB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