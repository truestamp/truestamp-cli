import { getOpenOptions, isFileOptions, } from "./_fs_common.ts";
import { notImplemented } from "../_utils.ts";
import { fromFileUrl } from "../path.ts";
export function appendFile(pathOrRid, data, optionsOrCallback, callback) {
    pathOrRid = pathOrRid instanceof URL ? fromFileUrl(pathOrRid) : pathOrRid;
    const callbackFn = optionsOrCallback instanceof Function ? optionsOrCallback : callback;
    const options = optionsOrCallback instanceof Function ? undefined : optionsOrCallback;
    if (!callbackFn) {
        throw new Error("No callback function supplied");
    }
    validateEncoding(options);
    let rid = -1;
    const buffer = data instanceof Uint8Array
        ? data
        : new TextEncoder().encode(data);
    new Promise((resolve, reject) => {
        if (typeof pathOrRid === "number") {
            rid = pathOrRid;
            Deno.write(rid, buffer).then(resolve, reject);
        }
        else {
            const mode = isFileOptions(options)
                ? options.mode
                : undefined;
            const flag = isFileOptions(options)
                ? options.flag
                : undefined;
            if (mode) {
                notImplemented("Deno does not yet support setting mode on create");
            }
            Deno.open(pathOrRid, getOpenOptions(flag))
                .then(({ rid: openedFileRid }) => {
                rid = openedFileRid;
                return Deno.write(openedFileRid, buffer);
            })
                .then(resolve, reject);
        }
    })
        .then(() => {
        closeRidIfNecessary(typeof pathOrRid === "string", rid);
        callbackFn(null);
    }, (err) => {
        closeRidIfNecessary(typeof pathOrRid === "string", rid);
        callbackFn(err);
    });
}
function closeRidIfNecessary(isPathString, rid) {
    if (isPathString && rid != -1) {
        Deno.close(rid);
    }
}
export function appendFileSync(pathOrRid, data, options) {
    let rid = -1;
    validateEncoding(options);
    pathOrRid = pathOrRid instanceof URL ? fromFileUrl(pathOrRid) : pathOrRid;
    try {
        if (typeof pathOrRid === "number") {
            rid = pathOrRid;
        }
        else {
            const mode = isFileOptions(options)
                ? options.mode
                : undefined;
            const flag = isFileOptions(options)
                ? options.flag
                : undefined;
            if (mode) {
                notImplemented("Deno does not yet support setting mode on create");
            }
            const file = Deno.openSync(pathOrRid, getOpenOptions(flag));
            rid = file.rid;
        }
        const buffer = data instanceof Uint8Array
            ? data
            : new TextEncoder().encode(data);
        Deno.writeSync(rid, buffer);
    }
    finally {
        closeRidIfNecessary(typeof pathOrRid === "string", rid);
    }
}
function validateEncoding(encodingOption) {
    if (!encodingOption)
        return;
    if (typeof encodingOption === "string") {
        if (encodingOption !== "utf8") {
            throw new Error("Only 'utf8' encoding is currently supported");
        }
    }
    else if (encodingOption.encoding && encodingOption.encoding !== "utf8") {
        throw new Error("Only 'utf8' encoding is currently supported");
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX2FwcGVuZEZpbGUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfZnNfYXBwZW5kRmlsZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBRUwsY0FBYyxFQUNkLGFBQWEsR0FFZCxNQUFNLGlCQUFpQixDQUFDO0FBQ3pCLE9BQU8sRUFBYSxjQUFjLEVBQUUsTUFBTSxjQUFjLENBQUM7QUFDekQsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLFlBQVksQ0FBQztBQU16QyxNQUFNLFVBQVUsVUFBVSxDQUN4QixTQUFnQyxFQUNoQyxJQUF5QixFQUN6QixpQkFBbUUsRUFDbkUsUUFBNEI7SUFFNUIsU0FBUyxHQUFHLFNBQVMsWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxTQUFTLENBQUMsQ0FBQyxDQUFDLENBQUMsU0FBUyxDQUFDO0lBQzFFLE1BQU0sVUFBVSxHQUNkLGlCQUFpQixZQUFZLFFBQVEsQ0FBQyxDQUFDLENBQUMsaUJBQWlCLENBQUMsQ0FBQyxDQUFDLFFBQVEsQ0FBQztJQUN2RSxNQUFNLE9BQU8sR0FDWCxpQkFBaUIsWUFBWSxRQUFRLENBQUMsQ0FBQyxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUMsaUJBQWlCLENBQUM7SUFDeEUsSUFBSSxDQUFDLFVBQVUsRUFBRTtRQUNmLE1BQU0sSUFBSSxLQUFLLENBQUMsK0JBQStCLENBQUMsQ0FBQztLQUNsRDtJQUVELGdCQUFnQixDQUFDLE9BQU8sQ0FBQyxDQUFDO0lBQzFCLElBQUksR0FBRyxHQUFHLENBQUMsQ0FBQyxDQUFDO0lBQ2IsTUFBTSxNQUFNLEdBQWUsSUFBSSxZQUFZLFVBQVU7UUFDbkQsQ0FBQyxDQUFDLElBQUk7UUFDTixDQUFDLENBQUMsSUFBSSxXQUFXLEVBQUUsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDbkMsSUFBSSxPQUFPLENBQUMsQ0FBQyxPQUFPLEVBQUUsTUFBTSxFQUFFLEVBQUU7UUFDOUIsSUFBSSxPQUFP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