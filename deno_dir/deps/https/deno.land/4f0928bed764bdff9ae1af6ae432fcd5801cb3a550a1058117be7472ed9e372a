import { Buffer } from "https://deno.land/std@0.79.0/node/buffer.ts";
import { ProviderError } from "../../property-provider/mod.ts";
type RequestOptions = any;

function timeout(promise: Promise<any>, timeoutMs: number, timeoutError: Error) {
  return new Promise(async (resolve, reject) => {
    const timer = setTimeout(() => {
      reject(timeoutError);
    }, timeoutMs);
    try {
      const result = await promise;
      clearTimeout(timer);
      resolve(result);
    } catch (error) {
      reject(error);
      clearTimeout(timer);
    }
  });
}

interface FetchTimeoutOptions extends RequestInit {
  timeout: number;
}

function fetchTimeout(url: string, options: FetchTimeoutOptions): Promise<any> {
  const { timeout: timeoutMs = 3000, ...fetchOptions } = options;

  return timeout(fetch(url, fetchOptions), timeoutMs, new Error("Request timed out"));
}

/**
 * @internal
 */
export async function httpRequest(options: RequestOptions): Promise<Buffer> {
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
  } catch (err) {
    throw Object.assign(new ProviderError("Unable to connect to instance metadata service"), err);
  }
}
