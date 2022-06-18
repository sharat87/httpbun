#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset
if [[ -n ${XTRACE-} ]]; then
	set -o xtrace
fi

echo "Starting curl based test suite, at $(date), with curl version:"
curl --version
echo

export HTTPBUN_BIND=localhost:30001

RED='\033[0;31m'
NC='\033[0m'

pcurl() {
	# We remove the Date header, since it's not static, and we don't care about it's value anyway.
	# We also remove `\r` chars at end of lines, to convert Windows style line-endings (used by headers) to Unix style.
	curl --silent --show-error \
		--location \
		--include \
		--user-agent curl \
		"$HTTPBUN_BIND$1" \
		| awk '!b && $1 == "Date:" {next} NF == 0 {b=1} {sub("\r$", ""); print}'
}

assert-eq() {
	local url="$1"
	local expected="$2"
	local got
	got="$(pcurl "$url")"
	if [[ $got != "$expected" ]]; then
		echo "${RED}Fail for $url (<Got >Expected)$NC"
		diff <(echo "$got") <(echo "$expected")
		return 1
	fi
}

go run ../main.go &>test-server.log &
server_pid="$!"
echo "Starting server at PID $server_pid"
sleep 5  # Wait for the server to be ready.

for test_file in ./test-*.sh; do
	echo "Running $test_file"
	# shellcheck source=/dev/null
	source "$test_file"
done

echo "Done"
