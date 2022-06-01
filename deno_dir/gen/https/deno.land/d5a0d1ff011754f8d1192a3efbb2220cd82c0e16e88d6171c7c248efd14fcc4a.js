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
        return this
            .description(() => {
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEMsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFDekMsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBRSxlQUFlLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDaEQsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBRSxxQkFBcUIsRUFBRSxNQUFNLFVBQVUsQ0FBQztBQUdqRCxNQUFNLE9BQU8sa0JBQW1CLFNBQVEsT0FBTztJQUM3QyxJQUFJLENBQVc7SUFFZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsT0FBTyxJQUFJO2FBQ1IsV0FBVyxDQUFDLEdBQUcsRUFBRTtZQUNoQixNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxPQUFPOzs4RUFHTCxHQUFHLENBQUMsTUFBTSxDQUFDLFdBQVcsQ0FBQyxDQUN6Qjs7TUFFRixHQUFHLENBQUMsTUFBTSxDQUFDLFlBQVksT0FBTyxDQUFDLE9BQU8sRUFBRSx1QkFBdUIsQ0FBQyxDQUFDOzsrQkFHN0QsR0FBRyxDQUFDLE1BQU0sQ0FBQyxHQUFHLE9BQU8sQ0FBQyxPQUFPLEVBQUUsNkJBQTZCLENBQUMsQ0FDL0Q7Q0FDUCxDQUFDO1FBQ0ksQ0FBQyxDQUFDO2FBQ0QsTUFBTSxDQUFDLEdBQUcsRUFBRSxDQUFDLElBQUksQ0FBQyxRQUFRLEVBQUUsQ0FBQzthQUM3QixPQUFPLENBQUMsTUFBTSxFQUFFLElBQUksc0JBQXNCLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO2FBQ3RELE9BQU8sQ0FBQyxNQUFNLEVBQUUsSUFBSSxzQkFBc0IsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7YUFDdEQsT0FBTyxDQUFDLEtBQUssRUFBRSxJQUFJLHFCQUFxQixDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQzthQUNwRCxPQUFPLENBQUMsVUFBVSxFQUFFLElBQUksZUFBZSxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQzthQUM1RCxLQUFLLEVBQUUsQ0FBQztJQUNiLENBQUM7Q0FDRiIsInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENvbW1hbmQgfSBmcm9tIFwiLi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHsgZGltLCBpdGFsaWMgfSBmcm9tIFwiLi4vZGVwcy50c1wiO1xuaW1wb3J0IHsgQmFzaENvbXBsZXRpb25zQ29tbWFuZCB9IGZyb20gXCIuL2Jhc2gudHNcIjtcbmltcG9ydCB7IENvbXBsZXRlQ29tbWFuZCB9IGZyb20gXCIuL2NvbXBsZXRlLnRzXCI7XG5pbXBvcnQgeyBGaXNoQ29tcGxldGlvbnNDb21tYW5kIH0gZnJvbSBcIi4vZmlzaC50c1wiO1xuaW1wb3J0IHsgWnNoQ29tcGxldGlvbnNDb21tYW5kIH0gZnJvbSBcIi4venNoLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgc2hlbGwgY29tcGxldGlvbiBzY3JpcHRzIGZvciB2YXJpb3VzIHNoZWxscy4gKi9cbmV4cG9ydCBjbGFzcyBDb21wbGV0aW9uc0NvbW1hbmQgZXh0ZW5kcyBDb21tYW5kIHtcbiAgI2NtZD86IENvbW1hbmQ7XG5cbiAgcHVibGljIGNvbnN0cnVjdG9yKGNtZD86IENvbW1hbmQpIHtcbiAgICBzdXBlcigpO1xuICAgIHRoaXMuI2NtZCA9IGNtZDtcbiAgICByZXR1cm4gdGhpc1xuICAgICAgLmRlc2NyaXB0aW9uKCgpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZUNtZCA9IHRoaXMuI2NtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgICAgIHJldHVybiBgR2VuZXJhdGUgc2hlbGwgY29tcGxldGlvbnMuXG5cblRvIGVuYWJsZSBzaGVsbCBjb21wbGV0aW9ucyBmb3IgdGhpcyBwcm9ncmFtIGFkZCB0aGUgZm9sbG93aW5nIGxpbmUgdG8geW91ciAke1xuICAgICAgICAgIGRpbShpdGFsaWMoXCJ+Ly5iYXNocmNcIikpXG4gICAgICAgIH0gb3Igc2ltaWxhcjpcblxuICAgICR7ZGltKGl0YWxpYyhgc291cmNlIDwoJHtiYXNlQ21kLmdldFBhdGgoKX0gY29tcGxldGlvbnMgW3NoZWxsXSlgKSl9XG5cbiAgICBGb3IgbW9yZSBpbmZvcm1hdGlvbiBydW4gJHtcbiAgICAgICAgICBkaW0oaXRhbGljKGAke2Jhc2VDbWQuZ2V0UGF0aCgpfSBjb21wbGV0aW9ucyBbc2hlbGxdIC0taGVscGApKVxuICAgICAgICB9XG5gO1xuICAgICAgfSlcbiAgICAgIC5hY3Rpb24oKCkgPT4gdGhpcy5zaG93SGVscCgpKVxuICAgICAgLmNvbW1hbmQoXCJiYXNoXCIsIG5ldyBCYXNoQ29tcGxldGlvbnNDb21tYW5kKHRoaXMuI2NtZCkpXG4gICAgICAuY29tbWFuZChcImZpc2hcIiwgbmV3IEZpc2hDb21wbGV0aW9uc0NvbW1hbmQodGhpcy4jY21kKSlcbiAgICAgIC5jb21tYW5kKFwienNoXCIsIG5ldyBac2hDb21wbGV0aW9uc0NvbW1hbmQodGhpcy4jY21kKSlcbiAgICAgIC5jb21tYW5kKFwiY29tcGxldGVcIiwgbmV3IENvbXBsZXRlQ29tbWFuZCh0aGlzLiNjbWQpLmhpZGRlbigpKVxuICAgICAgLnJlc2V0KCk7XG4gIH1cbn1cbiJdfQ==