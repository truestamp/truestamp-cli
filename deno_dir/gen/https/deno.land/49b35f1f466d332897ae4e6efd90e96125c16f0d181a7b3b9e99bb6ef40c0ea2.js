class NodeFalsyValueRejectionError extends Error {
    reason;
    code = "ERR_FALSY_VALUE_REJECTION";
    constructor(reason) {
        super("Promise was rejected with falsy value");
        this.reason = reason;
    }
}
class NodeInvalidArgTypeError extends TypeError {
    code = "ERR_INVALID_ARG_TYPE";
    constructor(argumentName) {
        super(`The ${argumentName} argument must be of type function.`);
    }
}
function callbackify(original) {
    if (typeof original !== "function") {
        throw new NodeInvalidArgTypeError('"original"');
    }
    const callbackified = function (...args) {
        const maybeCb = args.pop();
        if (typeof maybeCb !== "function") {
            throw new NodeInvalidArgTypeError("last");
        }
        const cb = (...args) => {
            maybeCb.apply(this, args);
        };
        original.apply(this, args).then((ret) => {
            queueMicrotask(cb.bind(this, null, ret));
        }, (rej) => {
            rej = rej || new NodeFalsyValueRejectionError(rej);
            queueMicrotask(cb.bind(this, rej));
        });
    };
    const descriptors = Object.getOwnPropertyDescriptors(original);
    if (typeof descriptors.length.value === "number") {
        descriptors.length.value++;
    }
    if (typeof descriptors.name.value === "string") {
        descriptors.name.value += "Callbackified";
    }
    Object.defineProperties(callbackified, descriptors);
    return callbackified;
}
export { callbackify };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxfY2FsbGJhY2tpZnkuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfdXRpbF9jYWxsYmFja2lmeS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUF3QkEsTUFBTSw0QkFBNkIsU0FBUSxLQUFLO0lBQ3ZDLE1BQU0sQ0FBVTtJQUNoQixJQUFJLEdBQUcsMkJBQTJCLENBQUM7SUFDMUMsWUFBWSxNQUFlO1FBQ3pCLEtBQUssQ0FBQyx1Q0FBdUMsQ0FBQyxDQUFDO1FBQy9DLElBQUksQ0FBQyxNQUFNLEdBQUcsTUFBTSxDQUFDO0lBQ3ZCLENBQUM7Q0FDRjtBQUNELE1BQU0sdUJBQXdCLFNBQVEsU0FBUztJQUN0QyxJQUFJLEdBQUcsc0JBQXNCLENBQUM7SUFDckMsWUFBWSxZQUFvQjtRQUM5QixLQUFLLENBQUMsT0FBTyxZQUFZLHFDQUFxQyxDQUFDLENBQUM7SUFDbEUsQ0FBQztDQUNGO0FBa0RELFNBQVMsV0FBVyxDQUFDLFFBQWE7SUFDaEMsSUFBSSxPQUFPLFFBQVEsS0FBSyxVQUFVLEVBQUU7UUFDbEMsTUFBTSxJQUFJLHVCQUF1QixDQUFDLFlBQVksQ0FBQyxDQUFDO0tBQ2pEO0lBRUQsTUFBTSxhQUFhLEdBQUcsVUFBeUIsR0FBRyxJQUFlO1FBQy9ELE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxHQUFHLEVBQUUsQ0FBQztRQUMzQixJQUFJLE9BQU8sT0FBTyxLQUFLLFVBQVUsRUFBRTtZQUNqQyxNQUFNLElBQUksdUJBQXVCLENBQUMsTUFBTSxDQUFDLENBQUM7U0FDM0M7UUFDRCxNQUFNLEVBQUUsR0FBRyxDQUFDLEdBQUcsSUFBZSxFQUFRLEVBQUU7WUFDdEMsT0FBTyxDQUFDLEtBQUssQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLENBQUM7UUFDNUIsQ0FBQyxDQUFDO1FBQ0YsUUFBUSxDQUFDLEtBQUssQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLENBQUMsSUFBSSxDQUM3QixDQUFDLEdBQVksRUFBRSxFQUFFO1lBQ2YsY0FBYyxDQUFDLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxFQUFFLElBQUksRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDO1FBQzNDLENBQUMsRUFDRCxDQUFDLEdBQVksRUFBRSxFQUFFO1lBQ2YsR0FBRyxHQUFHLEdBQUcsSUFBSSxJQUFJLDRCQUE0QixDQUFDLEdBQUcsQ0FBQyxDQUFDO1lBQ25ELGNBQWMsQ0FBQyxFQUFFLENBQUMsSUFBSSxDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDO1FBQ3JDLENBQUMsQ0FDRixDQUFDO0lBQ0osQ0FBQyxDQUFDO0lBRUYsTUFBTSxXQUFXLEdBQUcsTUFBTSxDQUFDLHlCQUF5QixDQUFDLFFBQVEsQ0FBQyxDQUFDO0lBRy9ELElBQUksT0FBTyxXQUFXLENBQUMsTUFBTSxDQUFDLEtBQUssS0FBSyxRQUFRLEVBQUU7UUFDaEQsV0FBVyxDQUFDLE1BQU0sQ0FBQyxLQUFLLEVBQUUsQ0FBQztLQUM1QjtJQUNELElBQUksT0FBTyxXQUFXLENBQUMsSUFBSSxDQUFDLEtBQUssS0FBSyxRQUFRLEVBQUU7UUFDOUMsV0FBVyxDQUFDLElBQUksQ0FBQyxLQUFLLElBQUksZUFBZSxDQUFDO0tBQzNDO0lBQ0QsTUFBTSxDQUFDLGdCQUFnQixDQUFDLGFBQWEsRUFBRSxXQUFXLENBQUMsQ0FBQztJQUNwRCxPQUFPLGFBQWEsQ0FBQztBQUN2QixDQUFDO0FBRUQsT0FBTyxFQUFFLFdBQVcsRUFBRSxDQUFDIn0=