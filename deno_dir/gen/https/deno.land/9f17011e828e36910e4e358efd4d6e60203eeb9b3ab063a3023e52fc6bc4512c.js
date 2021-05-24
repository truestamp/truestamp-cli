import { GetCallerIdentityRequest, GetCallerIdentityResponse } from "../models/models_0.ts";
import { deserializeAws_queryGetCallerIdentityCommand, serializeAws_queryGetCallerIdentityCommand, } from "../protocols/Aws_query.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { getAwsAuthPlugin } from "../../middleware-signing/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class GetCallerIdentityCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getAwsAuthPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "STSClient";
        const commandName = "GetCallerIdentityCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: GetCallerIdentityRequest.filterSensitiveLog,
            outputFilterSensitiveLog: GetCallerIdentityResponse.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_queryGetCallerIdentityCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_queryGetCallerIdentityCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiR2V0Q2FsbGVySWRlbnRpdHlDb21tYW5kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiR2V0Q2FsbGVySWRlbnRpdHlDb21tYW5kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSx5QkFBeUIsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQzVGLE9BQU8sRUFDTCw0Q0FBNEMsRUFDNUMsMENBQTBDLEdBQzNDLE1BQU0sMkJBQTJCLENBQUM7QUFDbkMsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLCtCQUErQixDQUFDO0FBQy9ELE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxNQUFNLGlDQUFpQyxDQUFDO0FBRW5FLE9BQU8sRUFBRSxPQUFPLElBQUksUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUF3Q2pFLE1BQU0sT0FBTyx3QkFBeUIsU0FBUSxRQUk3QztJQUlzQjtJQUFyQixZQUFxQixLQUFvQztRQUV2RCxLQUFLLEVBQUUsQ0FBQztRQUZXLFVBQUssR0FBTCxLQUFLLENBQStCO0lBSXpELENBQUM7SUFLRCxpQkFBaUIsQ0FDZixXQUFtRSxFQUNuRSxhQUFzQyxFQUN0QyxPQUE4QjtRQUU5QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUM7UUFDMUYsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsZ0JBQWdCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUUxRCxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFdBQVcsQ0FBQztRQUMvQixNQUFNLFdBQVcsR0FBRywwQkFBMEIsQ0FBQztRQUMvQyxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSx3QkFBd0IsQ0FBQyxrQkFBa0I7WUFDcEUsd0JBQXdCLEVBQUUseUJBQXlCLENBQUMsa0JBQWtCO1NBQ3ZFLENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FBQyxLQUFvQyxFQUFFLE9BQXVCO1FBQzdFLE9BQU8sMENBQTBDLENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3BFLENBQUM7SUFFTyxXQUFXLENBQUMsTUFBc0IsRUFBRSxPQUF1QjtRQUNqRSxPQUFPLDRDQUE0QyxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN2RSxDQUFDO0NBSUYifQ==