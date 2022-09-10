import { Command } from "../command.ts";
import { UnknownCommand } from "../_errors.ts";
import { CommandType } from "../types/command.ts";
/** Generates well formatted and colored help output for specified command. */ export class HelpCommand extends Command {
    constructor(cmd){
        super();
        return this.type("command", new CommandType()).arguments("[command:command]").description("Show this help or the help of a sub-command.").noGlobals().action(async (_, name)=>{
            if (!cmd) {
                cmd = name ? this.getGlobalParent()?.getBaseCommand(name) : this.getGlobalParent();
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
            if (this.shouldExit()) {
                Deno.exit(0);
            }
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC9oZWxwL21vZC50cyJdLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB7IFVua25vd25Db21tYW5kIH0gZnJvbSBcIi4uL19lcnJvcnMudHNcIjtcbmltcG9ydCB7IENvbW1hbmRUeXBlIH0gZnJvbSBcIi4uL3R5cGVzL2NvbW1hbmQudHNcIjtcblxuLyoqIEdlbmVyYXRlcyB3ZWxsIGZvcm1hdHRlZCBhbmQgY29sb3JlZCBoZWxwIG91dHB1dCBmb3Igc3BlY2lmaWVkIGNvbW1hbmQuICovXG5leHBvcnQgY2xhc3MgSGVscENvbW1hbmRcbiAgZXh0ZW5kcyBDb21tYW5kPHZvaWQsIHZvaWQsIHZvaWQsIFtjb21tYW5kTmFtZT86IENvbW1hbmRUeXBlXT4ge1xuICBwdWJsaWMgY29uc3RydWN0b3IoY21kPzogQ29tbWFuZCkge1xuICAgIHN1cGVyKCk7XG4gICAgcmV0dXJuIHRoaXNcbiAgICAgIC50eXBlKFwiY29tbWFuZFwiLCBuZXcgQ29tbWFuZFR5cGUoKSlcbiAgICAgIC5hcmd1bWVudHMoXCJbY29tbWFuZDpjb21tYW5kXVwiKVxuICAgICAgLmRlc2NyaXB0aW9uKFwiU2hvdyB0aGlzIGhlbHAgb3IgdGhlIGhlbHAgb2YgYSBzdWItY29tbWFuZC5cIilcbiAgICAgIC5ub0dsb2JhbHMoKVxuICAgICAgLmFjdGlvbihhc3luYyAoXywgbmFtZT86IHN0cmluZykgPT4ge1xuICAgICAgICBpZiAoIWNtZCkge1xuICAgICAgICAgIGNtZCA9IG5hbWVcbiAgICAgICAgICAgID8gdGhpcy5nZXRHbG9iYWxQYXJlbnQoKT8uZ2V0QmFzZUNvbW1hbmQobmFtZSlcbiAgICAgICAgICAgIDogdGhpcy5nZXRHbG9iYWxQYXJlbnQoKTtcbiAgICAgICAgfVxuICAgICAgICBpZiAoIWNtZCkge1xuICAgICAgICAgIGNvbnN0IGNtZHMgPSB0aGlzLmdldEdsb2JhbFBhcmVudCgpPy5nZXRDb21tYW5kcygpO1xuICAgICAgICAgIHRocm93IG5ldyBVbmtub3duQ29tbWFuZChuYW1lID8/IFwiXCIsIGNtZHMgPz8gW10sIFtcbiAgICAgICAgICAgIHRoaXMuZ2V0TmFtZSgpLFxuICAgICAgICAgICAgLi4udGhpcy5nZXRBbGlhc2VzKCksXG4gICAgICAgICAgXSk7XG4gICAgICAgIH1cbiAgICAgICAgYXdhaXQgY21kLmNoZWNrVmVyc2lvbigpO1xuICAgICAgICBjbWQuc2hvd0hlbHAoKTtcbiAgICAgICAgaWYgKHRoaXMuc2hvdWxkRXhpdCgpKSB7XG4gICAgICAgICAgRGVuby5leGl0KDApO1xuICAgICAgICB9XG4gICAgICB9KTtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLFNBQVMsT0FBTyxRQUFRLGVBQWUsQ0FBQztBQUN4QyxTQUFTLGNBQWMsUUFBUSxlQUFlLENBQUM7QUFDL0MsU0FBUyxXQUFXLFFBQVEscUJBQXFCLENBQUM7QUFFbEQsNEVBQTRFLEdBQzVFLE9BQU8sTUFBTSxXQUFXLFNBQ2QsT0FBTztJQUNmLFlBQW1CLEdBQWEsQ0FBRTtRQUNoQyxLQUFLLEVBQUUsQ0FBQztRQUNSLE9BQU8sSUFBSSxDQUNSLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxXQUFXLEVBQUUsQ0FBQyxDQUNsQyxTQUFTLENBQUMsbUJBQW1CLENBQUMsQ0FDOUIsV0FBVyxDQUFDLDhDQUE4QyxDQUFDLENBQzNELFNBQVMsRUFBRSxDQUNYLE1BQU0sQ0FBQyxPQUFPLENBQUMsRUFBRSxJQUFhLEdBQUs7WUFDbEMsSUFBSSxDQUFDLEdBQUcsRUFBRTtnQkFDUixHQUFHLEdBQUcsSUFBSSxHQUNOLElBQUksQ0FBQyxlQUFlLEVBQUUsRUFBRSxjQUFjLENBQUMsSUFBSSxDQUFDLEdBQzVDLElBQUksQ0FBQyxlQUFlLEVBQUUsQ0FBQztZQUM3QixDQUFDO1lBQ0QsSUFBSSxDQUFDLEdBQUcsRUFBRTtnQkFDUixNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsZUFBZSxFQUFFLEVBQUUsV0FBVyxFQUFFLEFBQUM7Z0JBQ25ELE1BQU0sSUFBSSxjQUFjLENBQUMsSUFBSSxJQUFJLEVBQUUsRUFBRSxJQUFJLElBQUksRUFBRSxFQUFFO29CQUMvQyxJQUFJLENBQUMsT0FBTyxFQUFFO3VCQUNYLElBQUksQ0FBQyxVQUFVLEVBQUU7aUJBQ3JCLENBQUMsQ0FBQztZQUNMLENBQUM7WUFDRCxNQUFNLEdBQUcsQ0FBQyxZQUFZLEVBQUUsQ0FBQztZQUN6QixHQUFHLENBQUMsUUFBUSxFQUFFLENBQUM7WUFDZixJQUFJLElBQUksQ0FBQyxVQUFVLEVBQUUsRUFBRTtnQkFDckIsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztZQUNmLENBQUM7UUFDSCxDQUFDLENBQUMsQ0FBQztJQUNQO0NBQ0QifQ==