import { notImplemented, } from "../_utils.ts";
export function isFileOptions(fileOptions) {
    if (!fileOptions)
        return false;
    return (fileOptions.encoding != undefined ||
        fileOptions.flag != undefined ||
        fileOptions.mode != undefined);
}
export function getEncoding(optOrCallback) {
    if (!optOrCallback || typeof optOrCallback === "function") {
        return null;
    }
    const encoding = typeof optOrCallback === "string"
        ? optOrCallback
        : optOrCallback.encoding;
    if (!encoding)
        return null;
    return encoding;
}
export function checkEncoding(encoding) {
    if (!encoding)
        return null;
    encoding = encoding.toLowerCase();
    if (["utf8", "hex", "base64"].includes(encoding))
        return encoding;
    if (encoding === "utf-8") {
        return "utf8";
    }
    if (encoding === "binary") {
        return "binary";
    }
    const notImplementedEncodings = ["utf16le", "latin1", "ascii", "ucs2"];
    if (notImplementedEncodings.includes(encoding)) {
        notImplemented(`"${encoding}" encoding`);
    }
    throw new Error(`The value "${encoding}" is invalid for option "encoding"`);
}
export function getOpenOptions(flag) {
    if (!flag) {
        return { create: true, append: true };
    }
    let openOptions;
    switch (flag) {
        case "a": {
            openOptions = { create: true, append: true };
            break;
        }
        case "ax": {
            openOptions = { createNew: true, write: true, append: true };
            break;
        }
        case "a+": {
            openOptions = { read: true, create: true, append: true };
            break;
        }
        case "ax+": {
            openOptions = { read: true, createNew: true, append: true };
            break;
        }
        case "r": {
            openOptions = { read: true };
            break;
        }
        case "r+": {
            openOptions = { read: true, write: true };
            break;
        }
        case "w": {
            openOptions = { create: true, write: true, truncate: true };
            break;
        }
        case "wx": {
            openOptions = { createNew: true, write: true };
            break;
        }
        case "w+": {
            openOptions = { create: true, write: true, truncate: true, read: true };
            break;
        }
        case "wx+": {
            openOptions = { createNew: true, write: true, read: true };
            break;
        }
        case "as": {
            openOptions = { create: true, append: true };
            break;
        }
        case "as+": {
            openOptions = { create: true, read: true, append: true };
            break;
        }
        case "rs+": {
            openOptions = { create: true, read: true, write: true };
            break;
        }
        default: {
            throw new Error(`Unrecognized file system flag: ${flag}`);
        }
    }
    return openOptions;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX2NvbW1vbi5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9mc19jb21tb24udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUdMLGNBQWMsR0FFZixNQUFNLGNBQWMsQ0FBQztBQXFCdEIsTUFBTSxVQUFVLGFBQWEsQ0FDM0IsV0FBa0Q7SUFFbEQsSUFBSSxDQUFDLFdBQVc7UUFBRSxPQUFPLEtBQUssQ0FBQztJQUUvQixPQUFPLENBQ0osV0FBMkIsQ0FBQyxRQUFRLElBQUksU0FBUztRQUNqRCxXQUEyQixDQUFDLElBQUksSUFBSSxTQUFTO1FBQzdDLFdBQWdDLENBQUMsSUFBSSxJQUFJLFNBQVMsQ0FDcEQsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLFVBQVUsV0FBVyxDQUN6QixhQU1RO0lBRVIsSUFBSSxDQUFDLGFBQWEsSUFBSSxPQUFPLGFBQWEsS0FBSyxVQUFVLEVBQUU7UUFDekQsT0FBTyxJQUFJLENBQUM7S0FDYjtJQUVELE1BQU0sUUFBUSxHQUFHLE9BQU8sYUFBYSxLQUFLLFFBQVE7UUFDaEQsQ0FBQyxDQUFDLGFBQWE7UUFDZixDQUFDLENBQUMsYUFBYSxDQUFDLFFBQVEsQ0FBQztJQUMzQixJQUFJLENBQUMsUUFBUTtRQUFFLE9BQU8sSUFBSSxDQUFDO0lBQzNCLE9BQU8sUUFBUSxDQUFDO0FBQ2xCLENBQUM7QUFFRCxNQUFNLFVBQVUsYUFBYSxDQUFDLFFBQTBCO0lBQ3RELElBQUksQ0FBQyxRQUFRO1FBQUUsT0FBTyxJQUFJLENBQUM7SUFFM0IsUUFBUSxHQUFHLFFBQVEsQ0FBQyxXQUFXLEVBQWUsQ0FBQztJQUMvQyxJQUFJLENBQUMsTUFBTSxFQUFFLEtBQUssRUFBRSxRQUFRLENBQUMsQ0FBQyxRQUFRLENBQUMsUUFBUSxDQUFDO1FBQUUsT0FBTyxRQUFRLENBQUM7SUFFbEUsSUFBSSxRQUFRLEtBQUssT0FBTyxFQUFFO1FBQ3hCLE9BQU8sTUFBTSxDQUFDO0tBQ2Y7SUFDRCxJQUFJLFFBQVEsS0FB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