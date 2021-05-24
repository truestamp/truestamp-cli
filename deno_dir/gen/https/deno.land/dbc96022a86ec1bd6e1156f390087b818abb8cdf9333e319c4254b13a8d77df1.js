import { ListBucketInventoryConfigurationsOutput, ListBucketInventoryConfigurationsRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlListBucketInventoryConfigurationsCommand, serializeAws_restXmlListBucketInventoryConfigurationsCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class ListBucketInventoryConfigurationsCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getBucketEndpointPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "S3Client";
        const commandName = "ListBucketInventoryConfigurationsCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: ListBucketInventoryConfigurationsRequest.filterSensitiveLog,
            outputFilterSensitiveLog: ListBucketInventoryConfigurationsOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlListBucketInventoryConfigurationsCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlListBucketInventoryConfigurationsCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdEJ1Y2tldEludmVudG9yeUNvbmZpZ3VyYXRpb25zQ29tbWFuZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkxpc3RCdWNrZXRJbnZlbnRvcnlDb25maWd1cmF0aW9uc0NvbW1hbmQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLHVDQUF1QyxFQUFFLHdDQUF3QyxFQUFFLE1BQU0sdUJBQXVCLENBQUM7QUFDMUgsT0FBTyxFQUNMLDhEQUE4RCxFQUM5RCw0REFBNEQsR0FDN0QsTUFBTSw2QkFBNkIsQ0FBQztBQUNyQyxPQUFPLEVBQUUsdUJBQXVCLEVBQUUsTUFBTSx5Q0FBeUMsQ0FBQztBQUNsRixPQUFPLEVBQUUsY0FBYyxFQUFFLE1BQU0sK0JBQStCLENBQUM7QUFFL0QsT0FBTyxFQUFFLE9BQU8sSUFBSSxRQUFRLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQXVFakUsTUFBTSxPQUFPLHdDQUF5QyxTQUFRLFFBSTdEO0lBSXNCO0lBQXJCLFlBQXFCLEtBQW9EO1FBRXZFLEtBQUssRUFBRSxDQUFDO1FBRlcsVUFBSyxHQUFMLEtBQUssQ0FBK0M7SUFJekUsQ0FBQztJQUtELGlCQUFpQixDQUNmLFdBQW1FLEVBQ25FLGFBQXFDLEVBQ3JDLE9BQThCO1FBRTlCLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGNBQWMsQ0FBQyxhQUFhLEVBQUUsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsV0FBVyxDQUFDLENBQUMsQ0FBQztRQUMxRixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyx1QkFBdUIsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBRWpFLE1BQU0sS0FBSyxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLGVBQWUsQ0FBQyxDQUFDO1FBRXZELE1BQU0sRUFBRSxNQUFNLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDakMsTUFBTSxVQUFVLEdBQUcsVUFBVSxDQUFDO1FBQzlCLE1BQU0sV0FBVyxHQUFHLDBDQUEwQyxDQUFDO1FBQy9ELE1BQU0sdUJBQXVCLEdBQTRCO1lBQ3ZELE1BQU07WUFDTixVQUFVO1lBQ1YsV0FBVztZQUNYLHVCQUF1QixFQUFFLHdDQUF3QyxDQUFDLGtCQUFrQjtZQUNwRix3QkFBd0IsRUFBRSx1Q0FBdUMsQ0FBQyxrQkFBa0I7U0FDckYsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUNmLEtBQW9ELEVBQ3BELE9BQXVCO1FBRXZCLE9BQU8sNERBQTRELENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3RGLENBQUM7SUFFTyxXQUFXLENBQ2pCLE1BQXNCLEVBQ3RCLE9BQXVCO1FBRXZCLE9BQU8sOERBQThELENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3pGLENBQUM7Q0FJRiJ9