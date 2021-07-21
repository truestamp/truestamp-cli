export function locationConstraintMiddleware(options) {
    return (next) => async (args) => {
        const { CreateBucketConfiguration } = args.input;
        const region = await options.region();
        if (!CreateBucketConfiguration || !CreateBucketConfiguration.LocationConstraint) {
            args = {
                ...args,
                input: {
                    ...args.input,
                    CreateBucketConfiguration: region === "us-east-1" ? undefined : { LocationConstraint: region },
                },
            };
        }
        return next(args);
    };
}
export const locationConstraintMiddlewareOptions = {
    step: "initialize",
    tags: ["LOCATION_CONSTRAINT", "CREATE_BUCKET_CONFIGURATION"],
    name: "locationConstraintMiddleware",
    override: true,
};
export const getLocationConstraintPlugin = (config) => ({
    applyToStack: (clientStack) => {
        clientStack.add(locationConstraintMiddleware(config), locationConstraintMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWtCQSxNQUFNLFVBQVUsNEJBQTRCLENBQzFDLE9BQXlDO0lBRXpDLE9BQU8sQ0FBZ0MsSUFBb0MsRUFBa0MsRUFBRSxDQUM3RyxLQUFLLEVBQUUsSUFBcUMsRUFBNEMsRUFBRTtRQUN4RixNQUFNLEVBQUUseUJBQXlCLEVBQUUsR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDO1FBRWpELE1BQU0sTUFBTSxHQUFHLE1BQU0sT0FBTyxDQUFDLE1BQU0sRUFBRSxDQUFDO1FBQ3RDLElBQUksQ0FBQyx5QkFBeUIsSUFBSSxDQUFDLHlCQUF5QixDQUFDLGtCQUFrQixFQUFFO1lBQy9FLElBQUksR0FBRztnQkFDTCxHQUFHLElBQUk7Z0JBQ1AsS0FBSyxFQUFFO29CQUNMLEdBQUcsSUFBSSxDQUFDLEtBQUs7b0JBQ2IseUJBQXlCLEVBQUUsTUFBTSxLQUFLLFdBQVcsQ0FBQyxDQUFDLENBQUMsU0FBUyxDQUFDLENBQUMsQ0FBQyxFQUFFLGtCQUFrQixFQUFFLE1BQU0sRUFBRTtpQkFDL0Y7YUFDRixDQUFDO1NBQ0g7UUFFRCxPQUFPLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUNwQixDQUFDLENBQUM7QUFDTixDQUFDO0FBRUQsTUFBTSxDQUFDLE1BQU0sbUNBQW1DLEdBQTZCO0lBQzNFLElBQUksRUFBRSxZQUFZO0lBQ2xCLElBQUksRUFBRSxDQUFDLHFCQUFxQixFQUFFLDZCQUE2QixDQUFDO0lBQzVELElBQUksRUFBRSw4QkFBOEI7SUFDcEMsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sMkJBQTJCLEdBQUcsQ0FBQyxNQUF3QyxFQUF1QixFQUFFLENBQUMsQ0FBQztJQUM3RyxZQUFZLEVBQUUsQ0FBQyxXQUFXLEVBQUUsRUFBRTtRQUM1QixXQUFXLENBQUMsR0FBRyxDQUFDLDRCQUE0QixDQUFDLE1BQU0sQ0FBQyxFQUFFLG1DQUFtQyxDQUFDLENBQUM7SUFDN0YsQ0FBQztDQUNGLENBQUMsQ0FBQyJ9