import { StringType } from "./string.ts";
export class CommandType extends StringType {
    complete(_cmd, parent) {
        return parent?.getCommands(false)
            .map((cmd) => cmd.getName()) || [];
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29tbWFuZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImNvbW1hbmQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLFVBQVUsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUd6QyxNQUFNLE9BQU8sV0FBWSxTQUFRLFVBQVU7SUFFbEMsUUFBUSxDQUFDLElBQWEsRUFBRSxNQUFnQjtRQUM3QyxPQUFPLE1BQU0sRUFBRSxXQUFXLENBQUMsS0FBSyxDQUFDO2FBQzlCLEdBQUcsQ0FBQyxDQUFDLEdBQVksRUFBRSxFQUFFLENBQUMsR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDLElBQUksRUFBRSxDQUFDO0lBQ2hELENBQUM7Q0FDRiJ9