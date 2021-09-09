import {
  Command,
  createTruestampClient,
  getConfigRefreshToken,
  ValidationError
} from "../../deps.ts";

const MAX_DESCRIPTION_LENGTH = 256;

const apiKeyNew = new Command()
  .description("Create a new API key.")
  .option(
    "-d, --description [description:string]",
    `A description to be stored in the key's metadata. (max length: ${MAX_DESCRIPTION_LENGTH})`,
    {
      required: false,
      default: ""
    },
  )
  .option(
    "-t, --ttl [ttl:integer]",
    "Key Time To Live value (seconds). (min: 60s, 0 for no expiration)",
    {
      required: false,
      default: 0
    },
  )
  .action(async (options) => {
    const refreshToken = getConfigRefreshToken(options.env);

    if (!refreshToken) {
      console.error("logged out, you need to 'truestamp auth login' first");
      Deno.exit(1);
    }

    if (options.description.length > MAX_DESCRIPTION_LENGTH) {
      throw new ValidationError(
        `key description max length of ${MAX_DESCRIPTION_LENGTH} characters exceeded`,
      );
    }

    if (options.ttl !== 0 && options.ttl < 60) {
      throw new ValidationError(
        `key TTL must be at least 60 seconds (or '0' for no expiration)`,
      );
    }

    const ts = await createTruestampClient(options.env);

    let keyResp
    try {
      keyResp = await ts.createApiKey({ refreshToken: refreshToken, description: options.description, ttl: options.ttl });
    } catch (error) {
      throw new Error(`auth key creation failed : ${error.message}`);
    }

    console.log(JSON.stringify(keyResp));
  })

export const apiKeys = new Command()
  .description(`Manage API keys.

    API keys are not needed to use the CLI or web interface, and are provided as a
    convenience when using non-interactive clients like cURL or in machine-to-machine
    contexts.

    Keys can be provided with a description or a TTL (Time To Live) value. Once the
    TTL has expired the key will be automatically destroyed.`)
  .action(() => {
    apiKeys.showHelp();
  })
  .command("new", apiKeyNew)
