import { EventStreamMarshaller as EventMarshaller } from "../eventstream-marshaller/mod.ts";
import { getChunkedStream } from "./getChunkedStream.ts";
import { getUnmarshalledStream } from "./getUnmarshalledStream.ts";
export class EventStreamMarshaller {
    eventMarshaller;
    utfEncoder;
    constructor({ utf8Encoder, utf8Decoder }) {
        this.eventMarshaller = new EventMarshaller(utf8Encoder, utf8Decoder);
        this.utfEncoder = utf8Encoder;
    }
    deserialize(body, deserializer) {
        const chunkedStream = getChunkedStream(body);
        const unmarshalledStream = getUnmarshalledStream(chunkedStream, {
            eventMarshaller: this.eventMarshaller,
            deserializer,
            toUtf8: this.utfEncoder,
        });
        return unmarshalledStream;
    }
    serialize(input, serializer) {
        const self = this;
        const serializedIterator = async function* () {
            for await (const chunk of input) {
                const payloadBuf = self.eventMarshaller.marshall(serializer(chunk));
                yield payloadBuf;
            }
            yield new Uint8Array(0);
        };
        return {
            [Symbol.asyncIterator]: serializedIterator,
        };
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRXZlbnRTdHJlYW1NYXJzaGFsbGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiRXZlbnRTdHJlYW1NYXJzaGFsbGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxxQkFBcUIsSUFBSSxlQUFlLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUc1RixPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUN6RCxPQUFPLEVBQUUscUJBQXFCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQVNuRSxNQUFNLE9BQU8scUJBQXFCO0lBQ2YsZUFBZSxDQUFrQjtJQUNqQyxVQUFVLENBQVU7SUFDckMsWUFBWSxFQUFFLFdBQVcsRUFBRSxXQUFXLEVBQWdDO1FBQ3BFLElBQUksQ0FBQyxlQUFlLEdBQUcsSUFBSSxlQUFlLENBQUMsV0FBVyxFQUFFLFdBQVcsQ0FBQyxDQUFDO1FBQ3JFLElBQUksQ0FBQyxVQUFVLEdBQUcsV0FBVyxDQUFDO0lBQ2hDLENBQUM7SUFFRCxXQUFXLENBQ1QsSUFBK0IsRUFDL0IsWUFBaUU7UUFFakUsTUFBTSxhQUFhLEdBQUcsZ0JBQWdCLENBQUMsSUFBSSxDQUFDLENBQUM7UUFDN0MsTUFBTSxrQkFBa0IsR0FBRyxxQkFBcUIsQ0FBQyxhQUFhLEVBQUU7WUFDOUQsZUFBZSxFQUFFLElBQUksQ0FBQyxlQUFlO1lBQ3JDLFlBQVk7WUFDWixNQUFNLEVBQUUsSUFBSSxDQUFDLFVBQVU7U0FDeEIsQ0FBQyxDQUFDO1FBQ0gsT0FBTyxrQkFBa0IsQ0FBQztJQUM1QixDQUFDO0lBRUQsU0FBUyxDQUFJLEtBQXVCLEVBQUUsVUFBaUM7UUFFckUsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2xCLE1BQU0sa0JBQWtCLEdBQUcsS0FBSyxTQUFTLENBQUM7WUFDeEMsSUFBSSxLQUFLLEVBQUUsTUFBTSxLQUFLLElBQUksS0FBSyxFQUFFO2dCQUMvQixNQUFNLFVBQVUsR0FBRyxJQUFJLENBQUMsZUFBZSxDQUFDLFFBQVEsQ0FBQyxVQUFVLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQztnQkFDcEUsTUFBTSxVQUFVLENBQUM7YUFDbEI7WUFFRCxNQUFNLElBQUksVUFBVSxDQUFDLENBQUMsQ0FBQyxDQUFDO1FBQzFCLENBQUMsQ0FBQztRQUNGLE9BQU87WUFDTCxDQUFDLE1BQU0sQ0FBQyxhQUFhLENBQUMsRUFBRSxrQkFBa0I7U0FDM0MsQ0FBQztJQUNKLENBQUM7Q0FDRiJ9