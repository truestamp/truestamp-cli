// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

import { getEnv, logSelectedOutputFormat } from "../utils.ts";

const commitmentsRead = new Command()
  .description("Read an existing Commitment for an Item.")
  .option("-i, --id [id:string]", "An Item Id to retrieve the Commitment for.", {
    required: true,
  })
  .example(
    "Read a Commitment",
    `Using a previously generated test Item ID:

$ truestamp commitments read --id T11_01G43NSA31APN5B04AYWMMNH6Q_1653685923413000_C8A56C8E32F434382ACFC1398275D505

`,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(getEnv(options), options.apiKey);

    try {
      const commitment = await ts.getCommitment(options.id);

      logSelectedOutputFormat(options, { text: JSON.stringify(commitment, null, 2), json: commitment });
    } catch (error) {
      // throw new Error(`Commitment not found : ${error.message}`);
      const { key, value, type, response } = error

      if (key || value || type) {
        // is a StructError
        if (value === undefined) {
          throw new Error(`attribute ${key} is required`)
        } else if (type === 'never') {
          throw new Error(`attribute ${key} is unknown`)
        } else {
          throw new Error(`${key} ${value} is invalid`)
        }
      } else if (response) {
        // is a HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json()
        throw new Error(`${status} : ${code} : ${description}`)
      } else {
        // is a generic Error
        throw error
      }
    }

  });

const commitmentsVerify = new Command()
  .description("Verify an existing Commitment for an Item.")
  .option("-i, --id [id:string]", "An Item Id to retrieve the Commitment for.", {
    required: true,
  })
  .example(
    "Verify a Commitment",
    `Using a previously generated test Item ID:

$ truestamp commitments verify --id T11_01G43NSA31APN5B04AYWMMNH6Q_1653685923413000_C8A56C8E32F434382ACFC1398275D505

`,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(getEnv(options), options.apiKey);

    try {
      const verification = await ts.getCommitmentVerification(options.id);

      logSelectedOutputFormat(options, { text: verification.ok ? "Verification: OK" : `Verification: FAILED (run command with --output=json for details)`, json: verification });
    } catch (error) {
      // throw new Error(`Commitment verification not found : ${error.message}`);
      const { key, value, type, response } = error

      if (key || value || type) {
        // is a StructError
        if (value === undefined) {
          throw new Error(`attribute ${key} is required`)
        } else if (type === 'never') {
          throw new Error(`attribute ${key} is unknown`)
        } else {
          throw new Error(`${key} ${value} is invalid`)
        }
      } else if (response) {
        // is a HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json()
        throw new Error(`${status} : ${code} : ${description}`)
      } else {
        // is a generic Error
        throw error
      }
    }
  });

export const commitments = new Command()
  .description("Read or verify Commitments for Items.")
  .action(() => {
    commitments.showHelp();
  })
  .command("read", commitmentsRead)
  .command("verify", commitmentsVerify);
