import { Type } from "../type.ts";
import { InvalidTypeError } from "../../flags/_errors.ts";
export class EnumType extends Type {
    allowedValues;
    constructor(values) {
        super();
        this.allowedValues = values;
    }
    parse(type) {
        for (const value of this.allowedValues) {
            if (value.toString() === type.value) {
                return value;
            }
        }
        throw new InvalidTypeError(type, this.allowedValues.slice());
    }
    values() {
        return this.allowedValues.slice();
    }
    complete() {
        return this.values();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW51bS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImVudW0udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLElBQUksRUFBRSxNQUFNLFlBQVksQ0FBQztBQUVsQyxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsTUFBTSx3QkFBd0IsQ0FBQztBQUcxRCxNQUFNLE9BQU8sUUFBb0MsU0FBUSxJQUFPO0lBQzdDLGFBQWEsQ0FBbUI7SUFFakQsWUFBWSxNQUF3QjtRQUNsQyxLQUFLLEVBQUUsQ0FBQztRQUNSLElBQUksQ0FBQyxhQUFhLEdBQUcsTUFBTSxDQUFDO0lBQzlCLENBQUM7SUFFTSxLQUFLLENBQUMsSUFBZTtRQUMxQixLQUFLLE1BQU0sS0FBSyxJQUFJLElBQUksQ0FBQyxhQUFhLEVBQUU7WUFDdEMsSUFBSSxLQUFLLENBQUMsUUFBUSxFQUFFLEtBQUssSUFBSSxDQUFDLEtBQUssRUFBRTtnQkFDbkMsT0FBTyxLQUFLLENBQUM7YUFDZDtTQUNGO1FBRUQsTUFBTSxJQUFJLGdCQUFnQixDQUFDLElBQUksRUFBRSxJQUFJLENBQUMsYUFBYSxDQUFDLEtBQUssRUFBRSxDQUFDLENBQUM7SUFDL0QsQ0FBQztJQUVNLE1BQU07UUFDWCxPQUFPLElBQUksQ0FBQyxhQUFhLENBQUMsS0FBSyxFQUFFLENBQUM7SUFDcEMsQ0FBQztJQUVNLFFBQVE7UUFDYixPQUFPLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQztJQUN2QixDQUFDO0NBQ0YifQ==