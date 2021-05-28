import { boolean } from "../../flags/types/boolean.ts";
import { Type } from "../type.ts";
export class BooleanType extends Type {
    parse(type) {
        return boolean(type);
    }
    complete() {
        return ["true", "false"];
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYm9vbGVhbi5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImJvb2xlYW4udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLE9BQU8sRUFBRSxNQUFNLDhCQUE4QixDQUFDO0FBRXZELE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFHbEMsTUFBTSxPQUFPLFdBQVksU0FBUSxJQUFhO0lBRXJDLEtBQUssQ0FBQyxJQUFlO1FBQzFCLE9BQU8sT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDO0lBQ3ZCLENBQUM7SUFHTSxRQUFRO1FBQ2IsT0FBTyxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUMzQixDQUFDO0NBQ0YifQ==