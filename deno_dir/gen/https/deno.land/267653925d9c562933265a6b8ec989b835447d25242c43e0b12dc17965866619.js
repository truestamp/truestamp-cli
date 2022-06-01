import { Command } from "../command.ts";
import { UnknownCommand } from "../_errors.ts";
import { CommandType } from "../types/command.ts";
export class HelpCommand extends Command {
    constructor(cmd) {
        super();
        return this
            .type("command", new CommandType())
            .arguments("[command:command]")
            .description("Show this help or the help of a sub-command.")
            .action(async (_, name) => {
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
            await cmd.checkVersion();
            cmd.showHelp();
            Deno.exit(0);
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUMvQyxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFHbEQsTUFBTSxPQUFPLFdBQ1gsU0FBUSxPQUFzRDtJQUM5RCxZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsT0FBTyxJQUFJO2FBQ1IsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLFdBQVcsRUFBRSxDQUFDO2FBQ2xDLFNBQVMsQ0FBQyxtQkFBbUIsQ0FBQzthQUM5QixXQUFXLENBQUMsOENBQThDLENBQUM7YUFDM0QsTUFBTSxDQUFDLEtBQUssRUFBRSxDQUFDLEVBQUUsSUFBYSxFQUFFLEVBQUU7WUFDakMsSUFBSSxDQUFDLEdBQUcsRUFBRTtnQkFDUixHQUFHLEdBQUcsSUFBSTtvQkFDUixDQUFDLENBQUMsSUFBSSxDQUFDLGVBQWUsRUFBRSxFQUFFLGNBQWMsQ0FBQyxJQUFJLENBQUM7b0JBQzlDLENBQUMsQ0FBQyxJQUFJLENBQUMsZUFBZSxFQUFFLENBQUM7YUFDNUI7WUFDRCxJQUFJLENBQUMsR0FBRyxFQUFFO2dCQUNSLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxlQUFlLEVBQUUsRUFBRSxXQUFXLEVBQUUsQ0FBQztnQkFDbkQsTUFBTSxJQUFJLGNBQWMsQ0FBQyxJQUFJLElBQUksRUFBRSxFQUFFLElBQUksSUFBSSxFQUFFLEVBQUU7b0JBQy9DLElBQUksQ0FBQyxPQUFPLEVBQUU7b0JBQ2QsR0FBRyxJQUFJLENBQUMsVUFBVSxFQUFFO2lCQUNyQixDQUFDLENBQUM7YUFDSjtZQUNELE1BQU0sR0FBRyxDQUFDLFlBQVksRUFBRSxDQUFDO1lBQ3pCLEdBQUcsQ0FBQyxRQUFRLEVBQUUsQ0FBQztZQUNmLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUM7UUFDZixDQUFDLENBQUMsQ0FBQztJQUNQLENBQUM7Q0FDRiIsInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENvbW1hbmQgfSBmcm9tIFwiLi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHsgVW5rbm93bkNvbW1hbmQgfSBmcm9tIFwiLi4vX2Vycm9ycy50c1wiO1xuaW1wb3J0IHsgQ29tbWFuZFR5cGUgfSBmcm9tIFwiLi4vdHlwZXMvY29tbWFuZC50c1wiO1xuXG4vKiogR2VuZXJhdGVzIHdlbGwgZm9ybWF0dGVkIGFuZCBjb2xvcmVkIGhlbHAgb3V0cHV0IGZvciBzcGVjaWZpZWQgY29tbWFuZC4gKi9cbmV4cG9ydCBjbGFzcyBIZWxwQ29tbWFuZFxuICBleHRlbmRzIENvbW1hbmQ8dm9pZCwgdm9pZCwgdm9pZCwgW2NvbW1hbmROYW1lPzogQ29tbWFuZFR5cGVdPiB7XG4gIHB1YmxpYyBjb25zdHJ1Y3RvcihjbWQ/OiBDb21tYW5kKSB7XG4gICAgc3VwZXIoKTtcbiAgICByZXR1cm4gdGhpc1xuICAgICAgLnR5cGUoXCJjb21tYW5kXCIsIG5ldyBDb21tYW5kVHlwZSgpKVxuICAgICAgLmFyZ3VtZW50cyhcIltjb21tYW5kOmNvbW1hbmRdXCIpXG4gICAgICAuZGVzY3JpcHRpb24oXCJTaG93IHRoaXMgaGVscCBvciB0aGUgaGVscCBvZiBhIHN1Yi1jb21tYW5kLlwiKVxuICAgICAgLmFjdGlvbihhc3luYyAoXywgbmFtZT86IHN0cmluZykgPT4ge1xuICAgICAgICBpZiAoIWNtZCkge1xuICAgICAgICAgIGNtZCA9IG5hbWVcbiAgICAgICAgICAgID8gdGhpcy5nZXRHbG9iYWxQYXJlbnQoKT8uZ2V0QmFzZUNvbW1hbmQobmFtZSlcbiAgICAgICAgICAgIDogdGhpcy5nZXRHbG9iYWxQYXJlbnQoKTtcbiAgICAgICAgfVxuICAgICAgICBpZiAoIWNtZCkge1xuICAgICAgICAgIGNvbnN0IGNtZHMgPSB0aGlzLmdldEdsb2JhbFBhcmVudCgpPy5nZXRDb21tYW5kcygpO1xuICAgICAgICAgIHRocm93IG5ldyBVbmtub3duQ29tbWFuZChuYW1lID8/IFwiXCIsIGNtZHMgPz8gW10sIFtcbiAgICAgICAgICAgIHRoaXMuZ2V0TmFtZSgpLFxuICAgICAgICAgICAgLi4udGhpcy5nZXRBbGlhc2VzKCksXG4gICAgICAgICAgXSk7XG4gICAgICAgIH1cbiAgICAgICAgYXdhaXQgY21kLmNoZWNrVmVyc2lvbigpO1xuICAgICAgICBjbWQuc2hvd0hlbHAoKTtcbiAgICAgICAgRGVuby5leGl0KDApO1xuICAgICAgfSk7XG4gIH1cbn1cbiJdfQ==