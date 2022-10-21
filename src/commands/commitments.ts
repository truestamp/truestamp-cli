// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient, Table } from "../deps.ts"

import {
  logSelectedOutputFormat,
  RFC7807ErrorSchema,
  throwApiError,
} from "../utils.ts"

import { environmentType, outputType } from "../cli.ts"

import { Commitment, CommitmentVerification, verify } from "@truestamp/verify"

const commitmentsRead = new Command<{
  env: typeof environmentType
  apiKey?: string
  output: typeof outputType
}>()
  .description("Read an existing Commitment for an Item.")
  .option(
    "-i, --id <id:string>",
    "An Item Id to retrieve the Commitment for.",
    {
      required: true,
    }
  )
  .example(
    "Read a Commitment",
    `Using a previously generated test Item ID:

$ truestamp commitments read --id ts_11SHyexF6pqKpTgvnxu5UvHveboF763B41JsZCYcjveSNgqXnL2k7K4LrUuy

`
  )
  .action(async (options) => {
    const truestamp = await createTruestampClient(options.env, options.apiKey)
    const commitmentResp = await truestamp.getCommitment(options.id)

    if (!commitmentResp.success) {
      const parseResult = RFC7807ErrorSchema.safeParse(commitmentResp.data)
      throwApiError(
        "get commitment error",
        parseResult.success ? parseResult.data : undefined
      )
    }

    const { data } = commitmentResp

    logSelectedOutputFormat(
      {
        text: JSON.stringify(data, null, 2),
        json: data,
      },
      options.output
    )
  })

const commitmentsVerify = new Command<{
  env: typeof environmentType
  apiKey?: string
  output: typeof outputType
}>()
  .description("Verify a Commitment for an Item.")
  .option(
    "-i, --id <id:string>",
    "An Item Id to retrieve the Commitment for.",
    {
      required: true,
    }
  )
  .option(
    "-l, --local [local:boolean]",
    "Locally verify cryptographic operations, and on-chain verifications.",
    { default: false }
  )
  .example(
    "Verify a Commitment",
    `Using a previously generated test Item ID:

All cryptographic operations, and on-chain verifications, performed via API server by default:

  $ truestamp commitments verify --id ts_11SHyexF6pqKpTgvnxu5UvHveboF763B41JsZCYcjveSNgqXnL2k7K4LrUuy

`
  )
  .example(
    "Verify a Commitment Locally",
    `Using a previously generated test Item ID:

All cryptographic operations, and on-chain verifications, are performed from this local client.

Verified public keys will be retrieved from https://keys.truestamp.com.

HTTP request to third-party blockchain API servers will originate from this local client.

  $ truestamp commitments verify --local --id ts_11SHyexF6pqKpTgvnxu5UvHveboF763B41JsZCYcjveSNgqXnL2k7K4LrUuy

`
  )
  .action(async (options) => {
    const truestamp = await createTruestampClient(options.env, options.apiKey)

    let verification: CommitmentVerification
    if (options.local) {
      const commitmentResp = await truestamp.getCommitment(options.id)
      if (!commitmentResp.success) {
        const parseResult = RFC7807ErrorSchema.safeParse(commitmentResp.data)
        throwApiError(
          "get commitment verification error",
          parseResult.success ? parseResult.data : undefined
        )
      }

      const { data } = commitmentResp

      verification = await verify(data as Commitment)
    } else {
      const verificationResp = await truestamp.getCommitmentVerification(
        options.id
      )
      if (!verificationResp.success) {
        const parseResult = RFC7807ErrorSchema.safeParse(verificationResp.data)
        throwApiError(
          "get commitment verification error",
          parseResult.success ? parseResult.data : undefined
        )
      }

      const { data } = verificationResp
      verification = data as CommitmentVerification
    }

    if (verification.verified) {
      logSelectedOutputFormat(
        {
          text: "Verification Results \n",
          json: verification,
        },
        options.output
      )

      if (options.output === "text") {
        const table: Table = Table.from([])
        table.push(["Verified?", "Yes"])

        const verifyUrlBase =
          options.env === "development"
            ? "http://localhost:3000"
            : options.env === "staging"
            ? "https://staging-verify.truestamp.com"
            : options.env === "production"
            ? "https://verify.truestamp.com"
            : (() => {
                throw new Error(`Unknown environment: ${options.env}`)
              })()

        table.push([
          "Verify URL",
          `${verifyUrlBase}/${options.id.replace("truestamp-", "")}`,
        ])

        table.push(["ID", options.id])

        if (verification.commitsTo?.timestamps.submittedAfter) {
          table.push([
            "Submitted After",
            verification.commitsTo.timestamps.submittedAfter,
          ])
        }

        if (verification.commitsTo?.timestamps.submittedAt) {
          table.push([
            "Submitted At",
            verification.commitsTo.timestamps.submittedAt,
          ])
        }

        if (verification.commitsTo?.timestamps.submittedBefore) {
          table.push([
            "Submitted Before",
            verification.commitsTo.timestamps.submittedBefore.join("\n"),
          ])
        }

        if (verification.commitsTo?.timestamps.submitWindowMilliseconds) {
          table.push([
            "Submitted Window (ms)",
            verification.commitsTo.timestamps.submitWindowMilliseconds,
          ])
        }

        if (verification.commitsTo?.observableEntropy) {
          table.push([
            "Observable Entropy Hash",
            verification.commitsTo.observableEntropy,
          ])
        }

        if (verification.commitsTo?.hashes) {
          table.push([
            "Hashes",
            verification.commitsTo.hashes
              .map((t: Record<string, string>) => `${t.hash} [${t.hashType}]`)
              .join("\n"),
          ])
        }

        table.indent(2)
        table.render()
      }
    } else {
      logSelectedOutputFormat(
        {
          text: `verification error : ${verification.error}`,
          json: verification,
        },
        options.output
      )
    }
  })

export const commitments = new Command<{
  env: typeof environmentType
  apiKey?: string
  output: typeof outputType
}>()
  .description("Read or verify Commitments for Items.")
  .action(() => {
    commitments.showHelp()
  })
  .command("read", commitmentsRead)
  .command("verify", commitmentsVerify)
