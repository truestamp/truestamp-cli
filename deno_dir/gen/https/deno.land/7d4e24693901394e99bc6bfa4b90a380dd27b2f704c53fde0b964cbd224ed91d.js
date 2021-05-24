export const serializerMiddleware = (options, serializer) => (next, context) => async (args) => {
    const request = await serializer(args.input, options);
    return next({
        ...args,
        request,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2VyaWFsaXplck1pZGRsZXdhcmUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJzZXJpYWxpemVyTWlkZGxld2FyZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFVQSxNQUFNLENBQUMsTUFBTSxvQkFBb0IsR0FBRyxDQUNsQyxPQUFxQixFQUNyQixVQUFnRCxFQUNaLEVBQUUsQ0FBQyxDQUN2QyxJQUFxQyxFQUNyQyxPQUFnQyxFQUNDLEVBQUUsQ0FBQyxLQUFLLEVBQ3pDLElBQXNDLEVBQ0csRUFBRTtJQUMzQyxNQUFNLE9BQU8sR0FBRyxNQUFNLFVBQVUsQ0FBQyxJQUFJLENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3RELE9BQU8sSUFBSSxDQUFDO1FBQ1YsR0FBRyxJQUFJO1FBQ1AsT0FBTztLQUNSLENBQUMsQ0FBQztBQUNMLENBQUMsQ0FBQyJ9