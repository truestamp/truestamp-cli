import { StringType } from "./string.ts";
/** String type with auto completion of child command names. */ export class ChildCommandType extends StringType {
    #cmd;
    constructor(cmd){
        super();
        this.#cmd = cmd;
    }
    /** Complete child command names. */ complete(cmd) {
        return (this.#cmd ?? cmd)?.getCommands(false).map((cmd)=>cmd.getName()) || [];
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvY29tbWFuZC90eXBlcy9jaGlsZF9jb21tYW5kLnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB0eXBlIHsgQ29tbWFuZCB9IGZyb20gXCIuLi9jb21tYW5kLnRzXCI7XG5pbXBvcnQgeyBTdHJpbmdUeXBlIH0gZnJvbSBcIi4vc3RyaW5nLnRzXCI7XG5cbi8qKiBTdHJpbmcgdHlwZSB3aXRoIGF1dG8gY29tcGxldGlvbiBvZiBjaGlsZCBjb21tYW5kIG5hbWVzLiAqL1xuZXhwb3J0IGNsYXNzIENoaWxkQ29tbWFuZFR5cGUgZXh0ZW5kcyBTdHJpbmdUeXBlIHtcbiAgI2NtZD86IENvbW1hbmQ7XG5cbiAgY29uc3RydWN0b3IoY21kPzogQ29tbWFuZCkge1xuICAgIHN1cGVyKCk7XG4gICAgdGhpcy4jY21kID0gY21kO1xuICB9XG5cbiAgLyoqIENvbXBsZXRlIGNoaWxkIGNvbW1hbmQgbmFtZXMuICovXG4gIHB1YmxpYyBjb21wbGV0ZShjbWQ6IENvbW1hbmQpOiBzdHJpbmdbXSB7XG4gICAgcmV0dXJuICh0aGlzLiNjbWQgPz8gY21kKT8uZ2V0Q29tbWFuZHMoZmFsc2UpXG4gICAgICAubWFwKChjbWQ6IENvbW1hbmQpID0+IGNtZC5nZXROYW1lKCkpIHx8IFtdO1xuICB9XG59XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsU0FBUyxVQUFVLFFBQVEsYUFBYSxDQUFDO0FBRXpDLCtEQUErRCxDQUMvRCxPQUFPLE1BQU0sZ0JBQWdCLFNBQVMsVUFBVTtJQUM5QyxDQUFDLEdBQUcsQ0FBVztJQUVmLFlBQVksR0FBYSxDQUFFO1FBQ3pCLEtBQUssRUFBRSxDQUFDO1FBQ1IsSUFBSSxDQUFDLENBQUMsR0FBRyxHQUFHLEdBQUcsQ0FBQztLQUNqQjtJQUVELG9DQUFvQyxDQUNwQyxBQUFPLFFBQVEsQ0FBQyxHQUFZLEVBQVk7UUFDdEMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLEdBQUcsSUFBSSxHQUFHLENBQUMsRUFBRSxXQUFXLENBQUMsS0FBSyxDQUFDLENBQzFDLEdBQUcsQ0FBQyxDQUFDLEdBQVksR0FBSyxHQUFHLENBQUMsT0FBTyxFQUFFLENBQUMsSUFBSSxFQUFFLENBQUM7S0FDL0M7Q0FDRiJ9