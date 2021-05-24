export function notImplemented(msg) {
    const message = msg ? `Not implemented: ${msg}` : "Not implemented";
    throw new Error(message);
}
export const _TextDecoder = TextDecoder;
export const _TextEncoder = TextEncoder;
export function intoCallbackAPI(func, cb, ...args) {
    func(...args)
        .then((value) => cb && cb(null, value))
        .catch((err) => cb && cb(err, null));
}
export function intoCallbackAPIWithIntercept(func, interceptor, cb, ...args) {
    func(...args)
        .then((value) => cb && cb(null, interceptor(value)))
        .catch((err) => cb && cb(err, null));
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX3V0aWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE1BQU0sVUFBVSxjQUFjLENBQUMsR0FBWTtJQUN6QyxNQUFNLE9BQU8sR0FBRyxHQUFHLENBQUMsQ0FBQyxDQUFDLG9CQUFvQixHQUFHLEVBQUUsQ0FBQyxDQUFDLENBQUMsaUJBQWlCLENBQUM7SUFDcEUsTUFBTSxJQUFJLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztBQUMzQixDQUFDO0FBR0QsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLFdBQVcsQ0FBQztBQUd4QyxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQUcsV0FBVyxDQUFDO0FBUXhDLE1BQU0sVUFBVSxlQUFlLENBRTdCLElBQW9DLEVBQ3BDLEVBQXFFLEVBRXJFLEdBQUcsSUFBVztJQUVkLElBQUksQ0FBQyxHQUFHLElBQUksQ0FBQztTQUNWLElBQUksQ0FBQyxDQUFDLEtBQUssRUFBRSxFQUFFLENBQUMsRUFBRSxJQUFJLEVBQUUsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUM7U0FDdEMsS0FBSyxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxFQUFFLElBQUksRUFBRSxDQUFDLEdBQUcsRUFBRSxJQUFJLENBQUMsQ0FBQyxDQUFDO0FBQ3pDLENBQUM7QUFFRCxNQUFNLFVBQVUsNEJBQTRCLENBRTFDLElBQXFDLEVBQ3JDLFdBQTBCLEVBQzFCLEVBQXNFLEVBRXRFLEdBQUcsSUFBVztJQUVkLElBQUksQ0FBQyxHQUFHLElBQUksQ0FBQztTQUNWLElBQUksQ0FBQyxDQUFDLEtBQUssRUFBRSxFQUFFLENBQUMsRUFBRSxJQUFJLEVBQUUsQ0FBQyxJQUFJLEVBQUUsV0FBVyxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUM7U0FDbkQsS0FBSyxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxFQUFFLElBQUksRUFBRSxDQUFDLEdBQUcsRUFBRSxJQUFJ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