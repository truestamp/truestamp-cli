import { existsSync } from "./_fs_exists.ts";
import { mkdir, mkdirSync } from "./_fs_mkdir.ts";
import { ERR_INVALID_CALLBACK, ERR_INVALID_OPT_VALUE_ENCODING, } from "../_errors.ts";
export function mkdtemp(prefix, optionsOrCallback, maybeCallback) {
    const callback = typeof optionsOrCallback == "function" ? optionsOrCallback : maybeCallback;
    if (!callback)
        throw new ERR_INVALID_CALLBACK(callback);
    const encoding = parseEncoding(optionsOrCallback);
    const path = tempDirPath(prefix);
    mkdir(path, { recursive: false, mode: 0o700 }, (err) => {
        if (err)
            callback(err);
        else
            callback(null, decode(path, encoding));
    });
}
export function mkdtempSync(prefix, options) {
    const encoding = parseEncoding(options);
    const path = tempDirPath(prefix);
    mkdirSync(path, { recursive: false, mode: 0o700 });
    return decode(path, encoding);
}
function parseEncoding(optionsOrCallback) {
    let encoding;
    if (typeof optionsOrCallback == "function")
        encoding = undefined;
    else if (optionsOrCallback instanceof Object) {
        encoding = optionsOrCallback?.encoding;
    }
    else
        encoding = optionsOrCallback;
    if (encoding) {
        try {
            new TextDecoder(encoding);
        }
        catch {
            throw new ERR_INVALID_OPT_VALUE_ENCODING(encoding);
        }
    }
    return encoding;
}
function decode(str, encoding) {
    if (!encoding)
        return str;
    else {
        const decoder = new TextDecoder(encoding);
        const encoder = new TextEncoder();
        return decoder.decode(encoder.encode(str));
    }
}
const CHARS = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ";
function randomName() {
    return [...Array(6)].map(() => CHARS[Math.floor(Math.random() * CHARS.length)]).join("");
}
function tempDirPath(prefix) {
    let path;
    do {
        path = prefix + randomName();
    } while (existsSync(path));
    return path;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX21rZHRlbXAuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfZnNfbWtkdGVtcC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsVUFBVSxFQUFFLE1BQU0saUJBQWlCLENBQUM7QUFDN0MsT0FBTyxFQUFFLEtBQUssRUFBRSxTQUFTLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQUNsRCxPQUFPLEVBQ0wsb0JBQW9CLEVBQ3BCLDhCQUE4QixHQUMvQixNQUFNLGVBQWUsQ0FBQztBQWN2QixNQUFNLFVBQVUsT0FBTyxDQUNyQixNQUFjLEVBQ2QsaUJBQWtFLEVBQ2xFLGFBQStCO0lBRS9CLE1BQU0sUUFBUSxHQUNaLE9BQU8saUJBQWlCLElBQUksVUFBVSxDQUFDLENBQUMsQ0FBQyxpQkFBaUIsQ0FBQyxDQUFDLENBQUMsYUFBYSxDQUFDO0lBQzdFLElBQUksQ0FBQyxRQUFRO1FBQUUsTUFBTSxJQUFJLG9CQUFvQixDQUFDLFFBQVEsQ0FBQyxDQUFDO0lBRXhELE1BQU0sUUFBUSxHQUF1QixhQUFhLENBQUMsaUJBQWlCLENBQUMsQ0FBQztJQUN0RSxNQUFNLElBQUksR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLENBQUM7SUFFakMsS0FBSyxDQUNILElBQUksRUFDSixFQUFFLFNBQVMsRUFBRSxLQUFLLEVBQUUsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUNqQyxDQUFDLEdBQTZCLEVBQUUsRUFBRTtRQUNoQyxJQUFJLEdBQUc7WUFBRSxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUM7O1lBQ2xCLFFBQVEsQ0FBQyxJQUFJLEVBQUUsTUFBTSxDQUFDLElBQUksRUFBRSxRQUFRLENBQUMsQ0FBQyxDQUFDO0lBQzlDLENBQUMsQ0FDRixDQUFDO0FBQ0osQ0FBQztBQUdELE1BQU0sVUFBVSxXQUFXLENBQ3pCLE1BQWMsRUFDZCxPQUF1QztJQUV2QyxNQUFNLFFBQVEsR0FBdUIsYUFBYSxDQUFDLE9BQU8sQ0FBQyxDQUFDO0lBQzVELE1BQU0sSUFBSSxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsQ0FBQztJQUVqQyxTQUFTLENBQUMsSUFBSSxFQUFFLEVBQUUsU0FBUyxFQUFFLEtBQUssRUFBRSxJQUFJLEVBQUUsS0FBSyxFQUFFLENBQUMsQ0FBQztJQUNuRCxPQUFPLE1BQU0sQ0FBQyxJQUFJLEVBQUUsUUFBUSxDQUFDLENBQUM7QUFDaEMsQ0FBQztBQUVELFNBQVMsYUFBYSxDQUNwQixpQkFBbUU7SUFFbkUsSUFBSSxRQUE0QixDQUFDO0lBQ2pDLElBQUksT0FBTyxpQkFBaUIsSUFBSSxVQUFVO1FBQUUsUUFBUSxHQUFHLFNBQVMsQ0FBQztTQUM1RCxJQUFJLGlCQUFpQixZQUFZLE1BQU0sRUFBRTtRQUM1QyxRQUFRLEdBQUcsaUJBQWlCLEVBQUUsUUFBUSxDQUFDO0tBQ3hDOztRQUFNLFFBQVEsR0FBRyxpQkFBaUIsQ0FBQztJQUVwQyxJQUFJLFFBQVEsRUFBRTtRQUNaLElBQUk7WUFDRixJQUFJLFdBQVcsQ0FBQyxRQUFRLENBQUMsQ0FBQztTQUMzQjtRQUFDLE1BQU07WUFDTixNQUFNLElBQUksOEJBQThCLENBQUMsUUFBUSxDQUFDLENBQUM7U0FDcEQ7S0FDRjtJQUVELE9BQU8sUUFBUSxDQUFDO0FBQ2xCLENBQUM7QUFFRCxTQUFTLE1BQU0sQ0FBQyxHQUFXLEVBQUUsUUFBaUI7SUFDNUMsSUFBSSxDQUFDLFFBQVE7UUFBRSxPQUFPLEdBQUcsQ0FBQztTQUNyQjtRQUNILE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxDQUFDLFFBQVEsQ0FBQyxDQUFDO1FBQzFDLE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUM7UUFDbEMsT0FBTyxPQUFPLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQztLQUM1QztBQUNILENBQUM7QUFFRCxNQUFNLEtBQUssR0FBRyxnRUFBZ0UsQ0FBQztBQUMvRSxTQUFTLFVBQVU7SUFDakIsT0FBTyxDQUFDLEdBQUcsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLEdBQUcsRUFBRSxDQUM1QixLQUFLLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUMsTUFBTSxFQUFFLEdBQUcsS0FBSyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQ2hELENBQUMsSUFBSSxDQUFDLEVBQUUsQ0FBQyxDQUFDO0FBQ2IsQ0FBQztBQUVELFNBQVMsV0FBVyxDQUFDLE1BQWM7SUFDakMsSUFBSSxJQUFZLENBQUM7SUFDakIsR0FBRztRQUNELElBQUksR0FBRyxNQUFNLEdBQUcsVUFBVSxFQUFFLENBQUM7S0FDOUIsUUFBUSxVQUFVLENBQUMsSUFBSSxDQUFDLEVBQUU7SUFFM0IsT0FBTyxJQUFJLENBQUM7QUFDZCxDQUFDIn0=