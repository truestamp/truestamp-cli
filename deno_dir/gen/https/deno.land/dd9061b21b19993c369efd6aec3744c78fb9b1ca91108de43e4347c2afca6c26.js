export const deserializerMiddleware = (options, deserializer) => (next, context) => async (args) => {
    const { response } = await next(args);
    const parsed = await deserializer(response, options);
    return {
        response,
        output: parsed,
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVzZXJpYWxpemVyTWlkZGxld2FyZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImRlc2VyaWFsaXplck1pZGRsZXdhcmUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBU0EsTUFBTSxDQUFDLE1BQU0sc0JBQXNCLEdBQUcsQ0FDcEMsT0FBcUIsRUFDckIsWUFBMEQsRUFDcEIsRUFBRSxDQUFDLENBQ3pDLElBQXVDLEVBQ3ZDLE9BQWdDLEVBQ0csRUFBRSxDQUFDLEtBQUssRUFDM0MsSUFBd0MsRUFDRyxFQUFFO0lBQzdDLE1BQU0sRUFBRSxRQUFRLEVBQUUsR0FBRyxNQUFNLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUN0QyxNQUFNLE1BQU0sR0FBRyxNQUFNLFlBQVksQ0FBQyxRQUFRLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDckQsT0FBTztRQUNMLFFBQVE7UUFDUixNQUFNLEVBQUUsTUFBZ0I7S0FDekIsQ0FBQztBQUNKLENBQUMsQ0FBQyJ9