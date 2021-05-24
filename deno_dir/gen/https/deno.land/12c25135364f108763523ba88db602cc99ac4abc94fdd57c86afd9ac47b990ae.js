import { AssumeRoleRequest, AssumeRoleResponse } from "../models/models_0.ts";
import { deserializeAws_queryAssumeRoleCommand, serializeAws_queryAssumeRoleCommand } from "../protocols/Aws_query.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { getAwsAuthPlugin } from "../../middleware-signing/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class AssumeRoleCommand extends $Command {
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
        const commandName = "AssumeRoleCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: AssumeRoleRequest.filterSensitiveLog,
            outputFilterSensitiveLog: AssumeRoleResponse.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_queryAssumeRoleCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_queryAssumeRoleCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQXNzdW1lUm9sZUNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJBc3N1bWVSb2xlQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsaUJBQWlCLEVBQUUsa0JBQWtCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUM5RSxPQUFPLEVBQUUscUNBQXFDLEVBQUUsbUNBQW1DLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUN2SCxPQUFPLEVBQUUsY0FBYyxFQUFFLE1BQU0sK0JBQStCLENBQUM7QUFDL0QsT0FBTyxFQUFFLGdCQUFnQixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFFbkUsT0FBTyxFQUFFLE9BQU8sSUFBSSxRQUFRLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQWtIakUsTUFBTSxPQUFPLGlCQUFrQixTQUFRLFFBSXRDO0lBSXNCO0lBQXJCLFlBQXFCLEtBQTZCO1FBRWhELEtBQUssRUFBRSxDQUFDO1FBRlcsVUFBSyxHQUFMLEtBQUssQ0FBd0I7SUFJbEQsQ0FBQztJQUtELGlCQUFpQixDQUNmLFdBQW1FLEVBQ25FLGFBQXNDLEVBQ3RDLE9BQThCO1FBRTlCLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGNBQWMsQ0FBQyxhQUFhLEVBQUUsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsV0FBVyxDQUFDLENBQUMsQ0FBQztRQUMxRixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxnQkFBZ0IsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBRTFELE1BQU0sS0FBSyxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLGVBQWUsQ0FBQyxDQUFDO1FBRXZELE1BQU0sRUFBRSxNQUFNLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDakMsTUFBTSxVQUFVLEdBQUcsV0FBVyxDQUFDO1FBQy9CLE1BQU0sV0FBVyxHQUFHLG1CQUFtQixDQUFDO1FBQ3hDLE1BQU0sdUJBQXVCLEdBQTRCO1lBQ3ZELE1BQU07WUFDTixVQUFVO1lBQ1YsV0FBVztZQUNYLHVCQUF1QixFQUFFLGlCQUFpQixDQUFDLGtCQUFrQjtZQUM3RCx3QkFBd0IsRUFBRSxrQkFBa0IsQ0FBQyxrQkFBa0I7U0FDaEUsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUFDLEtBQTZCLEVBQUUsT0FBdUI7UUFDdEUsT0FBTyxtQ0FBbUMsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDN0QsQ0FBQztJQUVPLFdBQVcsQ0FBQyxNQUFzQixFQUFFLE9BQXVCO1FBQ2pFLE9BQU8scUNBQXFDLENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ2hFLENBQUM7Q0FJRiJ9