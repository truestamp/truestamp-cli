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
            console.log(BashCompletionsGenerator.generate(baseCmd));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYmFzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImJhc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4QyxPQUFPLEVBQUUsR0FBRyxFQUFFLE1BQU0sRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RSxNQUFNLE9BQU8sc0JBQXVCLFNBQVEsT0FBYTtJQUN2RCxJQUFJLENBQVc7SUFDZixZQUFtQixHQUFhO1FBQzlCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7UUFDaEIsSUFBSSxDQUFDLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDcEIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLENBQUM7WUFDbkQsT0FBTzs7eUVBR0wsR0FBRyxDQUFDLE1BQU0sQ0FBQyxXQUFXLENBQUMsQ0FDekI7O01BRUEsR0FBRyxDQUFDLE1BQU0sQ0FBQyxZQUFZLE9BQU8sQ0FBQyxPQUFPLEVBQUUsb0JBQW9CLENBQUMsQ0FBQyxFQUFFLENBQUM7UUFDbkUsQ0FBQyxDQUFDO2FBQ0MsTUFBTSxDQUFDLEdBQUcsRUFBRTtZQUNYLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLGNBQWMsRUFBRSxDQUFDO1lBQ25ELE9BQU8sQ0FBQyxHQUFHLENBQUMsd0JBQXdCLENBQUMsUUFBUSxDQUFDLE9BQU8sQ0FBQyxDQUFDLENBQUM7UUFDMUQsQ0FBQyxDQUFDLENBQUM7SUFDUCxDQUFDO0NBQ0YiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBDb21tYW5kIH0gZnJvbSBcIi4uL2NvbW1hbmQudHNcIjtcbmltcG9ydCB7IGRpbSwgaXRhbGljIH0gZnJvbSBcIi4uL2RlcHMudHNcIjtcbmltcG9ydCB7IEJhc2hDb21wbGV0aW9uc0dlbmVyYXRvciB9IGZyb20gXCIuL19iYXNoX2NvbXBsZXRpb25zX2dlbmVyYXRvci50c1wiO1xuXG4vKiogR2VuZXJhdGVzIGJhc2ggY29tcGxldGlvbnMgc2NyaXB0LiAqL1xuZXhwb3J0IGNsYXNzIEJhc2hDb21wbGV0aW9uc0NvbW1hbmQgZXh0ZW5kcyBDb21tYW5kPHZvaWQ+IHtcbiAgI2NtZD86IENvbW1hbmQ7XG4gIHB1YmxpYyBjb25zdHJ1Y3RvcihjbWQ/OiBDb21tYW5kKSB7XG4gICAgc3VwZXIoKTtcbiAgICB0aGlzLiNjbWQgPSBjbWQ7XG4gICAgdGhpcy5kZXNjcmlwdGlvbigoKSA9PiB7XG4gICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgIHJldHVybiBgR2VuZXJhdGUgc2hlbGwgY29tcGxldGlvbnMgZm9yIGJhc2guXG5cblRvIGVuYWJsZSBiYXNoIGNvbXBsZXRpb25zIGZvciB0aGlzIHByb2dyYW0gYWRkIGZvbGxvd2luZyBsaW5lIHRvIHlvdXIgJHtcbiAgICAgICAgZGltKGl0YWxpYyhcIn4vLmJhc2hyY1wiKSlcbiAgICAgIH06XG5cbiAgICAke2RpbShpdGFsaWMoYHNvdXJjZSA8KCR7YmFzZUNtZC5nZXRQYXRoKCl9IGNvbXBsZXRpb25zIGJhc2gpYCkpfWA7XG4gICAgfSlcbiAgICAgIC5hY3Rpb24oKCkgPT4ge1xuICAgICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgICAgY29uc29sZS5sb2coQmFzaENvbXBsZXRpb25zR2VuZXJhdG9yLmdlbmVyYXRlKGJhc2VDbWQpKTtcbiAgICAgIH0pO1xuICB9XG59XG4iXX0=