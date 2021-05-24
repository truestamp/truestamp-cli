export function rmdir(path, optionsOrCallback, maybeCallback) {
    const callback = typeof optionsOrCallback === "function"
        ? optionsOrCallback
        : maybeCallback;
    const options = typeof optionsOrCallback === "object"
        ? optionsOrCallback
        : undefined;
    if (!callback)
        throw new Error("No callback function supplied");
    Deno.remove(path, { recursive: options?.recursive })
        .then((_) => callback(), callback);
}
export function rmdirSync(path, options) {
    Deno.removeSync(path, { recursive: options?.recursive });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX3JtZGlyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2ZzX3JtZGlyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWNBLE1BQU0sVUFBVSxLQUFLLENBQ25CLElBQWtCLEVBQ2xCLGlCQUErQyxFQUMvQyxhQUE2QjtJQUU3QixNQUFNLFFBQVEsR0FBRyxPQUFPLGlCQUFpQixLQUFLLFVBQVU7UUFDdEQsQ0FBQyxDQUFDLGlCQUFpQjtRQUNuQixDQUFDLENBQUMsYUFBYSxDQUFDO0lBQ2xCLE1BQU0sT0FBTyxHQUFHLE9BQU8saUJBQWlCLEtBQUssUUFBUTtRQUNuRCxDQUFDLENBQUMsaUJBQWlCO1FBQ25CLENBQUMsQ0FBQyxTQUFTLENBQUM7SUFFZCxJQUFJLENBQUMsUUFBUTtRQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsK0JBQStCLENBQUMsQ0FBQztJQUVoRSxJQUFJLENBQUMsTUFBTSxDQUFDLElBQUksRUFBRSxFQUFFLFNBQVMsRUFBRSxPQUFPLEVBQUUsU0FBUyxFQUFFLENBQUM7U0FDakQsSUFBSSxDQUFDLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FBQyxRQUFRLEVBQUUsRUFBRSxRQUFRLENBQUMsQ0FBQztBQUN2QyxDQUFDO0FBRUQsTUFBTSxVQUFVLFNBQVMsQ0FBQyxJQUFrQixFQUFFLE9BQXNCO0lBQ2xFLElBQUksQ0FBQyxVQUFVLENBQUMsSUFBSSxFQUFFLEVBQUUsU0FBUyxFQUFFLE9BQU8sRUFBRSxTQUFTLEVBQUUsQ0FBQyxDQUFDO0FBQzNELENBQUMifQ==