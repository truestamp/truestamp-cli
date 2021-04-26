import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts"

function getApiBaseUriForEnv(env: string): string {
  switch (env) {
    case "development":
      return "https://dev-api.truestamp.com/v1/"

    case "staging":
      return "https://staging-api.truestamp.com/v1/"

    case "production":
      return "https://api.truestamp.com/v1/"

    default:
      throw new Error(`invalid environment : '${env}'`)
  }
}

export async function createTruestampClient(env: string) {
  // FIXME : https://stackoverflow.com/questions/43623461/new-expression-whose-target-lacks-a-construct-signature-in-typescript
  const client = new (Truestamp as any)({
    apiKey: await getAccessTokenWithPrompts(env),
    apiBaseUrl: getApiBaseUriForEnv(env),
  })

  return client
}
