import { ListBucketMetricsConfigurationsOutput, ListBucketMetricsConfigurationsRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlListBucketMetricsConfigurationsCommand, serializeAws_restXmlListBucketMetricsConfigurationsCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class ListBucketMetricsConfigurationsCommand extends $Command {
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
        const commandName = "ListBucketMetricsConfigurationsCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: ListBucketMetricsConfigurationsRequest.filterSensitiveLog,
            outputFilterSensitiveLog: ListBucketMetricsConfigurationsOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlListBucketMetricsConfigurationsCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlListBucketMetricsConfigurationsCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdEJ1Y2tldE1ldHJpY3NDb25maWd1cmF0aW9uc0NvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJMaXN0QnVja2V0TWV0cmljc0NvbmZpZ3VyYXRpb25zQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUscUNBQXFDLEVBQUUsc0NBQXNDLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUN0SCxPQUFPLEVBQ0wsNERBQTRELEVBQzVELDBEQUEwRCxHQUMzRCxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBeUVqRSxNQUFNLE9BQU8sc0NBQXVDLFNBQVEsUUFJM0Q7SUFJc0I7SUFBckIsWUFBcUIsS0FBa0Q7UUFFckUsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUE2QztJQUl2RSxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsd0NBQXdDLENBQUM7UUFDN0QsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsc0NBQXNDLENBQUMsa0JBQWtCO1lBQ2xGLHdCQUF3QixFQUFFLHFDQUFxQyxDQUFDLGtCQUFrQjtTQUNuRixDQUFDO1FBQ0YsTUFBTSxFQUFFLGNBQWMsRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUN6QyxPQUFPLEtBQUssQ0FBQyxPQUFPLENBQ2xCLENBQUMsT0FBc0MsRUFBRSxFQUFFLENBQ3pDLGNBQWMsQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLE9BQXdCLEVBQUUsT0FBTyxJQUFJLEVBQUUsQ0FBQyxFQUN4RSx1QkFBdUIsQ0FDeEIsQ0FBQztJQUNKLENBQUM7SUFFTyxTQUFTLENBQ2YsS0FBa0QsRUFDbEQsT0FBdUI7UUFFdkIsT0FBTywwREFBMEQsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDcEYsQ0FBQztJQUVPLFdBQVcsQ0FDakIsTUFBc0IsRUFDdEIsT0FBdUI7UUFFdkIsT0FBTyw0REFBNEQsQ0FBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDdkYsQ0FBQztDQUlGIn0=