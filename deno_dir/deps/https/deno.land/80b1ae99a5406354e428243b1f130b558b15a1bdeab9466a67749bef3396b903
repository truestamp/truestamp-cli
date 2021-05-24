import {
  DeserializeHandlerOptions,
  EndpointBearer,
  MetadataBearer,
  MiddlewareStack,
  Pluggable,
  RequestSerializer,
  ResponseDeserializer,
  SerializeHandlerOptions,
} from "../types/mod.ts";

import { deserializerMiddleware } from "./deserializerMiddleware.ts";
import { serializerMiddleware } from "./serializerMiddleware.ts";

export const deserializerMiddlewareOption: DeserializeHandlerOptions = {
  name: "deserializerMiddleware",
  step: "deserialize",
  tags: ["DESERIALIZER"],
  override: true,
};

export const serializerMiddlewareOption: SerializeHandlerOptions = {
  name: "serializerMiddleware",
  step: "serialize",
  tags: ["SERIALIZER"],
  override: true,
};

export function getSerdePlugin<
  InputType extends object,
  SerDeContext extends EndpointBearer,
  OutputType extends MetadataBearer
>(
  config: SerDeContext,
  serializer: RequestSerializer<any, SerDeContext>,
  deserializer: ResponseDeserializer<OutputType, any, SerDeContext>
): Pluggable<InputType, OutputType> {
  return {
    applyToStack: (commandStack: MiddlewareStack<InputType, OutputType>) => {
      commandStack.add(deserializerMiddleware(config, deserializer), deserializerMiddlewareOption);
      commandStack.add(serializerMiddleware(config, serializer), serializerMiddlewareOption);
    },
  };
}
