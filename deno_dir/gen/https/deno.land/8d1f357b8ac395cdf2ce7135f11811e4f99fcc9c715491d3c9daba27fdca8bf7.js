import { Buffer } from "https://deno.land/std@0.79.0/node/buffer.ts";
import { ProviderError } from "../../property-provider/mod.ts";
function timeout(promise, timeoutMs, timeoutError) {
    return new Promise(async (resolve, reject) => {
        const timer = setTimeout(() => {
            reject(timeoutError);
        }, timeoutMs);
        try {
            const result = await promise;
            clearTimeout(timer);
            resolve(result);
        }
        catch (error) {
            reject(error);
            clearTimeout(timer);
        }
    });
}
function fetchTimeout(url, options) {
    const { timeout: timeoutMs = 3000, ...fetchOptions } = options;
    return timeout(fetch(url, fetchOptions), timeoutMs, new Error("Request timed out"));
}
export async function httpRequest(options) {
    const { host, port = 80, path, method = "GET", headers = {}, timeout } = options;
    try {
        const response = await fetchTimeout(`http://${host}:${port}${path}`, {
            method,
            headers: new Headers(headers),
            timeout,
        });
        const { status: statusCode = 400 } = response;
        if (statusCode < 200 || 300 <= statusCode) {
            throw Object.assign(new ProviderError("Error response received from instance metadata service"), { statusCode });
        }
        return Buffer.from(await response.text());
    }
    catch (err) {
        throw Object.assign(new ProviderError("Unable to connect to instance metadata service"), err);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaHR0cFJlcXVlc3QuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJodHRwUmVxdWVzdC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sNkNBQTZDLENBQUM7QUFDckUsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLGdDQUFnQyxDQUFDO0FBRy9ELFNBQVMsT0FBTyxDQUFDLE9BQXFCLEVBQUUsU0FBaUIsRUFBRSxZQUFtQjtJQUM1RSxPQUFPLElBQUksT0FBTyxDQUFDLEtBQUssRUFBRSxPQUFPLEVBQUUsTUFBTSxFQUFFLEVBQUU7UUFDM0MsTUFBTSxLQUFLLEdBQUcsVUFBVSxDQUFDLEdBQUcsRUFBRTtZQUM1QixNQUFNLENBQUMsWUFBWSxDQUFDLENBQUM7UUFDdkIsQ0FBQyxFQUFFLFNBQVMsQ0FBQyxDQUFDO1FBQ2QsSUFBSTtZQUNGLE1BQU0sTUFBTSxHQUFHLE1BQU0sT0FBTyxDQUFDO1lBQzdCLFlBQVksQ0FBQyxLQUFLLENBQUMsQ0FBQztZQUNwQixPQUFPLENBQUMsTUFBTSxDQUFDLENBQUM7U0FDakI7UUFBQyxPQUFPLEtBQUssRUFBRTtZQUNkLE1BQU0sQ0FBQyxLQUFLLENBQUMsQ0FBQztZQUNkLFlBQVksQ0FBQyxLQUFLLENBQUMsQ0FBQztTQUNyQjtJQUNILENBQUMsQ0FBQyxDQUFDO0FBQ0wsQ0FBQztBQU1ELFNBQVMsWUFBWSxDQUFDLEdBQVcsRUFBRSxPQUE0QjtJQUM3RCxNQUFNLEVBQUUsT0FBTyxFQUFFLFNBQVMsR0FBRyxJQUFJLEVBQUUsR0FBRyxZQUFZLEVBQUUsR0FBRyxPQUFPLENBQUM7SUFFL0QsT0FBTyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsRUFBRSxZQUFZLENBQUMsRUFBRSxTQUFTLEVBQUUsSUFBSSxLQUFLLENBQUMsbUJBQW1CLENBQUMsQ0FBQyxDQUFDO0FBQ3RGLENBQUM7QUFLRCxNQUFNLENBQUMsS0FBSyxVQUFVLFdBQVcsQ0FBQyxPQUF1QjtJQUN2RCxNQUFNLEVBQUUsSUFBSSxFQUFFLElBQUksR0FBRyxFQUFFLEVBQUUsSUFBSSxFQUFFLE1BQU0sR0FBRyxLQUFLLEVBQUUsT0FBTyxHQUFHLEVBQUUsRUFBRSxPQUFPLEVBQUUsR0FBRyxPQUFPLENBQUM7SUFFakYsSUFBSTtRQUNGLE1BQU0sUUFBUSxHQUFHLE1BQU0sWUFBWSxDQUFDLFVBQVUsSUFBSSxJQUFJLElBQUksR0FBRyxJQUFJLEVBQUUsRUFBRTtZQUNuRSxNQUFNO1lBQ04sT0FBTyxFQUFFLElBQUksT0FBTyxDQUFDLE9BQU8sQ0FBQztZQUM3QixPQUFPO1NBQ1IsQ0FBQyxDQUFDO1FBRUgsTUFBTSxFQUFFLE1BQU0sRUFBRSxVQUFVLEdBQUcsR0FBRyxFQUFFLEdBQUcsUUFBUSxDQUFDO1FBQzlDLElBQUksVUFBVSxHQUFHLEdBQUcsSUFBSSxHQUFHLElBQUksVUFBVSxFQUFFO1lBQ3pDLE1BQU0sTUFBTSxDQUFDLE1BQU0sQ0FBQyxJQUFJLGFBQWEsQ0FBQyx3REFBd0QsQ0FBQyxFQUFFLEVBQUUsVUFBVSxFQUFFLENBQUMsQ0FBQztTQUNsSDtRQUVELE9BQU8sTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLFFBQVEsQ0FBQyxJQUFJLEVBQUUsQ0FBQyxDQUFDO0tBQzNDO0lBQUMsT0FBTyxHQUFHLEVBQUU7UUFDWixNQUFNLE1BQU0sQ0FBQyxNQUFNLENBQUMsSUFBSSxhQUFhLENBQUMsZ0RBQWdELENBQUMsRUFBRSxHQUFHLENBQUMsQ0FBQztLQUMvRjtBQUNILENBQUMifQ==