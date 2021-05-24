import { Decoder, Encoder, Hash } from "../types/mod.ts";

export interface SsecMiddlewareInputConfig {}

interface PreviouslyResolved {
  base64Encoder: Encoder;
  md5: { new (): Hash };
  utf8Decoder: Decoder;
}

export interface ResolvedSsecMiddlewareConfig {
  base64Encoder: Encoder;
  md5: { new (): Hash };
  utf8Decoder: Decoder;
}

export function resolveSsecMiddlewareConfig<T>(
  input: T & PreviouslyResolved & SsecMiddlewareInputConfig
): T & ResolvedSsecMiddlewareConfig {
  return {
    ...input,
  };
}
