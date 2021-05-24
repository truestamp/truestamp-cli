export function getUnmarshalledStream(source, options) {
    return {
        [Symbol.asyncIterator]: async function* () {
            for await (const chunk of source) {
                const message = options.eventMarshaller.unmarshall(chunk);
                const { value: messageType } = message.headers[":message-type"];
                if (messageType === "error") {
                    const unmodeledError = new Error(message.headers[":error-message"].value || "UnknownError");
                    unmodeledError.name = message.headers[":error-code"].value;
                    throw unmodeledError;
                }
                else if (messageType === "exception") {
                    const code = message.headers[":exception-type"].value;
                    const exception = { [code]: message };
                    const deserializedException = await options.deserializer(exception);
                    if (deserializedException.$unknown) {
                        const error = new Error(options.toUtf8(message.body));
                        error.name = code;
                        throw error;
                    }
                    throw deserializedException[code];
                }
                else if (messageType === "event") {
                    const event = {
                        [message.headers[":event-type"].value]: message,
                    };
                    const deserialized = await options.deserializer(event);
                    if (deserialized.$unknown)
                        continue;
                    yield deserialized;
                }
                else {
                    throw Error(`Unrecognizable event type: ${message.headers[":event-type"].value}`);
                }
            }
        },
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZ2V0VW5tYXJzaGFsbGVkU3RyZWFtLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZ2V0VW5tYXJzaGFsbGVkU3RyZWFtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQVNBLE1BQU0sVUFBVSxxQkFBcUIsQ0FDbkMsTUFBaUMsRUFDakMsT0FBcUM7SUFFckMsT0FBTztRQUNMLENBQUMsTUFBTSxDQUFDLGFBQWEsQ0FBQyxFQUFFLEtBQUssU0FBUyxDQUFDO1lBQ3JDLElBQUksS0FBSyxFQUFFLE1BQU0sS0FBSyxJQUFJLE1BQU0sRUFBRTtnQkFDaEMsTUFBTSxPQUFPLEdBQUcsT0FBTyxDQUFDLGVBQWUsQ0FBQyxVQUFVLENBQUMsS0FBSyxDQUFDLENBQUM7Z0JBQzFELE1BQU0sRUFBRSxLQUFLLEVBQUUsV0FBVyxFQUFFLEdBQUcsT0FBTyxDQUFDLE9BQU8sQ0FBQyxlQUFlLENBQUMsQ0FBQztnQkFDaEUsSUFBSSxXQUFXLEtBQUssT0FBTyxFQUFFO29CQUUzQixNQUFNLGNBQWMsR0FBRyxJQUFJLEtBQUssQ0FBRSxPQUFPLENBQUMsT0FBTyxDQUFDLGdCQUFnQixDQUFDLENBQUMsS0FBZ0IsSUFBSSxjQUFjLENBQUMsQ0FBQztvQkFDeEcsY0FBYyxDQUFDLElBQUksR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLGFBQWEsQ0FBQyxDQUFDLEtBQWUsQ0FBQztvQkFDckUsTUFBTSxjQUFjLENBQUM7aUJBQ3RCO3FCQUFNLElBQUksV0FBVyxLQUFLLFdBQVcsRUFBRTtvQkFFdEMsTUFBTSxJQUFJLEdBQUcsT0FBTyxDQUFDLE9BQU8sQ0FBQyxpQkFBaUIsQ0FBQyxDQUFDLEtBQWUsQ0FBQztvQkFDaEUsTUFBTSxTQUFTLEdBQUcsRUFBRSxDQUFDLElBQUksQ0FBQyxFQUFFLE9BQU8sRUFBRSxDQUFDO29CQUV0QyxNQUFNLHFCQUFxQixHQUFHLE1BQU0sT0FBTyxDQUFDLFlBQVksQ0FBQyxTQUFTLENBQUMsQ0FBQztvQkFDcEUsSUFBSSxxQkFBcUIsQ0FBQyxRQUFRLEVBQUU7d0JBRWxDLE1BQU0sS0FBSyxHQUFHLElBQUksS0FBSyxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUM7d0JBQ3RELEtBQUssQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO3dCQUNsQixNQUFNLEtBQUssQ0FBQztxQkFDYjtvQkFDRCxNQUFNLHFCQUFxQixDQUFDLElBQUksQ0FBQyxDQUFDO2lCQUNuQztxQkFBTSxJQUFJLFdBQVcsS0FBSyxPQUFPLEVBQUU7b0JBQ2xDLE1BQU0sS0FBSyxHQUFHO3dCQUNaLENBQUMsT0FBTyxDQUFDLE9BQU8sQ0FBQyxhQUFhLENBQUMsQ0FBQyxLQUFlLENBQUMsRUFBRSxPQUFPO3FCQUMxRCxDQUFDO29CQUNGLE1BQU0sWUFBWSxHQUFHLE1BQU0sT0FBTyxDQUFDLFlBQVksQ0FBQyxLQUFLLENBQUMsQ0FBQztvQkFDdkQsSUFBSSxZQUFZLENBQUMsUUFBUTt3QkFBRSxTQUFTO29CQUNwQyxNQUFNLFlBQVksQ0FBQztpQkFDcEI7cUJBQU07b0JBQ0wsTUFBTSxLQUFLLENBQUMsOEJBQThCLE9BQU8sQ0FBQyxPQUFPLENBQUMsYUFBYSxDQUFDLENBQUMsS0FBSyxFQUFFLENBQUMsQ0FBQztpQkFDbkY7YUFDRjtRQUNILENBQUM7S0FDRixDQUFDO0FBQ0osQ0FBQyJ9