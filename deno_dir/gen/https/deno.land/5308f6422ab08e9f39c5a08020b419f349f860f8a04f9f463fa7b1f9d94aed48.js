import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { ZshCompletionsGenerator } from "./_zsh_completions_generator.ts";
export class ZshCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        this.description(() => {
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoienNoLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsienNoLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDekMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFHMUUsTUFBTSxPQUFPLHFCQUFzQixTQUFRLE9BQWE7SUFDdEQsSUFBSSxDQUFXO0lBQ2YsWUFBbUIsR0FBYTtRQUM5QixLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxJQUFJLEdBQUcsR0FBRyxDQUFDO1FBQ2hCLElBQUksQ0FBQyxXQUFXLENBQUMsR0FBRyxFQUFFO1lBQ3BCLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLGNBQWMsRUFBRSxDQUFDO1lBQ25ELE9BQU87O3dFQUdMLEdBQUcsQ0FBQyxNQUFNLENBQUMsVUFBVSxDQUFDLENBQ3hCOztNQUVBLEdBQUcsQ0FBQyxNQUFNLENBQUMsWUFBWSxPQUFPLENBQUMsT0FBTyxFQUFFLG1CQUFtQixDQUFDLENBQUMsRUFBRSxDQUFDO1FBQ2xFLENBQUMsQ0FBQzthQUNDLE1BQU0sQ0FBQyxHQUFHLEVBQUU7WUFDWCxNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxPQUFPLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLFFBQVEsQ0FBQyxPQUFPLENBQUMsQ0FBQyxDQUFDO1FBQ3pELENBQUMsQ0FBQyxDQUFDO0lBQ1AsQ0FBQztDQUNGIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBkaW0sIGl0YWxpYyB9IGZyb20gXCIuLi9kZXBzLnRzXCI7XG5pbXBvcnQgeyBac2hDb21wbGV0aW9uc0dlbmVyYXRvciB9IGZyb20gXCIuL196c2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgenNoIGNvbXBsZXRpb25zIHNjcmlwdC4gKi9cbmV4cG9ydCBjbGFzcyBac2hDb21wbGV0aW9uc0NvbW1hbmQgZXh0ZW5kcyBDb21tYW5kPHZvaWQ+IHtcbiAgI2NtZD86IENvbW1hbmQ7XG4gIHB1YmxpYyBjb25zdHJ1Y3RvcihjbWQ/OiBDb21tYW5kKSB7XG4gICAgc3VwZXIoKTtcbiAgICB0aGlzLiNjbWQgPSBjbWQ7XG4gICAgdGhpcy5kZXNjcmlwdGlvbigoKSA9PiB7XG4gICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgIHJldHVybiBgR2VuZXJhdGUgc2hlbGwgY29tcGxldGlvbnMgZm9yIHpzaC5cblxuVG8gZW5hYmxlIHpzaCBjb21wbGV0aW9ucyBmb3IgdGhpcyBwcm9ncmFtIGFkZCBmb2xsb3dpbmcgbGluZSB0byB5b3VyICR7XG4gICAgICAgIGRpbShpdGFsaWMoXCJ+Ly56c2hyY1wiKSlcbiAgICAgIH06XG5cbiAgICAke2RpbShpdGFsaWMoYHNvdXJjZSA8KCR7YmFzZUNtZC5nZXRQYXRoKCl9IGNvbXBsZXRpb25zIHpzaClgKSl9YDtcbiAgICB9KVxuICAgICAgLmFjdGlvbigoKSA9PiB7XG4gICAgICAgIGNvbnN0IGJhc2VDbWQgPSB0aGlzLiNjbWQgfHwgdGhpcy5nZXRNYWluQ29tbWFuZCgpO1xuICAgICAgICBjb25zb2xlLmxvZyhac2hDb21wbGV0aW9uc0dlbmVyYXRvci5nZW5lcmF0ZShiYXNlQ21kKSk7XG4gICAgICB9KTtcbiAgfVxufVxuIl19