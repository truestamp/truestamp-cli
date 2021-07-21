import { isArrayBuffer } from "../is-array-buffer/mod.ts";
import { HttpRequest } from "../protocol-http/mod.ts";
export const applyMd5BodyChecksumMiddleware = (options) => (next) => async (args) => {
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
const hasHeader = (soughtHeader, headers) => {
    soughtHeader = soughtHeader.toLowerCase();
    for (const headerName of Object.keys(headers)) {
        if (soughtHeader === headerName.toLowerCase()) {
            return true;
        }
    }
    return false;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXBwbHlNZDVCb2R5Q2hlY2tzdW1NaWRkbGV3YXJlLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiYXBwbHlNZDVCb2R5Q2hlY2tzdW1NaWRkbGV3YXJlLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUMxRCxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFjdEQsTUFBTSxDQUFDLE1BQU0sOEJBQThCLEdBQ3pDLENBQUMsT0FBc0MsRUFBNkIsRUFBRSxDQUN0RSxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQzlGLEtBQUssRUFBRSxJQUFnQyxFQUF1QyxFQUFFO0lBQzlFLElBQUksRUFBRSxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUM7SUFDdkIsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ25DLE1BQU0sRUFBRSxJQUFJLEVBQUUsT0FBTyxFQUFFLEdBQUcsT0FBTyxDQUFDO1FBQ2xDLElBQUksQ0FBQyxTQUFTLENBQUMsYUFBYSxFQUFFLE9BQU8sQ0FBQyxFQUFFO1lBQ3RDLElBQUksTUFBMkIsQ0FBQztZQUNoQyxJQUFJLElBQUksS0FBSyxTQUFTLElBQUksT0FBTyxJQUFJLEtBQUssUUFBUSxJQUFJLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksYUFBYSxDQUFDLElBQUksQ0FBQyxFQUFFO2dCQUNyRyxNQUFNLElBQUksR0FBRyxJQUFJLE9BQU8sQ0FBQyxHQUFHLEVBQUUsQ0FBQztnQkFDL0IsSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLElBQUksRUFBRSxDQUFDLENBQUM7Z0JBQ3hCLE1BQU0sR0FBRyxJQUFJLENBQUMsTUFBTSxFQUFFLENBQUM7YUFDeEI7aUJBQU07Z0JBQ0wsTUFBTSxHQUFHLE9BQU8sQ0FBQyxZQUFZLENBQUMsT0FBTyxDQUFDLEdBQUcsRUFBRSxJQUFJLENBQUMsQ0FBQzthQUNsRDtZQUVELE9BQU8sR0FBRztnQkFDUixHQUFHLE9BQU87Z0JBQ1YsT0FBTyxFQUFFO29CQUNQLEdBQUcsT0FBTztvQkFDVixhQUFhLEVBQUUsT0FBTyxDQUFDLGFBQWEsQ0FBQyxNQUFNLE1BQU0sQ0FBQztpQkFDbkQ7YUFDRixDQUFDO1NBQ0g7S0FDRjtJQUNELE9BQU8sSUFBSSxDQUFDO1FBQ1YsR0FBRyxJQUFJO1FBQ1AsT0FBTztLQUNSLENBQUMsQ0FBQztBQUNMLENBQUMsQ0FBQztBQUVKLE1BQU0sQ0FBQyxNQUFNLHFDQUFxQyxHQUF3QjtJQUN4RSxJQUFJLEVBQUUsZ0NBQWdDO0lBQ3RDLElBQUksRUFBRSxPQUFPO0lBQ2IsSUFBSSxFQUFFLENBQUMsaUJBQWlCLEVBQUUsZUFBZSxDQUFDO0lBQzFDLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLDZCQUE2QixHQUFHLENBQUMsTUFBcUMsRUFBdUIsRUFBRSxDQUFDLENBQUM7SUFDNUcsWUFBWSxFQUFFLENBQUMsV0FBVyxFQUFFLEVBQUU7UUFDNUIsV0FBVyxDQUFDLEdBQUcsQ0FBQyw4QkFBOEIsQ0FBQyxNQUFNLENBQUMsRUFBRSxxQ0FBcUMsQ0FBQyxDQUFDO0lBQ2pHLENBQUM7Q0FDRixDQUFDLENBQUM7QUFFSCxNQUFNLFNBQVMsR0FBRyxDQUFDLFlBQW9CLEVBQUUsT0FBa0IsRUFBVyxFQUFFO0lBQ3RFLFlBQVksR0FBRyxZQUFZLENBQUMsV0FBVyxFQUFFLENBQUM7SUFDMUMsS0FBSyxNQUFNLFVBQVUsSUFBSSxNQUFNLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQzdDLElBQUksWUFBWSxLQUFLLFVBQVUsQ0FBQyxXQUFXLEVBQUUsRUFBRTtZQUM3QyxPQUFPLElBQUksQ0FBQztTQUNiO0tBQ0Y7SUFFRCxPQUFPLEtBQUssQ0FBQztBQUNmLENBQUMsQ0FBQyJ9