import { deferred } from "../async/mod.ts";
import { assert, assertStringIncludes, fail } from "../testing/asserts.ts";
import { readAll } from "../io/util.ts";
export function notImplemented(msg) {
    const message = msg ? `Not implemented: ${msg}` : "Not implemented";
    throw new Error(message);
}
export const _TextDecoder = TextDecoder;
export const _TextEncoder = TextEncoder;
export function intoCallbackAPI(func, cb, ...args) {
    func(...args).then((value) => cb && cb(null, value), (err) => cb && cb(err));
}
export function intoCallbackAPIWithIntercept(func, interceptor, cb, ...args) {
    func(...args).then((value) => cb && cb(null, interceptor(value)), (err) => cb && cb(err));
}
export function spliceOne(list, index) {
    for (; index + 1 < list.length; index++)
        list[index] = list[index + 1];
    list.pop();
}
export function normalizeEncoding(enc) {
    if (enc == null || enc === "utf8" || enc === "utf-8")
        return "utf8";
    return slowCases(enc);
}
function slowCases(enc) {
    switch (enc.length) {
        case 4:
            if (enc === "UTF8")
                return "utf8";
            if (enc === "ucs2" || enc === "UCS2")
                return "utf16le";
            enc = `${enc}`.toLowerCase();
            if (enc === "utf8")
                return "utf8";
            if (enc === "ucs2")
                return "utf16le";
            break;
        case 3:
            if (enc === "hex" || enc === "HEX" || `${enc}`.toLowerCase() === "hex") {
                return "hex";
            }
            break;
        case 5:
            if (enc === "ascii")
                return "ascii";
            if (enc === "ucs-2")
                return "utf16le";
            if (enc === "UTF-8")
                return "utf8";
            if (enc === "ASCII")
                return "ascii";
            if (enc === "UCS-2")
                return "utf16le";
            enc = `${enc}`.toLowerCase();
            if (enc === "utf-8")
                return "utf8";
            if (enc === "ascii")
                return "ascii";
            if (enc === "ucs-2")
                return "utf16le";
            break;
        case 6:
            if (enc === "base64")
                return "base64";
            if (enc === "latin1" || enc === "binary")
                return "latin1";
            if (enc === "BASE64")
                return "base64";
            if (enc === "LATIN1" || enc === "BINARY")
                return "latin1";
            enc = `${enc}`.toLowerCase();
            if (enc === "base64")
                return "base64";
            if (enc === "latin1" || enc === "binary")
                return "latin1";
            break;
        case 7:
            if (enc === "utf16le" ||
                enc === "UTF16LE" ||
                `${enc}`.toLowerCase() === "utf16le") {
                return "utf16le";
            }
            break;
        case 8:
            if (enc === "utf-16le" ||
                enc === "UTF-16LE" ||
                `${enc}`.toLowerCase() === "utf-16le") {
                return "utf16le";
            }
            break;
        default:
            if (enc === "")
                return "utf8";
    }
}
export function validateIntegerRange(value, name, min = -2147483648, max = 2147483647) {
    if (!Number.isInteger(value)) {
        throw new Error(`${name} must be 'an integer' but was ${value}`);
    }
    if (value < min || value > max) {
        throw new Error(`${name} must be >= ${min} && <= ${max}. Value was ${value}`);
    }
}
export function once(callback) {
    let called = false;
    return function (...args) {
        if (called)
            return;
        called = true;
        callback.apply(this, args);
    };
}
export function mustCall(fn = () => { }, expectedExecutions = 1, timeout = 1000) {
    if (expectedExecutions < 1) {
        throw new Error("Expected executions can't be lower than 1");
    }
    let timesExecuted = 0;
    const completed = deferred();
    const abort = setTimeout(() => completed.reject(), timeout);
    function callback(...args) {
        timesExecuted++;
        if (timesExecuted === expectedExecutions) {
            completed.resolve();
        }
        fn.apply(this, args);
    }
    const result = completed
        .then(() => clearTimeout(abort))
        .catch(() => fail(`Async operation not completed: Expected ${expectedExecutions}, executed ${timesExecuted}`));
    return [
        result,
        callback,
    ];
}
export async function assertCallbackErrorUncaught({ prelude, invocation, cleanup }) {
    const p = Deno.run({
        cmd: [
            Deno.execPath(),
            "eval",
            "--no-check",
            "--unstable",
            `${prelude ?? ""}

      ${invocation}(err) => {
        // If the bug is present and the callback is called again with an error,
        // don't throw another error, so if the subprocess fails we know it had the correct behaviour.
        if (!err) throw new Error("success");
      });`,
        ],
        stderr: "piped",
    });
    const status = await p.status();
    const stderr = new TextDecoder().decode(await readAll(p.stderr));
    p.close();
    p.stderr.close();
    await cleanup?.();
    assert(!status.success);
    assertStringIncludes(stderr, "Error: success");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxpQkFBaUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLG9CQUFvQixFQUFFLElBQUksRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQzNFLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFpQnhDLE1BQU0sVUFBVSxjQUFjLENBQUMsR0FBWTtJQUN6QyxNQUFNLE9BQU8sR0FBRyxHQUFHLENBQUMsQ0FBQyxDQUFDLG9CQUFvQixHQUFHLEVBQUUsQ0FBQyxDQUFDLENBQUMsaUJBQWlCLENBQUM7SUFDcEUsTUFBTSxJQUFJLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztBQUMzQixDQUFDO0FBR0QsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLFdBQVcsQ0FBQztBQUd4QyxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQUcsV0FBVyxDQUFDO0FBUXhDLE1BQU0sVUFBVSxlQUFlLENBRTdCLElBQW9DLEVBQ3BDLEVBQXNFLEVBRXRFLEdBQUcsSUFBVztJQUVkLElBQUksQ0FBQyxHQUFHLElBQUksQ0FBQyxDQUFDLElBQUksQ0FDaEIsQ0FBQyxLQUFLLEVBQUUsRUFBRSxDQUFDLEVBQUUsSUFBSSxFQUFFLENBQUMsSUFBSSxFQUFFLEtBQUssQ0FBQyxFQUNoQyxDQUFDLEdBQUcsRUFBRSxFQUFFLENBQUMsRUFBRSxJQUFJLEVBQUUsQ0FBQyxHQUFHLENBQUMsQ0FDdkIsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLFVBQVUsNEJBQTRCLENBRTFDLElBQXFDLEVBQ3JDLFdBQTBCLEVBQzFCLEVBQXVFLEVBRXZFLEdBQUcsSUFBVztJQUVkLElBQUksQ0FBQyxHQUFHLElBQUksQ0FBQyxDQUFDLElBQUksQ0FDaEIsQ0FBQyxLQUFLLEVBQUUsRUFBRSxDQUFDLEVBQUUsSUFBSSxFQUFFLENBQUMsSUFBSSxFQUFFLFdBQVcsQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUM3QyxDQUFDLEdBQUcsRUFBRSxFQUFFLENBQUMsRUFBRSxJQUFJLEVBQUUsQ0FBQyxHQUFHLENBQUMsQ0FDdkIsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLFVBQVUsU0FBUyxDQUFDLElBQWMsRUFBRSxLQUFhO0lBQ3JELE9BQU8sS0FBSyxHQUFHLENBQUMsR0FBRyxJQUFJLENBQUMsTUFBTSxFQUFFLEtBQUssRUFBRTtRQUFFLElBQUksQ0FBQyxLQUFLLENBQUMsR0FBRyxJQUFJLENBQUMsS0FBSyxHQUFHLENBQUMsQ0FBQyxDQUFDO0lBQ3ZFLElBQUksQ0FBQyxHQUFHLEVBQUUsQ0FBQztBQUNiLENBQUM7QUFNRCxNQUFNLFVBQVUsaUJBQWlCLENBQy9CLEdBQWtCO0lBRWxCLElBQUksR0FBRyxJQUFJLElBQUksSUFBSSxHQUFHLEtBQUssTUFBTSxJQUFJLEdBQUcsS0FBSyxPQUFPO1FBQUUsT0FBTyxNQUFNLENBQUM7SUFDcEUsT0FBTyxTQUFTLENBQUMsR0FBRyxDQUFDLENBQUM7QUFDeEIsQ0FBQztBQUdELFNBQVMsU0FBUyxDQUFDLEdBQVc7SUFDNUIsUUFBUSxHQUFHLENBQUMsTUFBTSxFQUFFO1FBQ2xCLEtBQUssQ0FBQztZQUNKLElBQUksR0FBRyxLQUFLLE1BQU07Z0JBQUUsT0FBTyxNQUFNLENBQUM7WUFDbEMsSUFBSSxHQUFHLEtBQUssTUFBTSxJQUFJLEdBQUcsS0FBSyxNQUFNO2dCQUFFLE9BQU8sU0FBUyxDQUFDO1lBQ3ZELEdBQUcsR0FBRyxHQUFHLEdBQUcsRUFBRSxDQUFDLFdBQVcsRUFBRSxDQUFDO1lBQzdCLElBQUksR0FBRyxLQUFLLE1BQU07Z0JBQUUsT0FBTyxNQUFNLENBQUM7WUFDbEMsSUFBSSxHQUFHLEtBQUssTUFBTTtnQkFBRSxPQUFPLFNBQVMsQ0FBQztZQUNyQyxNQUFNO1FBQ1IsS0FBSyxDQUFDO1lBQ0osSUFBSSxHQUFHLEtBQUssS0FBSyxJQUFJLEdBQUcsS0FBSyxLQUFLLElBQUksR0FBRyxHQUFHLEVBQUUsQ0FBQyxXQUFXLEVBQUUsS0FBSyxLQUFLLEVBQUU7Z0JBQ3RFLE9BQU8sS0FBSyxDQUFDO2FBQ2Q7WUFDRCxNQUFNO1FBQ1IsS0FBSyxDQUFDO1lBQ0osSUFBSSxHQUFHLEtBQUssT0FBTztnQkFBRSxPQUFPLE9BQU8sQ0FBQztZQUNwQyxJQUFJLEdBQUcsS0FBSyxPQUFPO2dCQUFFLE9BQU8sU0FBUyxDQUFDO1lBQ3RDLElBQUksR0FBRyxLQUFLLE9BQU87Z0JBQUUsT0FBTyxNQUFNLENBQUM7WUFDbkMsSUFBSSxHQUFHLEtBQUssT0FBTztnQkFBRSxPQUFPLE9BQU8sQ0FBQztZQUNwQyxJQUFJLEdBQUcsS0FBSyxPQUFPO2dCQUFFLE9BQU8sU0FBUyxDQUFDO1lBQ3RDLEdBQUcsR0FBRyxHQUFHLEdBQUcsRUFBRSxDQUFDLFdBQVcsRUFBRSxDQUFDO1lBQzdCLElBQUksR0FBRyxLQUFLLE9BQU87Z0JBQUUsT0FBTyxNQUFNLENBQUM7WUFDbkMsSUFBSSxHQUFHLEtBQUssT0FBTztnQkFBRSxPQUFPLE9BQU8sQ0FBQztZQUNwQyxJQUFJLEdBQUcsS0FBSyxPQUFPO2dCQUFFLE9BQU8sU0FBUyxDQUFDO1lBQ3RDLE1BQU07UUFDUixLQUFLLENBQUM7WUFDSixJQUFJLEdBQUcsS0FBSyxRQUFRO2dCQUFFLE9BQU8sUUFBUSxDQUFDO1lBQ3RDLElBQUksR0FBRyxLQUFLLFFBQVEsSUFBSSxHQUFHLEtBQUssUUFBUTtnQkFBRSxPQUFPLFFBQVEsQ0FBQztZQUMxRCxJQUFJLEdBQUcsS0FBSyxRQUFRO2dCQUFFLE9BQU8sUUFBUSxDQUFDO1lBQ3RDLElBQUksR0FBRyxLQUFLLFFBQVEsSUFBSSxHQUFHLEtBQUssUUFBUTtnQkFBRSxPQUFPLFFBQVEsQ0FBQztZQUMxRCxHQUFHLEdBQUcsR0FBRyxHQUFHLEVBQUUsQ0FBQyxXQUFXLEVBQUUsQ0FBQztZQUM3QixJQUFJLEdBQUcsS0FBSyxRQUFRO2dCQUFFLE9BQU8sUUFBUSxDQUFDO1lBQ3RDLElBQUksR0FBRyxLQUFLLFFBQVEsSUFBSSxHQUFHLEtBQUssUUFBUTtnQkFBRSxPQUFPLFFBQVEsQ0FBQztZQUMxRCxNQUFNO1FBQ1IsS0FBSyxDQUFDO1lBQ0osSUFDRSxHQUFHLEtBQUssU0FBUztnQkFDakIsR0FBRyxLQUFLLFNBQVM7Z0JBQ2pCLEdBQUcsR0FBRyxFQUFFLENBQUMsV0FBVyxFQUFFLEtBQUssU0FBUyxFQUNwQztnQkFDQSxPQUFPLFNBQVMsQ0FBQzthQUNsQjtZQUNELE1BQU07UUFDUixLQUFLLENBQUM7WUFDSixJQUNFLEdBQUcsS0FBSyxVQUFVO2dCQUNsQixHQUFHLEtBQUssVUFBVTtnQkFDbEIsR0FBRyxHQUFHLEVBQUUsQ0FBQyxXQUFXLEVBQUUsS0FBSyxVQUFVLEVBQ3JDO2dCQUNBLE9BQU8sU0FBUyxDQUFDO2FBQ2xCO1lBQ0QsTUFBTTtRQUNSO1lBQ0UsSUFBSSxHQUFHLEtBQUssRUFBRTtnQkFBRSxPQUFPLE1BQU0sQ0FBQztLQUNqQztBQUNILENBQUM7QUFFRCxNQUFNLFVBQVUsb0JBQW9CLENBQ2xDLEtBQWEsRUFDYixJQUFZLEVBQ1osR0FBRyxHQUFHLENBQUMsVUFBVSxFQUNqQixHQUFHLEdBQUcsVUFBVTtJQUdoQixJQUFJLENBQUMsTUFBTSxDQUFDLFNBQVMsQ0FBQyxLQUFLLENBQUMsRUFBRTtRQUM1QixNQUFNLElBQUksS0FBSyxDQUFDLEdBQUcsSUFBSSxpQ0FBaUMsS0FBSyxFQUFFLENBQUMsQ0FBQztLQUNsRTtJQUVELElBQUksS0FBSyxHQUFHLEdBQUcsSUFBSSxLQUFLLEdBQUcsR0FBRyxFQUFFO1FBQzlCLE1BQU0sSUFBSSxLQUFLLENBQ2IsR0FBRyxJQUFJLGVBQWUsR0FBRyxVQUFVLEdBQUcsZUFBZSxLQUFLLEVBQUUsQ0FDN0QsQ0FBQztLQUNIO0FBQ0gsQ0FBQztBQUtELE1BQU0sVUFBVSxJQUFJLENBQ2xCLFFBQThDO0lBRTlDLElBQUksTUFBTSxHQUFHLEtBQUssQ0FBQztJQUNuQixPQUFPLFVBQXlCLEdBQUcsSUFBdUI7UUFDeEQsSUFBSSxNQUFNO1lBQUUsT0FBTztRQUNuQixNQUFNLEdBQUcsSUFBSSxDQUFDO1FBQ2QsUUFBUSxDQUFDLEtBQUssQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLENBQUM7SUFDN0IsQ0FBQyxDQUFDO0FBQ0osQ0FBQztBQU1ELE1BQU0sVUFBVSxRQUFRLENBQ3RCLEtBQTZCLEdBQUcsRUFBRSxHQUFFLENBQUMsRUFDckMsa0JBQWtCLEdBQUcsQ0FBQyxFQUN0QixPQUFPLEdBQUcsSUFBSTtJQUVkLElBQUksa0JBQWtCLEdBQUcsQ0FBQyxFQUFFO1FBQzFCLE1BQU0sSUFBSSxLQUFLLENBQUMsMkNBQTJDLENBQUMsQ0FBQztLQUM5RDtJQUNELElBQUksYUFBYSxHQUFHLENBQUMsQ0FBQztJQUN0QixNQUFNLFNBQVMsR0FBRyxRQUFRLEVBQUUsQ0FBQztJQUU3QixNQUFNLEtBQUssR0FBRyxVQUFVLENBQUMsR0FBRyxFQUFFLENBQUMsU0FBUyxDQUFDLE1BQU0sRUFBRSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBRTVELFNBQVMsUUFBUSxDQUFnQixHQUFHLElBQU87UUFDekMsYUFBYSxFQUFFLENBQUM7UUFDaEIsSUFBSSxhQUFhLEtBQUssa0JBQWtCLEVBQUU7WUFDeEMsU0FBUyxDQUFDLE9BQU8sRUFBRSxDQUFDO1NBQ3JCO1FBQ0QsRUFBRSxDQUFDLEtBQUssQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLENBQUM7SUFDdkIsQ0FBQztJQUVELE1BQU0sTUFBTSxHQUFHLFNBQVM7U0FDckIsSUFBSSxDQUFDLEdBQUcsRUFBRSxDQUFDLFlBQVksQ0FBQyxLQUFLLENBQUMsQ0FBQztTQUMvQixLQUFLLENBQUMsR0FBRyxFQUFFLENBQ1YsSUFBSSxDQUNGLDJDQUEyQyxrQkFBa0IsY0FBYyxhQUFhLEVBQUUsQ0FDM0YsQ0FDRixDQUFDO0lBRUosT0FBTztRQUNMLE1BQU07UUFDTixRQUFRO0tBQ1QsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLENBQUMsS0FBSyxVQUFVLDJCQUEyQixDQUMvQyxFQUFFLE9BQU8sRUFBRSxVQUFVLEVBQUUsT0FBTyxFQVU3QjtJQUlELE1BQU0sQ0FBQyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUM7UUFDakIsR0FBRyxFQUFFO1lBQ0gsSUFBSSxDQUFDLFFBQVEsRUFBRTtZQUNmLE1BQU07WUFDTixZQUFZO1lBQ1osWUFBWTtZQUNaLEdBQUcsT0FBTyxJQUFJLEVBQUU7O1FBRWQsVUFBVTs7OztVQUlSO1NBQ0w7UUFDRCxNQUFNLEVBQUUsT0FBTztLQUNoQixDQUFDLENBQUM7SUFDSCxNQUFNLE1BQU0sR0FBRyxNQUFNLENBQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQztJQUNoQyxNQUFNLE1BQU0sR0FBRyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxNQUFNLE9BQU8sQ0FBQyxDQUFDLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztJQUNqRSxDQUFDLENBQUMsS0FBSyxFQUFFLENBQUM7SUFDVixDQUFDLENBQUMsTUFBTSxDQUFDLEtBQUssRUFBRSxDQUFDO0lBQ2pCLE1BQU0sT0FBTyxFQUFFLEVBQUUsQ0FBQztJQUNsQixNQUFNLENBQUMsQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLENBQUM7SUFDeEIsb0JBQW9CLENBQUMsTUFBTSxFQUFFLGdCQUFnQixDQUFDLENBQUM7QUFDakQsQ0FBQyJ9