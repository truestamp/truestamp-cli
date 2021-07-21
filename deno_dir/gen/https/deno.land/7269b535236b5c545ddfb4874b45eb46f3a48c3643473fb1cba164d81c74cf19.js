export const serializerMiddleware = (options, serializer) => (next, context) => async (args) => {
    const request = await serializer(args.input, options);
    return next({
        ...args,
        request,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2VyaWFsaXplck1pZGRsZXdhcmUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJzZXJpYWxpemVyTWlkZGxld2FyZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFVQSxNQUFNLENBQUMsTUFBTSxvQkFBb0IsR0FDL0IsQ0FDRSxPQUFxQixFQUNyQixVQUFnRCxFQUNaLEVBQUUsQ0FDeEMsQ0FBQyxJQUFxQyxFQUFFLE9BQWdDLEVBQW1DLEVBQUUsQ0FDN0csS0FBSyxFQUFFLElBQXNDLEVBQTJDLEVBQUU7SUFDeEYsTUFBTSxPQUFPLEdBQUcsTUFBTSxVQUFVLENBQUMsSUFBSSxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN0RCxPQUFPLElBQUksQ0FBQztRQUNWLEdBQUcsSUFBSTtRQUNQLE9BQU87S0FDUixDQUFDLENBQUM7QUFDTCxDQUFDLENBQUMifQ==