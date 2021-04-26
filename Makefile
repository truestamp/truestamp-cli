compile:
	DENO_DIR=./deno_dir deno compile --unstable --allow-env=HOME --allow-net=api.truestamp.com,staging-api.truestamp.com,dev-api.truestamp.com,truestamp.auth0.com,truestamp-staging.auth0.com,truestamp-dev.auth0.com --allow-read --allow-write --lock=lock.json --cached-only --output=truestamp src/cli.ts
compile-lite:
	DENO_DIR=./deno_dir deno compile --unstable --lite --allow-env=HOME --allow-net=api.truestamp.com,staging-api.truestamp.com,dev-api.truestamp.com,truestamp.auth0.com,truestamp-staging.auth0.com,truestamp-dev.auth0.com --allow-read --allow-write --lock=lock.json --cached-only --output=truestamp src/cli.ts
lock:
	DENO_DIR=./deno_dir deno cache --lock=lock.json --lock-write src/deps.ts
cache:
	DENO_DIR=./deno_dir deno cache src/deps.ts && make lock
cache-reload:
	DENO_DIR=./deno_dir deno cache --reload src/deps.ts && make lock
test:
	DENO_DIR=./deno_dir deno test --lock=lock.json --cached-only src
format:
	DENO_DIR=./deno_dir deno fmt src
format-watch:
	DENO_DIR=./deno_dir deno fmt --unstable --watch src
