import { deferred } from "../async/mod.ts";
import { assert, assertStringIncludes, fail } from "../testing/asserts.ts";
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
    const stderr = new TextDecoder().decode(await Deno.readAll(p.stderr));
    p.close();
    p.stderr.close();
    await cleanup?.();
    assert(!status.success);
    assertStringIncludes(stderr, "Error: success");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxpQkFBaUIsQ0FBQztBQUMzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLG9CQUFvQixFQUFFLElBQUksRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBaUIzRSxNQUFNLFVBQVUsY0FBYyxDQUFDLEdBQVk7SUFDekMsTUFBTSxPQUFPLEdBQUcsR0FBRyxDQUFDLENBQUMsQ0FBQyxvQkFBb0IsR0FBRyxFQUFFLENBQUMsQ0FBQyxDQUFDLGlCQUFpQixDQUFDO0lBQ3BFLE1BQU0sSUFBSSxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7QUFDM0IsQ0FBQztBQUdELE1BQU0sQ0FBQyxNQUFNLFlBQVksR0FBRyxXQUFXLENBQUM7QUFHeEMsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLFdBQVcsQ0FBQztBQVF4QyxNQUFNLFVBQVUsZUFBZSxDQUU3QixJQUFvQyxFQUNwQyxFQUFzRSxFQUV0RSxHQUFHLElBQVc7SUFFZCxJQUFJLENBQUMsR0FBRyxJQUFJLENBQUMsQ0FBQyxJQUFJLENBQ2hCLENBQUMsS0FBSyxFQUFFLEVBQUUsQ0FBQyxFQUFFLElBQUksRUFBRSxDQUFDLElBQUksRUFBRSxLQUFLLENBQUMsRUFDaEMsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLEVBQUUsSUFBSSxFQUFFLENBQUMsR0FBRyxDQUFDLENBQ3ZCLENBQUM7QUFDSixDQUFDO0FBRUQsTUFBTSxVQUFVLDRCQUE0QixDQUUxQyxJQUFxQyxFQUNyQyxXQUEwQixFQUMxQixFQUF1RSxFQUV2RSxHQUFHLElBQVc7SUFFZCxJQUFJLENBQUMsR0FBRyxJQUFJLENBQUMsQ0FBQyxJQUFJLENBQ2hCLENBQUMsS0FBSyxFQUFFLEVBQUUsQ0FBQyxFQUFFLElBQUksRUFBRSxDQUFDLElBQUksRUFBRSxXQUFXLENBQUMsS0FBSyxDQUFD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