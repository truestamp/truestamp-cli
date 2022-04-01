import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { FishCompletionsGenerator } from "./_fish_completions_generator.ts";
export class FishCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        this.description(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions for fish.

To enable fish completions for this program add following line to your ${dim(italic("~/.config/fish/config.fish"))}:

    ${dim(italic(`source (${baseCmd.getPath()} completions fish | psub)`))}`;
        })
            .action(() => {
            const baseCmd = this.#cmd || this.getMainCommand();
            console.log(FishCompletionsGenerator.generate(baseCmd));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmlzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImZpc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4QyxPQUFPLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RSxNQUFNLE9BQU8sc0JBQXVCLFNBQVEsT0FBYTtJQUN2RCxJQUFJLENBQVc7SUFDZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDcEIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTzs7eUVBR0wsR0FBRyxDQUFDLE1BQU0sQ0FBQyw0QkFBNEIsQ0FBQyxDQUMxQzs7TUFFQSxHQUFHLENBQUMsTUFBTSxDQUFDLFdBQVcsT0FBTyxDQUFDLE9BQU8sRUFBRSwyQkFBMkIsQ0FBQyxDQUFDLEVBQUUsQ0FBQztRQUN6RSxDQUFDLENBQUM7YUFDQyxNQUFNLENBQUMsR0FBRyxFQUFFO1lBQ1gsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTyxDQUFDLEdBQUcsQ0FBQyx3QkFBd0IsQ0FBQyxRQUFRLENBQUMsT0FBTyxDQUFDLENBQUMsQ0FBQztRQUMxRCxDQUFDLENBQUMsQ0FBQztJQUNQLENBQUM7Q0FDRiIsInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENvbW1hbmQgfSBmcm9tIFwiLi4vY29tbWFuZC50c1wiO1xuaW1wb3J0IHsgZGltLCBpdGFsaWMgfSBmcm9tIFwiLi4vZGVwcy50c1wiO1xuaW1wb3J0IHsgRmlzaENvbXBsZXRpb25zR2VuZXJhdG9yIH0gZnJvbSBcIi4vX2Zpc2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgZmlzaCBjb21wbGV0aW9ucyBzY3JpcHQuICovXG5leHBvcnQgY2xhc3MgRmlzaENvbXBsZXRpb25zQ29tbWFuZCBleHRlbmRzIENvbW1hbmQ8dm9pZD4ge1xuICAjY21kPzogQ29tbWFuZDtcbiAgcHVibGljIGNvbnN0cnVjdG9yKGNtZD86IENvbW1hbmQpIHtcbiAgICBzdXBlcigpO1xuICAgIHRoaXMuI2NtZCA9IGNtZDtcbiAgICB0aGlzLmRlc2NyaXB0aW9uKCgpID0+IHtcbiAgICAgIGNvbnN0IGJhc2VDbWQgPSB0aGlzLiNjbWQgfHwgdGhpcy5nZXRNYWluQ29tbWFuZCgpO1xuICAgICAgcmV0dXJuIGBHZW5lcmF0ZSBzaGVsbCBjb21wbGV0aW9ucyBmb3IgZmlzaC5cblxuVG8gZW5hYmxlIGZpc2ggY29tcGxldGlvbnMgZm9yIHRoaXMgcHJvZ3JhbSBhZGQgZm9sbG93aW5nIGxpbmUgdG8geW91ciAke1xuICAgICAgICBkaW0oaXRhbGljKFwifi8uY29uZmlnL2Zpc2gvY29uZmlnLmZpc2hcIikpXG4gICAgICB9OlxuXG4gICAgJHtkaW0oaXRhbGljKGBzb3VyY2UgKCR7YmFzZUNtZC5nZXRQYXRoKCl9IGNvbXBsZXRpb25zIGZpc2ggfCBwc3ViKWApKX1gO1xuICAgIH0pXG4gICAgICAuYWN0aW9uKCgpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZUNtZCA9IHRoaXMuI2NtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgICAgIGNvbnNvbGUubG9nKEZpc2hDb21wbGV0aW9uc0dlbmVyYXRvci5nZW5lcmF0ZShiYXNlQ21kKSk7XG4gICAgICB9KTtcbiAgfVxufVxuIl19