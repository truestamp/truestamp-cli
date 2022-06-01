import { getDescription } from "../_utils.ts";
import { FileType } from "../types/file.ts";
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
            argsCommand += ` \\\n    '${++argIndex}:command:_commands'`;
        }
        if (command.hasArguments() || command.hasCommands(false)) {
            const args = [];
            for (const arg of command.getArguments().slice(1)) {
                const type = command.getType(arg.type);
                if (type && type.handler instanceof FileType) {
                    const fileCompletions = this.getFileCompletions(type);
                    if (arg.variadic) {
                        argIndex++;
                        for (let i = 0; i < 5; i++) {
                            args.push(`${argIndex + i}${arg.optionalValue ? "::" : ":"}${arg.name}:${fileCompletions}`);
                        }
                    }
                    else {
                        args.push(`${++argIndex}${arg.optionalValue ? "::" : ":"}${arg.name}:${fileCompletions}`);
                    }
                }
                else {
                    const completionsPath = path.split(" ").slice(1).join(" ");
                    const action = this.addAction(arg, completionsPath);
                    args.push(`${++argIndex}${arg.optionalValue ? "::" : ":"}${arg.name}:->${action.name}`);
                }
            }
            argsCommand += args.map((arg) => `\\\n    '${arg}'`).join("");
            if (command.hasCommands(false)) {
                argsCommand += ` \\\n    '*::sub command:->command_args'`;
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
            options.push(this.generateOption(command, option, completionsPath, excludedFlags));
        }
        return options;
    }
    generateOption(command, option, completionsPath, excludedOptions) {
        let excludedFlags = option.conflicts?.length
            ? [
                ...excludedOptions,
                ...option.conflicts.map((opt) => "--" + opt.replace(/^--/, "")),
            ]
            : excludedOptions;
        excludedFlags = option.collect ? excludedFlags : [
            ...excludedFlags,
            ...option.flags,
        ];
        let args = "";
        for (const arg of option.args) {
            const type = command.getType(arg.type);
            if (type && type.handler instanceof FileType) {
                const fileCompletions = this.getFileCompletions(type);
                args += `${arg.optionalValue ? "::" : ":"}${arg.name}:${fileCompletions}`;
            }
            else {
                const action = this.addAction(arg, completionsPath);
                args += `${arg.optionalValue ? "::" : ":"}${arg.name}:->${action.name}`;
            }
        }
        const description = getDescription(option.description, true)
            .replace(/\[/g, "\\[")
            .replace(/]/g, "\\]")
            .replace(/"/g, '\\"')
            .replace(/'/g, "'\"'\"'");
        const collect = option.collect ? "*" : "";
        const equalsSign = option.equalsSign ? "=" : "";
        const flags = option.flags.map((flag) => `${flag}${equalsSign}`).join(",");
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
    getFileCompletions(type) {
        if (!(type.handler instanceof FileType)) {
            return "";
        }
        return "_files";
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3pzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfenNoX2NvbXBsZXRpb25zX2dlbmVyYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsY0FBYyxFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRzlDLE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxrQkFBa0IsQ0FBQztBQVU1QyxNQUFNLE9BQU8sdUJBQXVCO0lBUUo7SUFQdEIsT0FBTyxHQUFtQyxJQUFJLEdBQUcsRUFBRSxDQUFDO0lBR3JELE1BQU0sQ0FBQyxRQUFRLENBQUMsR0FBWTtRQUNqQyxPQUFPLElBQUksdUJBQXVCLENBQUMsR0FBRyxDQUFDLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDckQsQ0FBQztJQUVELFlBQThCLEdBQVk7UUFBWixRQUFHLEdBQUgsR0FBRyxDQUFTO0lBQUcsQ0FBQztJQUd0QyxRQUFRO1FBQ2QsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQ0FBQztRQUNoQyxNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDO1FBQ2hDLE1BQU0sT0FBTyxHQUF1QixJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRTtZQUN2RCxDQUFDLENBQUMsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRSxFQUFFO1lBQzlCLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFFUCxPQUFPOytCQUNvQixJQUFJLEdBQUcsT0FBTzs7Ozs7bUJBSzFCLG1CQUFtQixDQUFDLElBQUksQ0FBQzthQUMvQixtQkFBbUIsQ0FBQyxJQUFJLENBQUM7Ozs7Ozs7Ozs7aUJBVXJCLElBQUk7Ozs7Ozs7Ozs7O0VBV25CLElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFOztLQUV0QyxtQkFBbUIsQ0FBQyxJQUFJ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LE1BQU07Z0JBQ1osR0FBRzthQUNKLENBQUMsQ0FBQztTQUNKO1FBRUQsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxNQUFNLENBQXNCLENBQUM7SUFDdkQsQ0FBQztJQUVPLGVBQWUsQ0FBQyxPQUFnQjtRQUN0QyxJQUFJLE9BQU8sR0FBYSxFQUFFLENBQUM7UUFFM0IsSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRTtZQUNyQixPQUFPLEdBQUcsS0FBSztpQkFDWixJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQztpQkFDbEIsR0FBRyxDQUFDLENBQUMsQ0FBQyxJQUFJLEVBQUUsTUFBTSxDQUFDLEVBQUUsRUFBRSxDQUN0QixHQUFHLElBQUksT0FDTCxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUN4QyxhQUFhLE1BQU0sQ0FBQyxHQUFHLENBQUMsSUFBSSxJQUFJLE1BQU0sQ0FBQyxHQUFHLENBQUMsTUFBTSxJQUFJLE1BQU0sQ0FBQyxHQUFHLEtBQUssQ0FDckUsQ0FBQztTQUNMO1FBRUQsSUFBSSxPQUFPLENBQUMsV0FBVyxDQUFDLEtBQUssQ0FBQyxFQUFFO1lBQzlCLE9BQU8sQ0FBQyxPQUFPLENBQUMsZ0NBQWdDLENBQUMsQ0FBQztTQUNuRDtRQUVELElBQUksT0FBTyxDQUFDLE1BQU0sRUFBRTtZQUNsQixPQUFPLCtCQUErQixPQUFPLENBQUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxVQUFVLENBQUM7U0FDeEU7UUFFRCxPQUFPLEVBQUUsQ0FBQztJQUNaLENBQUM7Q0FDRjtBQUVELFNBQVMsbUJBQW1CLENBQUMsR0FBVztJQUN0QyxPQUFPLEdBQUcsQ0FBQyxPQUFPLENBQUMsZUFBZSxFQUFFLEdBQUcsQ0FBQyxDQUFDO0FBQzNDLENBQUMiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBnZXREZXNjcmlwdGlvbiB9IGZyb20gXCIuLi9fdXRpbHMudHNcIjtcbmltcG9ydCB0eXBlIHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgdHlwZSB7IElBcmd1bWVudCwgSU9wdGlvbiwgSVR5cGUgfSBmcm9tIFwiLi4vdHlwZXMudHNcIjtcbmltcG9ydCB7IEZpbGVUeXBlIH0gZnJvbSBcIi4uL3R5cGVzL2ZpbGUudHNcIjtcblxuaW50ZXJmYWNlIElDb21wbGV0aW9uQWN0aW9uIHtcbiAgYXJnOiBJQXJndW1lbnQ7XG4gIGxhYmVsOiBzdHJpbmc7XG4gIG5hbWU6IHN0cmluZztcbiAgY21kOiBzdHJpbmc7XG59XG5cbi8qKiBHZW5lcmF0ZXMgenNoIGNvbXBsZXRpb25zIHNjcmlwdC4gKi9cbmV4cG9ydCBjbGFzcyBac2hDb21wbGV0aW9uc0dlbmVyYXRvciB7XG4gIHByaXZhdGUgYWN0aW9uczogTWFwPHN0cmluZywgSUNvbXBsZXRpb25BY3Rpb24+ID0gbmV3IE1hcCgpO1xuXG4gIC8qKiBHZW5lcmF0ZXMgenNoIGNvbXBsZXRpb25zIHNjcmlwdCBmb3IgZ2l2ZW4gY29tbWFuZC4gKi9cbiAgcHVibGljIHN0YXRpYyBnZW5lcmF0ZShjbWQ6IENvbW1hbmQpIHtcbiAgICByZXR1cm4gbmV3IFpzaENvbXBsZXRpb25zR2VuZXJhdG9yKGNtZCkuZ2VuZXJhdGUoKTtcbiAgfVxuXG4gIHByaXZhdGUgY29uc3RydWN0b3IocHJvdGVjdGVkIGNtZDogQ29tbWFuZCkge31cblxuICAvKiogR2VuZXJhdGVzIHpzaCBjb21wbGV0aW9ucyBjb2RlLiAqL1xuICBwcml2YXRlIGdlbmVyYXRlKCk6IHN0cmluZyB7XG4gICAgY29uc3QgcGF0aCA9IHRoaXMuY21kLmdldFBhdGgoKTtcbiAgICBjb25zdCBuYW1lID0gdGhpcy5jbWQuZ2V0TmFtZSgpO1xuICAgIGNvbnN0IHZlcnNpb246IHN0cmluZyB8IHVuZGVmaW5lZCA9IHRoaXMuY21kLmdldFZlcnNpb24oKVxuICAgICAgPyBgIHYke3RoaXMuY21kLmdldFZlcnNpb24oKX1gXG4gICAgICA6IFwiXCI7XG5cbiAgICByZXR1cm4gYCMhL3Vzci9iaW4vZW52IHpzaFxuIyB6c2ggY29tcGxldGlvbiBzdXBwb3J0IGZvciAke3BhdGh9JHt2ZXJzaW9ufVxuXG5hdXRvbG9hZCAtVSBpcy1hdC1sZWFzdFxuXG4jIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIxNTRcbigoICQrZnVuY3Rpb25zW19fJHtyZXBsYWNlU3BlY2lhbENoYXJzKG5hbWUpfV9jb21wbGV0ZV0gKSkgfHxcbmZ1bmN0aW9uIF9fJHtyZXBsYWNlU3BlY2lhbENoYXJzKG5hbWUpfV9jb21wbGV0ZSB7XG4gIGxvY2FsIG5hbWU9XCIkMVwiOyBzaGlmdFxuICBsb2NhbCBhY3Rpb249XCIkMVwiOyBzaGlmdFxuICBpbnRlZ2VyIHJldD0xXG4gIGxvY2FsIC1hIHZhbHVlc1xuICBsb2NhbCBleHBsIGxpbmVzXG4gIF90YWdzIFwiJG5hbWVcIlxuICB3aGlsZSBfdGFnczsgZG9cbiAgICBpZiBfcmVxdWVzdGVkIFwiJG5hbWVcIjsgdGhlblxuICAgICAgIyBzaGVsbGNoZWNrIGRpc2FibGU9U0MyMDM0XG4gICAgICBsaW5lcz1cIiQoJHtuYW1lfSBjb21wbGV0aW9ucyBjb21wbGV0ZSBcIlxcJHthY3Rpb259XCIgXCJcXCR7QH1cIilcIlxuICAgICAgdmFsdWVzPShcIlxcJHsocHM6XFxcXG46KWxpbmVzfVwiKVxuICAgICAgaWYgKCggXFwkeyN2YWx1ZXNbQF19ICkpOyB0aGVuXG4gICAgICAgIHdoaWxlIF9uZXh0X2xhYmVsIFwiJG5hbWVcIiBleHBsIFwiJGFjdGlvblwiOyBkb1xuICAgICAgICAgIGNvbXBhZGQgLVMgJycgXCJcXCR7ZXhwbFtAXX1cIiBcIlxcJHt2YWx1ZXNbQF19XCJcbiAgICAgICAgZG9uZVxuICAgICAgZmlcbiAgICBmaVxuICBkb25lXG59XG5cbiR7dGhpcy5nZW5lcmF0ZUNvbXBsZXRpb25zKHRoaXMuY21kKS50cmltKCl9XG5cbiMgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyhwYXRoKX0gXCJcXCR7QH1cIlxuXG5jb21wZGVmIF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9ICR7cGF0aH1gO1xuICB9XG5cbiAgLyoqIEdlbmVyYXRlcyB6c2ggY29tcGxldGlvbnMgbWV0aG9kIGZvciBnaXZlbiBjb21tYW5kIGFuZCBjaGlsZCBjb21tYW5kcy4gKi9cbiAgcHJpdmF0ZSBnZW5lcmF0ZUNvbXBsZXRpb25zKGNvbW1hbmQ6IENvbW1hbmQsIHBhdGggPSBcIlwiKTogc3RyaW5nIHtcbiAgICBpZiAoXG4gICAgICAhY29tbWFuZC5oYXNDb21tYW5kcyhmYWxzZSkgJiYgIWNvbW1hbmQuaGFzT3B0aW9ucyhmYWxzZSkgJiZcbiAgICAgICFjb21tYW5kLmhhc0FyZ3VtZW50cygpXG4gICAgKSB7XG4gICAgICByZXR1cm4gXCJcIjtcbiAgICB9XG5cbiAgICBwYXRoID0gKHBhdGggPyBwYXRoICsgXCIgXCIgOiBcIlwiKSArIGNvbW1hbmQuZ2V0TmFtZSgpO1xuXG4gICAgcmV0dXJuIGAjIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIxNTRcbigoICQrZnVuY3Rpb25zW18ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9XSApKSB8fFxuZnVuY3Rpb24gXyR7cmVwbGFjZVNwZWNpYWxDaGFycyhwYXRoKX0oKSB7YCArXG4gICAgICAoIWNvbW1hbmQuZ2V0UGFyZW50KClcbiAgICAgICAgPyBgXG4gIGxvY2FsIHN0YXRlYFxuICAgICAgICA6IFwiXCIpICtcbiAgICAgIHRoaXMuZ2VuZXJhdGVDb21tYW5kQ29tcGxldGlvbnMoY29tbWFuZCwgcGF0aCkgK1xuICAgICAgdGhpcy5nZW5lcmF0ZVN1YkNvbW1hbmRDb21wbGV0aW9ucyhjb21tYW5kLCBwYXRoKSArXG4gICAgICB0aGlzLmdlbmVyYXRlQXJndW1lbnRDb21wbGV0aW9ucyhjb21tYW5kLCBwYXRoKSArXG4gICAgICB0aGlzLmdlbmVyYXRlQWN0aW9ucyhjb21tYW5kKSArXG4gICAgICBgXFxufVxcblxcbmAgK1xuICAgICAgY29tbWFuZC5nZXRDb21tYW5kcyhmYWxzZSlcbiAgICAgICAgLmZpbHRlcigoc3ViQ29tbWFuZDogQ29tbWFuZCkgPT4gc3ViQ29tbWFuZCAhPT0gY29tbWFuZClcbiAgICAgICAgLm1hcCgoc3ViQ29tbWFuZDogQ29tbWFuZCkgPT5cbiAgICAgICAgICB0aGlzLmdlbmVyYXRlQ29tcGxldGlvbnMoc3ViQ29tbWFuZCwgcGF0aClcbiAgICAgICAgKVxuICAgICAgICAuam9pbihcIlwiKTtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVDb21tYW5kQ29tcGxldGlvbnMoY29tbWFuZDogQ29tbWFuZCwgcGF0aDogc3RyaW5nKTogc3RyaW5nIHtcbiAgICBjb25zdCBjb21tYW5kcyA9IGNvbW1hbmQuZ2V0Q29tbWFuZHMoZmFsc2UpO1xuXG4gICAgbGV0IGNvbXBsZXRpb25zOiBzdHJpbmcgPSBjb21tYW5kc1xuICAgICAgLm1hcCgoc3ViQ29tbWFuZDogQ29tbWFuZCkgPT5cbiAgICAgICAgYCcke3N1YkNvbW1hbmQuZ2V0TmFtZSgpfToke3N1YkNvbW1hbmQuZ2V0U2hvcnREZXNjcmlwdGlvbigpfSdgXG4gICAgICApXG4gICAgICAuam9pbihcIlxcbiAgICAgIFwiKTtcblxuICAgIGlmIChjb21wbGV0aW9ucykge1xuICAgICAgY29tcGxldGlvbnMgPSBgXG4gICAgbG9jYWwgLWEgY29tbWFuZHNcbiAgICAjIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIwMzRcbiAgICBjb21tYW5kcz0oXG4gICAgICAke2NvbXBsZXRpb25zfVxuICAgIClcbiAgICBfZGVzY3JpYmUgJ2NvbW1hbmQnIGNvbW1hbmRzYDtcbiAgICB9XG5cbiAgICAvLyBvbmx5IGNvbXBsZXRlIGZpcnN0IGFyZ3VtZW50LCByZXN0IGFyZ3VtZW50cyBhcmUgY29tcGxldGVkIHdpdGggX2FyZ3VtZW50cy5cbiAgICBpZiAoY29tbWFuZC5oYXNBcmd1bWVudHMoKSkge1xuICAgICAgY29uc3QgY29tcGxldGlvbnNQYXRoOiBzdHJpbmcgPSBwYXRoLnNwbGl0KFwiIFwiKS5zbGljZSgxKS5qb2luKFwiIFwiKTtcbiAgICAgIGNvbnN0IGFyZzogSUFyZ3VtZW50ID0gY29tbWFuZC5nZXRBcmd1bWVudHMoKVswXTtcbiAgICAgIGNvbnN0IGFjdGlvbiA9IHRoaXMuYWRkQWN0aW9uKGFyZywgY29tcGxldGlvbnNQYXRoKTtcbiAgICAgIGlmIChhY3Rpb24gJiYgY29tbWFuZC5nZXRDb21wbGV0aW9uKGFyZy5hY3Rpb24pKSB7XG4gICAgICAgIGNvbXBsZXRpb25zICs9IGBcXG4gICAgX18ke1xuICAgICAgICAgIHJlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKVxuICAgICAgICB9X2NvbXBsZXRlICR7YWN0aW9uLmFyZy5uYW1lfSAke2FjdGlvbi5hcmcuYWN0aW9ufSAke2FjdGlvbi5jbWR9YDtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBpZiAoY29tcGxldGlvbnMpIHtcbiAgICAgIGNvbXBsZXRpb25zID0gYFxcblxcbiAgZnVuY3Rpb24gX2NvbW1hbmRzKCkgeyR7Y29tcGxldGlvbnN9XFxuICB9YDtcbiAgICB9XG5cbiAgICByZXR1cm4gY29tcGxldGlvbnM7XG4gIH1cblxuICBwcml2YXRlIGdlbmVyYXRlU3ViQ29tbWFuZENvbXBsZXRpb25zKFxuICAgIGNvbW1hbmQ6IENvbW1hbmQsXG4gICAgcGF0aDogc3RyaW5nLFxuICApOiBzdHJpbmcge1xuICAgIGlmIChjb21tYW5kLmhhc0NvbW1hbmRzKGZhbHNlKSkge1xuICAgICAgY29uc3QgYWN0aW9uczogc3RyaW5nID0gY29tbWFuZFxuICAgICAgICAuZ2V0Q29tbWFuZHMoZmFsc2UpXG4gICAgICAgIC5tYXAoKGNvbW1hbmQ6IENvbW1hbmQpID0+XG4gICAgICAgICAgYCR7Y29tbWFuZC5nZXROYW1lKCl9KSBfJHtcbiAgICAgICAgICAgIHJlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCArIFwiIFwiICsgY29tbWFuZC5nZXROYW1lKCkpXG4gICAgICAgICAgfSA7O2BcbiAgICAgICAgKVxuICAgICAgICAuam9pbihcIlxcbiAgICAgIFwiKTtcblxuICAgICAgcmV0dXJuIGBcXG5cbiAgZnVuY3Rpb24gX2NvbW1hbmRfYXJncygpIHtcbiAgICBjYXNlIFwiXFwke3dvcmRzWzFdfVwiIGluXFxuICAgICAgJHthY3Rpb25zfVxcbiAgICBlc2FjXG4gIH1gO1xuICAgIH1cblxuICAgIHJldHVybiBcIlwiO1xuICB9XG5cbiAgcHJpdmF0ZSBnZW5lcmF0ZUFyZ3VtZW50Q29tcGxldGlvbnMoY29tbWFuZDogQ29tbWFuZCwgcGF0aDogc3RyaW5nKTogc3RyaW5nIHtcbiAgICAvKiBjbGVhciBhY3Rpb25zIGZyb20gcHJldmlvdXNseSBwYXJzZWQgY29tbWFuZC4gKi9cbiAgICB0aGlzLmFjdGlvbnMuY2xlYXIoKTtcblxuICAgIGNvbnN0IG9wdGlvbnM6IHN0cmluZ1tdID0gdGhpcy5nZW5lcmF0ZU9wdGlvbnMoY29tbWFuZCwgcGF0aCk7XG5cbiAgICBsZXQgYXJnSW5kZXggPSAwO1xuICAgIC8vIEBUT0RPOiBhZGQgc3RvcCBlYXJseSBvcHRpb246IC1BIFwiLSpcIlxuICAgIC8vIGh0dHA6Ly96c2guc291cmNlZm9yZ2UubmV0L0RvYy9SZWxlYXNlL0NvbXBsZXRpb24tU3lzdGVtLmh0bWxcbiAgICBsZXQgYXJnc0NvbW1hbmQgPSBcIlxcblxcbiAgX2FyZ3VtZW50cyAtdyAtcyAtUyAtQ1wiO1xuXG4gICAgaWYgKGNvbW1hbmQuaGFzT3B0aW9ucygpKSB7XG4gICAgICBhcmdzQ29tbWFuZCArPSBgIFxcXFxcXG4gICAgJHtvcHRpb25zLmpvaW4oXCIgXFxcXFxcbiAgICBcIil9YDtcbiAgICB9XG5cbiAgICBpZiAoXG4gICAgICBjb21tYW5kLmhhc0NvbW1hbmRzKGZhbHNlKSB8fCAoXG4gICAgICAgIGNvbW1hbmQuZ2V0QXJndW1lbnRzKClcbiAgICAgICAgICAuZmlsdGVyKChhcmcpID0+IGNvbW1hbmQuZ2V0Q29tcGxldGlvbihhcmcuYWN0aW9uKSkubGVuZ3RoXG4gICAgICApXG4gICAgKSB7XG4gICAgICBhcmdzQ29tbWFuZCArPSBgIFxcXFxcXG4gICAgJyR7KythcmdJbmRleH06Y29tbWFuZDpfY29tbWFuZHMnYDtcbiAgICB9XG5cbiAgICBpZiAoY29tbWFuZC5oYXNBcmd1bWVudHMoKSB8fCBjb21tYW5kLmhhc0NvbW1hbmRzKGZhbHNlKSkge1xuICAgICAgY29uc3QgYXJnczogc3RyaW5nW10gPSBbXTtcblxuICAgICAgLy8gZmlyc3QgYXJndW1lbnQgaXMgY29tcGxldGVkIHRvZ2V0aGVyIHdpdGggY29tbWFuZHMuXG4gICAgICBmb3IgKGNvbnN0IGFyZyBvZiBjb21tYW5kLmdldEFyZ3VtZW50cygpLnNsaWNlKDEpKSB7XG4gICAgICAgIGNvbnN0IHR5cGUgPSBjb21tYW5kLmdldFR5cGUoYXJnLnR5cGUpO1xuICAgICAgICBpZiAodHlwZSAmJiB0eXBlLmhhbmRsZXIgaW5zdGFuY2VvZiBGaWxlVHlwZSkge1xuICAgICAgICAgIGNvbnN0IGZpbGVDb21wbGV0aW9ucyA9IHRoaXMuZ2V0RmlsZUNvbXBsZXRpb25zKHR5cGUpO1xuICAgICAgICAgIGlmIChhcmcudmFyaWFkaWMpIHtcbiAgICAgICAgICAgIGFyZ0luZGV4Kys7XG4gICAgICAgICAgICBmb3IgKGxldCBpID0gMDsgaSA8IDU7IGkrKykge1xuICAgICAgICAgICAgICBhcmdzLnB1c2goXG4gICAgICAgICAgICAgICAgYCR7YXJnSW5kZXggKyBpfSR7XG4gICAgICAgICAgICAgICAgICBhcmcub3B0aW9uYWxWYWx1ZSA/IFwiOjpcIiA6IFwiOlwiXG4gICAgICAgICAgICAgICAgfSR7YXJnLm5hbWV9OiR7ZmlsZUNvbXBsZXRpb25zfWAsXG4gICAgICAgICAgICAgICk7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICAgIGFyZ3MucHVzaChcbiAgICAgICAgICAgICAgYCR7KythcmdJbmRleH0ke1xuICAgICAgICAgICAgICAgIGFyZy5vcHRpb25hbFZhbHVlID8gXCI6OlwiIDogXCI6XCJcbiAgICAgICAgICAgICAgfSR7YXJnLm5hbWV9OiR7ZmlsZUNvbXBsZXRpb25zfWAsXG4gICAgICAgICAgICApO1xuICAgICAgICAgIH1cbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBjb25zdCBjb21wbGV0aW9uc1BhdGg6IHN0cmluZyA9IHBhdGguc3BsaXQoXCIgXCIpLnNsaWNlKDEpLmpvaW4oXCIgXCIpO1xuICAgICAgICAgIGNvbnN0IGFjdGlvbiA9IHRoaXMuYWRkQWN0aW9uKGFyZywgY29tcGxldGlvbnNQYXRoKTtcbiAgICAgICAgICBhcmdzLnB1c2goXG4gICAgICAgICAgICBgJHsrK2FyZ0luZGV4fSR7XG4gICAgICAgICAgICAgIGFyZy5vcHRpb25hbFZhbHVlID8gXCI6OlwiIDogXCI6XCJcbiAgICAgICAgICAgIH0ke2FyZy5uYW1lfTotPiR7YWN0aW9uLm5hbWV9YCxcbiAgICAgICAgICApO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGFyZ3NDb21tYW5kICs9IGFyZ3MubWFwKChhcmc6IHN0cmluZykgPT4gYFxcXFxcXG4gICAgJyR7YXJnfSdgKS5qb2luKFwiXCIpO1xuXG4gICAgICBpZiAoY29tbWFuZC5oYXNDb21tYW5kcyhmYWxzZSkpIHtcbiAgICAgICAgYXJnc0NvbW1hbmQgKz0gYCBcXFxcXFxuICAgICcqOjpzdWIgY29tbWFuZDotPmNvbW1hbmRfYXJncydgO1xuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiBhcmdzQ29tbWFuZDtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb25zKGNvbW1hbmQ6IENvbW1hbmQsIHBhdGg6IHN0cmluZykge1xuICAgIGNvbnN0IG9wdGlvbnM6IHN0cmluZ1tdID0gW107XG4gICAgY29uc3QgY21kQXJnczogc3RyaW5nW10gPSBwYXRoLnNwbGl0KFwiIFwiKTtcbiAgICBjb25zdCBfYmFzZU5hbWU6IHN0cmluZyA9IGNtZEFyZ3Muc2hpZnQoKSBhcyBzdHJpbmc7XG4gICAgY29uc3QgY29tcGxldGlvbnNQYXRoOiBzdHJpbmcgPSBjbWRBcmdzLmpvaW4oXCIgXCIpO1xuXG4gICAgY29uc3QgZXhjbHVkZWRGbGFnczogc3RyaW5nW10gPSBjb21tYW5kLmdldE9wdGlvbnMoZmFsc2UpXG4gICAgICAubWFwKChvcHRpb24pID0+IG9wdGlvbi5zdGFuZGFsb25lID8gb3B0aW9uLmZsYWdzIDogZmFsc2UpXG4gICAgICAuZmxhdCgpXG4gICAgICAuZmlsdGVyKChmbGFnKSA9PiB0eXBlb2YgZmxhZyA9PT0gXCJzdHJpbmdcIikgYXMgc3RyaW5nW107XG5cbiAgICBmb3IgKGNvbnN0IG9wdGlvbiBvZiBjb21tYW5kLmdldE9wdGlvbnMoZmFsc2UpKSB7XG4gICAgICBvcHRpb25zLnB1c2goXG4gICAgICAgIHRoaXMuZ2VuZXJhdGVPcHRpb24oY29tbWFuZCwgb3B0aW9uLCBjb21wbGV0aW9uc1BhdGgsIGV4Y2x1ZGVkRmxhZ3MpLFxuICAgICAgKTtcbiAgICB9XG5cbiAgICByZXR1cm4gb3B0aW9ucztcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb24oXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBvcHRpb246IElPcHRpb24sXG4gICAgY29tcGxldGlvbnNQYXRoOiBzdHJpbmcsXG4gICAgZXhjbHVkZWRPcHRpb25zOiBzdHJpbmdbXSxcbiAgKTogc3RyaW5nIHtcbiAgICBsZXQgZXhjbHVkZWRGbGFncyA9IG9wdGlvbi5jb25mbGljdHM/Lmxlbmd0aFxuICAgICAgPyBbXG4gICAgICAgIC4uLmV4Y2x1ZGVkT3B0aW9ucyxcbiAgICAgICAgLi4ub3B0aW9uLmNvbmZsaWN0cy5tYXAoKG9wdCkgPT4gXCItLVwiICsgb3B0LnJlcGxhY2UoL14tLS8sIFwiXCIpKSxcbiAgICAgIF1cbiAgICAgIDogZXhjbHVkZWRPcHRpb25zO1xuICAgIGV4Y2x1ZGVkRmxhZ3MgPSBvcHRpb24uY29sbGVjdCA/IGV4Y2x1ZGVkRmxhZ3MgOiBbXG4gICAgICAuLi5leGNsdWRlZEZsYWdzLFxuICAgICAgLi4ub3B0aW9uLmZsYWdzLFxuICAgIF07XG5cbiAgICBsZXQgYXJncyA9IFwiXCI7XG4gICAgZm9yIChjb25zdCBhcmcgb2Ygb3B0aW9uLmFyZ3MpIHtcbiAgICAgIGNvbnN0IHR5cGUgPSBjb21tYW5kLmdldFR5cGUoYXJnLnR5cGUpO1xuICAgICAgaWYgKHR5cGUgJiYgdHlwZS5oYW5kbGVyIGluc3RhbmNlb2YgRmlsZVR5cGUpIHtcbiAgICAgICAgY29uc3QgZmlsZUNvbXBsZXRpb25zID0gdGhpcy5nZXRGaWxlQ29tcGxldGlvbnModHlwZSk7XG4gICAgICAgIGFyZ3MgKz0gYCR7XG4gICAgICAgICAgYXJnLm9wdGlvbmFsVmFsdWUgPyBcIjo6XCIgOiBcIjpcIlxuICAgICAgICB9JHthcmcubmFtZX06JHtmaWxlQ29tcGxldGlvbnN9YDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIGNvbnN0IGFjdGlvbiA9IHRoaXMuYWRkQWN0aW9uKGFyZywgY29tcGxldGlvbnNQYXRoKTtcbiAgICAgICAgYXJncyArPSBgJHthcmcub3B0aW9uYWxWYWx1ZSA/IFwiOjpcIiA6IFwiOlwifSR7YXJnLm5hbWV9Oi0+JHthY3Rpb24ubmFtZX1gO1xuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IGRlc2NyaXB0aW9uOiBzdHJpbmcgPSBnZXREZXNjcmlwdGlvbihvcHRpb24uZGVzY3JpcHRpb24sIHRydWUpXG4gICAgICAvLyBlc2NhcGUgYnJhY2tldHMgYW5kIHF1b3Rlc1xuICAgICAgLnJlcGxhY2UoL1xcWy9nLCBcIlxcXFxbXCIpXG4gICAgICAucmVwbGFjZSgvXS9nLCBcIlxcXFxdXCIpXG4gICAgICAucmVwbGFjZSgvXCIvZywgJ1xcXFxcIicpXG4gICAgICAucmVwbGFjZSgvJy9nLCBcIidcXFwiJ1xcXCInXCIpO1xuXG4gICAgY29uc3QgY29sbGVjdDogc3RyaW5nID0gb3B0aW9uLmNvbGxlY3QgPyBcIipcIiA6IFwiXCI7XG4gICAgY29uc3QgZXF1YWxzU2lnbiA9IG9wdGlvbi5lcXVhbHNTaWduID8gXCI9XCIgOiBcIlwiO1xuICAgIGNvbnN0IGZsYWdzID0gb3B0aW9uLmZsYWdzLm1hcCgoZmxhZykgPT4gYCR7ZmxhZ30ke2VxdWFsc1NpZ259YCkuam9pbihcIixcIik7XG5cbiAgICBpZiAob3B0aW9uLnN0YW5kYWxvbmUpIHtcbiAgICAgIHJldHVybiBgJygtICopJ3ske2NvbGxlY3R9JHtmbGFnc319J1ske2Rlc2NyaXB0aW9ufV0ke2FyZ3N9J2A7XG4gICAgfSBlbHNlIHtcbiAgICAgIGNvbnN0IGV4Y2x1ZGVkOiBzdHJpbmcgPSBleGNsdWRlZEZsYWdzLmxlbmd0aFxuICAgICAgICA/IGAnKCR7ZXhjbHVkZWRGbGFncy5qb2luKFwiIFwiKX0pJ2BcbiAgICAgICAgOiBcIlwiO1xuICAgICAgaWYgKGNvbGxlY3QgfHwgZmxhZ3MubGVuZ3RoID4gMSkge1xuICAgICAgICByZXR1cm4gYCR7ZXhjbHVkZWR9eyR7Y29sbGVjdH0ke2ZsYWdzfX0nWyR7ZGVzY3JpcHRpb259XSR7YXJnc30nYDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJldHVybiBgJHtleGNsdWRlZH0ke2ZsYWdzfSdbJHtkZXNjcmlwdGlvbn1dJHthcmdzfSdgO1xuICAgICAgfVxuICAgIH1cbiAgfVxuXG4gIHByaXZhdGUgZ2V0RmlsZUNvbXBsZXRpb25zKHR5cGU6IElUeXBlKSB7XG4gICAgaWYgKCEodHlwZS5oYW5kbGVyIGluc3RhbmNlb2YgRmlsZVR5cGUpKSB7XG4gICAgICByZXR1cm4gXCJcIjtcbiAgICB9XG4gICAgcmV0dXJuIFwiX2ZpbGVzXCI7XG4gICAgLy8gY29uc3QgZmlsZU9wdHMgPSB0eXBlLmhhbmRsZXIuZ2V0T3B0aW9ucygpO1xuICAgIC8vIGxldCBmaWxlQ29tcGxldGlvbnMgPSBcIl9maWxlc1wiO1xuICAgIC8vIGlmIChmaWxlT3B0cy5kaXJzT25seSkge1xuICAgIC8vICAgZmlsZUNvbXBsZXRpb25zICs9IFwiIC0vXCI7XG4gICAgLy8gfVxuICAgIC8vIGlmIChmaWxlT3B0cy5wYXR0ZXJuKSB7XG4gICAgLy8gICBmaWxlQ29tcGxldGlvbnMgKz0gJyAtZyBcIicgKyBmaWxlT3B0cy5wYXR0ZXJuICsgJ1wiJztcbiAgICAvLyB9XG4gICAgLy8gaWYgKGZpbGVPcHRzLmlnbm9yZSkge1xuICAgIC8vICAgZmlsZUNvbXBsZXRpb25zICs9IFwiIC1GIFwiICsgZmlsZU9wdHMuaWdub3JlO1xuICAgIC8vIH1cbiAgICAvLyByZXR1cm4gZmlsZUNvbXBsZXRpb25zO1xuICB9XG5cbiAgcHJpdmF0ZSBhZGRBY3Rpb24oYXJnOiBJQXJndW1lbnQsIGNtZDogc3RyaW5nKTogSUNvbXBsZXRpb25BY3Rpb24ge1xuICAgIGNvbnN0IGFjdGlvbiA9IGAke2FyZy5uYW1lfS0ke2FyZy5hY3Rpb259YDtcblxuICAgIGlmICghdGhpcy5hY3Rpb25zLmhhcyhhY3Rpb24pKSB7XG4gICAgICB0aGlzLmFjdGlvbnMuc2V0KGFjdGlvbiwge1xuICAgICAgICBhcmc6IGFyZyxcbiAgICAgICAgbGFiZWw6IGAke2FyZy5uYW1lfTogJHthcmcuYWN0aW9ufWAsXG4gICAgICAgIG5hbWU6IGFjdGlvbixcbiAgICAgICAgY21kLFxuICAgICAgfSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHRoaXMuYWN0aW9ucy5nZXQoYWN0aW9uKSBhcyBJQ29tcGxldGlvbkFjdGlvbjtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVBY3Rpb25zKGNvbW1hbmQ6IENvbW1hbmQpOiBzdHJpbmcge1xuICAgIGxldCBhY3Rpb25zOiBzdHJpbmdbXSA9IFtdO1xuXG4gICAgaWYgKHRoaXMuYWN0aW9ucy5zaXplKSB7XG4gICAgICBhY3Rpb25zID0gQXJyYXlcbiAgICAgICAgLmZyb20odGhpcy5hY3Rpb25zKVxuICAgICAgICAubWFwKChbbmFtZSwgYWN0aW9uXSkgPT5cbiAgICAgICAgICBgJHtuYW1lfSkgX18ke1xuICAgICAgICAgICAgcmVwbGFjZVNwZWNpYWxDaGFycyh0aGlzLmNtZC5nZXROYW1lKCkpXG4gICAgICAgICAgfV9jb21wbGV0ZSAke2FjdGlvbi5hcmcubmFtZX0gJHthY3Rpb24uYXJnLmFjdGlvbn0gJHthY3Rpb24uY21kfSA7O2BcbiAgICAgICAgKTtcbiAgICB9XG5cbiAgICBpZiAoY29tbWFuZC5oYXNDb21tYW5kcyhmYWxzZSkpIHtcbiAgICAgIGFjdGlvbnMudW5zaGlmdChgY29tbWFuZF9hcmdzKSBfY29tbWFuZF9hcmdzIDs7YCk7XG4gICAgfVxuXG4gICAgaWYgKGFjdGlvbnMubGVuZ3RoKSB7XG4gICAgICByZXR1cm4gYFxcblxcbiAgY2FzZSBcIiRzdGF0ZVwiIGluXFxuICAgICR7YWN0aW9ucy5qb2luKFwiXFxuICAgIFwiKX1cXG4gIGVzYWNgO1xuICAgIH1cblxuICAgIHJldHVybiBcIlwiO1xuICB9XG59XG5cbmZ1bmN0aW9uIHJlcGxhY2VTcGVjaWFsQ2hhcnMoc3RyOiBzdHJpbmcpOiBzdHJpbmcge1xuICByZXR1cm4gc3RyLnJlcGxhY2UoL1teYS16QS1aMC05XS9nLCBcIl9cIik7XG59XG4iXX0=