import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { BashCompletionsGenerator } from "./_bash_completions_generator.ts";
export class BashCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        return this
            .description(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions for bash.

To enable bash completions for this program add following line to your ${dim(italic("~/.bashrc"))}:

    ${dim(italic(`source <(${baseCmd.getPath()} completions bash)`))}`;
        })
            .action(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            console.log(BashCompletionsGenerator.generate(baseCmd));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYmFzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImJhc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4QyxPQUFPLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RSxNQUFNLE9BQU8sc0JBQXVCLFNBQVEsT0FBTztJQUNqRCxJQUFJLENBQVc7SUFDZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsT0FBTyxJQUFJO2FBQ1IsV0FBVyxDQUFDLEdBQUcsRUFBRTtZQUNoQixNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxPQUFPOzt5RUFHTCxHQUFHLENBQUMsTUFBTSxDQUFDLFdBQVcsQ0FBQyxDQUN6Qjs7TUFFRixHQUFHLENBQUMsTUFBTSxDQUFDLFlBQVksT0FBTyxDQUFDLE9BQU8sRUFBRSxvQkFBb0IsQ0FBQyxDQUFDLEVBQUUsQ0FBQztRQUNqRSxDQUFDLENBQUM7YUFDRCxNQUFNLENBQUMsR0FBRyxFQUFFO1lBQ1gsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTyxDQUFDLEdBQUcsQ0FBQyx3QkFBd0IsQ0FBQyxRQUFRLENBQUMsT0FBTyxDQUFDLENBQUMsQ0FBQztRQUMxRCxDQUFDLENBQUMsQ0FBQztJQUNQLENBQUM7Q0FDRiIsInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENvbW1hbmQgfSBmcm9tIFwiLi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHsgZGltLCBpdGFsaWMgfSBmcm9tIFwiLi4vZGVwcy50c1wiO1xuaW1wb3J0IHsgQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yIH0gZnJvbSBcIi4vX2Jhc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgYmFzaCBjb21wbGV0aW9ucyBzY3JpcHQuICovXG5leHBvcnQgY2xhc3MgQmFzaENvbXBsZXRpb25zQ29tbWFuZCBleHRlbmRzIENvbW1hbmQge1xuICAjY21kPzogQ29tbWFuZDtcbiAgcHVibGljIGNvbnN0cnVjdG9yKGNtZD86IENvbW1hbmQpIHtcbiAgICBzdXBlcigpO1xuICAgIHRoaXMuI2NtZCA9IGNtZDtcbiAgICByZXR1cm4gdGhpc1xuICAgICAgLmRlc2NyaXB0aW9uKCgpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZUNtZCA9IHRoaXMuI2NtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgICAgIHJldHVybiBgR2VuZXJhdGUgc2hlbGwgY29tcGxldGlvbnMgZm9yIGJhc2guXG5cblRvIGVuYWJsZSBiYXNoIGNvbXBsZXRpb25zIGZvciB0aGlzIHByb2dyYW0gYWRkIGZvbGxvd2luZyBsaW5lIHRvIHlvdXIgJHtcbiAgICAgICAgICBkaW0oaXRhbGljKFwifi8uYmFzaHJjXCIpKVxuICAgICAgICB9OlxuXG4gICAgJHtkaW0oaXRhbGljKGBzb3VyY2UgPCgke2Jhc2VDbWQuZ2V0UGF0aCgpfSBjb21wbGV0aW9ucyBiYXNoKWApKX1gO1xuICAgICAgfSlcbiAgICAgIC5hY3Rpb24oKCkgPT4ge1xuICAgICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgICAgY29uc29sZS5sb2coQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yLmdlbmVyYXRlKGJhc2VDbWQpKTtcbiAgICAgIH0pO1xuICB9XG59XG4iXX0=