import { InvalidTypeError } from "../_errors.ts";
/** Boolean type handler. Excepts `true`, `false`, `1`, `0` */ export const boolean = (type)=>{
    if (~[
        "1",
        "true"
    ].indexOf(type.value)) {
        return true;
    }
    if (~[
        "0",
        "false"
    ].indexOf(type.value)) {
        return false;
    }
    throw new InvalidTypeError(type);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvZmxhZ3MvdHlwZXMvYm9vbGVhbi50cyJdLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgdHlwZSB7IElUeXBlSGFuZGxlciwgSVR5cGVJbmZvIH0gZnJvbSBcIi4uL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBJbnZhbGlkVHlwZUVycm9yIH0gZnJvbSBcIi4uL19lcnJvcnMudHNcIjtcblxuLyoqIEJvb2xlYW4gdHlwZSBoYW5kbGVyLiBFeGNlcHRzIGB0cnVlYCwgYGZhbHNlYCwgYDFgLCBgMGAgKi9cbmV4cG9ydCBjb25zdCBib29sZWFuOiBJVHlwZUhhbmRsZXI8Ym9vbGVhbj4gPSAoXG4gIHR5cGU6IElUeXBlSW5mbyxcbik6IGJvb2xlYW4gPT4ge1xuICBpZiAofltcIjFcIiwgXCJ0cnVlXCJdLmluZGV4T2YodHlwZS52YWx1ZSkpIHtcbiAgICByZXR1cm4gdHJ1ZTtcbiAgfVxuXG4gIGlmICh+W1wiMFwiLCBcImZhbHNlXCJdLmluZGV4T2YodHlwZS52YWx1ZSkpIHtcbiAgICByZXR1cm4gZmFsc2U7XG4gIH1cblxuICB0aHJvdyBuZXcgSW52YWxpZFR5cGVFcnJvcih0eXBlKTtcbn07XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsU0FBUyxnQkFBZ0IsUUFBUSxlQUFlLENBQUM7QUFFakQsOERBQThELENBQzlELE9BQU8sTUFBTSxPQUFPLEdBQTBCLENBQzVDLElBQWUsR0FDSDtJQUNaLElBQUksQ0FBQztRQUFDLEdBQUc7UUFBRSxNQUFNO0tBQUMsQ0FBQyxPQUFPLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxFQUFFO1FBQ3RDLE9BQU8sSUFBSSxDQUFDO0tBQ2I7SUFFRCxJQUFJLENBQUM7UUFBQyxHQUFHO1FBQUUsT0FBTztLQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsRUFBRTtRQUN2QyxPQUFPLEtBQUssQ0FBQztLQUNkO0lBRUQsTUFBTSxJQUFJLGdCQUFnQixDQUFDLElBQUksQ0FBQyxDQUFDO0NBQ2xDLENBQUMifQ==