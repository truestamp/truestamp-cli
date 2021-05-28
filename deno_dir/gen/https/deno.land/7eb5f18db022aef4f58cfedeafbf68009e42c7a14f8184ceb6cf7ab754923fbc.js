import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { BashCompletionsGenerator } from "./_bash_completions_generator.ts";
export class BashCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        this.description(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions for bash.

To enable bash completions for this program add following line to your ${dim(italic("~/.bashrc"))}:

    ${dim(italic(`source <(${baseCmd.getPath()} completions bash)`))}`;
        })
            .action(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            Deno.stdout.writeSync(new TextEncoder().encode(BashCompletionsGenerator.generate(baseCmd)));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYmFzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImJhc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4QyxPQUFPLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RSxNQUFNLE9BQU8sc0JBQXVCLFNBQVEsT0FBYTtJQUN2RCxJQUFJLENBQVc7SUFDZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDcEIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTzs7eUVBR0wsR0FBRyxDQUFDLE1BQU0sQ0FBQyxXQUFXLENBQUMsQ0FDekI7O01BRUEsR0FBRyxDQUFDLE1BQU0sQ0FBQyxZQUFZLE9BQU8sQ0FBQyxPQUFPLEVBQUUsb0JBQW9CLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFDbkUsQ0FBQyxDQUFDO2FBQ0MsTUFBTSxDQUFDLEdBQUcsRUFBRTtZQUNYLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLGNBQWMsRUFBRSxDQUFDO1lBQ25ELElBQUksQ0FBQyxNQUFNLENBQUMsU0FBUyxDQUFDLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUM1Qyx3QkFBd0IsQ0FBQyxRQUFRLENBQUMsT0FBTyxDQUFDLENBQzNDLENBQUMsQ0FBQztRQUNMLENBQUMsQ0FBQyxDQUFDO0lBQ1AsQ0FBQztDQUNGIn0=