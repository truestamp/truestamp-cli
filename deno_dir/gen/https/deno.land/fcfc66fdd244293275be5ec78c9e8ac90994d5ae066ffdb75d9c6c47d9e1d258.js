import { Command } from "../command.ts";
import { UnknownCompletionCommand } from "../_errors.ts";
/** Execute auto completion method of command and action. */ export class CompleteCommand extends Command {
    constructor(cmd){
        super();
        return this.description("Get completions for given action from given command.").noGlobals().arguments("<action:string> [command...:string]").action(async (_, action, ...commandNames)=>{
            let parent;
            const completeCommand = commandNames?.reduce((cmd, name)=>{
                parent = cmd;
                const childCmd = cmd.getCommand(name, false);
                if (!childCmd) {
                    throw new UnknownCompletionCommand(name, cmd.getCommands());
                }
                return childCmd;
            }, cmd || this.getMainCommand()) ?? (cmd || this.getMainCommand());
            const completion = completeCommand.getCompletion(action);
            const result = await completion?.complete(completeCommand, parent) ?? [];
            if (result?.length) {
                Deno.stdout.writeSync(new TextEncoder().encode(result.join("\n")));
            }
        }).reset();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC9jb21wbGV0aW9ucy9jb21wbGV0ZS50cyJdLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB7IFVua25vd25Db21wbGV0aW9uQ29tbWFuZCB9IGZyb20gXCIuLi9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgdHlwZSB7IElDb21wbGV0aW9uIH0gZnJvbSBcIi4uL3R5cGVzLnRzXCI7XG5cbi8qKiBFeGVjdXRlIGF1dG8gY29tcGxldGlvbiBtZXRob2Qgb2YgY29tbWFuZCBhbmQgYWN0aW9uLiAqL1xuZXhwb3J0IGNsYXNzIENvbXBsZXRlQ29tbWFuZCBleHRlbmRzIENvbW1hbmQ8XG4gIHZvaWQsXG4gIHZvaWQsXG4gIHZvaWQsXG4gIFthY3Rpb246IHN0cmluZywgLi4uY29tbWFuZE5hbWVzOiBBcnJheTxzdHJpbmc+XVxuPiB7XG4gIHB1YmxpYyBjb25zdHJ1Y3RvcihjbWQ/OiBDb21tYW5kKSB7XG4gICAgc3VwZXIoKTtcbiAgICByZXR1cm4gdGhpc1xuICAgICAgLmRlc2NyaXB0aW9uKFxuICAgICAgICBcIkdldCBjb21wbGV0aW9ucyBmb3IgZ2l2ZW4gYWN0aW9uIGZyb20gZ2l2ZW4gY29tbWFuZC5cIixcbiAgICAgIClcbiAgICAgIC5ub0dsb2JhbHMoKVxuICAgICAgLmFyZ3VtZW50cyhcIjxhY3Rpb246c3RyaW5nPiBbY29tbWFuZC4uLjpzdHJpbmddXCIpXG4gICAgICAuYWN0aW9uKGFzeW5jIChfLCBhY3Rpb246IHN0cmluZywgLi4uY29tbWFuZE5hbWVzOiBBcnJheTxzdHJpbmc+KSA9PiB7XG4gICAgICAgIGxldCBwYXJlbnQ6IENvbW1hbmQgfCB1bmRlZmluZWQ7XG4gICAgICAgIGNvbnN0IGNvbXBsZXRlQ29tbWFuZDogQ29tbWFuZCA9IGNvbW1hbmROYW1lc1xuICAgICAgICAgID8ucmVkdWNlKChjbWQ6IENvbW1hbmQsIG5hbWU6IHN0cmluZyk6IENvbW1hbmQgPT4ge1xuICAgICAgICAgICAgcGFyZW50ID0gY21kO1xuICAgICAgICAgICAgY29uc3QgY2hpbGRDbWQ6IENvbW1hbmQgfCB1bmRlZmluZWQgPSBjbWQuZ2V0Q29tbWFuZChuYW1lLCBmYWxzZSk7XG4gICAgICAgICAgICBpZiAoIWNoaWxkQ21kKSB7XG4gICAgICAgICAgICAgIHRocm93IG5ldyBVbmtub3duQ29tcGxldGlvbkNvbW1hbmQobmFtZSwgY21kLmdldENvbW1hbmRzKCkpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgcmV0dXJuIGNoaWxkQ21kO1xuICAgICAgICAgIH0sIGNtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCkpID8/IChjbWQgfHwgdGhpcy5nZXRNYWluQ29tbWFuZCgpKTtcblxuICAgICAgICBjb25zdCBjb21wbGV0aW9uOiBJQ29tcGxldGlvbiB8IHVuZGVmaW5lZCA9IGNvbXBsZXRlQ29tbWFuZFxuICAgICAgICAgIC5nZXRDb21wbGV0aW9uKGFjdGlvbik7XG4gICAgICAgIGNvbnN0IHJlc3VsdDogQXJyYXk8c3RyaW5nIHwgbnVtYmVyIHwgYm9vbGVhbj4gPVxuICAgICAgICAgIGF3YWl0IGNvbXBsZXRpb24/LmNvbXBsZXRlKGNvbXBsZXRlQ29tbWFuZCwgcGFyZW50KSA/PyBbXTtcblxuICAgICAgICBpZiAocmVzdWx0Py5sZW5ndGgpIHtcbiAgICAgICAgICBEZW5vLnN0ZG91dC53cml0ZVN5bmMobmV3IFRleHRFbmNvZGVyKCkuZW5jb2RlKHJlc3VsdC5qb2luKFwiXFxuXCIpKSk7XG4gICAgICAgIH1cbiAgICAgIH0pXG4gICAgICAucmVzZXQoKTtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLFNBQVMsT0FBTyxRQUFRLGVBQWUsQ0FBQztBQUN4QyxTQUFTLHdCQUF3QixRQUFRLGVBQWUsQ0FBQztBQUd6RCwwREFBMEQsR0FDMUQsT0FBTyxNQUFNLGVBQWUsU0FBUyxPQUFPO0lBTTFDLFlBQW1CLEdBQWEsQ0FBRTtRQUNoQyxLQUFLLEVBQUUsQ0FBQztRQUNSLE9BQU8sSUFBSSxDQUNSLFdBQVcsQ0FDVixzREFBc0QsQ0FDdkQsQ0FDQSxTQUFTLEVBQUUsQ0FDWCxTQUFTLENBQUMscUNBQXFDLENBQUMsQ0FDaEQsTUFBTSxDQUFDLE9BQU8sQ0FBQyxFQUFFLE1BQWMsRUFBRSxHQUFHLFlBQVksQUFBZSxHQUFLO1lBQ25FLElBQUksTUFBTSxBQUFxQixBQUFDO1lBQ2hDLE1BQU0sZUFBZSxHQUFZLFlBQVksRUFDekMsTUFBTSxDQUFDLENBQUMsR0FBWSxFQUFFLElBQVksR0FBYztnQkFDaEQsTUFBTSxHQUFHLEdBQUcsQ0FBQztnQkFDYixNQUFNLFFBQVEsR0FBd0IsR0FBRyxDQUFDLFVBQVUsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDLEFBQUM7Z0JBQ2xFLElBQUksQ0FBQyxRQUFRLEVBQUU7b0JBQ2IsTUFBTSxJQUFJLHdCQUF3QixDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsV0FBVyxFQUFFLENBQUMsQ0FBQztnQkFDOUQsQ0FBQztnQkFDRCxPQUFPLFFBQVEsQ0FBQztZQUNsQixDQUFDLEVBQUUsR0FBRyxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQyxJQUFJLENBQUMsR0FBRyxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQyxBQUFDO1lBRXJFLE1BQU0sVUFBVSxHQUE0QixlQUFlLENBQ3hELGFBQWEsQ0FBQyxNQUFNLENBQUMsQUFBQztZQUN6QixNQUFNLE1BQU0sR0FDVixNQUFNLFVBQVUsRUFBRSxRQUFRLENBQUMsZUFBZSxFQUFFLE1BQU0sQ0FBQyxJQUFJLEVBQUUsQUFBQztZQUU1RCxJQUFJLE1BQU0sRUFBRSxNQUFNLEVBQUU7Z0JBQ2xCLElBQUksQ0FBQyxNQUFNLENBQUMsU0FBUyxDQUFDLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDO1lBQ3JFLENBQUM7UUFDSCxDQUFDLENBQUMsQ0FDRCxLQUFLLEVBQUUsQ0FBQztJQUNiO0NBQ0QifQ==