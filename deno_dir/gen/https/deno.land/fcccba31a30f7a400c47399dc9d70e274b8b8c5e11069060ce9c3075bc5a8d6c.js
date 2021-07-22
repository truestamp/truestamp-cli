import { HttpResponse } from "../protocol-http/mod.ts";
export const throw200ExceptionsMiddleware = (config) => (next) => async (args) => {
    const result = await next(args);
    const { response } = result;
    if (!HttpResponse.isInstance(response))
        return result;
    const { statusCode, body } = response;
    if (statusCode < 200 && statusCode >= 300)
        return result;
    const bodyBytes = await collectBody(body, config);
    const bodyString = await collectBodyString(bodyBytes, config);
    if (bodyBytes.length === 0) {
        const err = new Error("S3 aborted request");
        err.name = "InternalError";
        throw err;
    }
    if (bodyString && bodyString.match("<Error>")) {
        response.statusCode = 400;
    }
    response.body = bodyBytes;
    return result;
};
const collectBody = (streamBody = new Uint8Array(), context) => {
    if (streamBody instanceof Uint8Array) {
        return Promise.resolve(streamBody);
    }
    return context.streamCollector(streamBody) || Promise.resolve(new Uint8Array());
};
const collectBodyString = (streamBody, context) => collectBody(streamBody, context).then((body) => context.utf8Encoder(body));
export const throw200ExceptionsMiddlewareOptions = {
    relation: "after",
    toMiddleware: "deserializerMiddleware",
    tags: ["THROW_200_EXCEPTIONS", "S3"],
    name: "throw200ExceptionsMiddleware",
    override: true,
};
export const getThrow200ExceptionsPlugin = (config) => ({
    applyToStack: (clientStack) => {
        clientStack.addRelativeTo(throw200ExceptionsMiddleware(config), throw200ExceptionsMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidGhyb3ctMjAwLWV4Y2VwdGlvbnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJ0aHJvdy0yMDAtZXhjZXB0aW9ucy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsWUFBWSxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFhdkQsTUFBTSxDQUFDLE1BQU0sNEJBQTRCLEdBQUcsQ0FBQyxNQUEwQixFQUFtQyxFQUFFLENBQUMsQ0FDM0csSUFBSSxFQUNKLEVBQUUsQ0FBQyxLQUFLLEVBQUUsSUFBSSxFQUFFLEVBQUU7SUFDbEIsTUFBTSxNQUFNLEdBQUcsTUFBTSxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDaEMsTUFBTSxFQUFFLFFBQVEsRUFBRSxHQUFHLE1BQU0sQ0FBQztJQUM1QixJQUFJLENBQUMsWUFBWSxDQUFDLFVBQVUsQ0FBQyxRQUFRLENBQUM7UUFBRSxPQUFPLE1BQU0sQ0FBQztJQUN0RCxNQUFNLEVBQUUsVUFBVSxFQUFFLElBQUksRUFBRSxHQUFHLFFBQVEsQ0FBQztJQUN0QyxJQUFJLFVBQVUsR0FBRyxHQUFHLElBQUksVUFBVSxJQUFJLEdBQUc7UUFBRSxPQUFPLE1BQU0sQ0FBQztJQUd6RCxNQUFNLFNBQVMsR0FBRyxNQUFNLFdBQVcsQ0FBQyxJQUFJLEVBQUUsTUFBTSxDQUFDLENBQUM7SUFDbEQsTUFBTSxVQUFVLEdBQUcsTUFBTSxpQkFBaUIsQ0FBQyxTQUFTLEVBQUUsTUFBTSxDQUFDLENBQUM7SUFDOUQsSUFBSSxTQUFTLENBQUMsTUFBTSxLQUFLLENBQUMsRUFBRTtRQUMxQixNQUFNLEdBQUcsR0FBRyxJQUFJLEtBQUssQ0FBQyxvQkFBb0IsQ0FBQyxDQUFDO1FBQzVDLEdBQUcsQ0FBQyxJQUFJLEdBQUcsZUFBZSxDQUFDO1FBQzNCLE1BQU0sR0FBRyxDQUFDO0tBQ1g7SUFDRCxJQUFJLFVBQVUsSUFBSSxVQUFVLENBQUMsS0FBSyxDQUFDLFNBQVMsQ0FBQyxFQUFFO1FBRTdDLFFBQVEsQ0FBQyxVQUFVLEdBQUcsR0FBRyxDQUFDO0tBQzNCO0lBSUQsUUFBUSxDQUFDLElBQUksR0FBRyxTQUFTLENBQUM7SUFDMUIsT0FBTyxNQUFNLENBQUM7QUFDaEIsQ0FBQyxDQUFDO0FBR0YsTUFBTSxXQUFXLEdBQUcsQ0FBQyxhQUFrQixJQUFJLFVBQVUsRUFBRSxFQUFFLE9BQTJCLEVBQXVCLEVBQUU7SUFDM0csSUFBSSxVQUFVLFlBQVksVUFBVSxFQUFFO1FBQ3BDLE9BQU8sT0FBTyxDQUFDLE9BQU8sQ0FBQyxVQUFVLENBQUMsQ0FBQztLQUNwQztJQUNELE9BQU8sT0FBTyxDQUFDLGVBQWUsQ0FBQyxVQUFVLENBQUMsSUFBSSxPQUFPLENBQUMsT0FBTyxDQUFDLElBQUksVUFBVSxFQUFFLENBQUMsQ0FBQztBQUNsRixDQUFDLENBQUM7QUFHRixNQUFNLGlCQUFpQixHQUFHLENBQUMsVUFBZSxFQUFFLE9BQTJCLEVBQW1CLEVBQUUsQ0FDMUYsV0FBVyxDQUFDLFVBQVUsRUFBRSxPQUFPLENBQUMsQ0FBQyxJQUFJLENBQUMsQ0FBQyxJQUFJLEVBQUUsRUFBRSxDQUFDLE9BQU8sQ0FBQyxXQUFXLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztBQUs3RSxNQUFNLENBQUMsTUFBTSxtQ0FBbUMsR0FBOEI7SUFDNUUsUUFBUSxFQUFFLE9BQU87SUFDakIsWUFBWSxFQUFFLHdCQUF3QjtJQUN0QyxJQUFJLEVBQUUsQ0FBQyxzQkFBc0IsRUFBRSxJQUFJLENBQUM7SUFDcEMsSUFBSSxFQUFFLDhCQUE4QjtJQUNwQyxRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFNRixNQUFNLENBQUMsTUFBTSwyQkFBMkIsR0FBRyxDQUFDLE1BQTBCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQy9GLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxhQUFhLENBQUMsNEJBQTRCLENBQUMsTUFBTSxDQUFDLEVBQUUsbUNBQW1DLENBQUMsQ0FBQztJQUN2RyxDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=