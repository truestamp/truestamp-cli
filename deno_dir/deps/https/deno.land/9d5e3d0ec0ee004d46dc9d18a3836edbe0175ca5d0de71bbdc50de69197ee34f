const packageInfo = { version: "3.17.0" };

import { decorateDefaultCredentialProvider } from "../client-sts/mod.ts";
import { NODE_REGION_CONFIG_FILE_OPTIONS, NODE_REGION_CONFIG_OPTIONS } from "../config-resolver/mod.ts";
import { defaultProvider as credentialDefaultProvider } from "../credential-provider-node/mod.ts";
import { eventStreamSerdeProvider } from "../eventstream-serde-browser/mod.ts";
import { Hash } from "https://jspm.dev/@aws-sdk/hash-node";
import { blobHasher as streamHasher } from "../hash-blob-browser/mod.ts";
import { NODE_USE_ARN_REGION_CONFIG_OPTIONS } from "../middleware-bucket-endpoint/mod.ts";
import { NODE_MAX_ATTEMPT_CONFIG_OPTIONS } from "../middleware-retry/mod.ts";
import { loadConfig as loadNodeConfig } from "../node-config-provider/mod.ts";
import { FetchHttpHandler, streamCollector } from "../fetch-http-handler/mod.ts";
import { HashConstructor as __HashConstructor } from "../types/mod.ts";
import { fromBase64, toBase64 } from "../util-base64-node/mod.ts";
import { calculateBodyLength } from "../util-body-length-node/mod.ts";
import { defaultUserAgent } from "../util-user-agent-node/mod.ts";
import { fromUtf8, toUtf8 } from "../util-utf8-node/mod.ts";
import { ClientDefaults } from "./S3Client.ts";
import { ClientSharedValues } from "./runtimeConfig.shared.ts";

/**
 * @internal
 */
export const ClientDefaultValues: Required<ClientDefaults> = {
  ...ClientSharedValues,
  runtime: "deno",
  base64Decoder: fromBase64,
  base64Encoder: toBase64,
  bodyLengthChecker: calculateBodyLength,
  credentialDefaultProvider: decorateDefaultCredentialProvider(credentialDefaultProvider),
  defaultUserAgentProvider: defaultUserAgent({
    serviceId: ClientSharedValues.serviceId,
    clientVersion: packageInfo.version,
  }),
  eventStreamSerdeProvider,
  maxAttempts: loadNodeConfig(NODE_MAX_ATTEMPT_CONFIG_OPTIONS),
  md5: Hash.bind(null, "md5"),
  region: loadNodeConfig(NODE_REGION_CONFIG_OPTIONS, NODE_REGION_CONFIG_FILE_OPTIONS),
  requestHandler: new FetchHttpHandler(),
  sha256: Hash.bind(null, "sha256"),
  streamCollector,
  streamHasher,
  useArnRegion: loadNodeConfig(NODE_USE_ARN_REGION_CONFIG_OPTIONS),
  utf8Decoder: fromUtf8,
  utf8Encoder: toUtf8,
};
