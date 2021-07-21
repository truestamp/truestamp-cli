export const deserializerMiddleware = (options, deserializer) => (next, context) => async (args) => {
    const { response } = await next(args);
    const parsed = await deserializer(response, options);
    return {
        response,
        output: parsed,
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVzZXJpYWxpemVyTWlkZGxld2FyZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImRlc2VyaWFsaXplck1pZGRsZXdhcmUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBU0EsTUFBTSxDQUFDLE1BQU0sc0JBQXNCLEdBQ2pDLENBQ0UsT0FBcUIsRUFDckIsWUFBMEQsRUFDcEIsRUFBRSxDQUMxQyxDQUFDLElBQXVDLEVBQUUsT0FBZ0MsRUFBcUMsRUFBRSxDQUNqSCxLQUFLLEVBQUUsSUFBd0MsRUFBNkMsRUFBRTtJQUM1RixNQUFNLEVBQUUsUUFBUSxFQUFFLEdBQUcsTUFBTSxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDdEMsTUFBTSxNQUFNLEdBQUcsTUFBTSxZQUFZLENBQUMsUUFBUSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3JELE9BQU87UUFDTCxRQUFRO1FBQ1IsTUFBTSxFQUFFLE1BQWdCO0tBQ3pCLENBQUM7QUFDSixDQUFDLENBQUMifQ==