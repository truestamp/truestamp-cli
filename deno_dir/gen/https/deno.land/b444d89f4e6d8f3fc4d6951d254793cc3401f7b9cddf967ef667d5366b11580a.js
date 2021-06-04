import { StringType } from "./string.ts";
export class ActionListType extends StringType {
    cmd;
    constructor(cmd) {
        super();
        this.cmd = cmd;
    }
    complete() {
        return this.cmd.getCompletions()
            .map((type) => type.name)
            .filter((value, index, self) => self.indexOf(value) === index);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYWN0aW9uX2xpc3QuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJhY3Rpb25fbGlzdC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsVUFBVSxFQUFFLE1BQU0sYUFBYSxDQUFDO0FBR3pDLE1BQU0sT0FBTyxjQUFlLFNBQVEsVUFBVTtJQUN0QjtJQUF0QixZQUFzQixHQUFZO1FBQ2hDLEtBQUssRUFBRSxDQUFDO1FBRFksUUFBRyxHQUFILEdBQUcsQ0FBUztJQUVsQyxDQUFDO0lBR00sUUFBUTtRQUNiLE9BQU8sSUFBSSxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUU7YUFDN0IsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDO2FBRXhCLE1BQU0sQ0FBQyxDQUFDLEtBQUssRUFBRSxLQUFLLEVBQUUsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxLQUFLLEtBQUssQ0FBQyxDQUFDO0lBQ25FLENBQUM7Q0FDRiJ9