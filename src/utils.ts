// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { z } from "zod"

const OutputWrapper = z.object({
  text: z.string(),
  json: z.record(z.string().min(1), z.any()),
})

export type OutputWrapper = z.infer<typeof OutputWrapper>

export function logSelectedOutputFormat(
  data: OutputWrapper,
  outputFormat: "text" | "json" | "silent"
): void {
  switch (outputFormat) {
    case "text":
      console.log(data.text)
      break
    case "json":
      console.log(JSON.stringify(data.json, null, 2))
      break
    case "silent":
      // no-op : silent
      break
  }
}

export const RFC7807ErrorSchema = z.object({
  status: z.number(),
  type: z.optional(z.string().url()),
  title: z.optional(z.string()),
  detail: z.optional(z.string()),
  instance: z.optional(z.string().url()),
})

export type RFC7807Error = z.infer<typeof RFC7807ErrorSchema>

// RFC7807 Problem Details for HTTP APIs
// See : https://github.com/PDMLab/http-problem-details
export function throwApiError(
  errorMsgPrefix: string,
  errorData?: RFC7807Error
): void {
  const result = RFC7807ErrorSchema.safeParse(errorData)

  // Unknown/missing error format
  if (!result.success) {
    throw new Error(
      `${errorMsgPrefix} : invalid error response :\n${JSON.stringify(
        errorData,
        null,
        2
      )}`
    )
  }

  // RFC7807 error
  const { title, detail, status, type, instance } = result.data
  throw new Error(
    `${errorMsgPrefix} : ${title} ${detail} [${status}, ${type}, ${instance}]`
  )
}
