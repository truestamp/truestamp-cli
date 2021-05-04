SRC := src/cli.ts
DEPS := src/deps.ts
LOCK := lock.json
DENO_DIR := ./deno_dir
BUILD_DIR := ./build
ARGS := --unstable --allow-env=HOME --allow-net=api.truestamp.com,staging-api.truestamp.com,dev-api.truestamp.com,truestamp.auth0.com,truestamp-staging.auth0.com,truestamp-dev.auth0.com --allow-read --allow-write --lock=${LOCK} --cached-only

build: clean prep build-darwin-x86 build-darwin-x86-lite build-darwin-arm build-darwin-arm-lite build-windows build-windows-lite build-linux build-linux-lite compress

build-darwin-x86:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-x86_64 ${ARGS} ${SRC}

build-darwin-x86-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-x86_64-lite --lite ${ARGS} ${SRC}

build-darwin-arm:
	export DENO_DIR=${DENO_DIR} && deno compile --target=aarch64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-arm ${ARGS} ${SRC}

build-darwin-arm-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=aarch64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-arm-lite --lite ${ARGS} ${SRC}

build-windows:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-pc-windows-msvc --output=${BUILD_DIR}/truestamp-windows ${ARGS} ${SRC}

build-windows-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-pc-windows-msvc --output=${BUILD_DIR}/truestamp-windows-lite --lite ${ARGS} ${SRC}

build-linux:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-unknown-linux-gnu --output=${BUILD_DIR}/truestamp-linux-x86_64 ${ARGS} ${SRC}

build-linux-lite:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-unknown-linux-gnu --output=${BUILD_DIR}/truestamp-linux-x86_64-lite --lite ${ARGS} ${SRC}

clean:
	rm -rf ${BUILD_DIR}

prep:
	mkdir -p ${BUILD_DIR}

compress-darwin:
	for i in build/truestamp-darwin*; do tar -czf $$i.tar.gz $$i && rm $$i; done

compress-linux:
	for i in build/truestamp-linux*; do tar -czf $$i.tar.gz $$i && rm $$i; done

compress-windows:
	for i in build/truestamp-windows*; do zip -r $$i.zip $$i && rm $$i; done

compress: compress-darwin compress-linux compress-windows

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
