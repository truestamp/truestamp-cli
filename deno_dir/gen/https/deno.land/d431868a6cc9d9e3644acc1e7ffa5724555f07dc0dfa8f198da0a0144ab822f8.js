export function ssecMiddleware(options) {
    return (next) => async (args) => {
        let input = { ...args.input };
        const properties = [
            {
                target: "SSECustomerKey",
                hash: "SSECustomerKeyMD5",
            },
            {
                target: "CopySourceSSECustomerKey",
                hash: "CopySourceSSECustomerKeyMD5",
            },
        ];
        for (const prop of properties) {
            const value = input[prop.target];
            if (value) {
                const valueView = ArrayBuffer.isView(value)
                    ? new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
                    : typeof value === "string"
                        ? options.utf8Decoder(value)
                        : new Uint8Array(value);
                const encoded = options.base64Encoder(valueView);
                const hash = new options.md5();
                hash.update(valueView);
                input = {
                    ...input,
                    [prop.target]: encoded,
                    [prop.hash]: options.base64Encoder(await hash.digest()),
                };
            }
        }
        return next({
            ...args,
            input,
        });
    };
}
export const ssecMiddlewareOptions = {
    name: "ssecMiddleware",
    step: "initialize",
    tags: ["SSE"],
    override: true,
};
export const getSsecPlugin = (config) => ({
    applyToStack: (clientStack) => {
        clientStack.add(ssecMiddleware(config), ssecMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWFBLE1BQU0sVUFBVSxjQUFjLENBQUMsT0FBcUM7SUFDbEUsT0FBTyxDQUNMLElBQW9DLEVBQ0osRUFBRSxDQUFDLEtBQUssRUFDeEMsSUFBcUMsRUFDSyxFQUFFO1FBQzVDLElBQUksS0FBSyxHQUFHLEVBQUUsR0FBRyxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7UUFDOUIsTUFBTSxVQUFVLEdBQUc7WUFDakI7Z0JBQ0UsTUFBTSxFQUFFLGdCQUFnQjtnQkFDeEIsSUFBSSxFQUFFLG1CQUFtQjthQUMxQjtZQUNEO2dCQUNFLE1BQU0sRUFBRSwwQkFBMEI7Z0JBQ2xDLElBQUksRUFBRSw2QkFBNkI7YUFDcEM7U0FDRixDQUFDO1FBRUYsS0FBSyxNQUFNLElBQUksSUFBSSxVQUFVLEVBQUU7WUFDN0IsTUFBTSxLQUFLLEdBQTRCLEtBQWEsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUM7WUFDbEUsSUFBSSxLQUFLLEVBQUU7Z0JBQ1QsTUFBTSxTQUFTLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUM7b0JBQ3pDLENBQUMsQ0FBQyxJQUFJLFVBQVUsQ0FBQyxLQUFLLENBQUMsTUFBTSxFQUFFLEtBQUssQ0FBQyxVQUFVLEVBQUUsS0FBSyxDQUFDLFVBQVUsQ0FBQztvQkFDbEUsQ0FBQyxDQUFDLE9BQU8sS0FBSyxLQUFLLFFBQVE7d0JBQzNCLENBQUMsQ0FBQyxPQUFPLENBQUMsV0FBVyxDQUFDLEtBQUssQ0FBQzt3QkFDNUIsQ0FBQyxDQUFDLElBQUksVUFBVSxDQUFDLEtBQUssQ0FBQyxDQUFDO2dCQUMxQixNQUFNLE9BQU8sR0FBRyxPQUFPLENBQUMsYUFBYSxDQUFDLFNBQVMsQ0FBQyxDQUFDO2dCQUNqRCxNQUFNLElBQUksR0FBRyxJQUFJLE9BQU8sQ0FBQyxHQUFHLEVBQUUsQ0FBQztnQkFDL0IsSUFBSSxDQUFDLE1BQU0sQ0FBQyxTQUFTLENBQUMsQ0FBQztnQkFDdkIsS0FBSyxHQUFHO29CQUNOLEdBQUksS0FBYTtvQkFDakIsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEVBQUUsT0FBTztvQkFDdEIsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLEVBQUUsT0FBTyxDQUFDLGFBQWEsQ0FBQyxNQUFNLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQztpQkFDeEQsQ0FBQzthQUNIO1NBQ0Y7UUFFRCxPQUFPLElBQUksQ0FBQztZQUNWLEdBQUcsSUFBSTtZQUNQLEtBQUs7U0FDTixDQUFDLENBQUM7SUFDTCxDQUFDLENBQUM7QUFDSixDQUFDO0FBRUQsTUFBTSxDQUFDLE1BQU0scUJBQXFCLEdBQTZCO0lBQzdELElBQUksRUFBRSxnQkFBZ0I7SUFDdEIsSUFBSSxFQUFFLFlBQVk7SUFDbEIsSUFBSSxFQUFFLENBQUMsS0FBSyxDQUFDO0lBQ2IsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sYUFBYSxHQUFHLENBQUMsTUFBb0MsRUFBdUIsRUFBRSxDQUFDLENBQUM7SUFDM0YsWUFBWSxFQUFFLENBQUMsV0FBVyxFQUFFLEVBQUU7UUFDNUIsV0FBVyxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsTUFBTSxDQUFDLEVBQUUscUJBQXFCLENBQUMsQ0FBQztJQUNqRSxDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=