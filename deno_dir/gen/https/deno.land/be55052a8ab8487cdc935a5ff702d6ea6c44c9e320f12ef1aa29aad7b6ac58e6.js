import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { BashCompletionsCommand } from "./bash.ts";
import { CompleteCommand } from "./complete.ts";
import { FishCompletionsCommand } from "./fish.ts";
import { ZshCompletionsCommand } from "./zsh.ts";
/** Generates shell completion scripts for various shells. */ export class CompletionsCommand extends Command {
    #cmd;
    constructor(cmd){
        super();
        this.#cmd = cmd;
        return this.description(()=>{
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions.

To enable shell completions for this program add the following line to your ${dim(italic("~/.bashrc"))} or similar:

    ${dim(italic(`source <(${baseCmd.getPath()} completions [shell])`))}

    For more information run ${dim(italic(`${baseCmd.getPath()} completions [shell] --help`))}
`;
        }).noGlobals().action(()=>this.showHelp()).command("bash", new BashCompletionsCommand(this.#cmd)).command("fish", new FishCompletionsCommand(this.#cmd)).command("zsh", new ZshCompletionsCommand(this.#cmd)).command("complete", new CompleteCommand(this.#cmd).hidden()).reset();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC9jb21wbGV0aW9ucy9tb2QudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBkaW0sIGl0YWxpYyB9IGZyb20gXCIuLi9kZXBzLnRzXCI7XG5pbXBvcnQgeyBCYXNoQ29tcGxldGlvbnNDb21tYW5kIH0gZnJvbSBcIi4vYmFzaC50c1wiO1xuaW1wb3J0IHsgQ29tcGxldGVDb21tYW5kIH0gZnJvbSBcIi4vY29tcGxldGUudHNcIjtcbmltcG9ydCB7IEZpc2hDb21wbGV0aW9uc0NvbW1hbmQgfSBmcm9tIFwiLi9maXNoLnRzXCI7XG5pbXBvcnQgeyBac2hDb21wbGV0aW9uc0NvbW1hbmQgfSBmcm9tIFwiLi96c2gudHNcIjtcblxuLyoqIEdlbmVyYXRlcyBzaGVsbCBjb21wbGV0aW9uIHNjcmlwdHMgZm9yIHZhcmlvdXMgc2hlbGxzLiAqL1xuZXhwb3J0IGNsYXNzIENvbXBsZXRpb25zQ29tbWFuZCBleHRlbmRzIENvbW1hbmQge1xuICAjY21kPzogQ29tbWFuZDtcblxuICBwdWJsaWMgY29uc3RydWN0b3IoY21kPzogQ29tbWFuZCkge1xuICAgIHN1cGVyKCk7XG4gICAgdGhpcy4jY21kID0gY21kO1xuICAgIHJldHVybiB0aGlzXG4gICAgICAuZGVzY3JpcHRpb24oKCkgPT4ge1xuICAgICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgICAgcmV0dXJuIGBHZW5lcmF0ZSBzaGVsbCBjb21wbGV0aW9ucy5cblxuVG8gZW5hYmxlIHNoZWxsIGNvbXBsZXRpb25zIGZvciB0aGlzIHByb2dyYW0gYWRkIHRoZSBmb2xsb3dpbmcgbGluZSB0byB5b3VyICR7XG4gICAgICAgICAgZGltKGl0YWxpYyhcIn4vLmJhc2hyY1wiKSlcbiAgICAgICAgfSBvciBzaW1pbGFyOlxuXG4gICAgJHtkaW0oaXRhbGljKGBzb3VyY2UgPCgke2Jhc2VDbWQuZ2V0UGF0aCgpfSBjb21wbGV0aW9ucyBbc2hlbGxdKWApKX1cblxuICAgIEZvciBtb3JlIGluZm9ybWF0aW9uIHJ1biAke1xuICAgICAgICAgIGRpbShpdGFsaWMoYCR7YmFzZUNtZC5nZXRQYXRoKCl9IGNvbXBsZXRpb25zIFtzaGVsbF0gLS1oZWxwYCkpXG4gICAgICAgIH1cbmA7XG4gICAgICB9KVxuICAgICAgLm5vR2xvYmFscygpXG4gICAgICAuYWN0aW9uKCgpID0+IHRoaXMuc2hvd0hlbHAoKSlcbiAgICAgIC5jb21tYW5kKFwiYmFzaFwiLCBuZXcgQmFzaENvbXBsZXRpb25zQ29tbWFuZCh0aGlzLiNjbWQpKVxuICAgICAgLmNvbW1hbmQoXCJmaXNoXCIsIG5ldyBGaXNoQ29tcGxldGlvbnNDb21tYW5kKHRoaXMuI2NtZCkpXG4gICAgICAuY29tbWFuZChcInpzaFwiLCBuZXcgWnNoQ29tcGxldGlvbnNDb21tYW5kKHRoaXMuI2NtZCkpXG4gICAgICAuY29tbWFuZChcImNvbXBsZXRlXCIsIG5ldyBDb21wbGV0ZUNvbW1hbmQodGhpcy4jY21kKS5oaWRkZW4oKSlcbiAgICAgIC5yZXNldCgpO1xuICB9XG59XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsU0FBUyxPQUFPLFFBQVEsZUFBZSxDQUFDO0FBQ3hDLFNBQVMsR0FBRyxFQUFFLE1BQU0sUUFBUSxZQUFZLENBQUM7QUFDekMsU0FBUyxzQkFBc0IsUUFBUSxXQUFXLENBQUM7QUFDbkQsU0FBUyxlQUFlLFFBQVEsZUFBZSxDQUFDO0FBQ2hELFNBQVMsc0JBQXNCLFFBQVEsV0FBVyxDQUFDO0FBQ25ELFNBQVMscUJBQXFCLFFBQVEsVUFBVSxDQUFDO0FBRWpELDJEQUEyRCxHQUMzRCxPQUFPLE1BQU0sa0JBQWtCLFNBQVMsT0FBTztJQUM3QyxDQUFDLEdBQUcsQ0FBVztJQUVmLFlBQW1CLEdBQWEsQ0FBRTtRQUNoQyxLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUM7UUFDaEIsT0FBTyxJQUFJLENBQ1IsV0FBVyxDQUFDLElBQU07WUFDakIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLENBQUMsR0FBRyxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQUFBQztZQUNuRCxPQUFPLENBQUM7OzRFQUU0RCxFQUNsRSxHQUFHLENBQUMsTUFBTSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQ3pCOztJQUVMLEVBQUUsR0FBRyxDQUFDLE1BQU0sQ0FBQyxDQUFDLFNBQVMsRUFBRSxPQUFPLENBQUMsT0FBTyxFQUFFLENBQUMscUJBQXFCLENBQUMsQ0FBQyxDQUFDLENBQUM7OzZCQUUzQyxFQUNuQixHQUFHLENBQUMsTUFBTSxDQUFDLENBQUMsRUFBRSxPQUFPLENBQUMsT0FBTyxFQUFFLENBQUMsMkJBQTJCLENBQUMsQ0FBQyxDQUFDLENBQy9EO0FBQ1QsQ0FBQyxDQUFDO1FBQ0ksQ0FBQyxDQUFDLENBQ0QsU0FBUyxFQUFFLENBQ1gsTUFBTSxDQUFDLElBQU0sSUFBSSxDQUFDLFFBQVEsRUFBRSxDQUFDLENBQzdCLE9BQU8sQ0FBQyxNQUFNLEVBQUUsSUFBSSxzQkFBc0IsQ0FBQyxJQUFJLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUN0RCxPQUFPLENBQUMsTUFBTSxFQUFFLElBQUksc0JBQXNCLENBQUMsSUFBSSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FDdEQsT0FBTyxDQUFDLEtBQUssRUFBRSxJQUFJLHFCQUFxQixDQUFDLElBQUksQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQ3BELE9BQU8sQ0FBQyxVQUFVLEVBQUUsSUFBSSxlQUFlLENBQUMsSUFBSSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsQ0FDNUQsS0FBSyxFQUFFLENBQUM7SUFDYjtDQUNEIn0=