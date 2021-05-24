import { CopyObjectOutput, CopyObjectRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlCopyObjectCommand, serializeAws_restXmlCopyObjectCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getThrow200ExceptionsPlugin } from "../../middleware-sdk-s3/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { getSsecPlugin } from "../../middleware-ssec/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class CopyObjectCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getThrow200ExceptionsPlugin(configuration));
        this.middlewareStack.use(getSsecPlugin(configuration));
        this.middlewareStack.use(getBucketEndpointPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "S3Client";
        const commandName = "CopyObjectCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: CopyObjectRequest.filterSensitiveLog,
            outputFilterSensitiveLog: CopyObjectOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlCopyObjectCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlCopyObjectCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQ29weU9iamVjdENvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJDb3B5T2JqZWN0Q29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsaUJBQWlCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUM1RSxPQUFPLEVBQ0wsdUNBQXVDLEVBQ3ZDLHFDQUFxQyxHQUN0QyxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSwyQkFBMkIsRUFBRSxNQUFNLGdDQUFnQyxDQUFDO0FBQzdFLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUMvRCxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sOEJBQThCLENBQUM7QUFFN0QsT0FBTyxFQUFFLE9BQU8sSUFBSSxRQUFRLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQXdNakUsTUFBTSxPQUFPLGlCQUFrQixTQUFRLFFBSXRDO0lBSXNCO0lBQXJCLFlBQXFCLEtBQTZCO1FBRWhELEtBQUssRUFBRSxDQUFDO1FBRlcsVUFBSyxHQUFMLEtBQUssQ0FBd0I7SUFJbEQsQ0FBQztJQUtELGlCQUFpQixDQUNmLFdBQW1FLEVBQ25FLGFBQXFDLEVBQ3JDLE9BQThCO1FBRTlCLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGNBQWMsQ0FBQyxhQUFhLEVBQUUsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsV0FBVyxDQUFDLENBQUMsQ0FBQztRQUMxRixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQywyQkFBMkIsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBQ3JFLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGFBQWEsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBQ3ZELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsbUJBQW1CLENBQUM7UUFDeEMsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsaUJBQWlCLENBQUMsa0JBQWtCO1lBQzdELHdCQUF3QixFQUFFLGdCQUFnQixDQUFDLGtCQUFrQjtTQUM5RCxDQUFDO1FBQ0YsTUFBTSxFQUFFLGNBQWMsRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUN6QyxPQUFPLEtBQUssQ0FBQyxPQUFPLENBQ2xCLENBQUMsT0FBc0MsRUFBRSxFQUFFLENBQ3pDLGNBQWMsQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLE9BQXdCLEVBQUUsT0FBTyxJQUFJLEVBQUUsQ0FBQyxFQUN4RSx1QkFBdUIsQ0FDeEIsQ0FBQztJQUNKLENBQUM7SUFFTyxTQUFTLENBQUMsS0FBNkIsRUFBRSxPQUF1QjtRQUN0RSxPQUFPLHFDQUFxQyxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUMvRCxDQUFDO0lBRU8sV0FBVyxDQUFDLE1BQXNCLEVBQUUsT0FBdUI7UUFDakUsT0FBTyx1Q0FBdUMsQ0FBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDbEUsQ0FBQztDQUlGIn0=