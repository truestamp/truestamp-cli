import { Type } from "../type.ts";
import { integer } from "../../flags/types/integer.ts";
/** Integer type. */ export class IntegerType extends Type {
    /** Parse integer type. */ parse(type) {
        return integer(type);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvY29tbWFuZC90eXBlcy9pbnRlZ2VyLnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IFR5cGUgfSBmcm9tIFwiLi4vdHlwZS50c1wiO1xuaW1wb3J0IHR5cGUgeyBJVHlwZUluZm8gfSBmcm9tIFwiLi4vdHlwZXMudHNcIjtcbmltcG9ydCB7IGludGVnZXIgfSBmcm9tIFwiLi4vLi4vZmxhZ3MvdHlwZXMvaW50ZWdlci50c1wiO1xuXG4vKiogSW50ZWdlciB0eXBlLiAqL1xuZXhwb3J0IGNsYXNzIEludGVnZXJUeXBlIGV4dGVuZHMgVHlwZTxudW1iZXI+IHtcbiAgLyoqIFBhcnNlIGludGVnZXIgdHlwZS4gKi9cbiAgcHVibGljIHBhcnNlKHR5cGU6IElUeXBlSW5mbyk6IG51bWJlciB7XG4gICAgcmV0dXJuIGludGVnZXIodHlwZSk7XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLElBQUksUUFBUSxZQUFZLENBQUM7QUFFbEMsU0FBUyxPQUFPLFFBQVEsOEJBQThCLENBQUM7QUFFdkQsa0JBQWtCLEdBQ2xCLE9BQU8sTUFBTSxXQUFXLFNBQVMsSUFBSTtJQUNuQyx3QkFBd0IsR0FDakIsS0FBSyxDQUFDLElBQWUsRUFBVTtRQUNwQyxPQUFPLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUN2QjtDQUNEIn0=