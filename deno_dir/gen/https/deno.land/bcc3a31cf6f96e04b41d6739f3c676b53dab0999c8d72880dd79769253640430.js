import { HttpResponse } from "../protocol-http/mod.ts";
import { buildQueryString } from "../querystring-builder/mod.ts";
import { requestTimeout } from "./request-timeout.ts";
export class FetchHttpHandler {
    requestTimeout;
    constructor({ requestTimeout } = {}) {
        this.requestTimeout = requestTimeout;
    }
    destroy() {
    }
    handle(request, { abortSignal } = {}) {
        const requestTimeoutInMs = this.requestTimeout;
        if (abortSignal?.aborted) {
            const abortError = new Error("Request aborted");
            abortError.name = "AbortError";
            return Promise.reject(abortError);
        }
        let path = request.path;
        if (request.query) {
            const queryString = buildQueryString(request.query);
            if (queryString) {
                path += `?${queryString}`;
            }
        }
        const { port, method } = request;
        const url = `${request.protocol}//${request.hostname}${port ? `:${port}` : ""}${path}`;
        const body = method === "GET" || method === "HEAD" ? undefined : request.body;
        const requestOptions = {
            body,
            headers: new Headers(request.headers),
            method: method,
        };
        if (typeof AbortController !== "undefined") {
            requestOptions["signal"] = abortSignal;
        }
        const fetchRequest = new Request(url, requestOptions);
        const raceOfPromises = [
            fetch(fetchRequest).then((response) => {
                const fetchHeaders = response.headers;
                const transformedHeaders = {};
                for (const pair of fetchHeaders.entries()) {
                    transformedHeaders[pair[0]] = pair[1];
                }
                const hasReadableStream = response.body !== undefined;
                if (!hasReadableStream) {
                    return response.blob().then((body) => ({
                        response: new HttpResponse({
                            headers: transformedHeaders,
                            statusCode: response.status,
                            body,
                        }),
                    }));
                }
                return {
                    response: new HttpResponse({
                        headers: transformedHeaders,
                        statusCode: response.status,
                        body: response.body,
                    }),
                };
            }),
            requestTimeout(requestTimeoutInMs),
        ];
        if (abortSignal) {
            raceOfPromises.push(new Promise((resolve, reject) => {
                abortSignal.onabort = () => {
                    const abortError = new Error("Request aborted");
                    abortError.name = "AbortError";
                    reject(abortError);
                };
            }));
        }
        return Promise.race(raceOfPromises);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZmV0Y2gtaHR0cC1oYW5kbGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZmV0Y2gtaHR0cC1oYW5kbGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBNEIsWUFBWSxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFDakYsT0FBTyxFQUFFLGdCQUFnQixFQUFFLE1BQU0sK0JBQStCLENBQUM7QUFHakUsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLHNCQUFzQixDQUFDO0FBZXRELE1BQU0sT0FBTyxnQkFBZ0I7SUFDVixjQUFjLENBQVU7SUFFekMsWUFBWSxFQUFFLGNBQWMsS0FBOEIsRUFBRTtRQUMxRCxJQUFJLENBQUMsY0FBYyxHQUFHLGNBQWMsQ0FBQztJQUN2QyxDQUFDO0lBRUQsT0FBTztJQUVQLENBQUM7SUFFRCxNQUFNLENBQUMsT0FBb0IsRUFBRSxFQUFFLFdBQVcsS0FBeUIsRUFBRTtRQUNuRSxNQUFNLGtCQUFrQixHQUFHLElBQUksQ0FBQyxjQUFjLENBQUM7UUFHL0MsSUFBSSxXQUFXLEVBQUUsT0FBTyxFQUFFO1lBQ3hCLE1BQU0sVUFBVSxHQUFHLElBQUksS0FBSyxDQUFDLGlCQUFpQixDQUFDLENBQUM7WUFDaEQsVUFBVSxDQUFDLElBQUksR0FBRyxZQUFZLENBQUM7WUFDL0IsT0FBTyxPQUFPLENBQUMsTUFBTSxDQUFDLFVBQVUsQ0FBQyxDQUFDO1NBQ25DO1FBRUQsSUFBSSxJQUFJLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQztRQUN4QixJQUFJLE9BQU8sQ0FBQyxLQUFLLEVBQUU7WUFDakIsTUFBTSxXQUFXLEdBQUcsZ0JBQWdCLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxDQUFDO1lBQ3BELElBQUksV0FBVyxFQUFFO2dCQUNmLElBQUksSUFBSSxJQUFJLFdBQVcsRUFBRSxDQUFDO2FBQzNCO1NBQ0Y7UUFFRCxNQUFNLEVBQUUsSUFBSSxFQUFFLE1BQU0sRUFBRSxHQUFHLE9BQU8sQ0FBQztRQUNqQyxNQUFNLEdBQUcsR0FBRyxHQUFHLE9BQU8sQ0FBQyxRQUFRLEtBQUssT0FBTyxDQUFDLFFBQVEsR0FBRyxJQUFJLENBQUMsQ0FBQyxDQUFDLElBQUksSUFBSSxFQUFFLENBQUMsQ0FBQyxDQUFDLEVBQUUsR0FBRyxJQUFJLEVBQUUsQ0FBQztRQUd2RixNQUFNLElBQUksR0FBRyxNQUFNLEtBQUssS0FBSyxJQUFJLE1BQU0sS0FBSyxNQUFNLENBQUMsQ0FBQyxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQztRQUM5RSxNQUFNLGNBQWMsR0FBZ0I7WUFDbEMsSUFBSTtZQUNKLE9BQU8sRUFBRSxJQUFJLE9BQU8sQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDO1lBQ3JDLE1BQU0sRUFBRSxNQUFNO1NBQ2YsQ0FBQztRQUdGLElBQUksT0FBTyxlQUFlLEtBQUssV0FBVyxFQUFFO1lBQ3pDLGNBQXNCLENBQUMsUUFBUSxDQUFDLEdBQUcsV0FBVyxDQUFDO1NBQ2pEO1FBRUQsTUFBTSxZQUFZLEdBQUcsSUFBSSxPQUFPLENBQUMsR0FBRyxFQUFFLGNBQWMsQ0FBQyxDQUFDO1FBQ3RELE1BQU0sY0FBYyxHQUFHO1lBQ3JCLEtBQUssQ0FBQyxZQUFZLENBQUMsQ0FBQyxJQUFJLENBQUMsQ0FBQyxRQUFRLEVBQUUsRUFBRTtnQkFDcEMsTUFBTSxZQUFZLEdBQVEsUUFBUSxDQUFDLE9BQU8sQ0FBQztnQkFDM0MsTUFBTSxrQkFBa0IsR0FBYyxFQUFFLENBQUM7Z0JBRXpDLEtBQUssTUFBTSxJQUFJLElBQXFCLFlBQVksQ0FBQyxPQUFPLEVBQUUsRUFBRTtvQkFDMUQsa0JBQWtCLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLEdBQUcsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDO2lCQUN2QztnQkFFRCxNQUFNLGlCQUFpQixHQUFHLFFBQVEsQ0FBQyxJQUFJLEtBQUssU0FBUyxDQUFDO2dCQUd0RCxJQUFJLENBQUMsaUJBQWlCLEVBQUU7b0JBQ3RCLE9BQU8sUUFBUSxDQUFDLElBQUksRUFBRSxDQUFDLElBQUksQ0FBQyxDQUFDLElBQUksRUFBRSxFQUFFLENBQUMsQ0FBQzt3QkFDckMsUUFBUSxFQUFFLElBQUksWUFBWSxDQUFDOzRCQUN6QixPQUFPLEVBQUUsa0JBQWtCOzRCQUMzQixVQUFVLEVBQUUsUUFBUSxDQUFDLE1BQU07NEJBQzNCLElBQUk7eUJBQ0wsQ0FBQztxQkFDSCxDQUFDLENBQUMsQ0FBQztpQkFDTDtnQkFFRCxPQUFPO29CQUNMLFFBQVEsRUFBRSxJQUFJLFlBQVksQ0FBQzt3QkFDekIsT0FBTyxFQUFFLGtCQUFrQjt3QkFDM0IsVUFBVSxFQUFFLFFBQVEsQ0FBQyxNQUFNO3dCQUMzQixJQUFJLEVBQUUsUUFBUSxDQUFDLElBQUk7cUJBQ3BCLENBQUM7aUJBQ0gsQ0FBQztZQUNKLENBQUMsQ0FBQztZQUNGLGNBQWMsQ0FBQyxrQkFBa0IsQ0FBQztTQUNuQyxDQUFDO1FBQ0YsSUFBSSxXQUFXLEVBQUU7WUFDZixjQUFjLENBQUMsSUFBSSxDQUNqQixJQUFJLE9BQU8sQ0FBUSxDQUFDLE9BQU8sRUFBRSxNQUFNLEVBQUUsRUFBRTtnQkFDckMsV0FBVyxDQUFDLE9BQU8sR0FBRyxHQUFHLEVBQUU7b0JBQ3pCLE1BQU0sVUFBVSxHQUFHLElBQUksS0FBSyxDQUFDLGlCQUFpQixDQUFDLENBQUM7b0JBQ2hELFVBQVUsQ0FBQyxJQUFJLEdBQUcsWUFBWSxDQUFDO29CQUMvQixNQUFNLENBQUMsVUFBVSxDQUFDLENBQUM7Z0JBQ3JCLENBQUMsQ0FBQztZQUNKLENBQUMsQ0FBQyxDQUNILENBQUM7U0FDSDtRQUNELE9BQU8sT0FBTyxDQUFDLElBQUksQ0FBQyxjQUFjLENBQUMsQ0FBQztJQUN0QyxDQUFDO0NBQ0YifQ==