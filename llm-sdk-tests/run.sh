#!/usr/bin/env bash
# Runs the SDK tests against a freshly built httpbun, on a free port. Run from the repo root, or with `task llm-sdk-tests`.
set -o errexit -o nounset -o pipefail

mkdir -p bin
go build -o bin/httpbun-sdk-tests .

port="$(python3 -c 'import socket; s = socket.socket(); s.bind(("localhost", 0)); print(s.getsockname()[1])')"
bin/httpbun-sdk-tests --bind "localhost:$port" > /dev/null 2>&1 &
server_pid=$!
trap 'kill "$server_pid"' EXIT

for _ in $(seq 50); do
	curl -fsS "http://localhost:$port/get" > /dev/null 2>&1 && break
	sleep 0.1
done

cd llm-sdk-tests
BASE_URL="http://localhost:$port/llm" uv run pytest -q tests
