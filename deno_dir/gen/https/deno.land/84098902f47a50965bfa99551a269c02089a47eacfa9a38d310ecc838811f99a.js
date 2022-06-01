import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { ZshCompletionsGenerator } from "./_zsh_completions_generator.ts";
export class ZshCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        return this
            .description(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions for zsh.

To enable zsh completions for this program add following line to your ${dim(italic("~/.zshrc"))}:

    ${dim(italic(`source <(${baseCmd.getPath()} completions zsh)`))}`;
        })
            .action(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            console.log(ZshCompletionsGenerator.generate(baseCmd));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoienNoLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsienNoLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDekMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFHMUUsTUFBTSxPQUFPLHFCQUFzQixTQUFRLE9BQU87SUFDaEQsSUFBSSxDQUFXO0lBQ2YsWUFBbUIsR0FBYTtRQUM5QixLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxJQUFJLEdBQUcsR0FBRyxDQUFDO1FBQ2hCLE9BQU8sSUFBSTthQUNSLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDaEIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTzs7d0VBR0wsR0FBRyxDQUFDLE1BQU0sQ0FBQyxVQUFVLENBQUMsQ0FDeEI7O01BRUYsR0FBRyxDQUFDLE1BQU0sQ0FBQyxZQUFZLE9BQU8sQ0FBQyxPQUFPLEVBQUUsbUJBQW1CLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFDaEUsQ0FBQyxDQUFDO2FBQ0QsTUFBTSxDQUFDLEdBQUcsRUFBRTtZQUNYLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLGNBQWMsRUFBRSxDQUFDO1lBQ25ELE9BQU8sQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsUUFBUSxDQUFDLE9BQU8sQ0FBQyxDQUFDLENBQUM7UUFDekQsQ0FBQyxDQUFDLENBQUM7SUFDUCxDQUFDO0NBQ0YiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB7IGRpbSwgaXRhbGljIH0gZnJvbSBcIi4uL2RlcHMudHNcIjtcbmltcG9ydCB7IFpzaENvbXBsZXRpb25zR2VuZXJhdG9yIH0gZnJvbSBcIi4vX3pzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IudHNcIjtcblxuLyoqIEdlbmVyYXRlcyB6c2ggY29tcGxldGlvbnMgc2NyaXB0LiAqL1xuZXhwb3J0IGNsYXNzIFpzaENvbXBsZXRpb25zQ29tbWFuZCBleHRlbmRzIENvbW1hbmQge1xuICAjY21kPzogQ29tbWFuZDtcbiAgcHVibGljIGNvbnN0cnVjdG9yKGNtZD86IENvbW1hbmQpIHtcbiAgICBzdXBlcigpO1xuICAgIHRoaXMuI2NtZCA9IGNtZDtcbiAgICByZXR1cm4gdGhpc1xuICAgICAgLmRlc2NyaXB0aW9uKCgpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZUNtZCA9IHRoaXMuI2NtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgICAgIHJldHVybiBgR2VuZXJhdGUgc2hlbGwgY29tcGxldGlvbnMgZm9yIHpzaC5cblxuVG8gZW5hYmxlIHpzaCBjb21wbGV0aW9ucyBmb3IgdGhpcyBwcm9ncmFtIGFkZCBmb2xsb3dpbmcgbGluZSB0byB5b3VyICR7XG4gICAgICAgICAgZGltKGl0YWxpYyhcIn4vLnpzaHJjXCIpKVxuICAgICAgICB9OlxuXG4gICAgJHtkaW0oaXRhbGljKGBzb3VyY2UgPCgke2Jhc2VDbWQuZ2V0UGF0aCgpfSBjb21wbGV0aW9ucyB6c2gpYCkpfWA7XG4gICAgICB9KVxuICAgICAgLmFjdGlvbigoKSA9PiB7XG4gICAgICAgIGNvbnN0IGJhc2VDbWQgPSB0aGlzLiNjbWQgfHwgdGhpcy5nZXRNYWluQ29tbWFuZCgpO1xuICAgICAgICBjb25zb2xlLmxvZyhac2hDb21wbGV0aW9uc0dlbmVyYXRvci5nZW5lcmF0ZShiYXNlQ21kKSk7XG4gICAgICB9KTtcbiAgfVxufVxuIl19