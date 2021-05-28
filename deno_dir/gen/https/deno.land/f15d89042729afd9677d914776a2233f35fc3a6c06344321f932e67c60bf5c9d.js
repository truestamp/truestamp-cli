export class BashCompletionsGenerator {
    cmd;
    static generate(cmd) {
        return new BashCompletionsGenerator(cmd).generate();
    }
    constructor(cmd) {
        this.cmd = cmd;
    }
    generate() {
        const path = this.cmd.getPath();
        const version = this.cmd.getVersion()
            ? ` v${this.cmd.getVersion()}`
            : "";
        return `#!/usr/bin/env bash
# bash completion support for ${path}${version}

_${replaceSpecialChars(path)}() {
  local word cur prev
  local -a opts
  COMPREPLY=()
  cur="\${COMP_WORDS[COMP_CWORD]}"
  prev="\${COMP_WORDS[COMP_CWORD-1]}"
  cmd="_"
  opts=()

  _${replaceSpecialChars(this.cmd.getName())}_complete() {
    local action="$1"; shift
    mapfile -t values < <( ${this.cmd.getName()} completions complete "\${action}" "\${@}" )
    for i in "\${values[@]}"; do
      opts+=("$i")
    done
  }

  ${this.generateCompletions(this.cmd).trim()}

  for word in "\${COMP_WORDS[@]}"; do
    case "\${word}" in
      -*) ;;
      *)
        cmd_tmp="\${cmd}_\${word//[^[:alnum:]]/_}"
        if type "\${cmd_tmp}" &>/dev/null; then
          cmd="\${cmd_tmp}"
        fi
    esac
  done

  \${cmd}

  if [[ \${#opts[@]} -eq 0 ]]; then
    # shellcheck disable=SC2207
    COMPREPLY=($(compgen -f "\${cur}"))
    return 0
  fi

  local values
  values="$( printf "\\n%s" "\${opts[@]}" )"
  local IFS=$'\\n'
  # shellcheck disable=SC2207
  local result=($(compgen -W "\${values[@]}" -- "\${cur}"))
  if [[ \${#result[@]} -eq 0 ]]; then
    # shellcheck disable=SC2207
    COMPREPLY=($(compgen -f "\${cur}"))
  else
    # shellcheck disable=SC2207
    COMPREPLY=($(printf '%q\\n' "\${result[@]}"))
  fi

  return 0
}

complete -F _${replaceSpecialChars(path)} -o bashdefault -o default ${path}
`;
    }
    generateCompletions(command, path = "", index = 1) {
        path = (path ? path + " " : "") + command.getName();
        const commandCompletions = this.generateCommandCompletions(command, path, index);
        const childCommandCompletions = command.getCommands(false)
            .filter((subCommand) => subCommand !== command)
            .map((subCommand) => this.generateCompletions(subCommand, path, index + 1))
            .join("");
        return `${commandCompletions}

${childCommandCompletions}`;
    }
    generateCommandCompletions(command, path, index) {
        const flags = this.getFlags(command);
        const childCommandNames = command.getCommands(false)
            .map((childCommand) => childCommand.getName());
        const completionsPath = ~path.indexOf(" ")
            ? " " + path.split(" ").slice(1).join(" ")
            : "";
        const optionArguments = this.generateOptionArguments(command, completionsPath);
        const completionsCmd = this.generateCommandCompletionsCommand(command.getArguments(), completionsPath);
        return `  __${replaceSpecialChars(path)}() {
    opts=(${[...flags, ...childCommandNames].join(" ")})
    ${completionsCmd}
    if [[ \${cur} == -* || \${COMP_CWORD} -eq ${index} ]] ; then
      return 0
    fi
    ${optionArguments}
  }`;
    }
    getFlags(command) {
        return command.getOptions(false)
            .map((option) => option.flags)
            .flat();
    }
    generateOptionArguments(command, completionsPath) {
        let opts = "";
        const options = command.getOptions(false);
        if (options.length) {
            opts += 'case "${prev}" in';
            for (const option of options) {
                const flags = option.flags
                    .map((flag) => flag.trim())
                    .join("|");
                const completionsCmd = this.generateOptionCompletionsCommand(option.args, completionsPath, { standalone: option.standalone });
                opts += `\n      ${flags}) ${completionsCmd} ;;`;
            }
            opts += "\n    esac";
        }
        return opts;
    }
    generateCommandCompletionsCommand(args, path) {
        if (args.length) {
            return `_${replaceSpecialChars(this.cmd.getName())}_complete ${args[0].action}${path}`;
        }
        return "";
    }
    generateOptionCompletionsCommand(args, path, opts) {
        if (args.length) {
            return `opts=(); _${replaceSpecialChars(this.cmd.getName())}_complete ${args[0].action}${path}`;
        }
        if (opts?.standalone) {
            return "opts=()";
        }
        return "";
    }
}
function replaceSpecialChars(str) {
    return str.replace(/[^a-zA-Z0-9]/g, "_");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Jhc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2Jhc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUlBLE1BQU0sT0FBTyx3QkFBd0I7SUFNTDtJQUp2QixNQUFNLENBQUMsUUFBUSxDQUFDLEdBQVk7UUFDakMsT0FBTyxJQUFJLHdCQUF3QixDQUFDLEdBQUcsQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3RELENBQUM7SUFFRCxZQUE4QixHQUFZO1FBQVosUUFBRyxHQUFILEdBQUcsQ0FBUztJQUFHLENBQUM7SUFHdEMsUUFBUTtRQUNkLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUM7UUFDaEMsTUFBTSxPQUFPLEdBQXVCLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFO1lBQ3ZELENBQUMsQ0FBQyxLQUFLLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFLEVBQUU7WUFDOUIsQ0FBQyxDQUFDLEVBQUUsQ0FBQztRQUVQLE9BQU87Z0NBQ3FCLElBQUksR0FBRyxPQUFPOztHQUUzQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUM7Ozs7Ozs7OztLQVN2QixtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDOzs2QkFFZixJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRTs7Ozs7O0lBTTNDLElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O2VBcUM5QixtQkFBbUIsQ0FBQyxJQUFJLENBQUMsOEJBQThCLElBQUk7Q0FDekUsQ0FBQztJQUNBLENBQUM7SUFHTyxtQkFBbUIsQ0FBQyxPQUFnQixFQUFFLElBQUksR0FBRyxFQUFFLEVBQUUsS0FBSyxHQUFHLENB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