import {
  Command,
  CompletionsCommand,
  deleteSavedTokens,
  getSavedAccessToken,
  getSavedRefreshToken,
  getSavedIdTokenPayload,
  HelpCommand,
  ITypeInfo,
  createTruestampClient,
} from "./deps.ts"

function environmentType({ label, name, value }: ITypeInfo): string {
  const envs = ["development", "staging", "production"]
  if (!envs.includes(value.toLowerCase())) {
    throw new Error(
      `${label} "${name}" must be a valid environment [${envs}], but got "${value}".`
    )
  }

  return value.toLowerCase()
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
      const ts = await createTruestampClient(options.env)
      const hb = await ts.getHeartbeat()
      if (hb) {
        console.log("login successful")
      } else {
        throw new Error("auth login heartbeat check failed")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

const authLogout = new Command()
  .description("Log out of a Truestamp host")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action((options) => {
    deleteSavedTokens(options.env)
    console.log("logout complete")
    Deno.exit(0)
  })

const authStatus = new Command()
  .description("View authentication status")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    if (
      !getSavedAccessToken(options.env) ||
      !getSavedRefreshToken(options.env)
    ) {
      console.error("logged out")
      Deno.exit(1)
    }

    try {
      const ts = await createTruestampClient(options.env)
      const hb = await ts.getHeartbeat()
      if (!hb) {
        throw new Error("auth status heartbeat check failed")
      }

      const payload = getSavedIdTokenPayload(options.env)
      if (payload) {
        console.log(
          `logged into '${options.env}' environment as user '${payload.name} (${payload.email})'`
        )
      } else {
        throw new Error("id token missing or invalid")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

const auth = new Command()
  .description("Login, logout, and show status of your authentication.")
  .action(() => {
    auth.showHelp()
    Deno.exit(0)
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus)

const documentsNew = new Command()
  .description(
    `Create a new document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
  `
  )
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option(
    "-h, --hash [type:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    }
  )
  .option(
    "-t, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    }
  )
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env)
      const d = await ts.createDocument({
        hash: options.hash,
        type: options.type,
      })
      if (d) {
        console.log(JSON.stringify(d))
      } else {
        throw new Error("new document creation failed")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

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
      const ts = await createTruestampClient(options.env)
      const d = await ts.getDocument(options.id)
      if (d) {
        console.log(JSON.stringify(d))
      } else {
        throw new Error("document not found")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

const documentsUpdate = new Command()
  .description(
    `Update an existing document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
  `
  )
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .allowEmpty(false)
  .option("-i, --id [type:string]", "A document ID.", {
    required: true,
  })
  .option(
    "-h, --hash [type:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    }
  )
  .option(
    "-t, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    }
  )
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env)
      const d = await ts.updateDocument(options.id, {
        hash: options.hash,
        type: options.type,
      })
      if (d) {
        console.log(JSON.stringify(d))
      } else {
        throw new Error("document not found")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

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
      const ts = await createTruestampClient(options.env)
      const d = await ts.deleteDocument(options.id)
      if (d) {
        console.log(JSON.stringify(d))
      } else {
        throw new Error("document not found")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

const documentsList = new Command()
  .description("List all existing documents.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    try {
      const ts = await createTruestampClient(options.env)
      const d = await ts.getAllDocuments()
      if (d) {
        console.log(JSON.stringify(d))
      } else {
        throw new Error("documents not found")
      }
    } catch (error) {
      console.error("Error: ", error.message)
      Deno.exit(1)
    }

    Deno.exit(0)
  })

const documents = new Command()
  .description("Create, read, update, or destroy documents.")
  .action(() => {
    documents.showHelp()
    Deno.exit(0)
  })
  .command("new", documentsNew)
  .command("show", documentsShow)
  .command("update", documentsUpdate)
  .command("delete", documentsDelete)
  .command("list", documentsList)

const heartbeat = new Command()
  .description("Display results of API server heartbeat call.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env)
    const hb = await ts.getHeartbeat()
    console.log(JSON.stringify(hb))
    Deno.exit(0)
  })

// Top level command
const cmd = new Command()
  .name("truestamp")
  .version("0.0.0")
  .description("Truestamp CLI")
  .help({
    types: false,
    hints: true,
  })
  .action(() => {
    cmd.showHelp()
    Deno.exit(0)
  })
  .command("auth", auth)
  .command("completions", new CompletionsCommand())
  .command("documents", documents)
  .command("heartbeat", heartbeat)
  .command("help", new HelpCommand().global())

try {
  cmd.parse(Deno.args)
} catch (error) {
  console.error("Error: ", error.message)
  Deno.exit(1)
}
