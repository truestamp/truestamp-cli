import { Command } from "../command.ts";
import { dim, italic } from "../deps.ts";
import { ZshCompletionsGenerator } from "./_zsh_completions_generator.ts";
/** Generates zsh completions script. */ export class ZshCompletionsCommand extends Command {
    #cmd;
    constructor(cmd){
        super();
        this.#cmd = cmd;
        return this.description(()=>{
            const baseCmd = this.#cmd || this.getMainCommand();
            return `Generate shell completions for zsh.

To enable zsh completions for this program add following line to your ${dim(italic("~/.zshrc"))}:

    ${dim(italic(`source <(${baseCmd.getPath()} completions zsh)`))}`;
        }).action(()=>{
            const baseCmd = this.#cmd || this.getMainCommand();
            console.log(ZshCompletionsGenerator.generate(baseCmd));
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvY29tbWFuZC9jb21wbGV0aW9ucy96c2gudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBkaW0sIGl0YWxpYyB9IGZyb20gXCIuLi9kZXBzLnRzXCI7XG5pbXBvcnQgeyBac2hDb21wbGV0aW9uc0dlbmVyYXRvciB9IGZyb20gXCIuL196c2hfY29tcGxldGlvbnNfZ2VuZXJhdG9yLnRzXCI7XG5cbi8qKiBHZW5lcmF0ZXMgenNoIGNvbXBsZXRpb25zIHNjcmlwdC4gKi9cbmV4cG9ydCBjbGFzcyBac2hDb21wbGV0aW9uc0NvbW1hbmQgZXh0ZW5kcyBDb21tYW5kIHtcbiAgI2NtZD86IENvbW1hbmQ7XG4gIHB1YmxpYyBjb25zdHJ1Y3RvcihjbWQ/OiBDb21tYW5kKSB7XG4gICAgc3VwZXIoKTtcbiAgICB0aGlzLiNjbWQgPSBjbWQ7XG4gICAgcmV0dXJuIHRoaXNcbiAgICAgIC5kZXNjcmlwdGlvbigoKSA9PiB7XG4gICAgICAgIGNvbnN0IGJhc2VDbWQgPSB0aGlzLiNjbWQgfHwgdGhpcy5nZXRNYWluQ29tbWFuZCgpO1xuICAgICAgICByZXR1cm4gYEdlbmVyYXRlIHNoZWxsIGNvbXBsZXRpb25zIGZvciB6c2guXG5cblRvIGVuYWJsZSB6c2ggY29tcGxldGlvbnMgZm9yIHRoaXMgcHJvZ3JhbSBhZGQgZm9sbG93aW5nIGxpbmUgdG8geW91ciAke1xuICAgICAgICAgIGRpbShpdGFsaWMoXCJ+Ly56c2hyY1wiKSlcbiAgICAgICAgfTpcblxuICAgICR7ZGltKGl0YWxpYyhgc291cmNlIDwoJHtiYXNlQ21kLmdldFBhdGgoKX0gY29tcGxldGlvbnMgenNoKWApKX1gO1xuICAgICAgfSlcbiAgICAgIC5hY3Rpb24oKCkgPT4ge1xuICAgICAgICBjb25zdCBiYXNlQ21kID0gdGhpcy4jY21kIHx8IHRoaXMuZ2V0TWFpbkNvbW1hbmQoKTtcbiAgICAgICAgY29uc29sZS5sb2coWnNoQ29tcGxldGlvbnNHZW5lcmF0b3IuZ2VuZXJhdGUoYmFzZUNtZCkpO1xuICAgICAgfSk7XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLE9BQU8sUUFBUSxlQUFlLENBQUM7QUFDeEMsU0FBUyxHQUFHLEVBQUUsTUFBTSxRQUFRLFlBQVksQ0FBQztBQUN6QyxTQUFTLHVCQUF1QixRQUFRLGlDQUFpQyxDQUFDO0FBRTFFLHdDQUF3QyxDQUN4QyxPQUFPLE1BQU0scUJBQXFCLFNBQVMsT0FBTztJQUNoRCxDQUFDLEdBQUcsQ0FBVztJQUNmLFlBQW1CLEdBQWEsQ0FBRTtRQUNoQyxLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUM7UUFDaEIsT0FBTyxJQUFJLENBQ1IsV0FBVyxDQUFDLElBQU07WUFDakIsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLENBQUMsR0FBRyxJQUFJLElBQUksQ0FBQyxjQUFjLEVBQUUsQUFBQztZQUNuRCxPQUFPLENBQUM7O3NFQUVzRCxFQUM1RCxHQUFHLENBQUMsTUFBTSxDQUFDLFVBQVUsQ0FBQyxDQUFDLENBQ3hCOztJQUVMLEVBQUUsR0FBRyxDQUFDLE1BQU0sQ0FBQyxDQUFDLFNBQVMsRUFBRSxPQUFPLENBQUMsT0FBTyxFQUFFLENBQUMsaUJBQWlCLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO1NBQy9ELENBQUMsQ0FDRCxNQUFNLENBQUMsSUFBTTtZQUNaLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxDQUFDLEdBQUcsSUFBSSxJQUFJLENBQUMsY0FBYyxFQUFFLEFBQUM7WUFDbkQsT0FBTyxDQUFDLEdBQUcsQ0FBQyx1QkFBdUIsQ0FBQyxRQUFRLENBQUMsT0FBTyxDQUFDLENBQUMsQ0FBQztTQUN4RCxDQUFDLENBQUM7S0FDTjtDQUNGIn0=