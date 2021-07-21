import { intoCallbackAPIWithIntercept, notImplemented, } from "../_utils.ts";
import { fromFileUrl } from "../path.ts";
function maybeEncode(data, encoding) {
    if (encoding === "buffer") {
        return new TextEncoder().encode(data);
    }
    return data;
}
function getEncoding(optOrCallback) {
    if (!optOrCallback || typeof optOrCallback === "function") {
        return null;
    }
    else {
        if (optOrCallback.encoding) {
            if (optOrCallback.encoding === "utf8" ||
                optOrCallback.encoding === "utf-8") {
                return "utf8";
            }
            else if (optOrCallback.encoding === "buffer") {
                return "buffer";
            }
            else {
                notImplemented();
            }
        }
        return null;
    }
}
export function readlink(path, optOrCallback, callback) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    let cb;
    if (typeof optOrCallback === "function") {
        cb = optOrCallback;
    }
    else {
        cb = callback;
    }
    const encoding = getEncoding(optOrCallback);
    intoCallbackAPIWithIntercept(Deno.readLink, (data) => maybeEncode(data, encoding), cb, path);
}
export function readlinkSync(path, opt) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    return maybeEncode(Deno.readLinkSync(path), getEncoding(opt));
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX3JlYWRsaW5rLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2ZzX3JlYWRsaW5rLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFDTCw0QkFBNEIsRUFFNUIsY0FBYyxHQUNmLE1BQU0sY0FBYyxDQUFDO0FBQ3RCLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFXekMsU0FBUyxXQUFXLENBQ2xCLElBQVksRUFDWixRQUF1QjtJQUV2QixJQUFJLFFBQVEsS0FBSyxRQUFRLEVBQUU7UUFDekIsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsQ0FBQztLQUN2QztJQUNELE9BQU8sSUFBSSxDQUFDO0FBQ2QsQ0FBQztBQUVELFNBQVMsV0FBVyxDQUNsQixhQUFrRDtJQUVsRCxJQUFJLENBQUMsYUFBYSxJQUFJLE9BQU8sYUFBYSxLQUFLLFVBQVUsRUFBRTtRQUN6RCxPQUFPLElBQUksQ0FBQztLQUNiO1NBQU07UUFDTCxJQUFJLGFBQWEsQ0FBQyxRQUFRLEVBQUU7WUFDMUIsSUFDRSxhQUFhLENBQUMsUUFBUSxLQUFLLE1BQU07Z0JBQ2pDLGFBQWEsQ0FBQyxRQUFRLEtBQUssT0FBTyxFQUNsQztnQkFDQSxPQUFPLE1BQU0sQ0FBQzthQUNmO2lCQUFNLElBQUksYUFBYSxDQUFDLFFBQVEsS0FBSyxRQUFRLEVBQUU7Z0JBQzlDLE9BQU8sUUFBUSxDQUFDO2FBQ2pCO2lCQUFNO2dCQUNMLGNBQWMsRUFBRSxDQUFDO2FBQ2xCO1NBQ0Y7UUFDRCxPQUFPLElBQUksQ0FBQztLQUNiO0FBQ0gsQ0FBQztBQUVELE1BQU0sVUFBVSxRQUFRLENBQ3RCLElBQWtCLEVBQ2xCLGFBQWlELEVBQ2pELFFBQTJCO0lBRTNCLElBQUksR0FBRyxJQUFJLFlBQVksR0FBRyxDQUFDLENBQUMsQ0FBQyxXQUFXLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLElBQUksQ0FBQztJQUV0RCxJQUFJLEVBQWdDLENBQUM7SUFDckMsSUFBSSxPQUFPLGFBQWEsS0FBSyxVQUFVLEVBQUU7UUFDdkMsRUFBRSxHQUFHLGFBQWEsQ0FBQztLQUNwQjtTQUFNO1FBQ0wsRUFBRSxHQUFHLFFBQVEsQ0FBQztLQUNmO0lBRUQsTUFBTSxRQUFRLEdBQUcsV0FBVyxDQUFDLGFBQWEsQ0FBQyxDQUFDO0lBRTVDLDRCQUE0QixDQUMxQixJQUFJLENBQUMsUUFBUSxFQUNiLENBQUMsSUFBWSxFQUF1QixFQUFFLENBQUMsV0FBVyxDQUFDLElBQUksRUFBRSxRQUFRLENBQUMsRUFDbEUsRUFBRSxFQUNGLElBQUksQ0FDTCxDQUFDO0FBQ0osQ0FBQztBQUVELE1BQU0sVUFBVSxZQUFZLENBQzFCLElBQWtCLEVBQ2xCLEdBQXFCO0lBRXJCLElBQUksR0FBRyxJQUFJLFlBQVksR0FBRyxDQUFDLENBQUMsQ0FBQyxXQUFXLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLElBQUksQ0FBQztJQUV0RCxPQUFPLFdBQVcsQ0FBQyxJQUFJLENBQUMsWUFBWSxDQUFDLElBQUksQ0FBQyxFQUFFLFdBQVcsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDO0FBQ2hFLENBQUMifQ==