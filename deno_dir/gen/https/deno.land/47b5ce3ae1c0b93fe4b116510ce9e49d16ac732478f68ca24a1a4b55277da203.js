import { sleep } from "./utils/sleep.ts";
import { WaiterState } from "./waiter.ts";
const exponentialBackoffWithJitter = (minDelay, maxDelay, attemptCeiling, attempt) => {
    if (attempt > attemptCeiling)
        return maxDelay;
    const delay = minDelay * 2 ** (attempt - 1);
    return randomInRange(minDelay, delay);
};
const randomInRange = (min, max) => min + Math.random() * (max - min);
export const runPolling = async ({ minDelay, maxDelay, maxWaitTime, abortController, client, abortSignal }, input, acceptorChecks) => {
    const { state } = await acceptorChecks(client, input);
    if (state !== WaiterState.RETRY) {
        return { state };
    }
    let currentAttempt = 1;
    const waitUntil = Date.now() + maxWaitTime * 1000;
    const attemptCeiling = Math.log(maxDelay / minDelay) / Math.log(2) + 1;
    while (true) {
        if (abortController?.signal?.aborted || abortSignal?.aborted) {
            return { state: WaiterState.ABORTED };
        }
        const delay = exponentialBackoffWithJitter(minDelay, maxDelay, attemptCeiling, currentAttempt);
        if (Date.now() + delay * 1000 > waitUntil) {
            return { state: WaiterState.TIMEOUT };
        }
        await sleep(delay);
        const { state } = await acceptorChecks(client, input);
        if (state !== WaiterState.RETRY) {
            return { state };
        }
        currentAttempt += 1;
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicG9sbGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsicG9sbGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxLQUFLLEVBQUUsTUFBTSxrQkFBa0IsQ0FBQztBQUN6QyxPQUFPLEVBQStCLFdBQVcsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUt2RSxNQUFNLDRCQUE0QixHQUFHLENBQUMsUUFBZ0IsRUFBRSxRQUFnQixFQUFFLGNBQXNCLEVBQUUsT0FBZSxFQUFFLEVBQUU7SUFDbkgsSUFBSSxPQUFPLEdBQUcsY0FBYztRQUFFLE9BQU8sUUFBUSxDQUFDO0lBQzlDLE1BQU0sS0FBSyxHQUFHLFFBQVEsR0FBRyxDQUFDLElBQUksQ0FBQyxPQUFPLEdBQUcsQ0FBQyxDQUFDLENBQUM7SUFDNUMsT0FBTyxhQUFhLENBQUMsUUFBUSxFQUFFLEtBQUssQ0FBQyxDQUFDO0FBQ3hDLENBQUMsQ0FBQztBQUVGLE1BQU0sYUFBYSxHQUFHLENBQUMsR0FBVyxFQUFFLEdBQVcsRUFBRSxFQUFFLENBQUMsR0FBRyxHQUFHLElBQUksQ0FBQyxNQUFNLEVBQUUsR0FBRyxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUMsQ0FBQztBQVV0RixNQUFNLENBQUMsTUFBTSxVQUFVLEdBQUcsS0FBSyxFQUM3QixFQUFFLFFBQVEsRUFBRSxRQUFRLEVBQUUsV0FBVyxFQUFFLGVBQWUsRUFBRSxNQUFNLEVBQUUsV0FBVyxFQUF5QixFQUNoRyxLQUFZLEVBQ1osY0FBdUUsRUFDaEQsRUFBRTtJQUN6QixNQUFNLEVBQUUsS0FBSyxFQUFFLEdBQUcsTUFBTSxjQUFjLENBQUMsTUFBTSxFQUFFLEtBQUssQ0FBQyxDQUFDO0lBQ3RELElBQUksS0FBSyxLQUFLLFdBQVcsQ0FBQyxLQUFLLEVBQUU7UUFDL0IsT0FBTyxFQUFFLEtBQUssRUFBRSxDQUFDO0tBQ2xCO0lBRUQsSUFBSSxjQUFjLEdBQUcsQ0FBQyxDQUFDO0lBQ3ZCLE1BQU0sU0FBUyxHQUFHLElBQUksQ0FBQyxHQUFHLEVBQUUsR0FBRyxXQUFXLEdBQUcsSUFBSSxDQUFDO0lBR2xELE1BQU0sY0FBYyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsUUFBUSxHQUFHLFFBQVEsQ0FBQyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQ3ZFLE9BQU8sSUFBSSxFQUFFO1FBQ1gsSUFBSSxlQUFlLEVBQUUsTUFBTSxFQUFFLE9BQU8sSUFBSSxXQUFXLEVBQUUsT0FBTyxFQUFFO1lBQzVELE9BQU8sRUFBRSxLQUFLLEVBQUUsV0FBVyxDQUFDLE9BQU8sRUFBRSxDQUFDO1NBQ3ZDO1FBQ0QsTUFBTSxLQUFLLEdBQUcsNEJBQTRCLENBQUMsUUFBUSxFQUFFLFFBQVEsRUFBRSxjQUFjLEVBQUUsY0FBYyxDQUFDLENBQUM7UUFHL0YsSUFBSSxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsS0FBSyxHQUFHLElBQUksR0FBRyxTQUFTLEVBQUU7WUFDekMsT0FBTyxFQUFFLEtBQUssRUFBRSxXQUFXLENBQUMsT0FBTyxFQUFFLENBQUM7U0FDdkM7UUFDRCxNQUFNLEtBQUssQ0FBQyxLQUFLLENBQUMsQ0FBQztRQUNuQixNQUFNLEVBQUUsS0FBSyxFQUFFLEdBQUcsTUFBTSxjQUFjLENBQUMsTUFBTSxFQUFFLEtBQUssQ0FBQyxDQUFDO1FBQ3RELElBQUksS0FBSyxLQUFLLFdBQVcsQ0FBQyxLQUFLLEVBQUU7WUFDL0IsT0FBTyxFQUFFLEtBQUssRUFBRSxDQUFDO1NBQ2xCO1FBRUQsY0FBYyxJQUFJLENBQUMsQ0FBQztLQUNyQjtBQUNILENBQUMsQ0FBQyJ9