#!/usr/bin/env bash

assert-eq /get 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 147

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}'

assert-eq '/get?name=Sherlock' 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 187

{
  "args": {
    "name": "Sherlock"
  },
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get?name=Sherlock"
}'
