import {
  Command,
  CompletionsCommand,
  getAccessTokenWithPrompts,
  HelpCommand,
  ITypeInfo,
  Truestamp,
} from "./deps.ts";

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
  .action(async () => {
    try {
      const accessToken = await getAccessTokenWithPrompts();
      const ts = new Truestamp({ apiKey: accessToken });
      const hb = await ts.getHeartbeat();
      console.log(hb);
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
  .action(() => {
    authLogout.showHelp();
    Deno.exit(0);
  });

const authRefresh = new Command()
  .description("Refresh stored authentication credentials")
  .action(() => {
    authRefresh.showHelp();
    Deno.exit(0);
  });

const authStatus = new Command()
  .description("View authentication status")
  .action(() => {
    authStatus.showHelp();
    Deno.exit(0);
  });

const auth = new Command()
  .description("Login, logout, and refresh your authentication.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(() => {
    auth.showHelp();
    Deno.exit(0);
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("refresh", authRefresh)
  .command("status", authStatus);

const documents = new Command()
  .description("Create, read, update, or destroy documents.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .option("-s, --silent [silent:boolean]", "Disable output.")
  .action(() => {
    documents.showHelp();
    Deno.exit(0);
  });

const heartbeat = new Command()
  .description("Display results of API server heartbeat call.")
  .type("envType", environmentType, { global: true })
  .option("-E, --env [env:envType]", "API environment to use.", {
    hidden: false,
    default: "production",
  })
  .action(() => {
    heartbeat.showHelp();
    Deno.exit(0);
  });

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
    cmd.showHelp();
    Deno.exit(0);
  })
  .command("auth", auth)
  .command("completions", new CompletionsCommand())
  .command("documents", documents)
  .command("heartbeat", heartbeat)
  .command("help", new HelpCommand().global());

try {
  cmd.parse(Deno.args);
} catch (error) {
  console.error("Error: ", error.message);
  Deno.exit(1);
}
