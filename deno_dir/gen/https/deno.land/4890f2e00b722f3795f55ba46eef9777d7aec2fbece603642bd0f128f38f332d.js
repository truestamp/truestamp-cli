import { ListObjectsV2Output, ListObjectsV2Request } from "../models/models_0.ts";
import { deserializeAws_restXmlListObjectsV2Command, serializeAws_restXmlListObjectsV2Command, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class ListObjectsV2Command extends $Command {
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
        const commandName = "ListObjectsV2Command";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: ListObjectsV2Request.filterSensitiveLog,
            outputFilterSensitiveLog: ListObjectsV2Output.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlListObjectsV2Command(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlListObjectsV2Command(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdE9iamVjdHNWMkNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJMaXN0T2JqZWN0c1YyQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsbUJBQW1CLEVBQUUsb0JBQW9CLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNsRixPQUFPLEVBQ0wsMENBQTBDLEVBQzFDLHdDQUF3QyxHQUN6QyxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBd0VqRSxNQUFNLE9BQU8sb0JBQXFCLFNBQVEsUUFJekM7SUFJc0I7SUFBckIsWUFBcUIsS0FBZ0M7UUFFbkQsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUEyQjtJQUlyRCxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsc0JBQXNCLENBQUM7UUFDM0MsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsb0JBQW9CLENBQUMsa0JBQWtCO1lBQ2hFLHdCQUF3QixFQUFFLG1CQUFtQixDQUFDLGtCQUFrQjtTQUNqRSxDQUFDO1FBQ0YsTUFBTSxFQUFFLGNBQWMsRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUN6QyxPQUFPLEtBQUssQ0FBQyxPQUFPLENBQ2xCLENBQUMsT0FBc0MsRUFBRSxFQUFFLENBQ3pDLGNBQWMsQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLE9BQXdCLEVBQUUsT0FBTyxJQUFJLEVBQUUsQ0FBQyxFQUN4RSx1QkFBdUIsQ0FDeEIsQ0FBQztJQUNKLENBQUM7SUFFTyxTQUFTLENBQUMsS0FBZ0MsRUFBRSxPQUF1QjtRQUN6RSxPQUFPLHdDQUF3QyxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUNsRSxDQUFDO0lBRU8sV0FBVyxDQUFDLE1BQXNCLEVBQUUsT0FBdUI7UUFDakUsT0FBTywwQ0FBMEMsQ0FBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDckUsQ0FBQztDQUlGIn0=