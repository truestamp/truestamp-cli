export const loggerMiddleware = () => (next, context) => async (args) => {
    const { clientName, commandName, inputFilterSensitiveLog, logger, outputFilterSensitiveLog } = context;
    const response = await next(args);
    if (!logger) {
        return response;
    }
    if (typeof logger.info === "function") {
        const { $metadata, ...outputWithoutMetadata } = response.output;
        logger.info({
            clientName,
            commandName,
            input: inputFilterSensitiveLog(args.input),
            output: outputFilterSensitiveLog(outputWithoutMetadata),
            metadata: $metadata,
        });
    }
    return response;
};
export const loggerMiddlewareOptions = {
    name: "loggerMiddleware",
    tags: ["LOGGER"],
    step: "initialize",
    override: true,
};
export const getLoggerPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.add(loggerMiddleware(), loggerMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibG9nZ2VyTWlkZGxld2FyZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImxvZ2dlck1pZGRsZXdhcmUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBWUEsTUFBTSxDQUFDLE1BQU0sZ0JBQWdCLEdBQUcsR0FBRyxFQUFFLENBQUMsQ0FDcEMsSUFBb0MsRUFDcEMsT0FBZ0MsRUFDQSxFQUFFLENBQUMsS0FBSyxFQUN4QyxJQUFxQyxFQUNLLEVBQUU7SUFDNUMsTUFBTSxFQUFFLFVBQVUsRUFBRSxXQUFXLEVBQUUsdUJBQXVCLEVBQUUsTUFBTSxFQUFFLHdCQUF3QixFQUFFLEdBQUcsT0FBTyxDQUFDO0lBRXZHLE1BQU0sUUFBUSxHQUFHLE1BQU0sSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO0lBRWxDLElBQUksQ0FBQyxNQUFNLEVBQUU7UUFDWCxPQUFPLFFBQVEsQ0FBQztLQUNqQjtJQUVELElBQUksT0FBTyxNQUFNLENBQUMsSUFBSSxLQUFLLFVBQVUsRUFBRTtRQUNyQyxNQUFNLEVBQUUsU0FBUyxFQUFFLEdBQUcscUJBQXFCLEVBQUUsR0FBRyxRQUFRLENBQUMsTUFBTSxDQUFDO1FBQ2hFLE1BQU0sQ0FBQyxJQUFJLENBQUM7WUFDVixVQUFVO1lBQ1YsV0FBVztZQUNYLEtBQUssRUFBRSx1QkFBdUIsQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDO1lBQzFDLE1BQU0sRUFBRSx3QkFBd0IsQ0FBQyxxQkFBcUIsQ0FBQztZQUN2RCxRQUFRLEVBQUUsU0FBUztTQUNwQixDQUFDLENBQUM7S0FDSjtJQUVELE9BQU8sUUFBUSxDQUFDO0FBQ2xCLENBQUMsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLHVCQUF1QixHQUFnRDtJQUNsRixJQUFJLEVBQUUsa0JBQWtCO0lBQ3hCLElBQUksRUFBRSxDQUFDLFFBQVEsQ0FBQztJQUNoQixJQUFJLEVBQUUsWUFBWTtJQUNsQixRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFHRixNQUFNLENBQUMsTUFBTSxlQUFlLEdBQUcsQ0FBQyxPQUFZLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ3JFLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsZ0JBQWdCLEVBQUUsRUFBRSx1QkFBdUIsQ0FBQyxDQUFDO0lBQy9ELENBQUM7Q0FDRixDQUFDLENBQUMifQ==