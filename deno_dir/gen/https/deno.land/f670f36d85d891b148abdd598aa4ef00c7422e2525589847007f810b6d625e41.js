export const validateWaiterOptions = (options) => {
    if (options.maxWaitTime < 1) {
        throw new Error(`WaiterConfiguration.maxWaitTime must be greater than 0`);
    }
    else if (options.minDelay < 1) {
        throw new Error(`WaiterConfiguration.minDelay must be greater than 0`);
    }
    else if (options.maxDelay < 1) {
        throw new Error(`WaiterConfiguration.maxDelay must be greater than 0`);
    }
    else if (options.maxWaitTime <= options.minDelay) {
        throw new Error(`WaiterConfiguration.maxWaitTime [${options.maxWaitTime}] must be greater than WaiterConfiguration.minDelay [${options.minDelay}] for this waiter`);
    }
    else if (options.maxDelay < options.minDelay) {
        throw new Error(`WaiterConfiguration.maxDelay [${options.maxDelay}] must be greater than WaiterConfiguration.minDelay [${options.minDelay}] for this waiter`);
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidmFsaWRhdGUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJ2YWxpZGF0ZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFNQSxNQUFNLENBQUMsTUFBTSxxQkFBcUIsR0FBRyxDQUFTLE9BQThCLEVBQVEsRUFBRTtJQUNwRixJQUFJLE9BQU8sQ0FBQyxXQUFXLEdBQUcsQ0FBQyxFQUFFO1FBQzNCLE1BQU0sSUFBSSxLQUFLLENBQUMsd0RBQXdELENBQUMsQ0FBQztLQUMzRTtTQUFNLElBQUksT0FBTyxDQUFDLFFBQVEsR0FBRyxDQUFDLEVBQUU7UUFDL0IsTUFBTSxJQUFJLEtBQUssQ0FBQyxxREFBcUQsQ0FBQyxDQUFDO0tBQ3hFO1NBQU0sSUFBSSxPQUFPLENBQUMsUUFBUSxHQUFHLENBQUMsRUFBRTtRQUMvQixNQUFNLElBQUksS0FBSyxDQUFDLHFEQUFxRCxDQUFDLENBQUM7S0FDeEU7U0FBTSxJQUFJLE9BQU8sQ0FBQyxXQUFXLElBQUksT0FBTyxDQUFDLFFBQVEsRUFBRTtRQUNsRCxNQUFNLElBQUksS0FBSyxDQUNiLG9DQUFvQyxPQUFPLENBQUMsV0FBVyx3REFBd0QsT0FBTyxDQUFDLFFBQVEsbUJBQW1CLENBQ25KLENBQUM7S0FDSDtTQUFNLElBQUksT0FBTyxDQUFDLFFBQVEsR0FBRyxPQUFPLENBQUMsUUFBUSxFQUFFO1FBQzlDLE1BQU0sSUFBSSxLQUFLLENBQ2IsaUNBQWlDLE9BQU8sQ0FBQyxRQUFRLHdEQUF3RCxPQUFPLENBQUMsUUFBUSxtQkFBbUIsQ0FDN0ksQ0FBQztLQUNIO0FBQ0gsQ0FBQyxDQUFDIn0=