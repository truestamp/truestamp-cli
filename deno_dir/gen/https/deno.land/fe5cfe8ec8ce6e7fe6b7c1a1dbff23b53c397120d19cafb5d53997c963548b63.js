import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { FishCompletionsGenerator } from "./_fish_completions_generator.ts";
export class FishCompletionsCommand extends Command {
    #cmd;
    constructor(cmd) {
        super();
        this.#cmd = cmd;
        return this
            .description(() => {
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmlzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImZpc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4QyxPQUFPLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RSxNQUFNLE9BQU8sc0JBQXVCLFNBQVEsT0FBTztJQUNqRCxJQUFJLENBQVc7SUFDZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsT0FBTyxJQUFJO2FBQ1IsV0FBVyxDQUFDLEdBQUcsRUFBRTtZQUNoQixNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxPQUFPOzt5RUFHTCxHQUFHLENBQUMsTUFBTSxDQUFDLDRCQUE0QixDQUFDLENBQzFDOztNQUVGLEdBQUcsQ0FBQyxNQUFNLENBQUMsV0FBVyxPQUFPLENBQUMsT0FBTyxFQUFFLDJCQUEyQixDQUFDLENBQUMsRUFBRSxDQUFDO1FBQ3ZFLENBQUMsQ0FBQzthQUNELE1BQU0sQ0FBQyxHQUFHLEVBQUU7WUFDWCxNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQ0FBQztZQUNuRCxPQUFPLENBQUMsR0FBRyxDQUFDLHdCQUF3QixDQUFDLFFBQVEsQ0FBQyxPQUFPLENBQUMsQ0FBQyxDQUFDO1FBQzFELENBQUMsQ0FBQyxDQUFDO0lBQ1AsQ0FBQztDQUNGIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBkaW0sIGl0YWxpYyB9IGZyb20gXCIuLi9kZXBzLnRzXCI7XG5pbXBvcnQgeyBGaXNoQ29tcGxldGlvbnNHZW5lcmF0b3IgfSBmcm9tIFwiLi9fZmlzaF9jb21wbGV0aW9uc19nZW5lcmF0b3IudHNcIjtcblxuLyoqIEdlbmVyYXRlcyBmaXNoIGNvbXBsZXRpb25zIHNjcmlwdC4gKi9cbmV4cG9ydCBjbGFzcyBGaXNoQ29tcGxldGlvbnNDb21tYW5kIGV4dGVuZHMgQ29tbWFuZCB7XG4gICNjbWQ/OiBDb21tYW5kO1xuICBwdWJsaWMgY29uc3RydWN0b3IoY21kPzogQ29tbWFuZCkge1xuICAgIHN1cGVyKCk7XG4gICAgdGhpcy4jY21kID0gY21kO1xuICAgIHJldHVybiB0aGlzXG4gICAgICAuZGVzY3JpcHRpb24oKCkgPT4ge1xuICAgICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgICAgcmV0dXJuIGBHZW5lcmF0ZSBzaGVsbCBjb21wbGV0aW9ucyBmb3IgZmlzaC5cblxuVG8gZW5hYmxlIGZpc2ggY29tcGxldGlvbnMgZm9yIHRoaXMgcHJvZ3JhbSBhZGQgZm9sbG93aW5nIGxpbmUgdG8geW91ciAke1xuICAgICAgICAgIGRpbShpdGFsaWMoXCJ+Ly5jb25maWcvZmlzaC9jb25maWcuZmlzaFwiKSlcbiAgICAgICAgfTpcblxuICAgICR7ZGltKGl0YWxpYyhgc291cmNlICgke2Jhc2VDbWQuZ2V0UGF0aCgpfSBjb21wbGV0aW9ucyBmaXNoIHwgcHN1YilgKSl9YDtcbiAgICAgIH0pXG4gICAgICAuYWN0aW9uKCgpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZUNtZCA9IHRoaXMuI2NtZCB8fCB0aGlzLmdldE1haW5Db21tYW5kKCk7XG4gICAgICAgIGNvbnNvbGUubG9nKEZpc2hDb21wbGV0aW9uc0dlbmVyYXRvci5nZW5lcmF0ZShiYXNlQ21kKSk7XG4gICAgICB9KTtcbiAgfVxufVxuIl19