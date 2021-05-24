import { EventStreamMarshaller as EventMarshaller } from "../eventstream-marshaller/mod.ts";
import { EventStreamMarshaller as UniversalEventStreamMarshaller } from "../eventstream-serde-universal/mod.ts";
import { iterableToReadableStream, readableStreamtoIterable } from "./utils.ts";
export class EventStreamMarshaller {
    eventMarshaller;
    universalMarshaller;
    constructor({ utf8Encoder, utf8Decoder }) {
        this.eventMarshaller = new EventMarshaller(utf8Encoder, utf8Decoder);
        this.universalMarshaller = new UniversalEventStreamMarshaller({
            utf8Decoder,
            utf8Encoder,
        });
    }
    deserialize(body, deserializer) {
        const bodyIterable = isReadableStream(body) ? readableStreamtoIterable(body) : body;
        return this.universalMarshaller.deserialize(bodyIterable, deserializer);
    }
    serialize(input, serializer) {
        const serialziedIterable = this.universalMarshaller.serialize(input, serializer);
        return typeof ReadableStream === "function" ? iterableToReadableStream(serialziedIterable) : serialziedIterable;
    }
}
const isReadableStream = (body) => typeof ReadableStream === "function" && body instanceof ReadableStream;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRXZlbnRTdHJlYW1NYXJzaGFsbGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiRXZlbnRTdHJlYW1NYXJzaGFsbGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxxQkFBcUIsSUFBSSxlQUFlLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUM1RixPQUFPLEVBQUUscUJBQXFCLElBQUksOEJBQThCLEVBQUUsTUFBTSx1Q0FBdUMsQ0FBQztBQUdoSCxPQUFPLEVBQUUsd0JBQXdCLEVBQUUsd0JBQXdCLEVBQUUsTUFBTSxZQUFZLENBQUM7QUF5QmhGLE1BQU0sT0FBTyxxQkFBcUI7SUFDZixlQUFlLENBQWtCO0lBQ2pDLG1CQUFtQixDQUFpQztJQUNyRSxZQUFZLEVBQUUsV0FBVyxFQUFFLFdBQVcsRUFBZ0M7UUFDcEUsSUFBSSxDQUFDLGVBQWUsR0FBRyxJQUFJLGVBQWUsQ0FBQyxXQUFXLEVBQUUsV0FBVyxDQUFDLENBQUM7UUFDckUsSUFBSSxDQUFDLG1CQUFtQixHQUFHLElBQUksOEJBQThCLENBQUM7WUFDNUQsV0FBVztZQUNYLFdBQVc7U0FDWixDQUFDLENBQUM7SUFDTCxDQUFDO0lBRUQsV0FBVyxDQUNULElBQTRELEVBQzVELFlBQWlFO1FBRWpFLE1BQU0sWUFBWSxHQUFHLGdCQUFnQixDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQyx3QkFBd0IsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO1FBQ3BGLE9BQU8sSUFBSSxDQUFDLG1CQUFtQixDQUFDLFdBQVcsQ0FBQyxZQUFZLEVBQUUsWUFBWSxDQUFDLENBQUM7SUFDMUUsQ0FBQztJQVlELFNBQVMsQ0FBSSxLQUF1QixFQUFFLFVBQWlDO1FBQ3JFLE1BQU0sa0JBQWtCLEdBQUcsSUFBSSxDQUFDLG1CQUFtQixDQUFDLFNBQVMsQ0FBQyxLQUFLLEVBQUUsVUFBVSxDQUFDLENBQUM7UUFDakYsT0FBTyxPQUFPLGNBQWMsS0FBSyxVQUFVLENBQUMsQ0FBQyxDQUFDLHdCQUF3QixDQUFDLGtCQUFrQixDQUFDLENBQUMsQ0FBQyxDQUFDLGtCQUFrQixDQUFDO0lBQ2xILENBQUM7Q0FDRjtBQUVELE1BQU0sZ0JBQWdCLEdBQUcsQ0FBQyxJQUFTLEVBQTBCLEVBQUUsQ0FDN0QsT0FBTyxjQUFjLEtBQUssVUFBVSxJQUFJLElBQUksWUFBWSxjQUFjLENBQUMifQ==