import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { BashCompletionsCommand } from "./bash.ts";
import { CompleteCommand } from "./complete.ts";
import { FishCompletionsCommand } from "./fish.ts";
import { ZshCompletionsCommand } from "./zsh.ts";
export class CompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        this.description(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions.

To enable shell completions for this program add the following line to your ${dim(italic("~/.bashrc"))} or similar:

    ${dim(italic(`source <(${baseCmd.getPath()} completions [shell])`))}

    For more information run ${dim(italic(`${baseCmd.getPath()} completions [shell] --help`))}
`;
        })
            .action(() => this.showHelp())
            .command("bash", new BashCompletionsCommand(this.#cmd))
            .command("fish", new FishCompletionsCommand(this.#cmd))
            .command("zsh", new ZshCompletionsCommand(this.#cmd))
            .command("complete", new CompleteCommand(this.#cmd).hidden())
            .reset();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDekMsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBRSxlQUFlLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDaEQsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBRSxxQkFBcUIsRUFBRSxNQUFNLFVBQVUsQ0FBQztBQUdqRCxNQUFNLE9BQU8sa0JBQW1CLFNBQVEsT0FBYTtJQUNuRCxJQUFJLENBQVc7SUFFZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDcEIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTzs7OEVBR0wsR0FBRyxDQUFDLE1BQU0sQ0FBQyxXQUFXLENBQUMsQ0FDekI7O01BRUEsR0FBRyxDQUFDLE1BQU0sQ0FBQyxZQUFZLE9BQU8sQ0FBQyxPQUFPLEVBQUUsdUJBQXVCLENBQUMsQ0FBQzs7K0JBRy9ELEdBQUcsQ0FBQyxNQUFNLENBQUMsR0FBRyxPQUFPLENBQUMsT0FBTyxFQUFFLDZCQUE2QixDQUFDLENBQy9EO0NBQ0wsQ0FBQztRQUNFLENBQUMsQ0FBQzthQUNDLE1BQU0sQ0FBQyxHQUFHLEVBQUUsQ0FBQyxJQUFJLENBQUMsUUFBUSxFQUFFLENBQUM7YUFDN0IsT0FBTyxDQUFDLE1BQU0sRUFBRSxJQUFJLHNCQUFzQixDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQzthQUN0RCxPQUFPLENBQUMsTUFBTSxFQUFFLElBQUksc0JBQXNCLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO2FBQ3RELE9BQU8sQ0FBQyxLQUFLLEVBQUUsSUFBSSxxQkFBcUIsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7YUFDcEQsT0FBTyxDQUFDLFVBQVUsRUFBRSxJQUFJLGVBQWUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsTUFBTSxFQUFFLENBQUM7YUFDNUQsS0FBSyxFQUFFLENBQUM7SUFDYixDQUFDO0NBQ0YifQ==