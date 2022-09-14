// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient, Table } from "../deps.ts";

import { logSelectedOutputFormat } from "../utils.ts";

import { environmentType, outputType } from "../cli.ts";

import { verify } from "@truestamp/verify";

const commitmentsRead = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description("Read an existing Commitment for an Item.")
  .option(
    "-i, --id <id:string>",
    "An Item Id to retrieve the Commitment for.",
    {
      required: true,
    },
  )
  .example(
    "Read a Commitment",
    `Using a previously generated test Item ID:

$ truestamp commitments read --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

`,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    try {
      const commitment = await ts.getCommitment(options.id);

      logSelectedOutputFormat(
        {
          text: JSON.stringify(commitment, null, 2),
          json: commitment,
        },
        options.output,
      );
    } catch (error) {
      const { response } = error;

      if (response) {
        // HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json();
        throw new Error(`${status} : ${code} : ${description}`);
      } else {
        // is a generic Error
        throw error;
      }
    }
  });

const commitmentsVerify = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description("Verify a Commitment for an Item.")
  .option(
    "-i, --id <id:string>",
    "An Item Id to retrieve the Commitment for.",
    {
      required: true,
    },
  )
  .option(
    "-l, --local [local:boolean]",
    "Locally verify cryptographic operations, and on-chain verifications.",
    { default: false },
  )
  .example(
    "Verify a Commitment",
    `Using a previously generated test Item ID:

All cryptographic operations, and on-chain verifications, performed via API server by default:

  $ truestamp commitments verify --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

`,
  )
  .example(
    "Verify a Commitment Locally",
    `Using a previously generated test Item ID:

All cryptographic operations, and on-chain verifications, are performed from this local client.

Verified public keys will be retrieved from https://keys.truestamp.com.

HTTP request to third-party blockchain API servers will originate from this local client.

  $ truestamp commitments verify --local --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

`,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    let verification;
    try {
      if (options.local) {
        const commitment = await ts.getCommitment(options.id);
        verification = await verify(commitment);
      } else {
        verification = await ts.getCommitmentVerification(options.id);
      }

      if (verification.verified) {
        logSelectedOutputFormat(
          {
            text: "Verification Results \n",
            json: verification,
          },
          options.output,
        );

        if (options.output === "text") {
          const table: Table = Table.from([]);
          table.push(["Verified?", "Yes"]);

          let verifyUrlBase;
          switch (options.env) {
            case "development":
              verifyUrlBase = "http://localhost:3000";
              break;
            case "staging":
              verifyUrlBase = "https://staging-verify.truestamp.com";
              break;
            case "production":
              verifyUrlBase = "https://verify.truestamp.com";
              break;
            default:
              throw new Error(
                `Unknown environment: ${options.env}`,
              );
          }

          table.push([
            "Verify URL",
            `${verifyUrlBase}/${options.id.replace("truestamp-", "")}`,
          ]);

          table.push(["ID", options.id]);

          if (verification.commitsTo?.timestamps.submittedAfter) {
            table.push([
              "Submitted After",
              verification.commitsTo.timestamps.submittedAfter,
            ]);
          }

          if (verification.commitsTo?.timestamps.submittedAt) {
            table.push([
              "Submitted At",
              verification.commitsTo.timestamps.submittedAt,
            ]);
          }

          if (verification.commitsTo?.timestamps.submittedBefore) {
            table.push([
              "Submitted Before",
              verification.commitsTo.timestamps.submittedBefore
                .join("\n"),
            ]);
          }

          if (
            verification.commitsTo?.timestamps
              .submitWindowMilliseconds
          ) {
            table.push([
              "Submitted Window (ms)",
              verification.commitsTo.timestamps
                .submitWindowMilliseconds,
            ]);
          }

          if (verification.commitsTo?.observableEntropy) {
            table.push([
              "Observable Entropy Hash",
              verification.commitsTo.observableEntropy,
            ]);
          }

          if (verification.commitsTo?.hashes) {
            table.push([
              "Hashes",
              verification.commitsTo.hashes.map((
                t: Record<string, string>,
              ) => `${t.hash} [${t.hashType}]`).join("\n"),
            ]);
          }

          table.indent(2);
          table.render();
        }
      } else {
        logSelectedOutputFormat(
          {
            text: `Verification : FAILED : ${verification.error}`,
            json: verification,
          },
          options.output,
        );
      }
    } catch (error) {
      const { response } = error;

      if (response) {
        // HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json();
        throw new Error(`${status} : ${code} : ${description}`);
      } else {
        // is a generic Error
        throw error;
      }
    }
  });

export const commitments = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description("Read or verify Commitments for Items.")
  .action(() => {
    commitments.showHelp();
  })
  .command("read", commitmentsRead)
  .command("verify", commitmentsVerify);
