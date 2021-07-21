const kCustomPromisifiedSymbol = Symbol.for("nodejs.util.promisify.custom");
const kCustomPromisifyArgsSymbol = Symbol.for("nodejs.util.promisify.customArgs");
class NodeInvalidArgTypeError extends TypeError {
    code = "ERR_INVALID_ARG_TYPE";
    constructor(argumentName, type, received) {
        super(`The "${argumentName}" argument must be of type ${type}. Received ${typeof received}`);
    }
}
export function promisify(original) {
    if (typeof original !== "function") {
        throw new NodeInvalidArgTypeError("original", "Function", original);
    }
    if (original[kCustomPromisifiedSymbol]) {
        const fn = original[kCustomPromisifiedSymbol];
        if (typeof fn !== "function") {
            throw new NodeInvalidArgTypeError("util.promisify.custom", "Function", fn);
        }
        return Object.defineProperty(fn, kCustomPromisifiedSymbol, {
            value: fn,
            enumerable: false,
            writable: false,
            configurable: true,
        });
    }
    const argumentNames = original[kCustomPromisifyArgsSymbol];
    function fn(...args) {
        return new Promise((resolve, reject) => {
            original.call(this, ...args, (err, ...values) => {
                if (err) {
                    return reject(err);
                }
                if (argumentNames !== undefined && values.length > 1) {
                    const obj = {};
                    for (let i = 0; i < argumentNames.length; i++) {
                        obj[argumentNames[i]] = values[i];
                    }
                    resolve(obj);
                }
                else {
                    resolve(values[0]);
                }
            });
        });
    }
    Object.setPrototypeOf(fn, Object.getPrototypeOf(original));
    Object.defineProperty(fn, kCustomPromisifiedSymbol, {
        value: fn,
        enumerable: false,
        writable: false,
        configurable: true,
    });
    return Object.defineProperties(fn, Object.getOwnPropertyDescriptors(original));
}
promisify.custom = kCustomPromisifiedSymbol;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxfcHJvbWlzaWZ5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxfcHJvbWlzaWZ5LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQTBDQSxNQUFNLHdCQUF3QixHQUFHLE1BQU0sQ0FBQyxHQUFHLENBQUMsOEJBQThCLENBQUMsQ0FBQztBQUc1RSxNQUFNLDBCQUEwQixHQUFHLE1BQU0sQ0FBQyxHQUFHLENBQzNDLGtDQUFrQyxDQUNuQyxDQUFDO0FBRUYsTUFBTSx1QkFBd0IsU0FBUSxTQUFTO0lBQ3RDLElBQUksR0FBRyxzQkFBc0IsQ0FBQztJQUNyQyxZQUFZLFlBQW9CLEVBQUUsSUFBWSxFQUFFLFFBQWlCO1FBQy9ELEtBQUssQ0FDSCxRQUFRLFlBQVksOEJBQThCLElBQUksY0FBYyxPQUFPLFFBQVEsRUFBRSxDQUN0RixDQUFDO0lBQ0osQ0FBQztDQUNGO0FBRUQsTUFBTSxVQUFVLFNBQVMsQ0FFdkIsUUFBa0M7SUFHbEMsSUFBSSxPQUFPLFFBQVEsS0FBSyxVQUFVLEVBQUU7UUFDbEMsTUFBTSxJQUFJLHVCQUF1QixDQUFDLFVBQVUsRUFBRSxVQUFVLEVBQUUsUUFBUSxDQUFDLENBQUM7S0FDckU7SUFFRCxJQUFLLFFBQWdCLENBQUMsd0JBQXdCLENBQUMsRUFBRTtRQUUvQyxNQUFNLEVBQUUsR0FBSSxRQUFnQixDQUFDLHdCQUF3QixDQUFDLENBQUM7UUFDdkQsSUFBSSxPQUFPLEVBQUUsS0FBSyxVQUFVLEVBQUU7WUFDNUIsTUFBTSxJQUFJLHVCQUF1QixDQUMvQix1QkFBdUIsRUFDdkIsVUFBVSxFQUNWLEVBQUUsQ0FDSCxDQUFDO1NBQ0g7UUFDRCxPQUFPLE1BQU0sQ0FBQyxjQUFjLENBQUMsRUFBRSxFQUFFLHdCQUF3QixFQUFFO1lBQ3pELEtBQUssRUFBRSxFQUFFO1lBQ1QsVUFBVSxFQUFFLEtBQUs7WUFDakIsUUFBUSxFQUFFLEtBQUs7WUFDZixZQUFZLEVBQUUsSUFBSTtTQUNuQixDQUFDLENBQUM7S0FDSjtJQUtELE1BQU0sYUFBYSxHQUFJLFFBQWdCLENBQUMsMEJBQTBCLENBQUMsQ0FBQztJQUVwRSxTQUFTLEVBQUUsQ0FBWSxHQUFHLElBQWU7UUFDdkMsT0FBTyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxNQUFNLEVBQUUsRUFBRTtZQUNyQyxRQUFRLENBQUMsSUFBSSxDQUFDLElBQUksRUFBRSxHQUFHLElBQUksRUFBRSxDQUFDLEdBQVUsRUFBRSxHQUFHLE1BQWlCLEVBQUUsRUFBRTtnQkFDaEUsSUFBSSxHQUFHLEVBQUU7b0JBQ1AsT0FBTyxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUM7aUJBQ3BCO2dCQUNELElBQUksYUFBYSxLQUFLLFNBQVMsSUFBSSxNQUFNLENBQUMsTUFBTSxHQUFHLENBQUMsRUFBRTtvQkFDcEQsTUFBTSxHQUFHLEdBQUcsRUFBRSxDQUFDO29CQUNmLEtBQUssSUFBSSxDQUFDLEdBQUcsQ0FBQyxFQUFFLENBQUMsR0FBRyxhQUFhLENBQUMsTUFBTSxFQUFFLENBQUMsRUFBRSxFQUFFO3dCQUU1QyxHQUFXLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQyxDQUFDLEdBQUcsTUFBTSxDQUFDLENBQUMsQ0FBQyxDQUFDO3FCQUM1QztvQkFDRCxPQUFPLENBQUMsR0FBRyxDQUFDLENBQUM7aUJBQ2Q7cUJBQU07b0JBQ0wsT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO2lCQUNwQjtZQUNILENBQUMsQ0FBQyxDQUFDO1FBQ0wsQ0FBQyxDQUFDLENBQUM7SUFDTCxDQUFDO0lBRUQsTUFBTSxDQUFDLGNBQWMsQ0FBQyxFQUFFLEVBQUUsTUFBTSxDQUFDLGNBQWMsQ0FBQyxRQUFRLENBQUMsQ0FBQyxDQUFDO0lBRTNELE1BQU0sQ0FBQyxjQUFjLENBQUMsRUFBRSxFQUFFLHdCQUF3QixFQUFFO1FBQ2xELEtBQUssRUFBRSxFQUFFO1FBQ1QsVUFBVSxFQUFFLEtBQUs7UUFDakIsUUFBUSxFQUFFLEtBQUs7UUFDZixZQUFZLEVBQUUsSUFBSTtLQUNuQixDQUFDLENBQUM7SUFDSCxPQUFPLE1BQU0sQ0FBQyxnQkFBZ0IsQ0FDNUIsRUFBRSxFQUNGLE1BQU0sQ0FBQyx5QkFBeUIsQ0FBQyxRQUFRLENBQUMsQ0FDM0MsQ0FBQztBQUNKLENBQUM7QUFFRCxTQUFTLENBQUMsTUFBTSxHQUFHLHdCQUF3QixDQUFDIn0=