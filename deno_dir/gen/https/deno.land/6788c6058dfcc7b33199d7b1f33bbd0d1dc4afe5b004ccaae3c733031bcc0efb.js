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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQW1CQSxNQUFNLFVBQVUsY0FBYyxDQUFDLE9BQTJCO0lBQ3hELE9BQU8sQ0FBZ0MsSUFBb0MsRUFBa0MsRUFBRSxDQUM3RyxLQUFLLEVBQUUsSUFBcUMsRUFBNEMsRUFBRTtRQUN4RixJQUFJLEtBQUssR0FBRyxFQUFFLEdBQUcsSUFBSSxDQUFDLEtBQUssRUFBRSxDQUFDO1FBQzlCLE1BQU0sVUFBVSxHQUFHO1lBQ2pCO2dCQUNFLE1BQU0sRUFBRSxnQkFBZ0I7Z0JBQ3hCLElBQUksRUFBRSxtQkFBbUI7YUFDMUI7WUFDRDtnQkFDRSxNQUFNLEVBQUUsMEJBQTBCO2dCQUNsQyxJQUFJLEVBQUUsNkJBQTZCO2FBQ3BDO1NBQ0YsQ0FBQztRQUVGLEtBQUssTUFBTSxJQUFJLElBQUksVUFBVSxFQUFFO1lBQzdCLE1BQU0sS0FBSyxHQUE0QixLQUFhLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1lBQ2xFLElBQUksS0FBSyxFQUFFO2dCQUNULE1BQU0sU0FBUyxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDO29CQUN6QyxDQUFDLENBQUMsSUFBSSxVQUFVLENBQUMsS0FBSyxDQUFDLE1BQU0sRUFBRSxLQUFLLENBQUMsVUFBVSxFQUFFLEtBQUssQ0FBQyxVQUFVLENBQUM7b0JBQ2xFLENBQUMsQ0FBQyxPQUFPLEtBQUssS0FBSyxRQUFRO3dCQUMzQixDQUFDLENBQUMsT0FBTyxDQUFDLFdBQVcsQ0FBQyxLQUFLLENBQUM7d0JBQzVCLENBQUMsQ0FBQyxJQUFJLFVBQVUsQ0FBQyxLQUFLLENBQUMsQ0FBQztnQkFDMUIsTUFBTSxPQUFPLEdBQUcsT0FBTyxDQUFDLGFBQWEsQ0FBQyxTQUFTLENBQUMsQ0FBQztnQkFDakQsTUFBTSxJQUFJLEdBQUcsSUFBSSxPQUFPLENBQUMsR0FBRyxFQUFFLENBQUM7Z0JBQy9CLElBQUksQ0FBQyxNQUFNLENBQUMsU0FBUyxDQUFDLENBQUM7Z0JBQ3ZCLEtBQUssR0FBRztvQkFDTixHQUFJLEtBQWE7b0JBQ2pCLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxFQUFFLE9BQU87b0JBQ3RCLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxFQUFFLE9BQU8sQ0FBQyxhQUFhLENBQUMsTUFBTSxJQUFJLENBQUMsTUFBTSxFQUFFLENBQUM7aUJBQ3hELENBQUM7YUFDSDtTQUNGO1FBRUQsT0FBTyxJQUFJLENBQUM7WUFDVixHQUFHLElBQUk7WUFDUCxLQUFLO1NBQ04sQ0FBQyxDQUFDO0lBQ0wsQ0FBQyxDQUFDO0FBQ04sQ0FBQztBQUVELE1BQU0sQ0FBQyxNQUFNLHFCQUFxQixHQUE2QjtJQUM3RCxJQUFJLEVBQUUsZ0JBQWdCO0lBQ3RCLElBQUksRUFBRSxZQUFZO0lBQ2xCLElBQUksRUFBRSxDQUFDLEtBQUssQ0FBQztJQUNiLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLGFBQWEsR0FBRyxDQUFDLE1BQTBCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ2pGLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxFQUFFLHFCQUFxQixDQUFDLENBQUM7SUFDakUsQ0FBQztDQUNGLENBQUMsQ0FBQyJ9