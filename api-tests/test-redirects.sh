#!/usr/bin/env bash

assert-eq '/redirect/2' 'HTTP/1.1 302 Found
Location: ../redirect/1
X-Powered-By: httpbun
Content-Length: 200
Content-Type: text/html; charset=utf-8

HTTP/1.1 302 Found
Location: ../get
X-Powered-By: httpbun
Content-Length: 186
Content-Type: text/html; charset=utf-8

HTTP/1.1 200 OK
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

assert-eq '/redirect-to?url=http://'"$HTTPBUN_BIND"'/get' 'HTTP/1.1 302 Found
Location: http://'"$HTTPBUN_BIND"'/get
X-Powered-By: httpbun
Content-Length: 0

HTTP/1.1 200 OK
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
