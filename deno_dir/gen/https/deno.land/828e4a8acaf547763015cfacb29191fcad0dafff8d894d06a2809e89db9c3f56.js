import { Command } from "../command.ts";
import { UnknownCommand } from "../_errors.ts";
import { CommandType } from "../types/command.ts";
export class HelpCommand extends Command {
    constructor(cmd) {
        super();
        this.type("command", new CommandType())
            .arguments("[command:command]")
            .description("Show this help or the help of a sub-command.")
            .action((_, name) => {
            if (!cmd) {
                cmd = name
                    ? this.getGlobalParent()?.getBaseCommand(name)
                    : this.getGlobalParent();
            }
            if (!cmd) {
                const cmds = this.getGlobalParent()?.getCommands();
                throw new UnknownCommand(name ?? "", cmds ?? [], [
                    this.getName(),
                    ...this.getAliases(),
                ]);
            }
            cmd.showHelp();
            Deno.exit(0);
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUMvQyxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFHbEQsTUFBTSxPQUFPLFdBQVksU0FBUSxPQUFpQztJQUNoRSxZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxXQUFXLEVBQUUsQ0FBQzthQUNwQyxTQUFTLENBQUMsbUJBQW1CLENBQUM7YUFDOUIsV0FBVyxDQUFDLDhDQUE4QyxDQUFDO2FBQzNELE1BQU0sQ0FBQyxDQUFDLENBQUMsRUFBRSxJQUFhLEVBQUUsRUFBRTtZQUMzQixJQUFJLENBQUMsR0FBRyxFQUFFO2dCQUNSLEdBQUcsR0FBRyxJQUFJO29CQUNSLENBQUMsQ0FBQyxJQUFJLENBQUMsZUFBZSxFQUFFLEVBQUUsY0FBYyxDQUFDLElBQUksQ0FBQztvQkFDOUMsQ0FBQyxDQUFDLElBQUksQ0FBQyxlQUFlLEVBQUUsQ0FBQzthQUM1QjtZQUNELElBQUksQ0FBQyxHQUFHLEVBQUU7Z0JBQ1IsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLGVBQWUsRUFBRSxFQUFFLFdBQVcsRUFBRSxDQUFDO2dCQUNuRCxNQUFNLElBQUksY0FBYyxDQUFDLElBQUksSUFBSSxFQUFFLEVBQUUsSUFBSSxJQUFJLEVBQUUsRUFBRTtvQkFDL0MsSUFBSSxDQUFDLE9BQU8sRUFBRTtvQkFDZCxHQUFHLElBQUksQ0FBQyxVQUFVLEVBQUU7aUJBQ3JCLENBQUMsQ0FBQzthQUNKO1lBQ0QsR0FBRyxDQUFDLFFBQVEsRUFBRSxDQUFDO1lBQ2YsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztRQUNmLENBQUMsQ0FBQyxDQUFDO0lBQ1AsQ0FBQztDQUNGIn0=