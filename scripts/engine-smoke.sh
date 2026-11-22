#!/bin/sh
# Smoke-test the mcmodlist engine protocol over stdin/stdout.
set -eu

cd "$(dirname "$0")/.."

engine_bin="${MCMODLIST_ENGINE:-}"
if [ -z "$engine_bin" ]; then
	engine_bin="$(mktemp -d)/mcmodlist-engine"
	trap 'rm -rf "$(dirname "$engine_bin")"' EXIT
	go build -o "$engine_bin" ./cmd/mcmodlist-engine
fi

printf '%s\n' \
	'{"v":1,"id":1,"method":"hello","params":{"client":"smoke","clientVersion":"0"}}' \
	'{"v":1,"id":2,"method":"listInstances","params":{}}' \
	'{"v":1,"id":3,"method":"inspectPath","params":{"path":"/nonexistent/mcmodlist-smoke"}}' \
	'{"v":1,"id":4,"method":"shutdown","params":{}}' \
	| "$engine_bin"
