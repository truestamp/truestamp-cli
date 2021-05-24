import { S3ClientResolvedConfig, ServiceInputTypes, ServiceOutputTypes } from "../S3Client.ts";
import { GetObjectRetentionOutput, GetObjectRetentionRequest } from "../models/models_0.ts";
import {
  deserializeAws_restXmlGetObjectRetentionCommand,
  serializeAws_restXmlGetObjectRetentionCommand,
} from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { HttpRequest as __HttpRequest, HttpResponse as __HttpResponse } from "../../protocol-http/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
import {
  FinalizeHandlerArguments,
  Handler,
  HandlerExecutionContext,
  MiddlewareStack,
  HttpHandlerOptions as __HttpHandlerOptions,
  MetadataBearer as __MetadataBearer,
  SerdeContext as __SerdeContext,
} from "../../types/mod.ts";

export interface GetObjectRetentionCommandInput extends GetObjectRetentionRequest {}
export interface GetObjectRetentionCommandOutput extends GetObjectRetentionOutput, __MetadataBearer {}

/**
 * <p>Retrieves an object's retention settings. For more information, see <a href="https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lock.html">Locking Objects</a>.</p>
 *          <p>This action is not supported by Amazon S3 on Outposts.</p>
 * @example
 * Use a bare-bones client and the command you need to make an API call.
 * ```javascript
 * import { S3Client, GetObjectRetentionCommand } from "../../client-s3/mod.ts";
 * // const { S3Client, GetObjectRetentionCommand } = require("@aws-sdk/client-s3"); // CommonJS import
 * const client = new S3Client(config);
 * const command = new GetObjectRetentionCommand(input);
 * const response = await client.send(command);
 * ```
 *
 * @see {@link GetObjectRetentionCommandInput} for command's `input` shape.
 * @see {@link GetObjectRetentionCommandOutput} for command's `response` shape.
 * @see {@link S3ClientResolvedConfig | config} for command's `input` shape.
 *
 */
export class GetObjectRetentionCommand extends $Command<
  GetObjectRetentionCommandInput,
  GetObjectRetentionCommandOutput,
  S3ClientResolvedConfig
> {
  // Start section: command_properties
  // End section: command_properties

  constructor(readonly input: GetObjectRetentionCommandInput) {
    // Start section: command_constructor
    super();
    // End section: command_constructor
  }

  /**
   * @internal
   */
  resolveMiddleware(
    clientStack: MiddlewareStack<ServiceInputTypes, ServiceOutputTypes>,
    configuration: S3ClientResolvedConfig,
    options?: __HttpHandlerOptions
  ): Handler<GetObjectRetentionCommandInput, GetObjectRetentionCommandOutput> {
    this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
    this.middlewareStack.use(getBucketEndpointPlugin(configuration));

    const stack = clientStack.concat(this.middlewareStack);

    const { logger } = configuration;
    const clientName = "S3Client";
    const commandName = "GetObjectRetentionCommand";
    const handlerExecutionContext: HandlerExecutionContext = {
      logger,
      clientName,
      commandName,
      inputFilterSensitiveLog: GetObjectRetentionRequest.filterSensitiveLog,
      outputFilterSensitiveLog: GetObjectRetentionOutput.filterSensitiveLog,
    };
    const { requestHandler } = configuration;
    return stack.resolve(
      (request: FinalizeHandlerArguments<any>) =>
        requestHandler.handle(request.request as __HttpRequest, options || {}),
      handlerExecutionContext
    );
  }

  private serialize(input: GetObjectRetentionCommandInput, context: __SerdeContext): Promise<__HttpRequest> {
    return serializeAws_restXmlGetObjectRetentionCommand(input, context);
  }

  private deserialize(output: __HttpResponse, context: __SerdeContext): Promise<GetObjectRetentionCommandOutput> {
    return deserializeAws_restXmlGetObjectRetentionCommand(output, context);
  }

  // Start section: command_body_extra
  // End section: command_body_extra
}
