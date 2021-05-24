import { Decoder, Encoder, EventSigner, EventStreamSerdeProvider, Provider } from "../types/mod.ts";

import { EventStreamMarshaller } from "./EventStreamMarshaller.ts";

/** browser event stream serde utils provider */
export const eventStreamSerdeProvider: EventStreamSerdeProvider = (options: {
  utf8Encoder: Encoder;
  utf8Decoder: Decoder;
  eventSigner: EventSigner | Provider<EventSigner>;
}) => new EventStreamMarshaller(options);
