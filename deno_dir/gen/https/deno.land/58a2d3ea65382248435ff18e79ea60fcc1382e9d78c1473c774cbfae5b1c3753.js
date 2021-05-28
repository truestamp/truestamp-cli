export class ZshCompletionsGenerator {
    cmd;
    actions = new Map();
    static generate(cmd) {
        return new ZshCompletionsGenerator(cmd).generate();
    }
    constructor(cmd) {
        this.cmd = cmd;
    }
    generate() {
        const path = this.cmd.getPath();
        const name = this.cmd.getName();
        const version = this.cmd.getVersion()
            ? ` v${this.cmd.getVersion()}`
            : "";
        return `#!/usr/bin/env zsh
# zsh completion support for ${path}${version}

autoload -U is-at-least

# shellcheck disable=SC2154
(( $+functions[__${replaceSpecialChars(name)}_complete] )) ||
function __${replaceSpecialChars(name)}_complete {
  local name="$1"; shift
  local action="$1"; shift
  integer ret=1
  local -a values
  local expl lines
  _tags "$name"
  while _tags; do
    if _requested "$name"; then
      # shellcheck disable=SC2034
      lines="$(${name} completions complete "\${action}" "\${@}")"
      values=("\${(ps:\\n:)lines}")
      if (( \${#values[@]} )); then
        while _next_label "$name" expl "$action"; do
          compadd -S '' "\${expl[@]}" "\${values[@]}"
        done
      fi
    fi
  done
}

${this.generateCompletions(this.cmd).trim()}

# _${replaceSpecialChars(path)} "\${@}"

compdef _${replaceSpecialChars(path)} ${path}

`;
    }
    generateCompletions(command, path = "") {
        if (!command.hasCommands(false) && !command.hasOptions(false) &&
            !command.hasArguments()) {
            return "";
        }
        path = (path ? path + " " : "") + command.getName();
        return `# shellcheck disable=SC2154
(( $+functions[_${replaceSpecialChars(path)}] )) ||
function _${replaceSpecialChars(path)}() {` +
            (!command.getParent()
                ? `
  local state`
                : "") +
            this.generateCommandCompletions(command, path) +
            this.generateSubCommandCompletions(command, path) +
            this.generateArgumentCompletions(command, path) +
            this.generateActions(command) +
            `\n}\n\n` +
            command.getCommands(false)
                .filter((subCommand) => subCommand !== command)
                .map((subCommand) => this.generateCompletions(subCommand, path))
                .join("");
    }
    generateCommandCompletions(command, path) {
        const commands = command.getCommands(false);
        let completions = commands
            .map((subCommand) => `'${subCommand.getName()}:${subCommand.getShortDescription()}'`)
            .join("\n      ");
        if (completions) {
            completions = `
    local -a commands
    # shellcheck disable=SC2034
    commands=(
      ${completions}
    )
    _describe 'command' commands`;
        }
        if (command.hasArguments()) {
            const completionsPath = path.split(" ").slice(1).join(" ");
            const arg = command.getArguments()[0];
            const action = this.addAction(arg, completionsPath);
            if (action && command.getCompletion(arg.action)) {
                completions += `\n    __${replaceSpecialChars(this.cmd.getName())}_complete ${action.arg.name} ${action.arg.action} ${action.cmd}`;
            }
        }
        if (completions) {
            completions = `\n\n  function _commands() {${completions}\n  }`;
        }
        return completions;
    }
    generateSubCommandCompletions(command, path) {
        if (command.hasCommands(false)) {
            const actions = command
                .getCommands(false)
                .map((command) => `${command.getName()}) _${replaceSpecialChars(path + " " + command.getName())} ;;`)
                .join("\n      ");
            return `\n
  function _command_args() {
    case "\${words[1]}" in\n      ${actions}\n    esac
  }`;
        }
        return "";
    }
    generateArgumentCompletions(command, path) {
        this.actions.clear();
        const options = this.generateOptions(command, path);
        let argIndex = 0;
        let argsCommand = "\n\n  _arguments -w -s -S -C";
        if (command.hasOptions()) {
            argsCommand += ` \\\n    ${options.join(" \\\n    ")}`;
        }
        if (command.hasCommands(false) || (command.getArguments()
            .filter((arg) => command.getCompletion(arg.action)).length)) {
            argsCommand += ` \\\n    '${++argIndex}: :_commands'`;
        }
        if (command.hasArguments() || command.hasCommands(false)) {
            const args = [];
            for (const arg of command.getArguments().slice(1)) {
                const completionsPath = path.split(" ").slice(1).join(" ");
                const action = this.addAction(arg, completionsPath);
                args.push(`${++argIndex}${arg.optionalValue ? "::" : ":"}${action.name}`);
            }
            argsCommand += args.map((arg) => `\\\n    '${arg}'`).join("");
            if (command.hasCommands(false)) {
                argsCommand += ` \\\n    '*:: :->command_args'`;
            }
        }
        return argsCommand;
    }
    generateOptions(command, path) {
        const options = [];
        const cmdArgs = path.split(" ");
        const _baseName = cmdArgs.shift();
        const completionsPath = cmdArgs.join(" ");
        const excludedFlags = command.getOptions(false)
            .map((option) => option.standalone ? option.flags : false)
            .flat()
            .filter((flag) => typeof flag === "string");
        for (const option of command.getOptions(false)) {
            options.push(this.generateOption(option, completionsPath, excludedFlags));
        }
        return options;
    }
    generateOption(option, completionsPath, excludedOptions) {
        const flags = option.flags;
        let excludedFlags = option.conflicts?.length
            ? [
                ...excludedOptions,
                ...option.conflicts.map((opt) => "--" + opt.replace(/^--/, "")),
            ]
            : excludedOptions;
        excludedFlags = option.collect ? excludedFlags : [
            ...excludedFlags,
            ...flags,
        ];
        let args = "";
        for (const arg of option.args) {
            const action = this.addAction(arg, completionsPath);
            if (arg.variadic) {
                args += `${arg.optionalValue ? "::" : ":"}${arg.name}:->${action.name}`;
            }
            else {
                args += `${arg.optionalValue ? "::" : ":"}${arg.name}:->${action.name}`;
            }
        }
        let description = option.description
            .trim()
            .split("\n")
            .shift();
        description = description
            .replace(/\[/g, "\\[")
            .replace(/]/g, "\\]")
            .replace(/"/g, '\\"')
            .replace(/'/g, "'\"'\"'");
        const collect = option.collect ? "*" : "";
        if (option.standalone) {
            return `'(- *)'{${collect}${flags}}'[${description}]${args}'`;
        }
        else {
            const excluded = excludedFlags.length
                ? `'(${excludedFlags.join(" ")})'`
                : "";
            if (collect || flags.length > 1) {
                return `${excluded}{${collect}${flags}}'[${description}]${args}'`;
            }
            else {
                return `${excluded}${flags}'[${description}]${args}'`;
            }
        }
    }
    addAction(arg, cmd) {
        const action = `${arg.name}-${arg.action}`;
        if (!this.actions.has(action)) {
            this.actions.set(action, {
                arg: arg,
                label: `${arg.name}: ${arg.action}`,
                name: action,
                cmd,
            });
        }
        return this.actions.get(action);
    }
    generateActions(command) {
        let actions = [];
        if (this.actions.size) {
            actions = Array
                .from(this.actions)
                .map(([name, action]) => `${name}) __${replaceSpecialChars(this.cmd.getName())}_complete ${action.arg.name} ${action.arg.action} ${action.cmd} ;;`);
        }
        if (command.hasCommands(false)) {
            actions.unshift(`command_args) _command_args ;;`);
        }
        if (actions.length) {
            return `\n\n  case "$state" in\n    ${actions.join("\n    ")}\n  esac`;
        }
        return "";
    }
}
function replaceSpecialChars(str) {
    return str.replace(/[^a-zA-Z0-9]/g, "_");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3pzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfenNoX2NvbXBsZXRpb25zX2dlbmVyYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFXQSxNQUFNLE9BQU8sdUJBQXVCO0lBUUo7SUFQdEIsT0FBTyxHQUFtQyxJQUFJLEdBQUcsRUFBRSxDQUFDO0lBR3JELE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWTtRQUNqQyxPQUFPLElBQUksdUJBQXVCLENBQUMsR0FBRyxDQUFDLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDckQsQ0FBQztJQUVELFlBQThCLEdBQVk7UUFBWixRQUFHLEdBQUgsR0FBRyxDQUFTO0lBQUcsQ0FBQztJQUd0QyxRQUFRO1FBQ2QsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQ0FBQztRQUNoQyxNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDO1FBQ2hDLE1BQU0sT0FBTyxHQUF1QixJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRTtZQUN2RCxDQUFDLENBQUMsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRSxFQUFFO1lBQzlCLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFFUCxPQUFPOytCQUNvQixJQUFJLEdBQUcsT0FBTzs7Ozs7bUJBSzFCLG1CQUFtQixDQUFDLElBQUksQ0FBQzthQUMvQixtQkFBbUIsQ0FBQyxJQUFJLENBQUM7Ozs7Ozs7Ozs7aUJBVXJCLElBQUk7Ozs7Ozs7Ozs7O0VBV25CLElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFOztLQUV0QyxtQkFBbUIsQ0FBQyxJQUFJLENBQUM7O1dBRW5CLG1CQUFtQixDQUFDLElBQUksQ0FBQyxJQUFJLElBQUk7O0NBRTNDLENBQUM7SUFDQSxDQUFDO0lBR08sbUJBQW1CLENBQUMs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