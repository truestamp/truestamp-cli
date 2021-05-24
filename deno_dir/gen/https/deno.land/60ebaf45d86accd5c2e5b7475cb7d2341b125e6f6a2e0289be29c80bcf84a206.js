export const waiterServiceDefaults = {
    minDelay: 2,
    maxDelay: 120,
};
export var WaiterState;
(function (WaiterState) {
    WaiterState["ABORTED"] = "ABORTED";
    WaiterState["FAILURE"] = "FAILURE";
    WaiterState["SUCCESS"] = "SUCCESS";
    WaiterState["RETRY"] = "RETRY";
    WaiterState["TIMEOUT"] = "TIMEOUT";
})(WaiterState || (WaiterState = {}));
export const checkExceptions = (result) => {
    if (result.state === WaiterState.ABORTED) {
        const abortError = new Error(`${JSON.stringify({
            ...result,
            reason: "Request was aborted",
        })}`);
        abortError.name = "AbortError";
        throw abortError;
    }
    else if (result.state === WaiterState.TIMEOUT) {
        const timeoutError = new Error(`${JSON.stringify({
            ...result,
            reason: "Waiter has timed out",
        })}`);
        timeoutError.name = "TimeoutError";
        throw timeoutError;
    }
    else if (result.state !== WaiterState.SUCCESS) {
        throw new Error(`${JSON.stringify({ result })}`);
    }
    return result;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid2FpdGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsid2FpdGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQU9BLE1BQU0sQ0FBQyxNQUFNLHFCQUFxQixHQUFHO0lBQ25DLFFBQVEsRUFBRSxDQUFDO0lBQ1gsUUFBUSxFQUFFLEdBQUc7Q0FDZCxDQUFDO0FBUUYsTUFBTSxDQUFOLElBQVksV0FNWDtBQU5ELFdBQVksV0FBVztJQUNyQixrQ0FBbUIsQ0FBQTtJQUNuQixrQ0FBbUIsQ0FBQTtJQUNuQixrQ0FBbUIsQ0FBQTtJQUNuQiw4QkFBZSxDQUFBO0lBQ2Ysa0NBQW1CLENBQUE7QUFDckIsQ0FBQyxFQU5XLFdBQVcsS0FBWCxXQUFXLFFBTXRCO0FBZUQsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLENBQUMsTUFBb0IsRUFBZ0IsRUFBRTtJQUNwRSxJQUFJLE1BQU0sQ0FBQyxLQUFLLEtBQUssV0FBVyxDQUFDLE9BQU8sRUFBRTtRQUN4QyxNQUFNLFVBQVUsR0FBRyxJQUFJLEtBQUssQ0FDMUIsR0FBRyxJQUFJLENBQUMsU0FBUyxDQUFDO1lBQ2hCLEdBQUcsTUFBTTtZQUNULE1BQU0sRUFBRSxxQkFBcUI7U0FDOUIsQ0FBQyxFQUFFLENBQ0wsQ0FBQztRQUNGLFVBQVUsQ0FBQyxJQUFJLEdBQUcsWUFBWSxDQUFDO1FBQy9CLE1BQU0sVUFBVSxDQUFDO0tBQ2xCO1NBQU0sSUFBSSxNQUFNLENBQUMsS0FBSyxLQUFLLFdBQVcsQ0FBQyxPQUFPLEVBQUU7UUFDL0MsTUFBTSxZQUFZLEdBQUcsSUFBSSxLQUFLLENBQzVCLEdBQUcsSUFBSSxDQUFDLFNBQVMsQ0FBQztZQUNoQixHQUFHLE1BQU07WUFDVCxNQUFNLEVBQUUsc0JBQXNCO1NBQy9CLENBQUMsRUFBRSxDQUNMLENBQUM7UUFDRixZQUFZLENBQUMsSUFBSSxHQUFHLGNBQWMsQ0FBQztRQUNuQyxNQUFNLFlBQVksQ0FBQztLQUNwQjtTQUFNLElBQUksTUFBTSxDQUFDLEtBQUssS0FBSyxXQUFXLENBQUMsT0FBTyxFQUFFO1FBQy9DLE1BQU0sSUFBSSxLQUFLLENBQUMsR0FBRyxJQUFJLENBQUMsU0FBUyxDQUFDLEVBQUUsTUFBTSxFQUFFLENBQUMsRUFBRSxDQUFDLENBQUM7S0FDbEQ7SUFDRCxPQUFPLE1BQU0sQ0FBQztBQUNoQixDQUFDLENBQUMifQ==