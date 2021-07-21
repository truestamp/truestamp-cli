import { isArrayBuffer } from "../is-array-buffer/mod.ts";
import { HttpRequest } from "../protocol-http/mod.ts";
export function applyMd5BodyChecksumMiddleware(options) {
    return (next) => async (args) => {
        let { request } = args;
        if (HttpRequest.isInstance(request)) {
            const { body, headers } = request;
            if (!hasHeader("Content-MD5", headers)) {
                let digest;
                if (body === undefined || typeof body === "string" || ArrayBuffer.isView(body) || isArrayBuffer(body)) {
                    const hash = new options.md5();
                    hash.update(body || "");
                    digest = hash.digest();
                }
                else {
                    digest = options.streamHasher(options.md5, body);
                }
                request = {
                    ...request,
                    headers: {
                        ...headers,
                        "Content-MD5": options.base64Encoder(await digest),
                    },
                };
            }
        }
        return next({
            ...args,
            request,
        });
    };
}
export const applyMd5BodyChecksumMiddlewareOptions = {
    name: "applyMd5BodyChecksumMiddleware",
    step: "build",
    tags: ["SET_CONTENT_MD5", "BODY_CHECKSUM"],
    override: true,
};
export const getApplyMd5BodyChecksumPlugin = (config) => ({
    applyToStack: (clientStack) => {
        clientStack.add(applyMd5BodyChecksumMiddleware(config), applyMd5BodyChecksumMiddlewareOptions);
    },
});
function hasHeader(soughtHeader, headers) {
    soughtHeader = soughtHeader.toLowerCase();
    for (const headerName of Object.keys(headers)) {
        if (soughtHeader === headerName.toLowerCase()) {
            return true;
        }
    }
    return false;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXBwbHlNZDVCb2R5Q2hlY2tzdW1NaWRkbGV3YXJlLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiYXBwbHlNZDVCb2R5Q2hlY2tzdW1NaWRkbGV3YXJlLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUMxRCxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFjdEQsTUFBTSxVQUFVLDhCQUE4QixDQUFDLE9BQXNDO0lBQ25GLE9BQU8sQ0FBZ0MsSUFBK0IsRUFBNkIsRUFBRSxDQUFDLEtBQUssRUFDekcsSUFBZ0MsRUFDSyxFQUFFO1FBQ3ZDLElBQUksRUFBRSxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUM7UUFDdkIsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1lBQ25DLE1BQU0sRUFBRSxJQUFJLEVBQUUsT0FBTyxFQUFFLEdBQUcsT0FBTyxDQUFDO1lBQ2xDLElBQUksQ0FBQyxTQUFTLENBQUMsYUFBYSxFQUFFLE9BQU8sQ0FBQyxFQUFFO2dCQUN0QyxJQUFJLE1BQTJCLENBQUM7Z0JBQ2hDLElBQUksSUFBSSxLQUFLLFNBQVMsSUFBSSxPQUFPLElBQUksS0FBSyxRQUFRLElBQUksV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxhQUFhLENBQUMsSUFBSSxDQUFDLEVBQUU7b0JBQ3JHLE1BQU0sSUFBSSxHQUFHLElBQUksT0FBTyxDQUFDLEdBQUcsRUFBRSxDQUFDO29CQUMvQixJQUFJLENBQUMsTUFBTSxDQUFDLElBQUksSUFBSSxFQUFFLENBQUMsQ0FBQztvQkFDeEIsTUFBTSxHQUFHLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQztpQkFDeEI7cUJBQU07b0JBQ0wsTUFBTSxHQUFHLE9BQU8sQ0FBQyxZQUFZLENBQUMsT0FBTyxDQUFDLEdBQUcsRUFBRSxJQUFJLENBQUMsQ0FBQztpQkFDbEQ7Z0JBRUQsT0FBTyxHQUFHO29CQUNSLEdBQUcsT0FBTztvQkFDVixPQUFPLEVBQUU7d0JBQ1AsR0FBRyxPQUFPO3dCQUNWLGFBQWEsRUFBRSxPQUFPLENBQUMsYUFBYSxDQUFDLE1BQU0sTUFBTSxDQUFDO3FCQUNuRDtpQkFDRixDQUFDO2FBQ0g7U0FDRjtRQUNELE9BQU8sSUFBSSxDQUFDO1lBQ1YsR0FBRyxJQUFJO1lBQ1AsT0FBTztTQUNSLENBQUMsQ0FBQztJQUNMLENBQUMsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLENBQUMsTUFBTSxxQ0FBcUMsR0FBd0I7SUFDeEUsSUFBSSxFQUFFLGdDQUFnQztJQUN0QyxJQUFJLEVBQUUsT0FBTztJQUNiLElBQUksRUFBRSxDQUFDLGlCQUFpQixFQUFFLGVBQWUsQ0FBQztJQUMxQyxRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSw2QkFBNkIsR0FBRyxDQUFDLE1BQXFDLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQzVHLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsOEJBQThCLENBQUMsTUFBTSxDQUFDLEVBQUUscUNBQXFDLENBQUMsQ0FBQztJQUNqRyxDQUFDO0NBQ0YsQ0FBQyxDQUFDO0FBRUgsU0FBUyxTQUFTLENBQUMsWUFBb0IsRUFBRSxPQUFrQjtJQUN6RCxZQUFZLEdBQUcsWUFBWSxDQUFDLFdBQVcsRUFBRSxDQUFDO0lBQzFDLEtBQUssTUFBTSxVQUFVLElBQUksTUFBTSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsRUFBRTtRQUM3QyxJQUFJLFlBQVksS0FBSyxVQUFVLENBQUMsV0FBVyxFQUFFLEVBQUU7WUFDN0MsT0FBTyxJQUFJLENBQUM7U0FDYjtLQUNGO0lBRUQsT0FBTyxLQUFLLENBQUM7QUFDZixDQUFDIn0=