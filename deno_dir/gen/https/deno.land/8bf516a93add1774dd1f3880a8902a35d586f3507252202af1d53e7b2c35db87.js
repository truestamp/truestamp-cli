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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUMvQyxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFHbEQsTUFBTSxPQUFPLFdBQVksU0FBUSxPQUFpQztJQUNoRSxZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxXQUFXLEVBQUUsQ0FBQzthQUNwQyxTQUFTLENBQUMsbUJBQW1CLENBQUM7YUFDOUIsV0FBVyxDQUFDLDhDQUE4QyxDQUFDO2FBQzNELE1BQU0sQ0FBQyxDQUFDLENBQUMsRUFBRSxJQUFhLEVBQUUsRUFBRTtZQUMzQixJQUFJLENBQUMsR0FBRyxFQUFFO2dCQUNSLEdBQUcsR0FBRyxJQUFJO29CQUNSLENBQUMsQ0FBQyxJQUFJLENBQUMsZUFBZSxFQUFFLEVBQUUsY0FBYyxDQUFDLElBQUksQ0FBQztvQkFDOUMsQ0FBQyxDQUFDLElBQUksQ0FBQyxlQUFlLEVBQUUsQ0FBQzthQUM1QjtZQUNELElBQUksQ0FBQyxHQUFHLEVBQUU7Z0JBQ1IsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLGVBQWUsRUFBRSxFQUFFLFdBQVcsRUFBRSxDQUFDO2dCQUNuRCxNQUFNLElBQUksY0FBYyxDQUFDLElBQUksSUFBSSxFQUFFLEVBQUUsSUFBSSxJQUFJLEVBQUUsRUFBRTtvQkFDL0MsSUFBSSxDQUFDLE9BQU8sRUFBRTtvQkFDZCxHQUFHLElBQUksQ0FBQyxVQUFVLEVBQUU7aUJBQ3JCLENBQUMsQ0FBQzthQUNKO1lBQ0QsR0FBRyxDQUFDLFFBQVEsRUFBRSxDQUFDO1lBQ2YsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztRQUNmLENBQUMsQ0FBQyxDQUFDO0lBQ1AsQ0FBQztDQUNGIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBVbmtub3duQ29tbWFuZCB9IGZyb20gXCIuLi9fZXJyb3JzLnRzXCI7XG5pbXBvcnQgeyBDb21tYW5kVHlwZSB9IGZyb20gXCIuLi90eXBlcy9jb21tYW5kLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgd2VsbCBmb3JtYXR0ZWQgYW5kIGNvbG9yZWQgaGVscCBvdXRwdXQgZm9yIHNwZWNpZmllZCBjb21tYW5kLiAqL1xuZXhwb3J0IGNsYXNzIEhlbHBDb21tYW5kIGV4dGVuZHMgQ29tbWFuZDx2b2lkLCBbY29tbWFuZD86IHN0cmluZ10+IHtcbiAgcHVibGljIGNvbnN0cnVjdG9yKGNtZD86IENvbW1hbmQpIHtcbiAgICBzdXBlcigpO1xuICAgIHRoaXMudHlwZShcImNvbW1hbmRcIiwgbmV3IENvbW1hbmRUeXBlKCkpXG4gICAgICAuYXJndW1lbnRzKFwiW2NvbW1hbmQ6Y29tbWFuZF1cIilcbiAgICAgIC5kZXNjcmlwdGlvbihcIlNob3cgdGhpcyBoZWxwIG9yIHRoZSBoZWxwIG9mIGEgc3ViLWNvbW1hbmQuXCIpXG4gICAgICAuYWN0aW9uKChfLCBuYW1lPzogc3RyaW5nKSA9PiB7XG4gICAgICAgIGlmICghY21kKSB7XG4gICAgICAgICAgY21kID0gbmFtZVxuICAgICAgICAgICAgPyB0aGlzLmdldEdsb2JhbFBhcmVudCgpPy5nZXRCYXNlQ29tbWFuZChuYW1lKVxuICAgICAgICAgICAgOiB0aGlzLmdldEdsb2JhbFBhcmVudCgpO1xuICAgICAgICB9XG4gICAgICAgIGlmICghY21kKSB7XG4gICAgICAgICAgY29uc3QgY21kcyA9IHRoaXMuZ2V0R2xvYmFsUGFyZW50KCk/LmdldENvbW1hbmRzKCk7XG4gICAgICAgICAgdGhyb3cgbmV3IFVua25vd25Db21tYW5kKG5hbWUgPz8gXCJcIiwgY21kcyA/PyBbXSwgW1xuICAgICAgICAgICAgdGhpcy5nZXROYW1lKCksXG4gICAgICAgICAgICAuLi50aGlzLmdldEFsaWFzZXMoKSxcbiAgICAgICAgICBdKTtcbiAgICAgICAgfVxuICAgICAgICBjbWQuc2hvd0hlbHAoKTtcbiAgICAgICAgRGVuby5leGl0KDApO1xuICAgICAgfSk7XG4gIH1cbn1cbiJdfQ==