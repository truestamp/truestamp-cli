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
            Deno.stdout.writeSync(new TextEncoder().encode(ZshCompletionsGenerator.generate(baseCmd)));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoienNoLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsienNoLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDekMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFHMUUsTUFBTSxPQUFPLHFCQUFzQixTQUFRLE9BQWE7SUFDdEQsSUFBSSxDQUFXO0lBQ2YsWUFBbUIsR0FBYTtRQUM5QixLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxJQUFJLEdBQUcsR0FBRyxDQUFDO1FBQ2hCLElBQUksQ0FBQyxXQUFXLENBQUMsR0FBRyxFQUFFO1lBQ3BCLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLGNBQWMsRUFBRSxDQUFDO1lBQ25ELE9BQU87O3dFQUdMLEdBQUcsQ0FBQyxNQUFNLENBQUMsVUFBVSxDQUFDLENBQ3hCOztNQUVBLEdBQUcsQ0FBQyxNQUFNLENBQUMsWUFBWSxPQUFPLENBQUMsT0FBTyxFQUFFLG1CQUFtQixDQUFDLENBQUMsRUFBRSxDQUFDO1FBQ2xFLENBQUMsQ0FBQzthQUNDLE1BQU0sQ0FBQyxHQUFHLEVBQUU7WUFDWCxNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxJQUFJLENBQUMsTUFBTSxDQUFDLFNBQVMsQ0FBQyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FDNUMsdUJBQXVCLENBQUMsUUFBUSxDQUFDLE9BQU8sQ0FBQyxDQUMxQyxDQUFDLENBQUM7UUFDTCxDQUFDLENBQUMsQ0FBQztJQUNQLENBQUM7Q0FDRiJ9