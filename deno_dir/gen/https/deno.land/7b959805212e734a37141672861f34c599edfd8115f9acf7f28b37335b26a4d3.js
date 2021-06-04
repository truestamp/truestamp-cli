import { getFlag } from "../../flags/_utils.ts";
import { Table } from "../../table/table.ts";
import { parseArgumentsDefinition } from "../_utils.ts";
import { blue, bold, dim, green, italic, magenta, red, yellow, } from "../deps.ts";
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
        const type = this.cmd.getType(option.args[0]?.type)?.handler;
        if (type instanceof Type) {
            const possibleValues = type.values?.(this.cmd, this.cmd.getParent());
            if (possibleValues?.length) {
                hints.push(bold(`Values: `) +
                    possibleValues.map((value) => inspect(value)).join(", "));
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2hlbHBfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2hlbHBfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNoRCxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0MsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXhELE9BQU8sRUFDTCxJQUFJLEVBQ0osSUFBSSxFQUNKLEdBQUcsRUFDSCxLQUFLLEVBQ0wsTUFBTSxFQUNOLE9BQU8sRUFDUCxHQUFHLEVBQ0gsTUFBTSxHQUNQLE1BQU0sWUFBWSxDQUFDO0FBR3BCLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFRbEMsTUFBTSxPQUFPLGFBQWE7SUFTSTtJQVJwQixNQUFNLEdBQUcsQ0FBQyxDQUFDO0lBQ1gsT0FBTyxDQUFjO0lBR3RCLE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWSxFQUFFLE9BQXFCO1FBQ3hELE9BQU8sSUFBSSxhQUFhLENBQUMsR0FBRyxFQUFFLE9BQU8sQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3BELENBQUM7SUFFRCxZQUE0QixHQUFZLEVBQUUsVUFBdUIsRUFBRTtRQUF2QyxRQUFHLEdBQUgsR0FBRyxDQUFTO1FBQ3RDLElBQUksQ0FBQyxPQUFPLEdBQUc7WUFDYixLQUFLLEVBQUUsS0FBSztZQUNaLEtBQUssRUFBRSxJQUFJO1lBQ1gsR0FBRyxPQUFPO1NBQ1gsQ0FBQztJQUNKLENBQUM7SUFFTyxRQUFRO1FBQ2QsT0FBTyxJQUFJLENBQUMsY0FBYyxFQUFFO1lBQzFCLElBQUksQ0FBQyxtQkFBbUIsRUFBRTtZQUMxQixJQUFJLENBQUMsZUFBZSxFQUFFO1lBQ3RCLElBQUksQ0FBQyxnQkFBZ0IsRUFBRTtZQUN2QixJQUFJLENBQUMsNEJBQTRCLEVBQUU7WUFDbkMsSUFBSSxD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