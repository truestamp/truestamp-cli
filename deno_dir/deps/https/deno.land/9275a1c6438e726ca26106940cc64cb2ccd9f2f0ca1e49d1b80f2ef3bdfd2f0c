import { HttpRequest } from "../protocol-http/mod.ts";
import { isThrottlingError } from "../service-error-classification/mod.ts";
import { SdkError } from "../types/mod.ts";
import { FinalizeHandler, FinalizeHandlerArguments, MetadataBearer, Provider, RetryStrategy } from "../types/mod.ts";
import { v4 } from "../uuid/mod.ts";

import { DEFAULT_MAX_ATTEMPTS, RETRY_MODES } from "./config.ts";
import {
  DEFAULT_RETRY_DELAY_BASE,
  INITIAL_RETRY_TOKENS,
  INVOCATION_ID_HEADER,
  REQUEST_HEADER,
  THROTTLING_RETRY_DELAY_BASE,
} from "./constants.ts";
import { getDefaultRetryQuota } from "./defaultRetryQuota.ts";
import { defaultDelayDecider } from "./delayDecider.ts";
import { defaultRetryDecider } from "./retryDecider.ts";
import { DelayDecider, RetryDecider, RetryQuota } from "./types.ts";

/**
 * Strategy options to be passed to StandardRetryStrategy
 */
export interface StandardRetryStrategyOptions {
  retryDecider?: RetryDecider;
  delayDecider?: DelayDecider;
  retryQuota?: RetryQuota;
}

export class StandardRetryStrategy implements RetryStrategy {
  private retryDecider: RetryDecider;
  private delayDecider: DelayDecider;
  private retryQuota: RetryQuota;
  public mode: string = RETRY_MODES.STANDARD;

  constructor(private readonly maxAttemptsProvider: Provider<number>, options?: StandardRetryStrategyOptions) {
    this.retryDecider = options?.retryDecider ?? defaultRetryDecider;
    this.delayDecider = options?.delayDecider ?? defaultDelayDecider;
    this.retryQuota = options?.retryQuota ?? getDefaultRetryQuota(INITIAL_RETRY_TOKENS);
  }

  private shouldRetry(error: SdkError, attempts: number, maxAttempts: number) {
    return attempts < maxAttempts && this.retryDecider(error) && this.retryQuota.hasRetryTokens(error);
  }

  private async getMaxAttempts() {
    let maxAttempts: number;
    try {
      maxAttempts = await this.maxAttemptsProvider();
    } catch (error) {
      maxAttempts = DEFAULT_MAX_ATTEMPTS;
    }
    return maxAttempts;
  }

  async retry<Input extends object, Ouput extends MetadataBearer>(
    next: FinalizeHandler<Input, Ouput>,
    args: FinalizeHandlerArguments<Input>,
    options?: {
      beforeRequest: Function;
      afterRequest: Function;
    }
  ) {
    let retryTokenAmount;
    let attempts = 0;
    let totalDelay = 0;

    const maxAttempts = await this.getMaxAttempts();

    const { request } = args;
    if (HttpRequest.isInstance(request)) {
      request.headers[INVOCATION_ID_HEADER] = v4();
    }

    while (true) {
      try {
        if (HttpRequest.isInstance(request)) {
          request.headers[REQUEST_HEADER] = `attempt=${attempts + 1}; max=${maxAttempts}`;
        }

        if (options?.beforeRequest) {
          await options.beforeRequest();
        }
        const { response, output } = await next(args);
        if (options?.afterRequest) {
          options.afterRequest(response);
        }

        this.retryQuota.releaseRetryTokens(retryTokenAmount);
        output.$metadata.attempts = attempts + 1;
        output.$metadata.totalRetryDelay = totalDelay;

        return { response, output };
      } catch (e) {
        const err = asSdkError(e);
        attempts++;
        if (this.shouldRetry(err as SdkError, attempts, maxAttempts)) {
          retryTokenAmount = this.retryQuota.retrieveRetryTokens(err);
          const delay = this.delayDecider(
            isThrottlingError(err) ? THROTTLING_RETRY_DELAY_BASE : DEFAULT_RETRY_DELAY_BASE,
            attempts
          );
          totalDelay += delay;

          await new Promise((resolve) => setTimeout(resolve, delay));
          continue;
        }

        if (!err.$metadata) {
          err.$metadata = {};
        }

        err.$metadata.attempts = attempts;
        err.$metadata.totalRetryDelay = totalDelay;
        throw err;
      }
    }
  }
}

const asSdkError = (error: unknown): SdkError => {
  if (error instanceof Error) return error;
  if (error instanceof Object) return Object.assign(new Error(), error);
  if (typeof error === "string") return new Error(error);
  return new Error(`AWS SDK error wrapper for ${error}`);
};
