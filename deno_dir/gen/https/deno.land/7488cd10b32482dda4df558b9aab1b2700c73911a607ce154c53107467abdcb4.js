import { notImplemented } from "../_utils.ts";
export default class Dirent {
    entry;
    constructor(entry) {
        this.entry = entry;
    }
    isBlockDevice() {
        notImplemented("Deno does not yet support identification of block devices");
        return false;
    }
    isCharacterDevice() {
        notImplemented("Deno does not yet support identification of character devices");
        return false;
    }
    isDirectory() {
        return this.entry.isDirectory;
    }
    isFIFO() {
        notImplemented("Deno does not yet support identification of FIFO named pipes");
        return false;
    }
    isFile() {
        return this.entry.isFile;
    }
    isSocket() {
        notImplemented("Deno does not yet support identification of sockets");
        return false;
    }
    isSymbolicLink() {
        return this.entry.isSymlink;
    }
    get name() {
        return this.entry.name;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX2RpcmVudC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9mc19kaXJlbnQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUU5QyxNQUFNLENBQUMsT0FBTyxPQUFPLE1BQU07SUFDTDtJQUFwQixZQUFvQixLQUFvQjtRQUFwQixVQUFLLEdBQUwsS0FBSyxDQUFlO0lBQUcsQ0FBQztJQUU1QyxhQUFhO1FBQ1gsY0FBYyxDQUFDLDJEQUEyRCxDQUFDLENBQUM7UUFDNUUsT0FBTyxLQUFLLENBQUM7SUFDZixDQUFDO0lBRUQsaUJBQWlCO1FBQ2YsY0FBYyxDQUNaLCtEQUErRCxDQUNoRSxDQUFDO1FBQ0YsT0FBTyxLQUFLLENBQUM7SUFDZixDQUFDO0lBRUQsV0FBVztRQUNULE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxXQUFXLENBQUM7SUFDaEMsQ0FBQztJQUVELE1BQU07UUFDSixjQUFjLENBQ1osOERBQThELENBQy9ELENBQUM7UUFDRixPQUFPLEtBQUssQ0FBQztJQUNmLENBQUM7SUFFRCxNQUFNO1FBQ0osT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQztJQUMzQixDQUFDO0lBRUQsUUFBUTtRQUNOLGNBQWMsQ0FBQyxxREFBcUQsQ0FBQyxDQUFDO1FBQ3RFLE9BQU8sS0FBSyxDQUFDO0lBQ2YsQ0FBQztJQUVELGNBQWM7UUFDWixPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDO0lBQzlCLENBQUM7SUFFRCxJQUFJLElBQUk7UUFDTixPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDO0lBQ3pCLENBQUM7Q0FDRiJ9