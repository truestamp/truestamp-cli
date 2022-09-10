# Copyright © 2020-2022 Truestamp Inc. All rights reserved.

SRC := src/cli.ts
DEPS := src/deps.ts
LOCK := lock.json
DENO_DIR := ./deno_dir
BUILD_DIR := ./build
ARGS := --unstable --allow-env --allow-net --allow-read --allow-write --lock=${LOCK} --cached-only

build: clean prep build-darwin-x86 build-darwin-aarch64 build-windows build-linux compress

build-local:
	rm -f ./truestamp && export DENO_DIR=${DENO_DIR} && deno compile --output=./truestamp ${ARGS} ${SRC}

build-darwin-x86:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-x86_64 ${ARGS} ${SRC}

build-darwin-aarch64:
	export DENO_DIR=${DENO_DIR} && deno compile --target=aarch64-apple-darwin --output=${BUILD_DIR}/truestamp-darwin-aarch64 ${ARGS} ${SRC}

build-windows:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-pc-windows-msvc --output=${BUILD_DIR}/truestamp-windows ${ARGS} ${SRC}

build-linux:
	export DENO_DIR=${DENO_DIR} && deno compile --target=x86_64-unknown-linux-gnu --output=${BUILD_DIR}/truestamp-linux-x86_64 ${ARGS} ${SRC}

clean:
	rm -rf ${BUILD_DIR}

prep:
	mkdir -p ${BUILD_DIR}

compress-darwin:
	cd ${BUILD_DIR} && for i in truestamp-darwin*; do mv $$i truestamp && tar -czf $$i.tar.gz truestamp && rm truestamp; done && cd -

compress-linux:
	cd ${BUILD_DIR} && for i in truestamp-linux*; do mv $$i truestamp && tar -czf $$i.tar.gz truestamp && rm truestamp; done && cd -

compress-windows:
	cd ${BUILD_DIR} && for i in truestamp-windows*; do mv $$i truestamp.exe && zip -r truestamp-windows-x86_64.zip truestamp.exe && rm truestamp.exe; done && cd -

compress: compress-darwin compress-linux compress-windows

lock:
	export DENO_DIR=${DENO_DIR} && deno cache --unstable --lock=${LOCK} --lock-write ${DEPS}

cache:
	make cache-system && export DENO_DIR=${DENO_DIR} && deno cache --unstable ${DEPS} && make lock

cache-reload:
	export DENO_DIR=${DENO_DIR} && deno cache --unstable --reload ${DEPS} && make lock

cache-system:
	deno cache --unstable ${DEPS}

test:
	export DENO_DIR=${DENO_DIR} && deno test ${ARGS} src

format:
	export DENO_DIR=${DENO_DIR} && deno fmt src

format-watch:
	export DENO_DIR=${DENO_DIR} && deno fmt --unstable --watch src

# https://github.com/hayd/deno-udd
outdated-update:
	 udd src/deps.ts

# https://github.com/hayd/deno-udd
outdated-check:
	 udd --dry-run src/deps.ts

info:
	deno info src/cli.ts
