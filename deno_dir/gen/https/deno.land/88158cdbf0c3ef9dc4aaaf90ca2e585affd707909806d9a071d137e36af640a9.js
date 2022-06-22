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
      eval cur=$cur
      
    elif [[ "$cur" == \\~* ]]; then
      cur=\${cur#\\~}
      COMPREPLY=( $( compgen -P '~' -u $cur ) )
      return \${#COMPREPLY[@]}
    fi
  }

  _${replaceSpecialChars(this.cmd.getName())}_file_dir() {
    listFiles=1
    local IFS=$'\\t\\n' xspec #glob
    _${replaceSpecialChars(this.cmd.getName())}_expand || return 0
  
    if [ "\${1:-}" = -d ]; then
      COMPREPLY=( \${COMPREPLY[@]:-} $( compgen -d -- $cur ) )
      #eval "$glob"    # restore glob setting.
      return 0
    fi
  
    xspec=\${1:+"!*.$1"}	# set only if glob passed in as $1
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvY29tbWFuZC9jb21wbGV0aW9ucy9fYmFzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHR5cGUgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB0eXBlIHsgSUFyZ3VtZW50IH0gZnJvbSBcIi4uL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBGaWxlVHlwZSB9IGZyb20gXCIuLi90eXBlcy9maWxlLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgYmFzaCBjb21wbGV0aW9ucyBzY3JpcHQuICovXG5leHBvcnQgY2xhc3MgQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yIHtcbiAgLyoqIEdlbmVyYXRlcyBiYXNoIGNvbXBsZXRpb25zIHNjcmlwdCBmb3IgZ2l2ZW4gY29tbWFuZC4gKi9cbiAgcHVibGljIHN0YXRpYyBnZW5lcmF0ZShjbWQ6IENvbW1hbmQpIHtcbiAgICByZXR1cm4gbmV3IEJhc2hDb21wbGV0aW9uc0dlbmVyYXRvcihjbWQpLmdlbmVyYXRlKCk7XG4gIH1cblxuICBwcml2YXRlIGNvbnN0cnVjdG9yKHByb3RlY3RlZCBjbWQ6IENvbW1hbmQpIHt9XG5cbiAgLyoqIEdlbmVyYXRlcyBiYXNoIGNvbXBsZXRpb25zIGNvZGUuICovXG4gIHByaXZhdGUgZ2VuZXJhdGUoKTogc3RyaW5nIHtcbiAgICBjb25zdCBwYXRoID0gdGhpcy5jbWQuZ2V0UGF0aCgpO1xuICAgIGNvbnN0IHZlcnNpb246IHN0cmluZyB8IHVuZGVmaW5lZCA9IHRoaXMuY21kLmdldFZlcnNpb24oKVxuICAgICAgPyBgIHYke3RoaXMuY21kLmdldFZlcnNpb24oKX1gXG4gICAgICA6IFwiXCI7XG5cbiAgICByZXR1cm4gYCMhL3Vzci9iaW4vZW52IGJhc2hcbiMgYmFzaCBjb21wbGV0aW9uIHN1cHBvcnQgZm9yICR7cGF0aH0ke3ZlcnNpb259XG5cbl8ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9KCkge1xuICBsb2NhbCB3b3JkIGN1ciBwcmV2IGxpc3RGaWxlc1xuICBsb2NhbCAtYSBvcHRzXG4gIENPTVBSRVBMWT0oKVxuICBjdXI9XCJcXCR7Q09NUF9XT1JEU1tDT01QX0NXT1JEXX1cIlxuICBwcmV2PVwiXFwke0NPTVBfV09SRFNbQ09NUF9DV09SRC0xXX1cIlxuICBjbWQ9XCJfXCJcbiAgb3B0cz0oKVxuICBsaXN0RmlsZXM9MFxuXG4gIF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fY29tcGxldGUoKSB7XG4gICAgbG9jYWwgYWN0aW9uPVwiJDFcIjsgc2hpZnRcbiAgICBtYXBmaWxlIC10IHZhbHVlcyA8IDwoICR7dGhpcy5jbWQuZ2V0TmFtZSgpfSBjb21wbGV0aW9ucyBjb21wbGV0ZSBcIlxcJHthY3Rpb259XCIgXCJcXCR7QH1cIiApXG4gICAgZm9yIGkgaW4gXCJcXCR7dmFsdWVzW0BdfVwiOyBkb1xuICAgICAgb3B0cys9KFwiJGlcIilcbiAgICBkb25lXG4gIH1cblxuICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2V4cGFuZCgpIHtcbiAgICBbIFwiJGN1clwiICE9IFwiXFwke2N1ciVcXFxcXFxcXH1cIiBdICYmIGN1cj1cIiRjdXJcXFxcXFxcXFwiXG4gIFxuICAgICMgZXhwYW5kIH51c2VybmFtZSB0eXBlIGRpcmVjdG9yeSBzcGVjaWZpY2F0aW9uc1xuICAgIGlmIFtbIFwiJGN1clwiID09IFxcXFx+Ki8qIF1dOyB0aGVuXG4gICAgICBldmFsIGN1cj0kY3VyXG4gICAgICBcbiAgICBlbGlmIFtbIFwiJGN1clwiID09IFxcXFx+KiBdXTsgdGhlblxuICAgICAgY3VyPVxcJHtjdXIjXFxcXH59XG4gICAgICBDT01QUkVQTFk9KCAkKCBjb21wZ2VuIC1QICd+JyAtdSAkY3VyICkgKVxuICAgICAgcmV0dXJuIFxcJHsjQ09NUFJFUExZW0BdfVxuICAgIGZpXG4gIH1cblxuICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2ZpbGVfZGlyKCkge1xuICAgIGxpc3RGaWxlcz0xXG4gICAgbG9jYWwgSUZTPSQnXFxcXHRcXFxcbicgeHNwZWMgI2dsb2JcbiAgICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2V4cGFuZCB8fCByZXR1cm4gMFxuICBcbiAgICBpZiBbIFwiXFwkezE6LX1cIiA9IC1kIF07IHRoZW5cbiAgICAgIENPTVBSRVBMWT0oIFxcJHtDT01QUkVQTFlbQF06LX0gJCggY29tcGdlbiAtZCAtLSAkY3VyICkgKVxuICAgICAgI2V2YWwgXCIkZ2xvYlwiICAgICMgcmVzdG9yZSBnbG9iIHNldHRpbmcuXG4gICAgICByZXR1cm4gMFxuICAgIGZpXG4gIFxuICAgIHhzcGVjPVxcJHsxOitcIiEqLiQxXCJ9XHQjIHNldCBvbmx5IGlmIGdsb2IgcGFzc2VkIGluIGFzICQxXG4gICAgQ09NUFJFUExZPSggXFwke0NPTVBSRVBMWVtAXTotfSAkKCBjb21wZ2VuIC1mIC1YIFwiJHhzcGVjXCIgLS0gXCIkY3VyXCIgKSBcXFxuICAgICAgICAgICQoIGNvbXBnZW4gLWQgLS0gXCIkY3VyXCIgKSApXG4gIH1cblxuICAke3RoaXMuZ2VuZXJhdGVDb21wbGV0aW9ucyh0aGlzLmNtZCkudHJpbSgpfVxuXG4gIGZvciB3b3JkIGluIFwiXFwke0NPTVBfV09SRFNbQF19XCI7IGRvXG4gICAgY2FzZSBcIlxcJHt3b3JkfVwiIGluXG4gICAgICAtKikgOztcbiAgICAgICopXG4gICAgICAgIGNtZF90bXA9XCJcXCR7Y21kfV9cXCR7d29yZC8vW15bOmFsbnVtOl1dL199XCJcbiAgICAgICAgaWYgdHlwZSBcIlxcJHtjbWRfdG1wfVwiICY+L2Rldi9udWxsOyB0aGVuXG4gICAgICAgICAgY21kPVwiXFwke2NtZF90bXB9XCJcbiAgICAgICAgZmlcbiAgICBlc2FjXG4gIGRvbmVcblxuICBcXCR7Y21kfVxuXG4gIGlmIFtbIGxpc3RGaWxlcyAtZXEgMSBdXTsgdGhlblxuICAgIHJldHVybiAwXG4gIGZpXG5cbiAgaWYgW1sgXFwkeyNvcHRzW0BdfSAtZXEgMCBdXTsgdGhlblxuICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICAgIENPTVBSRVBMWT0oJChjb21wZ2VuIC1mIFwiXFwke2N1cn1cIikpXG4gICAgcmV0dXJuIDBcbiAgZmlcblxuICBsb2NhbCB2YWx1ZXNcbiAgdmFsdWVzPVwiJCggcHJpbnRmIFwiXFxcXG4lc1wiIFwiXFwke29wdHNbQF19XCIgKVwiXG4gIGxvY2FsIElGUz0kJ1xcXFxuJ1xuICAjIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIyMDdcbiAgbG9jYWwgcmVzdWx0PSgkKGNvbXBnZW4gLVcgXCJcXCR7dmFsdWVzW0BdfVwiIC0tIFwiXFwke2N1cn1cIikpXG4gIGlmIFtbIFxcJHsjcmVzdWx0W0BdfSAtZXEgMCBdXTsgdGhlblxuICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICAgIENPTVBSRVBMWT0oJChjb21wZ2VuIC1mIFwiXFwke2N1cn1cIikpXG4gIGVsc2VcbiAgICAjIHNoZWxsY2hlY2sgZGlzYWJsZT1TQzIyMDdcbiAgICBDT01QUkVQTFk9KCQocHJpbnRmICclcVxcXFxuJyBcIlxcJHtyZXN1bHRbQF19XCIpKVxuICBmaVxuXG4gIHJldHVybiAwXG59XG5cbmNvbXBsZXRlIC1GIF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9IC1vIGJhc2hkZWZhdWx0IC1vIGRlZmF1bHQgJHtwYXRofWA7XG4gIH1cblxuICAvKiogR2VuZXJhdGVzIGJhc2ggY29tcGxldGlvbnMgbWV0aG9kIGZvciBnaXZlbiBjb21tYW5kIGFuZCBjaGlsZCBjb21tYW5kcy4gKi9cbiAgcHJpdmF0ZSBnZW5lcmF0ZUNvbXBsZXRpb25zKGNvbW1hbmQ6IENvbW1hbmQsIHBhdGggPSBcIlwiLCBpbmRleCA9IDEpOiBzdHJpbmcge1xuICAgIHBhdGggPSAocGF0aCA/IHBhdGggKyBcIiBcIiA6IFwiXCIpICsgY29tbWFuZC5nZXROYW1lKCk7XG4gICAgY29uc3QgY29tbWFuZENvbXBsZXRpb25zID0gdGhpcy5nZW5lcmF0ZUNvbW1hbmRDb21wbGV0aW9ucyhcbiAgICAgIGNvbW1hbmQsXG4gICAgICBwYXRoLFxuICAgICAgaW5kZXgsXG4gICAgKTtcbiAgICBjb25zdCBjaGlsZENvbW1hbmRDb21wbGV0aW9uczogc3RyaW5nID0gY29tbWFuZC5nZXRDb21tYW5kcyhmYWxzZSlcbiAgICAgIC5maWx0ZXIoKHN1YkNvbW1hbmQ6IENvbW1hbmQpID0+IHN1YkNvbW1hbmQgIT09IGNvbW1hbmQpXG4gICAgICAubWFwKChzdWJDb21tYW5kOiBDb21tYW5kKSA9PlxuICAgICAgICB0aGlzLmdlbmVyYXRlQ29tcGxldGlvbnMoc3ViQ29tbWFuZCwgcGF0aCwgaW5kZXggKyAxKVxuICAgICAgKVxuICAgICAgLmpvaW4oXCJcIik7XG5cbiAgICByZXR1cm4gYCR7Y29tbWFuZENvbXBsZXRpb25zfVxuXG4ke2NoaWxkQ29tbWFuZENvbXBsZXRpb25zfWA7XG4gIH1cblxuICBwcml2YXRlIGdlbmVyYXRlQ29tbWFuZENvbXBsZXRpb25zKFxuICAgIGNvbW1hbmQ6IENvbW1hbmQsXG4gICAgcGF0aDogc3RyaW5nLFxuICAgIGluZGV4OiBudW1iZXIsXG4gICk6IHN0cmluZyB7XG4gICAgY29uc3QgZmxhZ3M6IHN0cmluZ1tdID0gdGhpcy5nZXRGbGFncyhjb21tYW5kKTtcblxuICAgIGNvbnN0IGNoaWxkQ29tbWFuZE5hbWVzOiBzdHJpbmdbXSA9IGNvbW1hbmQuZ2V0Q29tbWFuZHMoZmFsc2UpXG4gICAgICAubWFwKChjaGlsZENvbW1hbmQ6IENvbW1hbmQpID0+IGNoaWxkQ29tbWFuZC5nZXROYW1lKCkpO1xuXG4gICAgY29uc3QgY29tcGxldGlvbnNQYXRoOiBzdHJpbmcgPSB+cGF0aC5pbmRleE9mKFwiIFwiKVxuICAgICAgPyBcIiBcIiArIHBhdGguc3BsaXQoXCIgXCIpLnNsaWNlKDEpLmpvaW4oXCIgXCIpXG4gICAgICA6IFwiXCI7XG5cbiAgICBjb25zdCBvcHRpb25Bcmd1bWVudHMgPSB0aGlzLmdlbmVyYXRlT3B0aW9uQXJndW1lbnRzKFxuICAgICAgY29tbWFuZCxcbiAgICAgIGNvbXBsZXRpb25zUGF0aCxcbiAgICApO1xuXG4gICAgY29uc3QgY29tcGxldGlvbnNDbWQ6IHN0cmluZyA9IHRoaXMuZ2VuZXJhdGVDb21tYW5kQ29tcGxldGlvbnNDb21tYW5kKFxuICAgICAgY29tbWFuZCxcbiAgICAgIGNvbXBsZXRpb25zUGF0aCxcbiAgICApO1xuXG4gICAgcmV0dXJuIGAgIF9fJHtyZXBsYWNlU3BlY2lhbENoYXJzKHBhdGgpfSgpIHtcbiAgICBvcHRzPSgke1suLi5mbGFncywgLi4uY2hpbGRDb21tYW5kTmFtZXNdLmpvaW4oXCIgXCIpfSlcbiAgICAke2NvbXBsZXRpb25zQ21kfVxuICAgIGlmIFtbIFxcJHtjdXJ9ID09IC0qIHx8IFxcJHtDT01QX0NXT1JEfSAtZXEgJHtpbmRleH0gXV0gOyB0aGVuXG4gICAgICByZXR1cm4gMFxuICAgIGZpXG4gICAgJHtvcHRpb25Bcmd1bWVudHN9XG4gIH1gO1xuICB9XG5cbiAgcHJpdmF0ZSBnZXRGbGFncyhjb21tYW5kOiBDb21tYW5kKTogc3RyaW5nW10ge1xuICAgIHJldHVybiBjb21tYW5kLmdldE9wdGlvbnMoZmFsc2UpXG4gICAgICAubWFwKChvcHRpb24pID0+IG9wdGlvbi5mbGFncylcbiAgICAgIC5mbGF0KCk7XG4gIH1cblxuICBwcml2YXRlIGdlbmVyYXRlT3B0aW9uQXJndW1lbnRzKFxuICAgIGNvbW1hbmQ6IENvbW1hbmQsXG4gICAgY29tcGxldGlvbnNQYXRoOiBzdHJpbmcsXG4gICk6IHN0cmluZyB7XG4gICAgbGV0IG9wdHMgPSBcIlwiO1xuICAgIGNvbnN0IG9wdGlvbnMgPSBjb21tYW5kLmdldE9wdGlvbnMoZmFsc2UpO1xuICAgIGlmIChvcHRpb25zLmxlbmd0aCkge1xuICAgICAgb3B0cyArPSAnY2FzZSBcIiR7cHJldn1cIiBpbic7XG4gICAgICBmb3IgKGNvbnN0IG9wdGlvbiBvZiBvcHRpb25zKSB7XG4gICAgICAgIGNvbnN0IGZsYWdzOiBzdHJpbmcgPSBvcHRpb24uZmxhZ3NcbiAgICAgICAgICAubWFwKChmbGFnOiBzdHJpbmcpID0+IGZsYWcudHJpbSgpKVxuICAgICAgICAgIC5qb2luKFwifFwiKTtcblxuICAgICAgICBjb25zdCBjb21wbGV0aW9uc0NtZDogc3RyaW5nID0gdGhpcy5nZW5lcmF0ZU9wdGlvbkNvbXBsZXRpb25zQ29tbWFuZChcbiAgICAgICAgICBjb21tYW5kLFxuICAgICAgICAgIG9wdGlvbi5hcmdzLFxuICAgICAgICAgIGNvbXBsZXRpb25zUGF0aCxcbiAgICAgICAgICB7IHN0YW5kYWxvbmU6IG9wdGlvbi5zdGFuZGFsb25lIH0sXG4gICAgICAgICk7XG5cbiAgICAgICAgb3B0cyArPSBgXFxuICAgICAgJHtmbGFnc30pICR7Y29tcGxldGlvbnNDbWR9IDs7YDtcbiAgICAgIH1cbiAgICAgIG9wdHMgKz0gXCJcXG4gICAgZXNhY1wiO1xuICAgIH1cblxuICAgIHJldHVybiBvcHRzO1xuICB9XG5cbiAgcHJpdmF0ZSBnZW5lcmF0ZUNvbW1hbmRDb21wbGV0aW9uc0NvbW1hbmQoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBwYXRoOiBzdHJpbmcsXG4gICkge1xuICAgIGNvbnN0IGFyZ3M6IElBcmd1bWVudFtdID0gY29tbWFuZC5nZXRBcmd1bWVudHMoKTtcbiAgICBpZiAoYXJncy5sZW5ndGgpIHtcbiAgICAgIGNvbnN0IHR5cGUgPSBjb21tYW5kLmdldFR5cGUoYXJnc1swXS50eXBlKTtcbiAgICAgIGlmICh0eXBlICYmIHR5cGUuaGFuZGxlciBpbnN0YW5jZW9mIEZpbGVUeXBlKSB7XG4gICAgICAgIHJldHVybiBgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyh0aGlzLmNtZC5nZXROYW1lKCkpfV9maWxlX2RpcmA7XG4gICAgICB9XG4gICAgICAvLyBAVE9ETzogYWRkIHN1cHBvcnQgZm9yIG11bHRpcGxlIGFyZ3VtZW50c1xuICAgICAgcmV0dXJuIGBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2NvbXBsZXRlICR7XG4gICAgICAgIGFyZ3NbMF0uYWN0aW9uXG4gICAgICB9JHtwYXRofWA7XG4gICAgfVxuXG4gICAgcmV0dXJuIFwiXCI7XG4gIH1cblxuICBwcml2YXRlIGdlbmVyYXRlT3B0aW9uQ29tcGxldGlvbnNDb21tYW5kKFxuICAgIGNvbW1hbmQ6IENvbW1hbmQsXG4gICAgYXJnczogSUFyZ3VtZW50W10sXG4gICAgcGF0aDogc3RyaW5nLFxuICAgIG9wdHM/OiB7IHN0YW5kYWxvbmU/OiBib29sZWFuIH0sXG4gICkge1xuICAgIGlmIChhcmdzLmxlbmd0aCkge1xuICAgICAgY29uc3QgdHlwZSA9IGNvbW1hbmQuZ2V0VHlwZShhcmdzWzBdLnR5cGUpO1xuICAgICAgaWYgKHR5cGUgJiYgdHlwZS5oYW5kbGVyIGluc3RhbmNlb2YgRmlsZVR5cGUpIHtcbiAgICAgICAgcmV0dXJuIGBvcHRzPSgpOyBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2ZpbGVfZGlyYDtcbiAgICAgIH1cbiAgICAgIC8vIEBUT0RPOiBhZGQgc3VwcG9ydCBmb3IgbXVsdGlwbGUgYXJndW1lbnRzXG4gICAgICByZXR1cm4gYG9wdHM9KCk7IF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fY29tcGxldGUgJHtcbiAgICAgICAgYXJnc1swXS5hY3Rpb25cbiAgICAgIH0ke3BhdGh9YDtcbiAgICB9XG5cbiAgICBpZiAob3B0cz8uc3RhbmRhbG9uZSkge1xuICAgICAgcmV0dXJuIFwib3B0cz0oKVwiO1xuICAgIH1cblxuICAgIHJldHVybiBcIlwiO1xuICB9XG59XG5cbmZ1bmN0aW9uIHJlcGxhY2VTcGVjaWFsQ2hhcnMoc3RyOiBzdHJpbmcpOiBzdHJpbmcge1xuICByZXR1cm4gc3RyLnJl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