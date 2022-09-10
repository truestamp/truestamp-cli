import { FileType } from "../types/file.ts";
/** Generates bash completions script. */ export class BashCompletionsGenerator {
    /** Generates bash completions script for given command. */ static generate(cmd) {
        return new BashCompletionsGenerator(cmd).generate();
    }
    constructor(cmd){
        this.cmd = cmd;
    }
    /** Generates bash completions code. */ generate() {
        const path = this.cmd.getPath();
        const version = this.cmd.getVersion() ? ` v${this.cmd.getVersion()}` : "";
        return `#!/usr/bin/env bash
# bash completion support for ${path}${version}

_${replaceSpecialChars(path)}() {
  local word cur prev listFiles
  local -a opts
  COMPREPLY=()
  cur="\${COMP_WORDS[COMP_CWORD]}"
  prev="\${COMP_WORDS[COMP_CWORD-1]}"
  cmd="_"
  opts=()
  listFiles=0

  _${replaceSpecialChars(this.cmd.getName())}_complete() {
    local action="$1"; shift
    mapfile -t values < <( ${this.cmd.getName()} completions complete "\${action}" "\${@}" )
    for i in "\${values[@]}"; do
      opts+=("$i")
    done
  }

  _${replaceSpecialChars(this.cmd.getName())}_expand() {
    [ "$cur" != "\${cur%\\\\}" ] && cur="$cur\\\\"
  
    # expand ~username type directory specifications
    if [[ "$cur" == \\~*/* ]]; then
      # shellcheck disable=SC2086
      eval cur=$cur
      
    elif [[ "$cur" == \\~* ]]; then
      cur=\${cur#\\~}
      # shellcheck disable=SC2086,SC2207
      COMPREPLY=( $( compgen -P '~' -u $cur ) )
      return \${#COMPREPLY[@]}
    fi
  }

  # shellcheck disable=SC2120
  _${replaceSpecialChars(this.cmd.getName())}_file_dir() {
    listFiles=1
    local IFS=$'\\t\\n' xspec #glob
    _${replaceSpecialChars(this.cmd.getName())}_expand || return 0
  
    if [ "\${1:-}" = -d ]; then
      # shellcheck disable=SC2206,SC2207,SC2086
      COMPREPLY=( \${COMPREPLY[@]:-} $( compgen -d -- $cur ) )
      #eval "$glob"    # restore glob setting.
      return 0
    fi
  
    xspec=\${1:+"!*.$1"}	# set only if glob passed in as $1
    # shellcheck disable=SC2206,SC2207
    COMPREPLY=( \${COMPREPLY[@]:-} $( compgen -f -X "$xspec" -- "$cur" ) \
          $( compgen -d -- "$cur" ) )
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

  if [[ listFiles -eq 1 ]]; then
    return 0
  fi

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

complete -F _${replaceSpecialChars(path)} -o bashdefault -o default ${path}`;
    }
    /** Generates bash completions method for given command and child commands. */ generateCompletions(command, path = "", index = 1) {
        path = (path ? path + " " : "") + command.getName();
        const commandCompletions = this.generateCommandCompletions(command, path, index);
        const childCommandCompletions = command.getCommands(false).filter((subCommand)=>subCommand !== command).map((subCommand)=>this.generateCompletions(subCommand, path, index + 1)).join("");
        return `${commandCompletions}

${childCommandCompletions}`;
    }
    generateCommandCompletions(command, path, index) {
        const flags = this.getFlags(command);
        const childCommandNames = command.getCommands(false).map((childCommand)=>childCommand.getName());
        const completionsPath = ~path.indexOf(" ") ? " " + path.split(" ").slice(1).join(" ") : "";
        const optionArguments = this.generateOptionArguments(command, completionsPath);
        const completionsCmd = this.generateCommandCompletionsCommand(command, completionsPath);
        return `  __${replaceSpecialChars(path)}() {
    opts=(${[
            ...flags,
            ...childCommandNames
        ].join(" ")})
    ${completionsCmd}
    if [[ \${cur} == -* || \${COMP_CWORD} -eq ${index} ]] ; then
      return 0
    fi
    ${optionArguments}
  }`;
    }
    getFlags(command) {
        return command.getOptions(false).map((option)=>option.flags).flat();
    }
    generateOptionArguments(command, completionsPath) {
        let opts = "";
        const options = command.getOptions(false);
        if (options.length) {
            opts += 'case "${prev}" in';
            for (const option of options){
                const flags = option.flags.map((flag)=>flag.trim()).join("|");
                const completionsCmd = this.generateOptionCompletionsCommand(command, option.args, completionsPath, {
                    standalone: option.standalone
                });
                opts += `\n      ${flags}) ${completionsCmd} ;;`;
            }
            opts += "\n    esac";
        }
        return opts;
    }
    generateCommandCompletionsCommand(command, path) {
        const args = command.getArguments();
        if (args.length) {
            const type = command.getType(args[0].type);
            if (type && type.handler instanceof FileType) {
                return `_${replaceSpecialChars(this.cmd.getName())}_file_dir`;
            }
            // @TODO: add support for multiple arguments
            return `_${replaceSpecialChars(this.cmd.getName())}_complete ${args[0].action}${path}`;
        }
        return "";
    }
    generateOptionCompletionsCommand(command, args, path, opts) {
        if (args.length) {
            const type = command.getType(args[0].type);
            if (type && type.handler instanceof FileType) {
                return `opts=(); _${replaceSpecialChars(this.cmd.getName())}_file_dir`;
            }
            // @TODO: add support for multiple arguments
            return `opts=(); _${replaceSpecialChars(this.cmd.getName())}_complete ${args[0].action}${path}`;
        }
        if (opts?.standalone) {
            return "opts=()";
        }
        return "";
    }
    cmd;
}
function replaceSpecialChars(str) {
    return str.replace(/[^a-zA-Z0-9]/g, "_");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC9jb21wbGV0aW9ucy9fYmFzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHR5cGUgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB0eXBlIHsgSUFyZ3VtZW50IH0gZnJvbSBcIi4uL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBGaWxlVHlwZSB9IGZyb20gXCIuLi90eXBlcy9maWxlLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgYmFzaCBjb21wbGV0aW9ucyBzY3JpcHQuICovXG5leHBvcnQgY2xhc3MgQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yIHtcbiAgLyoqIEdlbmVyYXRlcyBiYXNoIGNvbXBsZXRpb25zIHNjcmlwdCBmb3IgZ2l2ZW4gY29tbWFuZC4gKi9cbiAgcHVibGljIHN0YXRpYyBnZW5lcmF0ZShjbWQ6IENvbW1hbmQpIHtcbiAgICByZXR1cm4gbmV3IEJhc2hDb21wbGV0aW9uc0dlbmVyYXRvcihjbWQpLmdlbmVyYXRlKCk7XG4gIH1cblxuICBwcml2YXRlIGNvbnN0cnVjdG9yKHByb3RlY3RlZCBjbWQ6IENvbW1hbmQpIHt9XG5cbiAgLyoqIEdlbmVyYXRlcyBiYXNoIGNvbXBsZXRpb25zIGNvZGUuICovXG4gIHByaXZhdGUgZ2VuZXJhdGUoKTogc3RyaW5nIHtcbiAgICBjb25zdCBwYXRoID0gdGhpcy5jbWQuZ2V0UGF0aCgpO1xuICAgIGNvbnN0IHZlcnNpb246IHN0cmluZyB8IHVuZGVmaW5lZCA9IHRoaXMuY21kLmdldFZlcnNpb24oKVxuICAgICAgPyBgIHYke3RoaXMuY21kLmdldFZlcnNpb24oKX1gXG4gICAgICA6IFwiXCI7XG5cbiAgICByZXR1cm4gYCMhL3Vzci9iaW4vZW52IGJhc2hcbiMgYmFzaCBjb21wbGV0aW9uIHN1cHBvcnQgZm9yICR7cGF0aH0ke3ZlcnNpb259XG5cbl8ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9KCkge1xuICBsb2NhbCB3b3JkIGN1ciBwcmV2IGxpc3RGaWxlc1xuICBsb2NhbCAtYSBvcHRzXG4gIENPTVBSRVBMWT0oKVxuICBjdXI9XCJcXCR7Q09NUF9XT1JEU1tDT01QX0NXT1JEXX1cIlxuICBwcmV2PVwiXFwke0NPTVBfV09SRFNbQ09NUF9DV09SRC0xXX1cIlxuICBjbWQ9XCJfXCJcbiAgb3B0cz0oKVxuICBsaXN0RmlsZXM9MFxuXG4gIF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fY29tcGxldGUoKSB7XG4gICAgbG9jYWwgYWN0aW9uPVwiJDFcIjsgc2hpZnRcbiAgICBtYXBmaWxlIC10IHZhbHVlcyA8IDwoICR7dGhpcy5jbWQuZ2V0TmFtZSgpfSBjb21wbGV0aW9ucyBjb21wbGV0ZSBcIlxcJHthY3Rpb259XCIgXCJcXCR7QH1cIiApXG4gICAgZm9yIGkgaW4gXCJcXCR7dmFsdWVzW0BdfVwiOyBkb1xuICAgICAgb3B0cys9KFwiJGlcIilcbiAgICBkb25lXG4gIH1cblxuICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2V4cGFuZCgpIHtcbiAgICBbIFwiJGN1clwiICE9IFwiXFwke2N1ciVcXFxcXFxcXH1cIiBdICYmIGN1cj1cIiRjdXJcXFxcXFxcXFwiXG4gIFxuICAgICMgZXhwYW5kIH51c2VybmFtZSB0eXBlIGRpcmVjdG9yeSBzcGVjaWZpY2F0aW9uc1xuICAgIGlmIFtbIFwiJGN1clwiID09IFxcXFx+Ki8qIF1dOyB0aGVuXG4gICAgICAjIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIwODZcbiAgICAgIGV2YWwgY3VyPSRjdXJcbiAgICAgIFxuICAgIGVsaWYgW1sgXCIkY3VyXCIgPT0gXFxcXH4qIF1dOyB0aGVuXG4gICAgICBjdXI9XFwke2N1ciNcXFxcfn1cbiAgICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjA4NixTQzIyMDdcbiAgICAgIENPTVBSRVBMWT0oICQoIGNvbXBnZW4gLVAgJ34nIC11ICRjdXIgKSApXG4gICAgICByZXR1cm4gXFwkeyNDT01QUkVQTFlbQF19XG4gICAgZmlcbiAgfVxuXG4gICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjEyMFxuICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2ZpbGVfZGlyKCkge1xuICAgIGxpc3RGaWxlcz0xXG4gICAgbG9jYWwgSUZTPSQnXFxcXHRcXFxcbicgeHNwZWMgI2dsb2JcbiAgICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2V4cGFuZCB8fCByZXR1cm4gMFxuICBcbiAgICBpZiBbIFwiXFwkezE6LX1cIiA9IC1kIF07IHRoZW5cbiAgICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwNixTQzIyMDcsU0MyMDg2XG4gICAgICBDT01QUkVQTFk9KCBcXCR7Q09NUFJFUExZW0BdOi19ICQoIGNvbXBnZW4gLWQgLS0gJGN1ciApIClcbiAgICAgICNldmFsIFwiJGdsb2JcIiAgICAjIHJlc3RvcmUgZ2xvYiBzZXR0aW5nLlxuICAgICAgcmV0dXJuIDBcbiAgICBmaVxuICBcbiAgICB4c3BlYz1cXCR7MTorXCIhKi4kMVwifVx0IyBzZXQgb25seSBpZiBnbG9iIHBhc3NlZCBpbiBhcyAkMVxuICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwNixTQzIyMDdcbiAgICBDT01QUkVQTFk9KCBcXCR7Q09NUFJFUExZW0BdOi19ICQoIGNvbXBnZW4gLWYgLVggXCIkeHNwZWNcIiAtLSBcIiRjdXJcIiApIFxcXG4gICAgICAgICAgJCggY29tcGdlbiAtZCAtLSBcIiRjdXJcIiApIClcbiAgfVxuXG4gICR7dGhpcy5nZW5lcmF0ZUNvbXBsZXRpb25zKHRoaXMuY21kKS50cmltKCl9XG5cbiAgZm9yIHdvcmQgaW4gXCJcXCR7Q09NUF9XT1JEU1tAXX1cIjsgZG9cbiAgICBjYXNlIFwiXFwke3dvcmR9XCIgaW5cbiAgICAgIC0qKSA7O1xuICAgICAgKilcbiAgICAgICAgY21kX3RtcD1cIlxcJHtjbWR9X1xcJHt3b3JkLy9bXls6YWxudW06XV0vX31cIlxuICAgICAgICBpZiB0eXBlIFwiXFwke2NtZF90bXB9XCIgJj4vZGV2L251bGw7IHRoZW5cbiAgICAgICAgICBjbWQ9XCJcXCR7Y21kX3RtcH1cIlxuICAgICAgICBmaVxuICAgIGVzYWNcbiAgZG9uZVxuXG4gIFxcJHtjbWR9XG5cbiAgaWYgW1sgbGlzdEZpbGVzIC1lcSAxIF1dOyB0aGVuXG4gICAgcmV0dXJuIDBcbiAgZmlcblxuICBpZiBbWyBcXCR7I29wdHNbQF19IC1lcSAwIF1dOyB0aGVuXG4gICAgIyBzaGVsbGNoZWNrIGRpc2FibGU9U0MyMjA3XG4gICAgQ09NUFJFUExZPSgkKGNvbXBnZW4gLWYgXCJcXCR7Y3VyfVwiKSlcbiAgICByZXR1cm4gMFxuICBmaVxuXG4gIGxvY2FsIHZhbHVlc1xuICB2YWx1ZXM9XCIkKCBwcmludGYgXCJcXFxcbiVzXCIgXCJcXCR7b3B0c1tAXX1cIiApXCJcbiAgbG9jYWwgSUZTPSQnXFxcXG4nXG4gICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICBsb2NhbCByZXN1bHQ9KCQoY29tcGdlbiAtVyBcIlxcJHt2YWx1ZXNbQF19XCIgLS0gXCJcXCR7Y3VyfVwiKSlcbiAgaWYgW1sgXFwkeyNyZXN1bHRbQF19IC1lcSAwIF1dOyB0aGVuXG4gICAgIyBzaGVsbGNoZWNrIGRpc2FibGU9U0MyMjA3XG4gICAgQ09NUFJFUExZPSgkKGNvbXBnZW4gLWYgXCJcXCR7Y3VyfVwiKSlcbiAgZWxzZVxuICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICAgIENPTVBSRVBMWT0oJChwcmludGYgJyVxXFxcXG4nIFwiXFwke3Jlc3VsdFtAXX1cIikpXG4gIGZpXG5cbiAgcmV0dXJuIDBcbn1cblxuY29tcGxldGUgLUYgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyhwYXRoKX0gLW8gYmFzaGRlZmF1bHQgLW8gZGVmYXVsdCAke3BhdGh9YDtcbiAgfVxuXG4gIC8qKiBHZW5lcmF0ZXMgYmFzaCBjb21wbGV0aW9ucyBtZXRob2QgZm9yIGdpdmVuIGNvbW1hbmQgYW5kIGNoaWxkIGNvbW1hbmRzLiAqL1xuICBwcml2YXRlIGdlbmVyYXRlQ29tcGxldGlvbnMoY29tbWFuZDogQ29tbWFuZCwgcGF0aCA9IFwiXCIsIGluZGV4ID0gMSk6IHN0cmluZyB7XG4gICAgcGF0aCA9IChwYXRoID8gcGF0aCArIFwiIFwiIDogXCJcIikgKyBjb21tYW5kLmdldE5hbWUoKTtcbiAgICBjb25zdCBjb21tYW5kQ29tcGxldGlvbnMgPSB0aGlzLmdlbmVyYXRlQ29tbWFuZENvbXBsZXRpb25zKFxuICAgICAgY29tbWFuZCxcbiAgICAgIHBhdGgsXG4gICAgICBpbmRleCxcbiAgICApO1xuICAgIGNvbnN0IGNoaWxkQ29tbWFuZENvbXBsZXRpb25zOiBzdHJpbmcgPSBjb21tYW5kLmdldENvbW1hbmRzKGZhbHNlKVxuICAgICAgLmZpbHRlcigoc3ViQ29tbWFuZDogQ29tbWFuZCkgPT4gc3ViQ29tbWFuZCAhPT0gY29tbWFuZClcbiAgICAgIC5tYXAoKHN1YkNvbW1hbmQ6IENvbW1hbmQpID0+XG4gICAgICAgIHRoaXMuZ2VuZXJhdGVDb21wbGV0aW9ucyhzdWJDb21tYW5kLCBwYXRoLCBpbmRleCArIDEpXG4gICAgICApXG4gICAgICAuam9pbihcIlwiKTtcblxuICAgIHJldHVybiBgJHtjb21tYW5kQ29tcGxldGlvbnN9XG5cbiR7Y2hpbGRDb21tYW5kQ29tcGxldGlvbnN9YDtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVDb21tYW5kQ29tcGxldGlvbnMoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBwYXRoOiBzdHJpbmcsXG4gICAgaW5kZXg6IG51bWJlcixcbiAgKTogc3RyaW5nIHtcbiAgICBjb25zdCBmbGFnczogc3RyaW5nW10gPSB0aGlzLmdldEZsYWdzKGNvbW1hbmQpO1xuXG4gICAgY29uc3QgY2hpbGRDb21tYW5kTmFtZXM6IHN0cmluZ1tdID0gY29tbWFuZC5nZXRDb21tYW5kcyhmYWxzZSlcbiAgICAgIC5tYXAoKGNoaWxkQ29tbWFuZDogQ29tbWFuZCkgPT4gY2hpbGRDb21tYW5kLmdldE5hbWUoKSk7XG5cbiAgICBjb25zdCBjb21wbGV0aW9uc1BhdGg6IHN0cmluZyA9IH5wYXRoLmluZGV4T2YoXCIgXCIpXG4gICAgICA/IFwiIFwiICsgcGF0aC5zcGxpdChcIiBcIikuc2xpY2UoMSkuam9pbihcIiBcIilcbiAgICAgIDogXCJcIjtcblxuICAgIGNvbnN0IG9wdGlvbkFyZ3VtZW50cyA9IHRoaXMuZ2VuZXJhdGVPcHRpb25Bcmd1bWVudHMoXG4gICAgICBjb21tYW5kLFxuICAgICAgY29tcGxldGlvbnNQYXRoLFxuICAgICk7XG5cbiAgICBjb25zdCBjb21wbGV0aW9uc0NtZDogc3RyaW5nID0gdGhpcy5nZW5lcmF0ZUNvbW1hbmRDb21wbGV0aW9uc0NvbW1hbmQoXG4gICAgICBjb21tYW5kLFxuICAgICAgY29tcGxldGlvbnNQYXRoLFxuICAgICk7XG5cbiAgICByZXR1cm4gYCAgX18ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9KCkge1xuICAgIG9wdHM9KCR7Wy4uLmZsYWdzLCAuLi5jaGlsZENvbW1hbmROYW1lc10uam9pbihcIiBcIil9KVxuICAgICR7Y29tcGxldGlvbnNDbWR9XG4gICAgaWYgW1sgXFwke2N1cn0gPT0gLSogfHwgXFwke0NPTVBfQ1dPUkR9IC1lcSAke2luZGV4fSBdXSA7IHRoZW5cbiAgICAgIHJldHVybiAwXG4gICAgZmlcbiAgICAke29wdGlvbkFyZ3VtZW50c31cbiAgfWA7XG4gIH1cblxuICBwcml2YXRlIGdldEZsYWdzKGNvbW1hbmQ6IENvbW1hbmQpOiBzdHJpbmdbXSB7XG4gICAgcmV0dXJuIGNvbW1hbmQuZ2V0T3B0aW9ucyhmYWxzZSlcbiAgICAgIC5tYXAoKG9wdGlvbikgPT4gb3B0aW9uLmZsYWdzKVxuICAgICAgLmZsYXQoKTtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb25Bcmd1bWVudHMoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBjb21wbGV0aW9uc1BhdGg6IHN0cmluZyxcbiAgKTogc3RyaW5nIHtcbiAgICBsZXQgb3B0cyA9IFwiXCI7XG4gICAgY29uc3Qgb3B0aW9ucyA9IGNvbW1hbmQuZ2V0T3B0aW9ucyhmYWxzZSk7XG4gICAgaWYgKG9wdGlvbnMubGVuZ3RoKSB7XG4gICAgICBvcHRzICs9ICdjYXNlIFwiJHtwcmV2fVwiIGluJztcbiAgICAgIGZvciAoY29uc3Qgb3B0aW9uIG9mIG9wdGlvbnMpIHtcbiAgICAgICAgY29uc3QgZmxhZ3M6IHN0cmluZyA9IG9wdGlvbi5mbGFnc1xuICAgICAgICAgIC5tYXAoKGZsYWc6IHN0cmluZykgPT4gZmxhZy50cmltKCkpXG4gICAgICAgICAgLmpvaW4oXCJ8XCIpO1xuXG4gICAgICAgIGNvbnN0IGNvbXBsZXRpb25zQ21kOiBzdHJpbmcgPSB0aGlzLmdlbmVyYXRlT3B0aW9uQ29tcGxldGlvbnNDb21tYW5kKFxuICAgICAgICAgIGNvbW1hbmQsXG4gICAgICAgICAgb3B0aW9uLmFyZ3MsXG4gICAgICAgICAgY29tcGxldGlvbnNQYXRoLFxuICAgICAgICAgIHsgc3RhbmRhbG9uZTogb3B0aW9uLnN0YW5kYWxvbmUgfSxcbiAgICAgICAgKTtcblxuICAgICAgICBvcHRzICs9IGBcXG4gICAgICAke2ZsYWdzfSkgJHtjb21wbGV0aW9uc0NtZH0gOztgO1xuICAgICAgfVxuICAgICAgb3B0cyArPSBcIlxcbiAgICBlc2FjXCI7XG4gICAgfVxuXG4gICAgcmV0dXJuIG9wdHM7XG4gIH1cblxuICBwcml2YXRlIGdlbmVyYXRlQ29tbWFuZENvbXBsZXRpb25zQ29tbWFuZChcbiAgICBjb21tYW5kOiBDb21tYW5kLFxuICAgIHBhdGg6IHN0cmluZyxcbiAgKSB7XG4gICAgY29uc3QgYXJnczogSUFyZ3VtZW50W10gPSBjb21tYW5kLmdldEFyZ3VtZW50cygpO1xuICAgIGlmIChhcmdzLmxlbmd0aCkge1xuICAgICAgY29uc3QgdHlwZSA9IGNvbW1hbmQuZ2V0VHlwZShhcmdzWzBdLnR5cGUpO1xuICAgICAgaWYgKHR5cGUgJiYgdHlwZS5oYW5kbGVyIGluc3RhbmNlb2YgRmlsZVR5cGUpIHtcbiAgICAgICAgcmV0dXJuIGBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2ZpbGVfZGlyYDtcbiAgICAgIH1cbiAgICAgIC8vIEBUT0RPOiBhZGQgc3VwcG9ydCBmb3IgbXVsdGlwbGUgYXJndW1lbnRzXG4gICAgICByZXR1cm4gYF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fY29tcGxldGUgJHtcbiAgICAgICAgYXJnc1swXS5hY3Rpb25cbiAgICAgIH0ke3BhdGh9YDtcbiAgICB9XG5cbiAgICByZXR1cm4gXCJcIjtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb25Db21wbGV0aW9uc0NvbW1hbmQoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBhcmdzOiBJQXJndW1lbnRbXSxcbiAgICBwYXRoOiBzdHJpbmcsXG4gICAgb3B0cz86IHsgc3RhbmRhbG9uZT86IGJvb2xlYW4gfSxcbiAgKSB7XG4gICAgaWYgKGFyZ3MubGVuZ3RoKSB7XG4gICAgICBjb25zdCB0eXBlID0gY29tbWFuZC5nZXRUeXBlKGFyZ3NbMF0udHlwZSk7XG4gICAgICBpZiAodHlwZSAmJiB0eXBlLmhhbmRsZXIgaW5zdGFuY2VvZiBGaWxlVHlwZSkge1xuICAgICAgICByZXR1cm4gYG9wdHM9KCk7IF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fZmlsZV9kaXJgO1xuICAgICAgfVxuICAgICAgLy8gQFRPRE86IGFkZCBzdXBwb3J0IGZvciBtdWx0aXBsZSBhcmd1bWVudHNcbiAgICAgIHJldHVybiBgb3B0cz0oKTsgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyh0aGlzLmNtZC5nZXROYW1lKCkpfV9jb21wbGV0ZSAke1xuICAgICAgICBhcmdzWzBdLmFjdGlvblxuICAgICAgfSR7cGF0aH1gO1xuICAgIH1cblxuICAgIGlmIChvcHRzPy5zdGFuZGFsb25lKSB7XG4gICAgICByZXR1cm4gXCJvcHRzPSgpXCI7XG4gICAgfVxuXG4gICAgcmV0dXJuIFwiXCI7XG4gIH1cbn1cblxuZnVuY3Rpb24gcmVwbGFjZVNwZWNpYWxDaGFycyhzdHI6IHN0cmluZyk6IHN0cmluZyB7XG4gIHJldHVybiBzdHIucmVwbGFjZSgvW15hLXpBLVowLTldL2csIFwiX1wiKTtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxTQUFTLFFBQVEsUUFBUSxrQkFBa0IsQ0FBQztBQUU1Qyx1Q0FBdUMsR0FDdkMsT0FBTyxNQUFNLHdCQUF3QjtJQUNuQyx5REFBeUQsVUFDM0MsUUFBUSxDQUFDLEdBQVksRUFBRTtRQUNuQyxPQUFPLElBQUksd0JBQXdCLENBQUMsR0FBRyxDQUFDLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDdEQ7SUFFQSxZQUE4QixHQUFZLENBQUU7UUFBZCxXQUFBLEdBQVksQ0FBQTtJQUFHO0lBRTdDLHFDQUFxQyxHQUM3QixRQUFRLEdBQVc7UUFDekIsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQUFBQztRQUNoQyxNQUFNLE9BQU8sR0FBdUIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsR0FDckQsQ0FBQyxFQUFFLEVBQUUsSUFBSSxDQUFDLEdBQUcsQ0FBQyxVQUFVLEVBQUUsQ0FBQyxDQUFDLEdBQzVCLEVBQUUsQUFBQztRQUVQLE9BQU8sQ0FBQzs4QkFDa0IsRUFBRSxJQUFJLENBQUMsRUFBRSxPQUFPLENBQUM7O0NBRTlDLEVBQUUsbUJBQW1CLENBQUMsSUFBSSxDQUFDLENBQUM7Ozs7Ozs7Ozs7R0FVMUIsRUFBRSxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDLENBQUM7OzJCQUVsQixFQUFFLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUM7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