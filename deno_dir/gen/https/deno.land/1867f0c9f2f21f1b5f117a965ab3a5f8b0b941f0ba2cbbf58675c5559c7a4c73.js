import { deserializerMiddleware } from "./deserializerMiddleware.ts";
import { serializerMiddleware } from "./serializerMiddleware.ts";
export const deserializerMiddlewareOption = {
    name: "deserializerMiddleware",
    step: "deserialize",
    tags: ["DESERIALIZER"],
    override: true,
};
export const serializerMiddlewareOption = {
    name: "serializerMiddleware",
    step: "serialize",
    tags: ["SERIALIZER"],
    override: true,
};
export function getSerdePlugin(config, serializer, deserializer) {
    return {
        applyToStack: (commandStack) => {
            commandStack.add(deserializerMiddleware(config, deserializer), deserializerMiddlewareOption);
            commandStack.add(serializerMiddleware(config, serializer), serializerMiddlewareOption);
        },
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2VyZGVQbHVnaW4uanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJzZXJkZVBsdWdpbi50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFXQSxPQUFPLEVBQUUsc0JBQXNCLEVBQUUsTUFBTSw2QkFBNkIsQ0FBQztBQUNyRSxPQUFPLEVBQUUsb0JBQW9CLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUVqRSxNQUFNLENBQUMsTUFBTSw0QkFBNEIsR0FBOEI7SUFDckUsSUFBSSxFQUFFLHdCQUF3QjtJQUM5QixJQUFJLEVBQUUsYUFBYTtJQUNuQixJQUFJLEVBQUUsQ0FBQyxjQUFjLENBQUM7SUFDdEIsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sMEJBQTBCLEdBQTRCO0lBQ2pFLElBQUksRUFBRSxzQkFBc0I7SUFDNUIsSUFBSSxFQUFFLFdBQVc7SUFDakIsSUFBSSxFQUFFLENBQUMsWUFBWSxDQUFDO0lBQ3BCLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sVUFBVSxjQUFjLENBSzVCLE1BQW9CLEVBQ3BCLFVBQWdELEVBQ2hELFlBQWlFO0lBRWpFLE9BQU87UUFDTCxZQUFZLEVBQUUsQ0FBQyxZQUFvRCxFQUFFLEVBQUU7WUFDckUsWUFBWSxDQUFDLEdBQUcsQ0FBQyxzQkFBc0IsQ0FBQyxNQUFNLEVBQUUsWUFBWSxDQUFDLEVBQUUsNEJBQTRCLENBQUMsQ0FBQztZQUM3RixZQUFZLENBQUMsR0FBRyxDQUFDLG9CQUFvQixDQUFDLE1BQU0sRUFBRSxVQUFVLENBQUMsRUFBRSwwQkFBMEIsQ0FBQyxDQUFDO1FBQ3pGLENBQUM7S0FDRixDQUFDO0FBQ0osQ0FBQyJ9