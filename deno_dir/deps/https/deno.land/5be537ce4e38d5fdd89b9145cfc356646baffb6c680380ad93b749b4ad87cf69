import { S3ClientResolvedConfig, ServiceInputTypes, ServiceOutputTypes } from "../S3Client.ts";
import { GetObjectLockConfigurationOutput, GetObjectLockConfigurationRequest } from "../models/models_0.ts";
import {
  deserializeAws_restXmlGetObjectLockConfigurationCommand,
  serializeAws_restXmlGetObjectLockConfigurationCommand,
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

export interface GetObjectLockConfigurationCommandInput extends GetObjectLockConfigurationRequest {}
export interface GetObjectLockConfigurationCommandOutput extends GetObjectLockConfigurationOutput, __MetadataBearer {}

/**
 * <p>Gets the Object Lock configuration for a bucket. The rule specified in the Object Lock
 *          configuration will be applied by default to every new object placed in the specified
 *          bucket. For more information, see <a href="https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lock.html">Locking
 *             Objects</a>.</p>
 * @example
 * Use a bare-bones client and the command you need to make an API call.
 * ```javascript
 * import { S3Client, GetObjectLockConfigurationCommand } from "../../client-s3/mod.ts";
 * // const { S3Client, GetObjectLockConfigurationCommand } = require("@aws-sdk/client-s3"); // CommonJS import
 * const client = new S3Client(config);
 * const command = new GetObjectLockConfigurationCommand(input);
 * const response = await client.send(command);
 * ```
 *
 * @see {@link GetObjectLockConfigurationCommandInput} for command's `input` shape.
 * @see {@link GetObjectLockConfigurationCommandOutput} for command's `response` shape.
 * @see {@link S3ClientResolvedConfig | config} for command's `input` shape.
 *
 */
export class GetObjectLockConfigurationCommand extends $Command<
  GetObjectLockConfigurationCommandInput,
  GetObjectLockConfigurationCommandOutput,
  S3ClientResolvedConfig
> {
  // Start section: command_properties
  // End section: command_properties

  constructor(readonly input: GetObjectLockConfigurationCommandInput) {
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
  ): Handler<GetObjectLockConfigurationCommandInput, GetObjectLockConfigurationCommandOutput> {
    this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
    this.middlewareStack.use(getBucketEndpointPlugin(configuration));

    const stack = clientStack.concat(this.middlewareStack);

    const { logger } = configuration;
    const clientName = "S3Client";
    const commandName = "GetObjectLockConfigurationCommand";
    const handlerExecutionContext: HandlerExecutionContext = {
      logger,
      clientName,
      commandName,
      inputFilterSensitiveLog: GetObjectLockConfigurationRequest.filterSensitiveLog,
      outputFilterSensitiveLog: GetObjectLockConfigurationOutput.filterSensitiveLog,
    };
    const { requestHandler } = configuration;
    return stack.resolve(
      (request: FinalizeHandlerArguments<any>) =>
        requestHandler.handle(request.request as __HttpRequest, options || {}),
      handlerExecutionContext
    );
  }

  private serialize(input: GetObjectLockConfigurationCommandInput, context: __SerdeContext): Promise<__HttpRequest> {
    return serializeAws_restXmlGetObjectLockConfigurationCommand(input, context);
  }

  private deserialize(
    output: __HttpResponse,
    context: __SerdeContext
  ): Promise<GetObjectLockConfigurationCommandOutput> {
    return deserializeAws_restXmlGetObjectLockConfigurationCommand(output, context);
  }

  // Start section: command_body_extra
  // End section: command_body_extra
}
