import { NO_RETRY_INCREMENT, RETRY_COST, TIMEOUT_RETRY_COST } from "./constants.ts";
export const getDefaultRetryQuota = (initialRetryTokens) => {
    const MAX_CAPACITY = initialRetryTokens;
    let availableCapacity = initialRetryTokens;
    const getCapacityAmount = (error) => (error.name === "TimeoutError" ? TIMEOUT_RETRY_COST : RETRY_COST);
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
        availableCapacity += capacityReleaseAmount ?? NO_RETRY_INCREMENT;
        availableCapacity = Math.min(availableCapacity, MAX_CAPACITY);
    };
    return Object.freeze({
        hasRetryTokens,
        retrieveRetryTokens,
        releaseRetryTokens,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVmYXVsdFJldHJ5UXVvdGEuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJkZWZhdWx0UmV0cnlRdW90YS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxPQUFPLEVBQUUsa0JBQWtCLEVBQUUsVUFBVSxFQUFFLGtCQUFrQixFQUFFLE1BQU0sZ0JBQWdCLENBQUM7QUFHcEYsTUFBTSxDQUFDLE1BQU0sb0JBQW9CLEdBQUcsQ0FBQyxrQkFBMEIsRUFBYyxFQUFFO0lBQzdFLE1BQU0sWUFBWSxHQUFHLGtCQUFrQixDQUFDO0lBQ3hDLElBQUksaUJBQWlCLEdBQUcsa0JBQWtCLENBQUM7SUFFM0MsTUFBTSxpQkFBaUIsR0FBRyxDQUFDLEtBQWUsRUFBRSxFQUFFLENBQUMsQ0FBQyxLQUFLLENBQUMsSUFBSSxLQUFLLGNBQWMsQ0FBQyxDQUFDLENBQUMsa0JBQWtCLENBQUMsQ0FBQyxDQUFDLFVBQVUsQ0FBQyxDQUFDO0lBRWpILE1BQU0sY0FBYyxHQUFHLENBQUMsS0FBZSxFQUFFLEVBQUUsQ0FBQyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsSUFBSSxpQkFBaUIsQ0FBQztJQUUxRixNQUFNLG1CQUFtQixHQUFHLENBQUMsS0FBZSxFQUFFLEVBQUU7UUFDOUMsSUFBSSxDQUFDLGNBQWMsQ0FBQyxLQUFLLENBQUMsRUFBRTtZQUUxQixNQUFNLElBQUksS0FBSyxDQUFDLDBCQUEwQixDQUFDLENBQUM7U0FDN0M7UUFDRCxNQUFNLGNBQWMsR0FBRyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsQ0FBQztRQUNoRCxpQkFBaUIsSUFBSSxjQUFjLENBQUM7UUFDcEMsT0FBTyxjQUFjLENBQUM7SUFDeEIsQ0FBQyxDQUFDO0lBRUYsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLHFCQUE4QixFQUFFLEVBQUU7UUFDNUQsaUJBQWlCLElBQUkscUJBQXFCLElBQUksa0JBQWtCLENBQUM7UUFDakUsaUJBQWlCLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsRUFBRSxZQUFZLENBQUMsQ0FBQztJQUNoRSxDQUFDLENBQUM7SUFFRixPQUFPLE1BQU0sQ0FBQyxNQUFNLENBQUM7UUFDbkIsY0FBYztRQUNkLG1CQUFtQjtRQUNuQixrQkFBa0I7S0FDbkIsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxDQUFDIn0=