import { InvalidTypeError } from "../_errors.ts";
/** Number type handler. Excepts any numeric value. */ export const integer = (type)=>{
    const value = Number(type.value);
    if (Number.isInteger(value)) {
        return value;
    }
    throw new InvalidTypeError(type);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvZmxhZ3MvdHlwZXMvaW50ZWdlci50cyJdLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgdHlwZSB7IElUeXBlSGFuZGxlciwgSVR5cGVJbmZvIH0gZnJvbSBcIi4uL3R5cGVzLnRzXCI7XG5pbXBvcnQgeyBJbnZhbGlkVHlwZUVycm9yIH0gZnJvbSBcIi4uL19lcnJvcnMudHNcIjtcblxuLyoqIE51bWJlciB0eXBlIGhhbmRsZXIuIEV4Y2VwdHMgYW55IG51bWVyaWMgdmFsdWUuICovXG5leHBvcnQgY29uc3QgaW50ZWdlcjogSVR5cGVIYW5kbGVyPG51bWJlcj4gPSAodHlwZTogSVR5cGVJbmZvKTogbnVtYmVyID0+IHtcbiAgY29uc3QgdmFsdWUgPSBOdW1iZXIodHlwZS52YWx1ZSk7XG4gIGlmIChOdW1iZXIuaXNJbnRlZ2VyKHZhbHVlKSkge1xuICAgIHJldHVybiB2YWx1ZTtcbiAgfVxuXG4gIHRocm93IG5ldyBJbnZhbGlkVHlwZUVycm9yKHR5cGUpO1xufTtcbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxTQUFTLGdCQUFnQixRQUFRLGVBQWUsQ0FBQztBQUVqRCxzREFBc0QsQ0FDdEQsT0FBTyxNQUFNLE9BQU8sR0FBeUIsQ0FBQyxJQUFlLEdBQWE7SUFDeEUsTUFBTSxLQUFLLEdBQUcsTUFBTSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQUFBQztJQUNqQyxJQUFJLE1BQU0sQ0FBQyxTQUFTLENBQUMsS0FBSyxDQUFDLEVBQUU7UUFDM0IsT0FBTyxLQUFLLENBQUM7S0FDZDtJQUVELE1BQU0sSUFBSSxnQkFBZ0IsQ0FBQyxJQUFJLENBQUMsQ0FBQztDQUNsQyxDQUFDIn0=