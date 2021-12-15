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

complete -F _${replaceSpecialChars(path)} -o bashdefault -o default ${path}`;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Jhc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2Jhc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUlBLE1BQU0sT0FBTyx3QkFBd0I7SUFNTDtJQUp2QixNQUFNLENBQUMsUUFBUSxDQUFDLEdBQVk7UUFDakMsT0FBTyxJQUFJLHdCQUF3QixDQUFDLEdBQUcsQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3RELENBQUM7SUFFRCxZQUE4QixHQUFZO1FBQVosUUFBRyxHQUFILEdBQUcsQ0FBUztJQUFHLENBQUM7SUFHdEMsUUFBUTtRQUNkLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUM7UUFDaEMsTUFBTSxPQUFPLEdBQXVCLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFO1lBQ3ZELENBQUMsQ0FBQyxLQUFLLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFLEVBQUU7WUFDOUIsQ0FBQyxDQUFDLEVBQUUsQ0FBQztRQUVQLE9BQU87Z0NBQ3FCLElBQUksR0FBRyxPQUFPOztHQUUzQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUM7Ozs7Ozs7OztLQVN2QixtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDOzs2QkFFZixJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRTs7Ozs7O0lBTTNDLElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O2VBcUM5QixtQkFBbUIsQ0FBQyxJQUFJLENBQUMsOEJBQThCLElBQUksRUFBRSxDQUFDO0lBQzNFLENBQUM7SUFHTyxtQkFBbUIsQ0FBQyxPQUFnQixFQUFFLElBQUksR0FBRyxFQUFFLEVBQUUsS0FBSyxHQUFHLENB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SSxFQUNYLGVBQWUsRUFDZixFQUFFLFVBQVUsRUFBRSxNQUFNLENBQUMsVUFBVSxFQUFFLENBQ2xDLENBQUM7Z0JBRUYsSUFBSSxJQUFJLFdBQVcsS0FBSyxLQUFLLGNBQWMsS0FBSyxDQUFDO2FBQ2xEO1lBQ0QsSUFBSSxJQUFJLFlBQVksQ0FBQztTQUN0QjtRQUVELE9BQU8sSUFBSSxDQUFDO0lBQ2QsQ0FBQztJQUVPLGlDQUFpQyxDQUN2QyxJQUFpQixFQUNqQixJQUFZO1FBRVosSUFBSSxJQUFJLENBQUMsTUFBTSxFQUFFO1lBRWYsT0FBTyxJQUFJLG1CQUFtQixDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUMsYUFDaEQsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLE1BQ1YsR0FBRyxJQUFJLEVBQUUsQ0FBQztTQUNYO1FBRUQsT0FBTyxFQUFFLENBQUM7SUFDWixDQUFDO0lBRU8sZ0NBQWdDLENBQ3RDLElBQWlCLEVBQ2pCLElBQVksRUFDWixJQUErQjtRQUUvQixJQUFJLElBQUksQ0FBQyxNQUFNLEVBQUU7WUFFZixPQUFPLGFBQWEsbUJBQW1CLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQ0FBQyxhQUN6RCxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsTUFDVixHQUFHLElBQUksRUFBRSxDQUFDO1NBQ1g7UUFFRCxJQUFJLElBQUksRUFBRSxVQUFVLEVBQUU7WUFDcEIsT0FBTyxTQUFTLENBQUM7U0FDbEI7UUFFRCxPQUFPLEVBQUUsQ0FBQztJQUNaLENBQUM7Q0FDRjtBQUVELFNBQVMsbUJBQW1CLENBQUMsR0FBVztJQUN0QyxPQUFPLEdBQUcsQ0FBQyxPQUFPLENBQUMsZUFBZSxFQUFFLEdBQUcsQ0FBQyxDQUFDO0FBQzNDLENBQUMiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgdHlwZSB7IENvbW1hbmQgfSBmcm9tIFwiLi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHR5cGUgeyBJQXJndW1lbnQgfSBmcm9tIFwiLi4vdHlwZXMudHNcIjtcblxuLyoqIEdlbmVyYXRlcyBiYXNoIGNvbXBsZXRpb25zIHNjcmlwdC4gKi9cbmV4cG9ydCBjbGFzcyBCYXNoQ29tcGxldGlvbnNHZW5lcmF0b3Ige1xuICAvKiogR2VuZXJhdGVzIGJhc2ggY29tcGxldGlvbnMgc2NyaXB0IGZvciBnaXZlbiBjb21tYW5kLiAqL1xuICBwdWJsaWMgc3RhdGljIGdlbmVyYXRlKGNtZDogQ29tbWFuZCkge1xuICAgIHJldHVybiBuZXcgQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yKGNtZCkuZ2VuZXJhdGUoKTtcbiAgfVxuXG4gIHByaXZhdGUgY29uc3RydWN0b3IocHJvdGVjdGVkIGNtZDogQ29tbWFuZCkge31cblxuICAvKiogR2VuZXJhdGVzIGJhc2ggY29tcGxldGlvbnMgY29kZS4gKi9cbiAgcHJpdmF0ZSBnZW5lcmF0ZSgpOiBzdHJpbmcge1xuICAgIGNvbnN0IHBhdGggPSB0aGlzLmNtZC5nZXRQYXRoKCk7XG4gICAgY29uc3QgdmVyc2lvbjogc3RyaW5nIHwgdW5kZWZpbmVkID0gdGhpcy5jbWQuZ2V0VmVyc2lvbigpXG4gICAgICA/IGAgdiR7dGhpcy5jbWQuZ2V0VmVyc2lvbigpfWBcbiAgICAgIDogXCJcIjtcblxuICAgIHJldHVybiBgIyEvdXNyL2Jpbi9lbnYgYmFzaFxuIyBiYXNoIGNvbXBsZXRpb24gc3VwcG9ydCBmb3IgJHtwYXRofSR7dmVyc2lvbn1cblxuXyR7cmVwbGFjZVNwZWNpYWxDaGFycyhwYXRoKX0oKSB7XG4gIGxvY2FsIHdvcmQgY3VyIHByZXZcbiAgbG9jYWwgLWEgb3B0c1xuICBDT01QUkVQTFk9KClcbiAgY3VyPVwiXFwke0NPTVBfV09SRFNbQ09NUF9DV09SRF19XCJcbiAgcHJldj1cIlxcJHtDT01QX1dPUkRTW0NPTVBfQ1dPUkQtMV19XCJcbiAgY21kPVwiX1wiXG4gIG9wdHM9KClcblxuICBfJHtyZXBsYWNlU3BlY2lhbENoYXJzKHRoaXMuY21kLmdldE5hbWUoKSl9X2NvbXBsZXRlKCkge1xuICAgIGxvY2FsIGFjdGlvbj1cIiQxXCI7IHNoaWZ0XG4gICAgbWFwZmlsZSAtdCB2YWx1ZXMgPCA8KCAke3RoaXMuY21kLmdldE5hbWUoKX0gY29tcGxldGlvbnMgY29tcGxldGUgXCJcXCR7YWN0aW9ufVwiIFwiXFwke0B9XCIgKVxuICAgIGZvciBpIGluIFwiXFwke3ZhbHVlc1tAXX1cIjsgZG9cbiAgICAgIG9wdHMrPShcIiRpXCIpXG4gICAgZG9uZVxuICB9XG5cbiAgJHt0aGlzLmdlbmVyYXRlQ29tcGxldGlvbnModGhpcy5jbWQpLnRyaW0oKX1cblxuICBmb3Igd29yZCBpbiBcIlxcJHtDT01QX1dPUkRTW0BdfVwiOyBkb1xuICAgIGNhc2UgXCJcXCR7d29yZH1cIiBpblxuICAgICAgLSopIDs7XG4gICAgICAqKVxuICAgICAgICBjbWRfdG1wPVwiXFwke2NtZH1fXFwke3dvcmQvL1teWzphbG51bTpdXS9ffVwiXG4gICAgICAgIGlmIHR5cGUgXCJcXCR7Y21kX3RtcH1cIiAmPi9kZXYvbnVsbDsgdGhlblxuICAgICAgICAgIGNtZD1cIlxcJHtjbWRfdG1wfVwiXG4gICAgICAgIGZpXG4gICAgZXNhY1xuICBkb25lXG5cbiAgXFwke2NtZH1cblxuICBpZiBbWyBcXCR7I29wdHNbQF19IC1lcSAwIF1dOyB0aGVuXG4gICAgIyBzaGVsbGNoZWNrIGRpc2FibGU9U0MyMjA3XG4gICAgQ09NUFJFUExZPSgkKGNvbXBnZW4gLWYgXCJcXCR7Y3VyfVwiKSlcbiAgICByZXR1cm4gMFxuICBmaVxuXG4gIGxvY2FsIHZhbHVlc1xuICB2YWx1ZXM9XCIkKCBwcmludGYgXCJcXFxcbiVzXCIgXCJcXCR7b3B0c1tAXX1cIiApXCJcbiAgbG9jYWwgSUZTPSQnXFxcXG4nXG4gICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICBsb2NhbCByZXN1bHQ9KCQoY29tcGdlbiAtVyBcIlxcJHt2YWx1ZXNbQF19XCIgLS0gXCJcXCR7Y3VyfVwiKSlcbiAgaWYgW1sgXFwkeyNyZXN1bHRbQF19IC1lcSAwIF1dOyB0aGVuXG4gICAgIyBzaGVsbGNoZWNrIGRpc2FibGU9U0MyMjA3XG4gICAgQ09NUFJFUExZPSgkKGNvbXBnZW4gLWYgXCJcXCR7Y3VyfVwiKSlcbiAgZWxzZVxuICAgICMgc2hlbGxjaGVjayBkaXNhYmxlPVNDMjIwN1xuICAgIENPTVBSRVBMWT0oJChwcmludGYgJyVxXFxcXG4nIFwiXFwke3Jlc3VsdFtAXX1cIikpXG4gIGZpXG5cbiAgcmV0dXJuIDBcbn1cblxuY29tcGxldGUgLUYgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyhwYXRoKX0gLW8gYmFzaGRlZmF1bHQgLW8gZGVmYXVsdCAke3BhdGh9YDtcbiAgfVxuXG4gIC8qKiBHZW5lcmF0ZXMgYmFzaCBjb21wbGV0aW9ucyBtZXRob2QgZm9yIGdpdmVuIGNvbW1hbmQgYW5kIGNoaWxkIGNvbW1hbmRzLiAqL1xuICBwcml2YXRlIGdlbmVyYXRlQ29tcGxldGlvbnMoY29tbWFuZDogQ29tbWFuZCwgcGF0aCA9IFwiXCIsIGluZGV4ID0gMSk6IHN0cmluZyB7XG4gICAgcGF0aCA9IChwYXRoID8gcGF0aCArIFwiIFwiIDogXCJcIikgKyBjb21tYW5kLmdldE5hbWUoKTtcbiAgICBjb25zdCBjb21tYW5kQ29tcGxldGlvbnMgPSB0aGlzLmdlbmVyYXRlQ29tbWFuZENvbXBsZXRpb25zKFxuICAgICAgY29tbWFuZCxcbiAgICAgIHBhdGgsXG4gICAgICBpbmRleCxcbiAgICApO1xuICAgIGNvbnN0IGNoaWxkQ29tbWFuZENvbXBsZXRpb25zOiBzdHJpbmcgPSBjb21tYW5kLmdldENvbW1hbmRzKGZhbHNlKVxuICAgICAgLmZpbHRlcigoc3ViQ29tbWFuZDogQ29tbWFuZCkgPT4gc3ViQ29tbWFuZCAhPT0gY29tbWFuZClcbiAgICAgIC5tYXAoKHN1YkNvbW1hbmQ6IENvbW1hbmQpID0+XG4gICAgICAgIHRoaXMuZ2VuZXJhdGVDb21wbGV0aW9ucyhzdWJDb21tYW5kLCBwYXRoLCBpbmRleCArIDEpXG4gICAgICApXG4gICAgICAuam9pbihcIlwiKTtcblxuICAgIHJldHVybiBgJHtjb21tYW5kQ29tcGxldGlvbnN9XG5cbiR7Y2hpbGRDb21tYW5kQ29tcGxldGlvbnN9YDtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVDb21tYW5kQ29tcGxldGlvbnMoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBwYXRoOiBzdHJpbmcsXG4gICAgaW5kZXg6IG51bWJlcixcbiAgKTogc3RyaW5nIHtcbiAgICBjb25zdCBmbGFnczogc3RyaW5nW10gPSB0aGlzLmdldEZsYWdzKGNvbW1hbmQpO1xuXG4gICAgY29uc3QgY2hpbGRDb21tYW5kTmFtZXM6IHN0cmluZ1tdID0gY29tbWFuZC5nZXRDb21tYW5kcyhmYWxzZSlcbiAgICAgIC5tYXAoKGNoaWxkQ29tbWFuZDogQ29tbWFuZCkgPT4gY2hpbGRDb21tYW5kLmdldE5hbWUoKSk7XG5cbiAgICBjb25zdCBjb21wbGV0aW9uc1BhdGg6IHN0cmluZyA9IH5wYXRoLmluZGV4T2YoXCIgXCIpXG4gICAgICA/IFwiIFwiICsgcGF0aC5zcGxpdChcIiBcIikuc2xpY2UoMSkuam9pbihcIiBcIilcbiAgICAgIDogXCJcIjtcblxuICAgIGNvbnN0IG9wdGlvbkFyZ3VtZW50cyA9IHRoaXMuZ2VuZXJhdGVPcHRpb25Bcmd1bWVudHMoXG4gICAgICBjb21tYW5kLFxuICAgICAgY29tcGxldGlvbnNQYXRoLFxuICAgICk7XG5cbiAgICBjb25zdCBjb21wbGV0aW9uc0NtZDogc3RyaW5nID0gdGhpcy5nZW5lcmF0ZUNvbW1hbmRDb21wbGV0aW9uc0NvbW1hbmQoXG4gICAgICBjb21tYW5kLmdldEFyZ3VtZW50cygpLFxuICAgICAgY29tcGxldGlvbnNQYXRoLFxuICAgICk7XG5cbiAgICByZXR1cm4gYCAgX18ke3JlcGxhY2VTcGVjaWFsQ2hhcnMocGF0aCl9KCkge1xuICAgIG9wdHM9KCR7Wy4uLmZsYWdzLCAuLi5jaGlsZENvbW1hbmROYW1lc10uam9pbihcIiBcIil9KVxuICAgICR7Y29tcGxldGlvbnNDbWR9XG4gICAgaWYgW1sgXFwke2N1cn0gPT0gLSogfHwgXFwke0NPTVBfQ1dPUkR9IC1lcSAke2luZGV4fSBdXSA7IHRoZW5cbiAgICAgIHJldHVybiAwXG4gICAgZmlcbiAgICAke29wdGlvbkFyZ3VtZW50c31cbiAgfWA7XG4gIH1cblxuICBwcml2YXRlIGdldEZsYWdzKGNvbW1hbmQ6IENvbW1hbmQpOiBzdHJpbmdbXSB7XG4gICAgcmV0dXJuIGNvbW1hbmQuZ2V0T3B0aW9ucyhmYWxzZSlcbiAgICAgIC5tYXAoKG9wdGlvbikgPT4gb3B0aW9uLmZsYWdzKVxuICAgICAgLmZsYXQoKTtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb25Bcmd1bWVudHMoXG4gICAgY29tbWFuZDogQ29tbWFuZCxcbiAgICBjb21wbGV0aW9uc1BhdGg6IHN0cmluZyxcbiAgKTogc3RyaW5nIHtcbiAgICBsZXQgb3B0cyA9IFwiXCI7XG4gICAgY29uc3Qgb3B0aW9ucyA9IGNvbW1hbmQuZ2V0T3B0aW9ucyhmYWxzZSk7XG4gICAgaWYgKG9wdGlvbnMubGVuZ3RoKSB7XG4gICAgICBvcHRzICs9ICdjYXNlIFwiJHtwcmV2fVwiIGluJztcbiAgICAgIGZvciAoY29uc3Qgb3B0aW9uIG9mIG9wdGlvbnMpIHtcbiAgICAgICAgY29uc3QgZmxhZ3M6IHN0cmluZyA9IG9wdGlvbi5mbGFnc1xuICAgICAgICAgIC5tYXAoKGZsYWc6IHN0cmluZykgPT4gZmxhZy50cmltKCkpXG4gICAgICAgICAgLmpvaW4oXCJ8XCIpO1xuXG4gICAgICAgIGNvbnN0IGNvbXBsZXRpb25zQ21kOiBzdHJpbmcgPSB0aGlzLmdlbmVyYXRlT3B0aW9uQ29tcGxldGlvbnNDb21tYW5kKFxuICAgICAgICAgIG9wdGlvbi5hcmdzLFxuICAgICAgICAgIGNvbXBsZXRpb25zUGF0aCxcbiAgICAgICAgICB7IHN0YW5kYWxvbmU6IG9wdGlvbi5zdGFuZGFsb25lIH0sXG4gICAgICAgICk7XG5cbiAgICAgICAgb3B0cyArPSBgXFxuICAgICAgJHtmbGFnc30pICR7Y29tcGxldGlvbnNDbWR9IDs7YDtcbiAgICAgIH1cbiAgICAgIG9wdHMgKz0gXCJcXG4gICAgZXNhY1wiO1xuICAgIH1cblxuICAgIHJldHVybiBvcHRzO1xuICB9XG5cbiAgcHJpdmF0ZSBnZW5lcmF0ZUNvbW1hbmRDb21wbGV0aW9uc0NvbW1hbmQoXG4gICAgYXJnczogSUFyZ3VtZW50W10sXG4gICAgcGF0aDogc3RyaW5nLFxuICApIHtcbiAgICBpZiAoYXJncy5sZW5ndGgpIHtcbiAgICAgIC8vIEBUT0RPOiBhZGQgc3VwcG9ydCBmb3IgbXVsdGlwbGUgYXJndW1lbnRzXG4gICAgICByZXR1cm4gYF8ke3JlcGxhY2VTcGVjaWFsQ2hhcnModGhpcy5jbWQuZ2V0TmFtZSgpKX1fY29tcGxldGUgJHtcbiAgICAgICAgYXJnc1swXS5hY3Rpb25cbiAgICAgIH0ke3BhdGh9YDtcbiAgICB9XG5cbiAgICByZXR1cm4gXCJcIjtcbiAgfVxuXG4gIHByaXZhdGUgZ2VuZXJhdGVPcHRpb25Db21wbGV0aW9uc0NvbW1hbmQoXG4gICAgYXJnczogSUFyZ3VtZW50W10sXG4gICAgcGF0aDogc3RyaW5nLFxuICAgIG9wdHM/OiB7IHN0YW5kYWxvbmU/OiBib29sZWFuIH0sXG4gICkge1xuICAgIGlmIChhcmdzLmxlbmd0aCkge1xuICAgICAgLy8gQFRPRE86IGFkZCBzdXBwb3J0IGZvciBtdWx0aXBsZSBhcmd1bWVudHNcbiAgICAgIHJldHVybiBgb3B0cz0oKTsgXyR7cmVwbGFjZVNwZWNpYWxDaGFycyh0aGlzLmNtZC5nZXROYW1lKCkpfV9jb21wbGV0ZSAke1xuICAgICAgICBhcmdzWzBdLmFjdGlvblxuICAgICAgfSR7cGF0aH1gO1xuICAgIH1cblxuICAgIGlmIChvcHRzPy5zdGFuZGFsb25lKSB7XG4gICAgICByZXR1cm4gXCJvcHRzPSgpXCI7XG4gICAgfVxuXG4gICAgcmV0dXJuIFwiXCI7XG4gIH1cbn1cblxuZnVuY3Rpb24gcmVwbGFjZVNwZWNpYWxDaGFycyhzdHI6IHN0cmluZyk6IHN0cmluZyB7XG4gIHJldHVybiBzdHIucmVwbGFjZSgvW15hLXpBLVowLTldL2csIFwiX1wiKTtcbn1cbiJdfQ==