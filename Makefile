SRC := src/cli.ts
DEPS := src/deps.ts
LOCK := lock.json
DENO_DIR := ./deno_dir
BUILD_DIR := ./build
ARGS := --unstable --allow-env=HOME --allow-net=api.truestamp.com,staging-api.truestamp.com,dev-api.truestamp.com,truestamp.auth0.com,truestamp-staging.auth0.com,truestamp-dev.auth0.com --allow-read --allow-write --lock=${LOCK} --cached-only

build: clean prep compile-darwin-x86 compile-darwin-x86-lite compile-darwin-arm compile-darwin-arm-lite compile-windows compile-windows-lite compile-linux compile-linux-lite

compile-darwin-x86:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-x86_64 ${ARGS} ${SRC}

compile-darwin-x86-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-x86_64-lite --lite ${ARGS} ${SRC}

compile-darwin-arm:
	export DENO_DIR=${DENO_DIR} && deno compile --target=aarch64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-arm ${ARGS} ${SRC}

compile-darwin-arm-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=aarch64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-arm-lite --lite ${ARGS} ${SRC}

compile-windows:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-pc-windows-msvc --output=${BUILD_DIR}/truestamp-windows ${ARGS} ${SRC}

compile-windows-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-pc-windows-msvc --output=${BUILD_DIR}/truestamp-windows-lite --lite ${ARGS} ${SRC}

compile-linux:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-unknown-linux-gnu --output=${BUILD_DIR}/truestamp-linux-x86_64 ${ARGS} ${SRC}

compile-linux-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-unknown-linux-gnu --output=${BUILD_DIR}/truestamp-linux-x86_64-lite --lite ${ARGS} ${SRC}

clean:
	rm -rf ${BUILD_DIR}

prep:
	mkdir -p ${BUILD_DIR}

lock:
	export DENO_DIR=${DENO_DIR} && deno cache --lock=${LOCK} --lock-write ${DEPS}

cache:
	export DENO_DIR=${DENO_DIR} && deno cache ${DEPS} && make lock

cache-reload:
	export DENO_DIR=${DENO_DIR} && deno cache --reload ${DEPS} && make lock

test:
	export DENO_DIR=${DENO_DIR} && deno test --lock=${LOCK} --cached-only src

format:
	export DENO_DIR=${DENO_DIR} && deno fmt src

format-watch:
	export DENO_DIR=${DENO_DIR} && deno fmt --unstable --watch src
