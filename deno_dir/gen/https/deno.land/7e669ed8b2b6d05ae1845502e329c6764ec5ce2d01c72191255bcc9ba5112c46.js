import { asyncIterableToCallback } from "./_fs_watch.ts";
import Dirent from "./_fs_dirent.ts";
import { fromFileUrl } from "../path.ts";
function toDirent(val) {
    return new Dirent(val);
}
export function readdir(path, optionsOrCallback, maybeCallback) {
    const callback = (typeof optionsOrCallback === "function"
        ? optionsOrCallback
        : maybeCallback);
    const options = typeof optionsOrCallback === "object"
        ? optionsOrCallback
        : null;
    const result = [];
    path = path instanceof URL ? fromFileUrl(path) : path;
    if (!callback)
        throw new Error("No callback function supplied");
    if (options?.encoding) {
        try {
            new TextDecoder(options.encoding);
        }
        catch {
            throw new Error(`TypeError [ERR_INVALID_OPT_VALUE_ENCODING]: The value "${options.encoding}" is invalid for option "encoding"`);
        }
    }
    try {
        asyncIterableToCallback(Deno.readDir(path), (val, done) => {
            if (typeof path !== "string")
                return;
            if (done) {
                callback(null, result);
                return;
            }
            if (options?.withFileTypes) {
                result.push(toDirent(val));
            }
            else
                result.push(decode(val.name));
        });
    }
    catch (error) {
        callback(error);
    }
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
export function readdirSync(path, options) {
    const result = [];
    path = path instanceof URL ? fromFileUrl(path) : path;
    if (options?.encoding) {
        try {
            new TextDecoder(options.encoding);
        }
        catch {
            throw new Error(`TypeError [ERR_INVALID_OPT_VALUE_ENCODING]: The value "${options.encoding}" is invalid for option "encoding"`);
        }
    }
    for (const file of Deno.readDirSync(path)) {
        if (options?.withFileTypes) {
            result.push(toDirent(file));
        }
        else
            result.push(decode(file.name));
    }
    return result;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX3JlYWRkaXIuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfZnNfcmVhZGRpci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsdUJBQXVCLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQUN6RCxPQUFPLE1BQU0sTUFBTSxpQkFBaUIsQ0FBQztBQUNyQyxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBRXpDLFNBQVMsUUFBUSxDQUFDLEdBQWtCO0lBQ2xDLE9BQU8sSUFBSSxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUM7QUFDekIsQ0FBQztBQTBCRCxNQUFNLFVBQVUsT0FBTyxDQUNyQixJQUFrQixFQUNsQixpQkFBMkUsRUFDM0UsYUFBdUQ7SUFFdkQsTUFBTSxRQUFRLEdBQ1osQ0FBQyxPQUFPLGlCQUFpQixLQUFLLFVBQVU7UUFDdEMsQ0FBQyxDQUFDLGlCQUFpQjtRQUNuQixDQUFDLENBQUMsYUFBYSxDQUE0QixDQUFDO0lBQ2hELE1BQU0sT0FBTyxHQUFHLE9BQU8saUJBQWlCLEtBQUssUUFBUTtRQUNuRCxDQUFDLENBQUMsaUJBQWlCO1FBQ25CLENBQUMsQ0FBQyxJQUFJLENBQUM7SUFDVCxNQUFNLE1BQU0sR0FBMkIsRUFBRSxDQUFDO0lBQzFDLElBQUksR0FBRyxJQUFJLFlBQVksR0FBRyxDQUFDLENBQUMsQ0FBQyxXQUFXLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLElBQUksQ0FBQztJQUV0RCxJQUFJLENBQUMsUUFBUTtRQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsK0JBQStCLENBQUMsQ0FBQztJQUVoRSxJQUFJLE9BQU8sRUFBRSxRQUFRLEVBQUU7UUFDckIsSUFBSTtZQUNGLElBQUksV0FBVyxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQztTQUNuQztRQUFDLE1BQU07WUFDTixNQUFNLElBQUksS0FBSyxDQUNiLDBEQUEwRCxPQUFPLENBQUMsUUFBUSxvQ0FBb0MsQ0FDL0csQ0FBQztTQUNIO0tBQ0Y7SUFFRCxJQUFJO1FBQ0YsdUJBQXVCLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsRUFBRSxDQUFDLEdBQUcsRUFBRSxJQUFJLEVBQUUsRUFBRTtZQUN4RCxJQUFJLE9BQU8sSUFBSSxLQUFLLFFBQVE7Z0JBQUUsT0FBTztZQUNyQyxJQUFJLElBQUksRUFBRTtnQkFDUixRQUFRLENBQUMsSUFBSSxFQUFFLE1BQU0sQ0FBQyxDQUFDO2dCQUN2QixPQUFPO2FBQ1I7WUFDRCxJQUFJLE9BQU8sRUFBRSxhQUFhLEVBQUU7Z0JBQzFCLE1BQU0sQ0FBQyxJQUFJLENBQUMsUUFBUSxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUM7YUFDNUI7O2dCQUFNLE1BQU0sQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDO1FBQ3ZDLENBQUMsQ0FBQyxDQUFDO0tBQ0o7SUFBQyxPQUFPLEtBQUssRUFBRTtRQUNkLFFBQVEsQ0FBQyxLQUFLLENBQUMsQ0FBQztLQUNqQjtBQUNILENBQUM7QUFFRCxTQUFTLE1BQU0sQ0FBQyxHQUFXLEVBQUUsUUFBaUI7SUFDNUMsSUFBSSxDQUFDLFFBQVE7UUFBRSxPQUFPLEdBQUcsQ0FBQztTQUNyQjtRQUNILE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxDQUFDLFFBQVEsQ0FBQyxDQUFDO1FBQzFDLE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUM7UUFDbEMsT0FBTyxPQUFPLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQztLQUM1QztBQUNILENBQUM7QUFVRCxNQUFNLFVBQVUsV0FBVyxDQUN6QixJQUFrQixFQUNsQixPQUF3QjtJQUV4QixNQUFNLE1BQU0sR0FBRyxFQUFFLENBQUM7SUFDbEIsSUFBSSxHQUFHLElBQUksWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO0lBRXRELElBQUksT0FBTyxFQUFFLFFBQVEsRUFBRTtRQUNyQixJQUFJO1lBQ0YsSUFBSSxXQUFXLENBQUMsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDO1NBQ25DO1FBQUMsTUFBTTtZQUNOLE1BQU0sSUFBSSxLQUFLLENBQ2IsMERBQTBELE9BQU8sQ0FBQyxRQUFRLG9DQUFvQyxDQUMvRyxDQUFDO1NBQ0g7S0FDRjtJQUVELEtBQUssTUFBTSxJQUFJLElBQUksSUFBSSxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsRUFBRTtRQUN6QyxJQUFJLE9BQU8sRUFBRSxhQUFhLEVBQUU7WUFDMUIsTUFBTSxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztTQUM3Qjs7WUFBTSxNQUFNLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztLQUN2QztJQUNELE9BQU8sTUFBTSxDQUFDO0FBQ2hCLENBQUMifQ==