/**
 * Base class for custom types.
 *
 * **Custom type example:**
 * ```
 * export class ColorType extends Type<string> {
 *   public parse({ label, name, value, type }: ITypeInfo): string {
 *     if (["red", "blue"].includes(value)) {
 *       trow new Error(
 *         `${label} "${name}" must be of type "${type}", but got "${value}".` +
 *         "Valid colors are: red, blue"
 *       );
 *     }
 *     return value;
 *   }
 *
 *   public complete(): string[] {
 *     return ["red", "blue"];
 *   }
 * }
 * ```
 */ export class Type {
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvY29tbWFuZC90eXBlLnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB0eXBlIHsgQ29tbWFuZCB9IGZyb20gXCIuL2NvbW1hbmQudHNcIjtcbmltcG9ydCB0eXBlIHtcbiAgQ29tcGxldGVIYW5kbGVyUmVzdWx0LFxuICBJVHlwZUluZm8sXG4gIFZhbHVlc0hhbmRsZXJSZXN1bHQsXG59IGZyb20gXCIuL3R5cGVzLnRzXCI7XG5cbi8qKlxuICogQmFzZSBjbGFzcyBmb3IgY3VzdG9tIHR5cGVzLlxuICpcbiAqICoqQ3VzdG9tIHR5cGUgZXhhbXBsZToqKlxuICogYGBgXG4gKiBleHBvcnQgY2xhc3MgQ29sb3JUeXBlIGV4dGVuZHMgVHlwZTxzdHJpbmc+IHtcbiAqICAgcHVibGljIHBhcnNlKHsgbGFiZWwsIG5hbWUsIHZhbHVlLCB0eXBlIH06IElUeXBlSW5mbyk6IHN0cmluZyB7XG4gKiAgICAgaWYgKFtcInJlZFwiLCBcImJsdWVcIl0uaW5jbHVkZXModmFsdWUpKSB7XG4gKiAgICAgICB0cm93IG5ldyBFcnJvcihcbiAqICAgICAgICAgYCR7bGFiZWx9IFwiJHtuYW1lfVwiIG11c3QgYmUgb2YgdHlwZSBcIiR7dHlwZX1cIiwgYnV0IGdvdCBcIiR7dmFsdWV9XCIuYCArXG4gKiAgICAgICAgIFwiVmFsaWQgY29sb3JzIGFyZTogcmVkLCBibHVlXCJcbiAqICAgICAgICk7XG4gKiAgICAgfVxuICogICAgIHJldHVybiB2YWx1ZTtcbiAqICAgfVxuICpcbiAqICAgcHVibGljIGNvbXBsZXRlKCk6IHN0cmluZ1tdIHtcbiAqICAgICByZXR1cm4gW1wicmVkXCIsIFwiYmx1ZVwiXTtcbiAqICAgfVxuICogfVxuICogYGBgXG4gKi9cbmV4cG9ydCBhYnN0cmFjdCBjbGFzcyBUeXBlPFQ+IHtcbiAgcHVibGljIGFic3RyYWN0IHBhcnNlKHR5cGU6IElUeXBlSW5mbyk6IFQ7XG5cbiAgLyoqXG4gICAqIFJldHVybnMgdmFsdWVzIGRpc3BsYXllZCBpbiBoZWxwIHRleHQuIElmIG5vIGNvbXBsZXRlIG1ldGhvZCBpcyBwcm92aWRlZCxcbiAgICogdGhlc2UgdmFsdWVzIGFyZSBhbHNvIHVzZWQgZm9yIHNoZWxsIGNvbXBsZXRpb25zLlxuICAgKi9cbiAgcHVibGljIHZhbHVlcz8oXG4gICAgY21kOiBDb21tYW5kLFxuICAgIHBhcmVudD86IENvbW1hbmQsXG4gICk6IFZhbHVlc0hhbmRsZXJSZXN1bHQ7XG5cbiAgLyoqXG4gICAqIFJldHVybnMgc2hlbGwgY29tcGxldGlvbiB2YWx1ZXMuIElmIG5vIGNvbXBsZXRlIG1ldGhvZCBpcyBwcm92aWRlZCxcbiAgICogdmFsdWVzIGZyb20gdGhlIHZhbHVlcyBtZXRob2QgYXJlIHVzZWQuXG4gICAqL1xuICBwdWJsaWMgY29tcGxldGU/KFxuICAgIGNtZDogQ29tbWFuZCxcbiAgICBwYXJlbnQ/OiBDb21tYW5kLFxuICApOiBDb21wbGV0ZUhhbmRsZXJSZXN1bHQ7XG59XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBT0E7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7OztHQXFCRyxDQUNILE9BQU8sTUFBZSxJQUFJO0NBb0J6QiJ9