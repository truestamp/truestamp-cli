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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibG9nZ2VyTWlkZGxld2FyZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImxvZ2dlck1pZGRsZXdhcmUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBWUEsTUFBTSxDQUFDLE1BQU0sZ0JBQWdCLEdBQzNCLEdBQUcsRUFBRSxDQUNMLENBQ0UsSUFBb0MsRUFDcEMsT0FBZ0MsRUFDQSxFQUFFLENBQ3BDLEtBQUssRUFBRSxJQUFxQyxFQUE0QyxFQUFFO0lBQ3hGLE1BQU0sRUFBRSxVQUFVLEVBQUUsV0FBVyxFQUFFLHVCQUF1QixFQUFFLE1BQU0sRUFBRSx3QkFBd0IsRUFBRSxHQUFHLE9BQU8sQ0FBQztJQUV2RyxNQUFNLFFBQVEsR0FBRyxNQUFNLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUVsQyxJQUFJLENBQUMsTUFBTSxFQUFFO1FBQ1gsT0FBTyxRQUFRLENBQUM7S0FDakI7SUFFRCxJQUFJLE9BQU8sTUFBTSxDQUFDLElBQUksS0FBSyxVQUFVLEVBQUU7UUFDckMsTUFBTSxFQUFFLFNBQVMsRUFBRSxHQUFHLHFCQUFxQixFQUFFLEdBQUcsUUFBUSxDQUFDLE1BQU0sQ0FBQztRQUNoRSxNQUFNLENBQUMsSUFBSSxDQUFDO1lBQ1YsVUFBVTtZQUNWLFdBQVc7WUFDWCxLQUFLLEVBQUUsdUJBQXVCLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQztZQUMxQyxNQUFNLEVBQUUsd0JBQXdCLENBQUMscUJBQXFCLENBQUM7WUFDdkQsUUFBUSxFQUFFLFNBQVM7U0FDcEIsQ0FBQyxDQUFDO0tBQ0o7SUFFRCxPQUFPLFFBQVEsQ0FBQztBQUNsQixDQUFDLENBQUM7QUFFSixNQUFNLENBQUMsTUFBTSx1QkFBdUIsR0FBZ0Q7SUFDbEYsSUFBSSxFQUFFLGtCQUFrQjtJQUN4QixJQUFJLEVBQUUsQ0FBQyxRQUFRLENBQUM7SUFDaEIsSUFBSSxFQUFFLFlBQVk7SUFDbEIsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBR0YsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLENBQUMsT0FBWSxFQUF1QixFQUFFLENBQUMsQ0FBQztJQUNyRSxZQUFZLEVBQUUsQ0FBQyxXQUFXLEVBQUUsRUFBRTtRQUM1QixXQUFXLENBQUMsR0FBRyxDQUFDLGdCQUFnQixFQUFFLEVBQUUsdUJBQXVCLENBQUMsQ0FBQztJQUMvRCxDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=