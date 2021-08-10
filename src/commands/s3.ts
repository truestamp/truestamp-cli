// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import {
  Buffer,
  Command,
  createHash,
  EnumType,
  getConfigForEnv,
  getConfigKeyForEnv,
  path,
  S3,
  S3ClientConfig,
  setConfigKeyForEnv,
  ValidationError,
} from "../deps.ts";

const awsRegions = [
  "af-south-1",
  "ap-east-1",
  "ap-northeast-1",
  "ap-northeast-2",
  "ap-northeast-3",
  "ap-south-1",
  "ap-southeast-1",
  "ap-southeast-2",
  "ca-central-1",
  "cn-north-1",
  "cn-northwest-1",
  "eu-central-1",
  "eu-north-1",
  "eu-south-1",
  "eu-west-1",
  "eu-west-2",
  "eu-west-3",
  "me-south-1",
  "sa-east-1",
  "us-east-1",
  "us-east-2",
  "us-gov-east-1",
  "us-gov-west-1",
  "us-west-1",
  "us-west-2",
].sort();

const s3ConfigSet = new Command()
  .description(`Set environment specific persistent config for AWS S3.`)
  .option(
    "-b, --bucket [bucket:string]",
    "Set the name of the AWS S3 bucket (must exist in same region client uses. Override with ).",
    {
      required: true,
    },
  )
  .action((options) => {
    if (options.bucket !== getConfigKeyForEnv(options.env, "aws_s3_bucket")) {
      setConfigKeyForEnv(options.env, "aws_s3_bucket", options.bucket);
    }
  });

const s3Config = new Command()
  .description(`View environment specific config for AWS S3.`)
  .action((options) => {
    const config = getConfigForEnv(options.env);
    console.log(JSON.stringify(config.store));
  })
  .command("set", s3ConfigSet);

const s3Upload = new Command()
  .description(
    `Upload a file to an AWS S3 bucket monitored by Truestamp.
  
    A valid AWS S3 region must be provided, and the destination bucket and
    the monitoring function must exist in that same region.
  
    By default only the base filename of a path will be used to determine the
    'key' the object will be stored at. This base filename can be overriden with
    the '--key' argument. An optional '--prefix' can also be provided to store key
    in a 'folder' in the S3 Bucket.
  
    Uploading a file to a pre-existing key in a bucket will result
    in the silent creation of a new version of the file in S3 in
    versioned buckets.
  
    The use of this command assumes an existing install of
    an appropriate Truestamp AWS S3 monitoring function. Without
    it, uploaded files will not be observed.
  
    All objects uploaded have an MD5 hash checksum included to ensure
    the end to end storage integrity of the object per AWS recommendation.
  
    All objects uploaded also have the Bas64 encoded SHA2-256 hash of the
    object attached as permanent object metadata.
    `,
  )
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
  .option(
    "-r, --aws-region [aws_region:aws_region_type]",
    "Set the AWS S3 region the bucket exists in. Overrides `AWS_REGION` or `AMAZON_REGION` environment variables if present.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    if (!getConfigKeyForEnv(options.env, "aws_s3_bucket")) {
      throw new ValidationError("missing aws s3 bucket config");
    }

    const awsS3Bucket = getConfigKeyForEnv(
      options.env,
      "aws_s3_bucket",
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
      throw new ValidationError(
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

    // The AWS client will automatically look for an `AWS_REGION` or `AMAZON_REGION`
    // environment variable. The region specified in the options will override this
    // if provided.
    if (
      !options.awsRegion && !Deno.env.get("AWS_REGION") &&
      !Deno.env.get("AMAZON_REGION")
    ) {
      throw new ValidationError(
        "missing AWS S3 region. Specify `--aws-region` or `AWS_REGION` or `AMAZON_REGION` env var.",
      );
    }

    // The AWS_* region env vars don't need to be specified here since the SDK
    // will look for them automatically.
    const awsS3RegionOverride = options.awsRegion ? options.awsRegion : null;
    const clientConfig: S3ClientConfig = {};
    if (awsS3RegionOverride) {
      clientConfig.region = awsS3RegionOverride;
    }
    const s3 = new S3(clientConfig);

    const resp = await s3.putObject(s3ObjParams);
    console.log(resp);

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
        "the MD5 hash end-to-end integrity check on the submitted data and the returned ETag failed",
      );
    }

    const objMeta = {
      region: options.awsRegion || Deno.env.get("AWS_REGION") ||
        Deno.env.get("AMAZON_REGION"),
      bucket: awsS3Bucket,
      key: keyName,
      versionId: resp.VersionId,
      eTag: eTagStripped,
      metadata: metadata,
    };
    console.log(JSON.stringify(objMeta));
  });

export const s3 = new Command()
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
  .type(
    "aws_region_type",
    new EnumType(awsRegions),
    {
      global: true,
    },
  )
  .env(
    "AWS_REGION=<aws_region:aws_region_type>",
    "The AWS region to configure the client to use.",
    {
      global: true,
      hidden: false,
    },
  )
  .env(
    "AMAZON_REGION=<aws_region:aws_region_type>",
    "The AWS region to configure the client to use. Alternate to `AWS_REGION`.",
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
