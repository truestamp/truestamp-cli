import { Command } from "../command.ts";
import { UnknownCompletionCommand } from "../_errors.ts";
export class CompleteCommand extends Command {
    constructor(cmd) {
        super();
        this.description("Get completions for given action from given command.")
            .arguments("<action:string> [command...:string]")
            .action(async (_, action, commandNames) => {
            let parent;
            const completeCommand = commandNames
                ?.reduce((cmd, name) => {
                parent = cmd;
                const childCmd = cmd.getCommand(name, false);
                if (!childCmd) {
                    throw new UnknownCompletionCommand(name, cmd.getCommands());
                }
                return childCmd;
            }, cmd || this.getMainCommand()) ?? (cmd || this.getMainCommand());
            const completion = completeCommand
                .getCompletion(action);
            const result = await completion?.complete(completeCommand, parent) ?? [];
            if (result?.length) {
                Deno.stdout.writeSync(new TextEncoder().encode(result.join("\n")));
            }
        })
            .reset();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29tcGxldGUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJjb21wbGV0ZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsT0FBTyxFQUFFLE1BQU0sZUFBZSxDQUFDO0FBQ3hDLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUl6RCxNQUFNLE9BQU8sZUFDWCxTQUFRLE9BQTZEO0lBQ3JFLFlBQW1CLEdBQWE7UUFDOUIsS0FBSyxFQUFFLENBQUM7UUFDUixJQUFJLENBQUMsV0FBVyxDQUFDLHNEQUFzRCxDQUFDO2FBQ3JFLFNBQVMsQ0FBQyxxQ0FBcUMsQ0FBQzthQUNoRCxNQUFNLENBQUMsS0FBSyxFQUFFLENBQUMsRUFBRSxNQUFjLEVBQUUsWUFBNEIsRUFBRSxFQUFFO1lBQ2hFLElBQUksTUFBMkIsQ0FBQztZQUNoQyxNQUFNLGVBQWUsR0FBWSxZQUFZO2dCQUMzQyxFQUFFLE1BQU0sQ0FBQyxDQUFDLEdBQVksRUFBRSxJQUFZLEVBQVcsRUFBRTtnQkFDL0MsTUFBTSxHQUFHLEdBQUcsQ0FBQztnQkFDYixNQUFNLFFBQVEsR0FBd0IsR0FBRyxDQUFDLFVBQVUsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUM7Z0JBQ2xFLElBQUksQ0FBQyxRQUFRLEVBQUU7b0JBQ2IsTUFBTSxJQUFJLHdCQUF3QixDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsV0FBVyxFQUFFLENBQUMsQ0FBQztpQkFDN0Q7Z0JBQ0QsT0FBTyxRQUFRLENBQUM7WUFDbEIsQ0FBQyxFQUFFLEdBQUcsSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUMsSUFBSSxDQUFDLEdBQUcsSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUMsQ0FBQztZQUVyRSxNQUFNLFVBQVUsR0FBNEIsZUFBZTtpQkFDeEQsYUFBYSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1lBQ3pCLE1BQU0sTUFBTSxHQUNWLE1BQU0sVUFBVSxFQUFFLFFBQVEsQ0FBQyxlQUFlLEVBQUUsTUFBTSxDQUFDLElBQUksRUFBRSxDQUFDO1lBRTVELElBQUksTUFBTSxFQUFFLE1BQU0sRUFBRTtnQkFDbEIsSUFBSSxDQUFDLE1BQU0sQ0FBQyxTQUFTLENBQUMsSUFBSSxXQUFXLEVBQUUsQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUM7YUFDcEU7UUFDSCxDQUFDLENBQUM7YUFDRCxLQUFLLEVBQUUsQ0FBQztJQUNiLENBQUM7Q0FDRiJ9