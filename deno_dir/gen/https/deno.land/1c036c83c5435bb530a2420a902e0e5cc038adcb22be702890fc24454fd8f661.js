import { runPolling } from "./poller.ts";
import { validateWaiterOptions } from "./utils/index.ts";
import { waiterServiceDefaults, WaiterState } from "./waiter.ts";
const abortTimeout = async (abortSignal) => {
    return new Promise((resolve) => {
        abortSignal.onabort = () => resolve({ state: WaiterState.ABORTED });
    });
};
export const createWaiter = async (options, input, acceptorChecks) => {
    const params = {
        ...waiterServiceDefaults,
        ...options,
    };
    validateWaiterOptions(params);
    const exitConditions = [runPolling(params, input, acceptorChecks)];
    if (options.abortController) {
        exitConditions.push(abortTimeout(options.abortController.signal));
    }
    if (options.abortSignal) {
        exitConditions.push(abortTimeout(options.abortSignal));
    }
    return Promise.race(exitConditions);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY3JlYXRlV2FpdGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiY3JlYXRlV2FpdGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSxhQUFhLENBQUM7QUFDekMsT0FBTyxFQUFFLHFCQUFxQixFQUFFLE1BQU0sa0JBQWtCLENBQUM7QUFDekQsT0FBTyxFQUErQixxQkFBcUIsRUFBRSxXQUFXLEVBQUUsTUFBTSxhQUFhLENBQUM7QUFFOUYsTUFBTSxZQUFZLEdBQUcsS0FBSyxFQUFFLFdBQXdCLEVBQXlCLEVBQUU7SUFDN0UsT0FBTyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxFQUFFO1FBQzdCLFdBQVcsQ0FBQyxPQUFPLEdBQUcsR0FBRyxFQUFFLENBQUMsT0FBTyxDQUFDLEVBQUUsS0FBSyxFQUFFLFdBQVcsQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO0lBQ3RFLENBQUMsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxDQUFDO0FBV0YsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLEtBQUssRUFDL0IsT0FBOEIsRUFDOUIsS0FBWSxFQUNaLGNBQXVFLEVBQ2hELEVBQUU7SUFDekIsTUFBTSxNQUFNLEdBQUc7UUFDYixHQUFHLHFCQUFxQjtRQUN4QixHQUFHLE9BQU87S0FDWCxDQUFDO0lBQ0YscUJBQXFCLENBQUMsTUFBTSxDQUFDLENBQUM7SUFFOUIsTUFBTSxjQUFjLEdBQUcsQ0FBQyxVQUFVLENBQWdCLE1BQU0sRUFBRSxLQUFLLEVBQUUsY0FBYyxDQUFDLENBQUMsQ0FBQztJQUNsRixJQUFJLE9BQU8sQ0FBQyxlQUFlLEVBQUU7UUFDM0IsY0FBYyxDQUFDLElBQUksQ0FBQyxZQUFZLENBQUMsT0FBTyxDQUFDLGVBQWUsQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO0tBQ25FO0lBRUQsSUFBSSxPQUFPLENBQUMsV0FBVyxFQUFFO1FBQ3ZCLGNBQWMsQ0FBQyxJQUFJLENBQUMsWUFBWSxDQUFDLE9BQU8sQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO0tBQ3hEO0lBRUQsT0FBTyxPQUFPLENBQUMsSUFBSSxDQUFDLGNBQWMsQ0FBQyxDQUFDO0FBQ3RDLENBQUMsQ0FBQyJ9