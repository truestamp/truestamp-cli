import { RETRY_MODES } from "./config.ts";
import { DefaultRateLimiter } from "./DefaultRateLimiter.ts";
import { StandardRetryStrategy } from "./StandardRetryStrategy.ts";
export class AdaptiveRetryStrategy extends StandardRetryStrategy {
    rateLimiter;
    constructor(maxAttemptsProvider, options) {
        const { rateLimiter, ...superOptions } = options ?? {};
        super(maxAttemptsProvider, superOptions);
        this.rateLimiter = rateLimiter ?? new DefaultRateLimiter();
        this.mode = RETRY_MODES.ADAPTIVE;
    }
    async retry(next, args) {
        return super.retry(next, args, {
            beforeRequest: async () => {
                return this.rateLimiter.getSendToken();
            },
            afterRequest: (response) => {
                this.rateLimiter.updateClientSendingRate(response);
            },
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQWRhcHRpdmVSZXRyeVN0cmF0ZWd5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiQWRhcHRpdmVSZXRyeVN0cmF0ZWd5LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSxhQUFhLENBQUM7QUFDMUMsT0FBTyxFQUFFLGtCQUFrQixFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFDN0QsT0FBTyxFQUFFLHFCQUFxQixFQUFnQyxNQUFNLDRCQUE0QixDQUFDO0FBVWpHLE1BQU0sT0FBTyxxQkFBc0IsU0FBUSxxQkFBcUI7SUFDdEQsV0FBVyxDQUFjO0lBRWpDLFlBQVksbUJBQXFDLEVBQUUsT0FBc0M7UUFDdkYsTUFBTSxFQUFFLFdBQVcsRUFBRSxHQUFHLFlBQVksRUFBRSxHQUFHLE9BQU8sSUFBSSxFQUFFLENBQUM7UUFDdkQsS0FBSyxDQUFDLG1CQUFtQixFQUFFLFlBQVksQ0FBQyxDQUFDO1FBQ3pDLElBQUksQ0FBQyxXQUFXLEdBQUcsV0FBVyxJQUFJLElBQUksa0JBQWtCLEVBQUUsQ0FBQztRQUMzRCxJQUFJLENBQUMsSUFBSSxHQUFHLFdBQVcsQ0FBQyxRQUFRLENBQUM7SUFDbkMsQ0FBQztJQUVELEtBQUssQ0FBQyxLQUFLLENBQ1QsSUFBbUMsRUFDbkMsSUFBcUM7UUFFckMsT0FBTyxLQUFLLENBQUMsS0FBSyxDQUFDLElBQUksRUFBRSxJQUFJLEVBQUU7WUFDN0IsYUFBYSxFQUFFLEtBQUssSUFBSSxFQUFFO2dCQUN4QixPQUFPLElBQUksQ0FBQyxXQUFXLENBQUMsWUFBWSxFQUFFLENBQUM7WUFDekMsQ0FBQztZQUNELFlBQVksRUFBRSxDQUFDLFFBQWEsRUFBRSxFQUFFO2dCQUM5QixJQUFJLENBQUMsV0FBVyxDQUFDLHVCQUF1QixDQUFDLFFBQVEsQ0FBQyxDQUFDO1lBQ3JELENBQUM7U0FDRixDQUFDLENBQUM7SUFDTCxDQUFDO0NBQ0YifQ==