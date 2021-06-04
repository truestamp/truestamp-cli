export class FishCompletionsGenerator {
    cmd;
    static generate(cmd) {
        return new FishCompletionsGenerator(cmd).generate();
    }
    constructor(cmd) {
        this.cmd = cmd;
    }
    generate() {
        const path = this.cmd.getPath();
        const version = this.cmd.getVersion()
            ? ` v${this.cmd.getVersion()}`
            : "";
        return `#!/usr/bin/env fish
# fish completion support for ${path}${version}

function __fish_${replaceSpecialChars(this.cmd.getName())}_using_command
  set cmds ${getCommandFnNames(this.cmd).join(" ")}
  set words (commandline -opc)
  set cmd "_"
  for word in $words
    switch $word
      case '-*'
        continue
      case '*'
        set word (string replace -r -a '\\W' '_' $word)
        set cmd_tmp $cmd"_$word"
        if contains $cmd_tmp $cmds
          set cmd $cmd_tmp
        end
    end
  end
  if [ "$cmd" = "$argv[1]" ]
    return 0
  end
  return 1
end

${this.generateCompletions(this.cmd).trim()}
`;
    }
    generateCompletions(command) {
        const parent = command.getParent();
        let result = ``;
        if (parent) {
            result += "\n" + this.complete(parent, {
                description: command.getShortDescription(),
                arguments: command.getName(),
            });
        }
        const commandArgs = command.getArguments();
        if (commandArgs.length) {
            result += "\n" + this.complete(command, {
                arguments: commandArgs.length
                    ? this.getCompletionCommand(commandArgs[0].action + " " + getCompletionsPath(command))
                    : undefined,
            });
        }
        for (const option of command.getOptions(false)) {
            result += "\n" + this.completeOption(command, option);
        }
        for (const subCommand of command.getCommands(false)) {
            result += this.generateCompletions(subCommand);
        }
        return result;
    }
    completeOption(command, option) {
        const shortOption = option.flags
            .find((flag) => flag.length === 2)
            ?.replace(/^(-)+/, "");
        const longOption = option.flags
            .find((flag) => flag.length > 2)
            ?.replace(/^(-)+/, "");
        return this.complete(command, {
            description: option.description,
            shortOption: shortOption,
            longOption: longOption,
            required: true,
            standalone: option.standalone,
            arguments: option.args.length
                ? this.getCompletionCommand(option.args[0].action + " " + getCompletionsPath(command))
                : undefined,
        });
    }
    complete(command, options) {
        const cmd = ["complete"];
        cmd.push("-c", this.cmd.getName());
        cmd.push("-n", `'__fish_${replaceSpecialChars(this.cmd.getName())}_using_command __${replaceSpecialChars(command.getPath())}'`);
        options.shortOption && cmd.push("-s", options.shortOption);
        options.longOption && cmd.push("-l", options.longOption);
        options.standalone && cmd.push("-x");
        cmd.push("-k");
        cmd.push("-f");
        if (options.arguments) {
            options.required && cmd.push("-r");
            cmd.push("-a", options.arguments);
        }
        options.description && cmd.push("-d", `'${options.description}'`);
        return cmd.join(" ");
    }
    getCompletionCommand(cmd) {
        return `'(${this.cmd.getName()} completions complete ${cmd.trim()})'`;
    }
}
function getCommandFnNames(cmd, cmds = []) {
    cmds.push(`__${replaceSpecialChars(cmd.getPath())}`);
    cmd.getCommands(false).forEach((command) => {
        getCommandFnNames(command, cmds);
    });
    return cmds;
}
function getCompletionsPath(command) {
    return command.getPath()
        .split(" ")
        .slice(1)
        .join(" ");
}
function replaceSpecialChars(str) {
    return str.replace(/[^a-zA-Z0-9]/g, "_");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Zpc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2Zpc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWNBLE1BQU0sT0FBTyx3QkFBd0I7SUFNTDtJQUp2QixNQUFNLENBQUMsUUFBUSxDQUFDLEdBQVk7UUFDakMsT0FBTyxJQUFJLHdCQUF3QixDQUFDLEdBQUcsQ0FBQyxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3RELENBQUM7SUFFRCxZQUE4QixHQUFZO1FBQVosUUFBRyxHQUFILEdBQUcsQ0FBUztJQUFHLENBQUM7SUFHdEMsUUFBUTtRQUNkLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUM7UUFDaEMsTUFBTSxPQUFPLEdBQXVCLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFO1lBQ3ZELENBQUMsQ0FBQyxLQUFLLElBQUksQ0FBQyxHQUFHLENBQUMsVUFBVSxFQUFFLEVBQUU7WUFDOUIsQ0FBQyxDQUFDLEVBQUUsQ0FBQztRQUVQLE9BQU87Z0NBQ3FCLElBQUksR0FBRyxPQUFPOztrQkFFNUIsbUJBQW1CLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxPQUFPLEVBQUUsQ0FBQzthQUM1QyxpQkFBaUIsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O0VBcUJoRCxJQUFJLENBQUMsbUJBQW1CLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQUksRUFBRTtDQUMxQyxDQUFDO0lBQ0EsQ0FBQztJQUVPLG1CQUFtQixDQUFDLE9BQWdCO1FBQzFDLE1BQU0sTUFBTSxHQUF3QixPQUFPLENBQUMsU0FBUyxFQUFFLENBQUM7UUFDeEQsSUFBSSxNQUFNLEdBQUcsRUFBRSxDQUFDO1FBRWhCLElBQUksTUFBTSxFQUFFO1lBRVYsTUFB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