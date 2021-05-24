import { SelectObjectContentOutput, SelectObjectContentRequest } from "../models/models_1.ts";
import { deserializeAws_restXmlSelectObjectContentCommand, serializeAws_restXmlSelectObjectContentCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { getSsecPlugin } from "../../middleware-ssec/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class SelectObjectContentCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getSsecPlugin(configuration));
        this.middlewareStack.use(getBucketEndpointPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "S3Client";
        const commandName = "SelectObjectContentCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: SelectObjectContentRequest.filterSensitiveLog,
            outputFilterSensitiveLog: SelectObjectContentOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlSelectObjectContentCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlSelectObjectContentCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiU2VsZWN0T2JqZWN0Q29udGVudENvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJTZWxlY3RPYmplY3RDb250ZW50Q29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUseUJBQXlCLEVBQUUsMEJBQTBCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUM5RixPQUFPLEVBQ0wsZ0RBQWdELEVBQ2hELDhDQUE4QyxHQUMvQyxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUMvRCxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sOEJBQThCLENBQUM7QUFFN0QsT0FBTyxFQUFFLE9BQU8sSUFBSSxRQUFRLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQXFKakUsTUFBTSxPQUFPLDBCQUEyQixTQUFRLFFBSS9DO0lBSXNCO0lBQXJCLFlBQXFCLEtBQXNDO1FBRXpELEtBQUssRUFBRSxDQUFDO1FBRlcsVUFBSyxHQUFMLEtBQUssQ0FBaUM7SUFJM0QsQ0FBQztJQUtELGlCQUFpQixDQUNmLFdBQW1FLEVBQ25FLGFBQXFDLEVBQ3JDLE9BQThCO1FBRTlCLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGNBQWMsQ0FBQyxhQUFhLEVBQUUsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsV0FBVyxDQUFDLENBQUMsQ0FBQztRQUMxRixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxhQUFhLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUN2RCxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyx1QkFBdUIsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBRWpFLE1BQU0sS0FBSyxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLGVBQWUsQ0FBQyxDQUFDO1FBRXZELE1BQU0sRUFBRSxNQUFNLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDakMsTUFBTSxVQUFVLEdBQUcsVUFBVSxDQUFDO1FBQzlCLE1BQU0sV0FBVyxHQUFHLDRCQUE0QixDQUFDO1FBQ2pELE1BQU0sdUJBQXVCLEdBQTRCO1lBQ3ZELE1BQU07WUFDTixVQUFVO1lBQ1YsV0FBVztZQUNYLHVCQUF1QixFQUFFLDBCQUEwQixDQUFDLGtCQUFrQjtZQUN0RSx3QkFBd0IsRUFBRSx5QkFBeUIsQ0FBQyxrQkFBa0I7U0FDdkUsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUFDLEtBQXNDLEVBQUUsT0FBdUI7UUFDL0UsT0FBTyw4Q0FBOEMsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDeEUsQ0FBQztJQUVPLFdBQVcsQ0FDakIsTUFBc0IsRUFDdEIsT0FBbUQ7UUFFbkQsT0FBTyxnREFBZ0QsQ0FBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDM0UsQ0FBQztDQUlGIn0=