import { NO_RETRY_INCREMENT, RETRY_COST, TIMEOUT_RETRY_COST } from "./constants.ts";
export const getDefaultRetryQuota = (initialRetryTokens, options) => {
    const MAX_CAPACITY = initialRetryTokens;
    const noRetryIncrement = options?.noRetryIncrement ?? NO_RETRY_INCREMENT;
    const retryCost = options?.retryCost ?? RETRY_COST;
    const timeoutRetryCost = options?.timeoutRetryCost ?? TIMEOUT_RETRY_COST;
    let availableCapacity = initialRetryTokens;
    const getCapacityAmount = (error) => (error.name === "TimeoutError" ? timeoutRetryCost : retryCost);
    const hasRetryTokens = (error) => getCapacityAmount(error) <= availableCapacity;
    const retrieveRetryTokens = (error) => {
        if (!hasRetryTokens(error)) {
            throw new Error("No retry token available");
        }
        const capacityAmount = getCapacityAmount(error);
        availableCapacity -= capacityAmount;
        return capacityAmount;
    };
    const releaseRetryTokens = (capacityReleaseAmount) => {
        availableCapacity += capacityReleaseAmount ?? noRetryIncrement;
        availableCapacity = Math.min(availableCapacity, MAX_CAPACITY);
    };
    return Object.freeze({
        hasRetryTokens,
        retrieveRetryTokens,
        releaseRetryTokens,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVmYXVsdFJldHJ5UXVvdGEuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJkZWZhdWx0UmV0cnlRdW90YS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxPQUFPLEVBQUUsa0JBQWtCLEVBQUUsVUFBVSxFQUFFLGtCQUFrQixFQUFFLE1BQU0sZ0JBQWdCLENBQUM7QUFzQnBGLE1BQU0sQ0FBQyxNQUFNLG9CQUFvQixHQUFHLENBQUMsa0JBQTBCLEVBQUUsT0FBa0MsRUFBYyxFQUFFO0lBQ2pILE1BQU0sWUFBWSxHQUFHLGtCQUFrQixDQUFDO0lBQ3hDLE1BQU0sZ0JBQWdCLEdBQUcsT0FBTyxFQUFFLGdCQUFnQixJQUFJLGtCQUFrQixDQUFDO0lBQ3pFLE1BQU0sU0FBUyxHQUFHLE9BQU8sRUFBRSxTQUFTLElBQUksVUFBVSxDQUFDO0lBQ25ELE1BQU0sZ0JBQWdCLEdBQUcsT0FBTyxFQUFFLGdCQUFnQixJQUFJLGtCQUFrQixDQUFDO0lBRXpFLElBQUksaUJBQWlCLEdBQUcsa0JBQWtCLENBQUM7SUFFM0MsTUFBTSxpQkFBaUIsR0FBRyxDQUFDLEtBQWUsRUFBRSxFQUFFLENBQUMsQ0FBQyxLQUFLLENBQUMsSUFBSSxLQUFLLGNBQWMsQ0FBQyxDQUFDLENBQUMsZ0JBQWdCLENBQUMsQ0FBQyxDQUFDLFNBQVMsQ0FBQyxDQUFDO0lBRTlHLE1BQU0sY0FBYyxHQUFHLENBQUMsS0FBZSxFQUFFLEVBQUUsQ0FBQyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsSUFBSSxpQkFBaUIsQ0FBQztJQUUxRixNQUFNLG1CQUFtQixHQUFHLENBQUMsS0FBZSxFQUFFLEVBQUU7UUFDOUMsSUFBSSxDQUFDLGNBQWMsQ0FBQyxLQUFLLENBQUMsRUFBRTtZQUUxQixNQUFNLElBQUksS0FBSyxDQUFDLDBCQUEwQixDQUFDLENBQUM7U0FDN0M7UUFDRCxNQUFNLGNBQWMsR0FBRyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsQ0FBQztRQUNoRCxpQkFBaUIsSUFBSSxjQUFjLENBQUM7UUFDcEMsT0FBTyxjQUFjLENBQUM7SUFDeEIsQ0FBQyxDQUFDO0lBRUYsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLHFCQUE4QixFQUFFLEVBQUU7UUFDNUQsaUJBQWlCLElBQUkscUJBQXFCLElBQUksZ0JBQWdCLENBQUM7UUFDL0QsaUJBQWlCLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsRUFBRSxZQUFZLENBQUMsQ0FBQztJQUNoRSxDQUFDLENBQUM7SUFFRixPQUFPLE1BQU0sQ0FBQyxNQUFNLENBQUM7UUFDbkIsY0FBYztRQUNkLG1CQUFtQjtRQUNuQixrQkFBa0I7S0FDbkIsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxDQUFDIn0=