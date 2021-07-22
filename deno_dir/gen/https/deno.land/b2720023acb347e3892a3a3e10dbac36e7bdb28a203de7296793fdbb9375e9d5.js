import { HttpRequest } from "../protocol-http/mod.ts";
import { INVOCATION_ID_HEADER, REQUEST_HEADER } from "./constants.ts";
export const omitRetryHeadersMiddleware = () => (next) => async (args) => {
    const { request } = args;
    if (HttpRequest.isInstance(request)) {
        delete request.headers[INVOCATION_ID_HEADER];
        delete request.headers[REQUEST_HEADER];
    }
    return next(args);
};
export const omitRetryHeadersMiddlewareOptions = {
    name: "omitRetryHeadersMiddleware",
    tags: ["RETRY", "HEADERS", "OMIT_RETRY_HEADERS"],
    relation: "before",
    toMiddleware: "awsAuthMiddleware",
    override: true,
};
export const getOmitRetryHeadersPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.addRelativeTo(omitRetryHeadersMiddleware(), omitRetryHeadersMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoib21pdFJldHJ5SGVhZGVyc01pZGRsZXdhcmUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJvbWl0UmV0cnlIZWFkZXJzTWlkZGxld2FyZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFVdEQsT0FBTyxFQUFFLG9CQUFvQixFQUFFLGNBQWMsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBRXRFLE1BQU0sQ0FBQyxNQUFNLDBCQUEwQixHQUNyQyxHQUFHLEVBQUUsQ0FDTCxDQUFpRCxJQUFrQyxFQUFnQyxFQUFFLENBQ3JILEtBQUssRUFBRSxJQUFtQyxFQUEwQyxFQUFFO0lBQ3BGLE1BQU0sRUFBRSxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUM7SUFDekIsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ25DLE9BQU8sT0FBTyxDQUFDLE9BQU8sQ0FBQyxvQkFBb0IsQ0FBQyxDQUFDO1FBQzdDLE9BQU8sT0FBTyxDQUFDLE9BQU8sQ0FBQyxjQUFjLENBQUMsQ0FBQztLQUN4QztJQUNELE9BQU8sSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO0FBQ3BCLENBQUMsQ0FBQztBQUVKLE1BQU0sQ0FBQyxNQUFNLGlDQUFpQyxHQUE4QjtJQUMxRSxJQUFJLEVBQUUsNEJBQTRCO0lBQ2xDLElBQUksRUFBRSxDQUFDLE9BQU8sRUFBRSxTQUFTLEVBQUUsb0JBQW9CLENBQUM7SUFDaEQsUUFBUSxFQUFFLFFBQVE7SUFDbEIsWUFBWSxFQUFFLG1CQUFtQjtJQUNqQyxRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSx5QkFBeUIsR0FBRyxDQUFDLE9BQWdCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ25GLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxhQUFhLENBQUMsMEJBQTBCLEVBQUUsRUFBRSxpQ0FBaUMsQ0FBQyxDQUFDO0lBQzdGLENBQUM7Q0FDRixDQUFDLENBQUMifQ==