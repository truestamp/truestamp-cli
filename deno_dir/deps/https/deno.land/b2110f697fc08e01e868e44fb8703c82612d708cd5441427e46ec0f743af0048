import { SSOClientResolvedConfig, ServiceInputTypes, ServiceOutputTypes } from "../SSOClient.ts";
import { ListAccountsRequest, ListAccountsResponse } from "../models/models_0.ts";
import {
  deserializeAws_restJson1ListAccountsCommand,
  serializeAws_restJson1ListAccountsCommand,
} from "../protocols/Aws_restJson1.ts";
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

export interface ListAccountsCommandInput extends ListAccountsRequest {}
export interface ListAccountsCommandOutput extends ListAccountsResponse, __MetadataBearer {}

/**
 * <p>Lists all AWS accounts assigned to the user. These AWS accounts are assigned by the
 *       administrator of the account. For more information, see <a href="https://docs.aws.amazon.com/singlesignon/latest/userguide/useraccess.html#assignusers">Assign User Access</a> in the <i>AWS SSO User Guide</i>. This operation
 *       returns a paginated response.</p>
 * @example
 * Use a bare-bones client and the command you need to make an API call.
 * ```javascript
 * import { SSOClient, ListAccountsCommand } from "../../client-sso/mod.ts";
 * // const { SSOClient, ListAccountsCommand } = require("@aws-sdk/client-sso"); // CommonJS import
 * const client = new SSOClient(config);
 * const command = new ListAccountsCommand(input);
 * const response = await client.send(command);
 * ```
 *
 * @see {@link ListAccountsCommandInput} for command's `input` shape.
 * @see {@link ListAccountsCommandOutput} for command's `response` shape.
 * @see {@link SSOClientResolvedConfig | config} for command's `input` shape.
 *
 */
export class ListAccountsCommand extends $Command<
  ListAccountsCommandInput,
  ListAccountsCommandOutput,
  SSOClientResolvedConfig
> {
  // Start section: command_properties
  // End section: command_properties

  constructor(readonly input: ListAccountsCommandInput) {
    // Start section: command_constructor
    super();
    // End section: command_constructor
  }

  /**
   * @internal
   */
  resolveMiddleware(
    clientStack: MiddlewareStack<ServiceInputTypes, ServiceOutputTypes>,
    configuration: SSOClientResolvedConfig,
    options?: __HttpHandlerOptions
  ): Handler<ListAccountsCommandInput, ListAccountsCommandOutput> {
    this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));

    const stack = clientStack.concat(this.middlewareStack);

    const { logger } = configuration;
    const clientName = "SSOClient";
    const commandName = "ListAccountsCommand";
    const handlerExecutionContext: HandlerExecutionContext = {
      logger,
      clientName,
      commandName,
      inputFilterSensitiveLog: ListAccountsRequest.filterSensitiveLog,
      outputFilterSensitiveLog: ListAccountsResponse.filterSensitiveLog,
    };
    const { requestHandler } = configuration;
    return stack.resolve(
      (request: FinalizeHandlerArguments<any>) =>
        requestHandler.handle(request.request as __HttpRequest, options || {}),
      handlerExecutionContext
    );
  }

  private serialize(input: ListAccountsCommandInput, context: __SerdeContext): Promise<__HttpRequest> {
    return serializeAws_restJson1ListAccountsCommand(input, context);
  }

  private deserialize(output: __HttpResponse, context: __SerdeContext): Promise<ListAccountsCommandOutput> {
    return deserializeAws_restJson1ListAccountsCommand(output, context);
  }

  // Start section: command_body_extra
  // End section: command_body_extra
}
