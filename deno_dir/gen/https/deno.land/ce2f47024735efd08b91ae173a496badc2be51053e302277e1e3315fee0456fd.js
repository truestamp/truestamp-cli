export const StringWrapper = function () {
    const Class = Object.getPrototypeOf(this).constructor;
    const Constructor = Function.bind.apply(String, [null, ...arguments]);
    const instance = new Constructor();
    Object.setPrototypeOf(instance, Class.prototype);
    return instance;
};
StringWrapper.prototype = Object.create(String.prototype, {
    constructor: {
        value: StringWrapper,
        enumerable: false,
        writable: true,
        configurable: true,
    },
});
Object.setPrototypeOf(StringWrapper, String);
export class LazyJsonString extends StringWrapper {
    deserializeJSON() {
        return JSON.parse(super.toString());
    }
    toJSON() {
        return super.toString();
    }
    static fromObject(object) {
        if (object instanceof LazyJsonString) {
            return object;
        }
        else if (object instanceof String || typeof object === "string") {
            return new LazyJsonString(object);
        }
        return new LazyJsonString(JSON.stringify(object));
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibGF6eS1qc29uLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibGF6eS1qc29uLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWVBLE1BQU0sQ0FBQyxNQUFNLGFBQWEsR0FBa0I7SUFFMUMsTUFBTSxLQUFLLEdBQUcsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLENBQUMsQ0FBQyxXQUFXLENBQUM7SUFDdEQsTUFBTSxXQUFXLEdBQUcsUUFBUSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsTUFBTSxFQUFFLENBQUMsSUFBVyxFQUFFLEdBQUcsU0FBUyxDQUFDLENBQUMsQ0FBQztJQUU3RSxNQUFNLFFBQVEsR0FBRyxJQUFJLFdBQVcsRUFBRSxDQUFDO0lBQ25DLE1BQU0sQ0FBQyxjQUFjLENBQUMsUUFBUSxFQUFFLEtBQUssQ0FBQyxTQUFTLENBQUMsQ0FBQztJQUNqRCxPQUFPLFFBQWtCLENBQUM7QUFDNUIsQ0FBQyxDQUFDO0FBQ0YsYUFBYSxDQUFDLFNBQVMsR0FBRyxNQUFNLENBQUMsTUFBTSxDQUFDLE1BQU0sQ0FBQyxTQUFTLEVBQUU7SUFDeEQsV0FBVyxFQUFFO1FBQ1gsS0FBSyxFQUFFLGFBQWE7UUFDcEIsVUFBVSxFQUFFLEtBQUs7UUFDakIsUUFBUSxFQUFFLElBQUk7UUFDZCxZQUFZLEVBQUUsSUFBSTtLQUNuQjtDQUNGLENBQUMsQ0FBQztBQUNILE1BQU0sQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLE1BQU0sQ0FBQyxDQUFDO0FBRTdDLE1BQU0sT0FBTyxjQUFlLFNBQVEsYUFBYTtJQUMvQyxlQUFlO1FBQ2IsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLEtBQUssQ0FBQyxRQUFRLEVBQUUsQ0FBQyxDQUFDO0lBQ3RDLENBQUM7SUFFRCxNQUFNO1FBQ0osT0FBTyxLQUFLLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDMUIsQ0FBQztJQUVELE1BQU0sQ0FBQyxVQUFVLENBQUMsTUFBVztRQUMzQixJQUFJLE1BQU0sWUFBWSxjQUFjLEVBQUU7WUFDcEMsT0FBTyxNQUFNLENBQUM7U0FDZjthQUFNLElBQUksTUFBTSxZQUFZLE1BQU0sSUFBSSxPQUFPLE1BQU0sS0FBSyxRQUFRLEVBQUU7WUFDakUsT0FBTyxJQUFJLGNBQWMsQ0FBQyxNQUFNLENBQUMsQ0FBQztTQUNuQztRQUNELE9BQU8sSUFBSSxjQUFjLENBQUMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO0lBQ3BELENBQUM7Q0FDRiJ9