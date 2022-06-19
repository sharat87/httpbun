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

cd "$(dirname "$0")"

export HTTPBUN_BIND=localhost:30001

RED='\033[0;31m'
NC='\033[0m'

pcurl() {
	# We remove the Date header, since it's not static, and we don't care about it's value anyway.
	# We also remove `\r` chars at end of lines, to convert Windows style line-endings (used by headers) to Unix style.
	curl --disable \
		--silent \
		--show-error \
		--location \
		--include \
		--user-agent curl \
		"${@:2:$#}" \
		"$HTTPBUN_BIND$1" \
		| awk '!b && $1 == "Date:" {next} NF == 0 {b=1} {sub("\r$", ""); print}'
}

assert-eq() {
	local ac=$(($# - 1))
	local curl_args=( "${@:1:$ac}" )
	local expected="${*: -1}."
	local got
	got="$(pcurl "${curl_args[@]}"; echo .)"
	if [[ "$got" != "$expected" ]]; then
		echo -e "${RED}Fail for '${curl_args[*]}' (-Got +Expected)$NC"
		diff --unified --label "From ""${curl_args[*]}" --label Expected <(echo "$got") <(echo "$expected")
	fi
}

assert-contains() {
	local ac=$(($# - 1))
	local curl_args=( "${@:1:$ac}" )
	local expected="${*: -1}"
	local got
	got="$(pcurl "${curl_args[@]}"; echo .)"
	got="${got%.}"
	if ! str-contains "$got" "$expected"; then
		echo -e "${RED}Fail for '${curl_args[*]}' (-Got +Expected)$NC"
		diff --unified --label "From ""${curl_args[*]}" --label Expected <(echo "$got") <(echo "$expected")
	fi
}

str-contains() {
	python -c 'import sys; sys.exit(0 if sys.argv[2] in sys.argv[1] else 1)' "$1" "$2"
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
