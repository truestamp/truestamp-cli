// © 2020-2021 Truestamp Inc. All rights reserved.

import {
  Buffer,
  Command,
  CompletionsCommand,
  createHash,
  createTruestampClient,
  deleteTokensInConfig,
  getConfigAccessToken,
  getConfigIdTokenPayload,
  getConfigRefreshToken,
  HelpCommand,
  ITypeInfo,
  path,
  S3,
} from "./deps.ts";

import {
  getConfigForEnv,
  getConfigKeyForEnv,
  setConfigKeyForEnv,
} from "./config.ts";

function environmentType({ label, name, value }: ITypeInfo): string {
  const envs = ["development", "staging", "production"];
  if (!envs.includes(value.toLowerCase())) {
    throw new Error(
      `${label} "${name}" must be a valid environment [${envs}], but got "${value}".`,
    );
  }

  return value.toLowerCase();
}

const authLogin = new Command()
  .description("Authenticate with a Truestamp host")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const hb = await ts.getHeartbeat();
      if (hb) {
        console.log("login successful");
      } else {
        throw new Error("auth login heartbeat check failed");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const authLogout = new Command()
  .description("Log out of a Truestamp host")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action((options) => {
    deleteTokensInConfig(options.env);
    console.log("logout complete");
    Deno.exit(0);
  });

const authStatus = new Command()
  .description("View authentication status")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    if (
      !getConfigAccessToken(options.env) ||
      !getConfigRefreshToken(options.env)
    ) {
      console.error("logged out");
      Deno.exit(1);
    }

    try {
      const ts = await createTruestampClient(options.env);
      const hb = await ts.getHeartbeat();
      if (!hb) {
        throw new Error("auth status heartbeat check failed");
      }

      const payload = getConfigIdTokenPayload(options.env);
      if (payload) {
        console.log(
          `logged into '${options.env}' environment as user '${payload.name} (${payload.email})'`,
        );
      } else {
        throw new Error("id token missing or invalid");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const auth = new Command()
  .description("Login, logout, and show status of your authentication.")
  .action(() => {
    auth.showHelp();
    Deno.exit(0);
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus);

const documentsNew = new Command()
  .description(
    `Create a new document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
  `,
  )
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option(
    "-H, --hash [hash:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    },
  )
  .option(
    "-T, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const d = await ts.createDocument({
        hash: options.hash,
        type: options.type,
      });
      if (d) {
        console.log(JSON.stringify(d));
      } else {
        throw new Error("new document creation failed");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const documentsShow = new Command()
  .description("Show an existing document.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option("-i, --id [type:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const d = await ts.getDocument(options.id);
      if (d) {
        console.log(JSON.stringify(d));
      } else {
        throw new Error("document not found");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const documentsUpdate = new Command()
  .description(
    `Update an existing document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
  `,
  )
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option("-i, --id [id:string]", "A document ID.", {
    required: true,
  })
  .option(
    "-H, --hash [hash:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    },
  )
  .option(
    "-T, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const d = await ts.updateDocument(options.id, {
        hash: options.hash,
        type: options.type,
      });
      if (d) {
        console.log(JSON.stringify(d));
      } else {
        throw new Error("document not found");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const documentsDelete = new Command()
  .description("Delete an existing document.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option("-i, --id [type:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const d = await ts.deleteDocument(options.id);
      if (d) {
        console.log(JSON.stringify(d));
      } else {
        throw new Error("document not found");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const documentsList = new Command()
  .description("List all existing documents.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env);
      const d = await ts.getAllDocuments();
      if (d) {
        console.log(JSON.stringify(d));
      } else {
        throw new Error("documents not found");
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const documents = new Command()
  .description("Create, read, update, or destroy documents.")
  .action(() => {
    documents.showHelp();
    Deno.exit(0);
  })
  .command("new", documentsNew)
  .command("show", documentsShow)
  .command("update", documentsUpdate)
  .command("delete", documentsDelete)
  .command("list", documentsList);

const s3ConfigSet = new Command()
  .description(`Set environment specific persistent config for AWS S3.`)
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option(
    "-r, --region [region:string]",
    "Set the AWS S3 region the bucket exists in.",
    {
      required: true,
      default: "us-east-1",
    },
  )
  .option(
    "-b, --bucket [bucket:string]",
    "Set the name of the AWS S3 bucket (must exist in region specified in '--region').",
    {
      required: true,
    },
  )
  .action((options) => {
    try {
      if (options.region !== getConfigKeyForEnv(options.env, "aws_s3_region")) {
        setConfigKeyForEnv(options.env, "aws_s3_region", options.region);
      }

      if (options.bucket !== getConfigKeyForEnv(options.env, "aws_s3_bucket")) {
        setConfigKeyForEnv(options.env, "aws_s3_bucket", options.bucket);
      }
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const s3Config = new Command()
  .description(`View environment specific config for AWS S3.`)
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .action((options) => {
    const config = getConfigForEnv(options.env);
    console.log(JSON.stringify(config.store));
    Deno.exit(0);
  })
  .command("set", s3ConfigSet);

const s3Upload = new Command()
  .description(
    `Upload a file to an AWS S3 bucket monitored by Truestamp.

  By default only the base filename of a path will be used to determine the
  'key' the object will be stored at. This base filename can be overriden with
  the '--key' argument. An optional '--prefix' can also be provided to store key
  in a 'folder'.

  Uploading a file to a pre-existing key in a bucket will result
  in the silent creation of a new version of the file in S3 in
  versioned buckets.

  The use of this command assumes an existing install of
  an appropriate Truestamp AWS S3 monitoring function. Without
  it, uploaded files will not be observed.
  `,
  )
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option(
    "-p, --path [path:string]",
    "The relative, or absolute, filesystem path of a local file to upload. The file basename will be used as the S3 object key. e.g. '/my/path/doc.txt' has a basename of 'doc.txt'.",
    {
      required: true,
    },
  )
  .option(
    "-P, --prefix [prefix:string]",
    "The prefix that preceeds the file '--path' basename or '--key'. The prefix can emulate a folder structure with slashes '/' as the delimiter. If no prefix is provided the object will be stored in the root of the bucket.",
    {
      required: false,
      default: "",
    },
  )
  .option(
    "-k, --key [key:string]",
    "The 'key' to store the object in the AWS S3 bucket. Overrides the file basename from the '--path' option.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    try {
      if (!getConfigKeyForEnv(options.env, "aws_s3_bucket")) {
        throw new Error("missing aws s3 bucket config");
      }

      const awsS3Bucket = getConfigKeyForEnv(
        options.env,
        "aws_s3_bucket",
      ) as string;

      if (!getConfigKeyForEnv(options.env, "aws_s3_region")) {
        throw new Error("missing aws s3 region config");
      }

      const awsS3Region = getConfigKeyForEnv(
        options.env,
        "aws_s3_region",
      ) as string;

      // ex: /path/to/my-picture.png becomes my-picture.png
      const fileBaseName = path.basename(options.path);

      const fileContents = await Deno.readFile(options.path);

      // If you want to save to "my-bucket/{prefix}/{filename}"
      //                    ex: "my-bucket/my-pictures-folder/my-picture.png"
      const resolvedFileName = options.key ? options.key : fileBaseName;

      const keyName = [options.prefix, resolvedFileName].join("");

      // https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html
      const keyRegex = /^[a-zA-Z0-9\/\!\-\_\.\*\'\(\)]+$/;
      if (!keyRegex.test(keyName)) {
        throw new Error(
          `object prefix + key must be valid, but got "${keyName}". Allowed RegEx ${keyRegex}`,
        );
      }

      // MD5 hash to be submitted with the content to
      // ensure integrity of file stored. This is
      // required for buckets using a retention period
      // configured using Amazon S3 Object Lock.
      // https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html
      const hashMD5 = createHash("md5");
      hashMD5.update(fileContents);
      const hashMD5InBase64 = hashMD5.toString("base64");

      // SHA256 hash (Base64 encoded) to submit as
      // user-defined metadata and stored with the S3 object
      const hashSHA256 = createHash("sha256");
      hashSHA256.update(fileContents);
      const hashSHA256InBase64 = hashSHA256.toString("base64");

      const metadata = {
        "hash-sha256": hashSHA256InBase64,
        creator: `truestamp-cli-v${s3Upload.getVersion()}`,
      };

      const s3ObjParams = {
        Bucket: awsS3Bucket,
        Body: fileContents,
        ContentMD5: hashMD5InBase64,
        Key: keyName,
        Metadata: metadata,
      };

      const client = new S3({
        region: awsS3Region,
      });

      const resp = await client.putObject(s3ObjParams);
      // console.log(resp)

      // The ETag is being returned wrapped in extra quotes
      // See : https://github.com/aws/aws-sdk-net/issues/815
      const eTagStripped = resp.ETag ? resp.ETag.replace(/(^"|"$)/g, "") : "";

      // compare() method compares two buffer objects and returns a number defining their differences
      // https://teppen.io/2018/06/23/aws_s3_etags/
      if (
        Buffer.compare(
          Buffer.from(hashMD5InBase64, "base64"),
          Buffer.from(eTagStripped, "hex"),
        ) != 0
      ) {
        throw new Error(
          "the md5 end-to-end integrity check on the submitted data and the returned ETag failed",
        );
      }

      const objMeta = {
        region: awsS3Region,
        bucket: awsS3Bucket,
        key: keyName,
        versionId: resp.VersionId,
        eTag: eTagStripped,
        metadata: metadata,
      };
      console.log(JSON.stringify(objMeta));
    } catch (error) {
      console.error("Error: ", error.message);
      Deno.exit(1);
    }

    Deno.exit(0);
  });

const s3 = new Command()
  .description(
    "Manage files stored in an AWS S3 bucket monitored by Truestamp.",
  )
  .hidden()
  .env(
    "AWS_PROFILE=<aws_client_credentials_profile:string>",
    "The AWS client credentials profile to use.",
    {
      global: true,
      hidden: false,
    },
  )
  .env(
    "AWS_ACCESS_KEY_ID=<aws_access_key_id:string>",
    "The AWS Access Key ID to use.",
    {
      global: true,
      hidden: false,
    },
  )
  .env(
    "AWS_SECRET_ACCESS_KEY=<aws_secret_access_key:string>",
    "The AWS Secret Access Key to use.",
    {
      global: true,
      hidden: false,
    },
  )
  .action(() => {
    s3.showHelp();
    Deno.exit(0);
  })
  .command("config", s3Config)
  .command("upload", s3Upload);

const heartbeat = new Command()
  .description("Display results of API server heartbeat call.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const hb = await ts.getHeartbeat();
    console.log(JSON.stringify(hb));
    Deno.exit(0);
  });

// Top level command
const cmd = new Command()
  .name("truestamp")
  .version("0.0.4")
  .description("Truestamp CLI")
  .help({
    types: false,
    hints: true,
  })
  .action(() => {
    cmd.showHelp();
    Deno.exit(0);
  })
  .command("auth", auth)
  .command("completions", new CompletionsCommand())
  .command("documents", documents)
  .command("heartbeat", heartbeat)
  .command("help", new HelpCommand().global())
  .command("s3", s3);

try {
  cmd.parse(Deno.args);
} catch (error) {
  console.error("Error: ", error.message);
  Deno.exit(1);
}
