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

compdef _${replaceSpecialChars(path)} ${path}`;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3pzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfenNoX2NvbXBsZXRpb25zX2dlbmVyYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFXQSxNQUFNLE9BQU8sdUJBQXVCO0lBUUo7SUFQdEIsT0FBTyxHQUFtQyxJQUFJLEdBQUcsRUFBRSxDQUFDO0lBR3JELE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWTtRQUNqQyxPQUFPLElBQUksdUJBQXVCLENBQUMsR0FBRyxDQUFDLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDckQsQ0FBQztJQUVELFlBQThCLEdBQVk7UUFBWixRQUFHLEdBQUgsR0FBRyxDQUFTO0lBQUcsQ0FBQztJQUd0QyxRQUFRO1FBQ2QsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQ0FBQztRQUNoQyxNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDO1FBQ2hDLE1BQU0sT0FBTyxHQUF1QixJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRTtZQUN2RCxDQUFDLENBQUMsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRSxFQUFFO1lBQzlCLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFFUCxPQUFPOytCQUNvQixJQUFJLEdBQUcsT0FBTzs7Ozs7bUJBSzFCLG1CQUFtQixDQUFDLElBQUksQ0FBQzthQUMvQixtQkFBbUIsQ0FBQyxJQUFJLENBQUM7Ozs7Ozs7Ozs7aUJBVXJCLElBQUk7Ozs7Ozs7Ozs7O0VBV25CLElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFOztLQUV0QyxtQkFBbUIsQ0FBQyxJQUFJLENBQUM7O1dBRW5CLG1CQUFtQixDQUFDLElBQUksQ0FBQyxJQUFJLElBQUksRUFBRSxDQUFDO0lBQzdDLENBQUM7SUFHTyxtQkFBbUIsQ0FBQyxP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