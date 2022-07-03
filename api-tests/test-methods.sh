#!/usr/bin/env bash

assert-eq '/get' 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 166

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "method": "GET",
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}
'

assert-eq '/get?name=Sherlock' 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 206

{
  "args": {
    "name": "Sherlock"
  },
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "method": "GET",
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get?name=Sherlock"
}
'

assert-eq '/get?first=Sherlock&last=Holmes' 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 247

{
  "args": {
    "first": "Sherlock",
    "last": "Holmes"
  },
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "method": "GET",
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get?first=Sherlock\u0026last=Holmes"
}
'

assert-eq '/get' -H x-custom:first-custom 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 198

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl",
    "X-Custom": "first-custom"
  },
  "method": "GET",
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}
'

assert-eq '/get' -H x-first:first-custom -H x-second:second-custom 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 230

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl",
    "X-First": "first-custom",
    "X-Second": "second-custom"
  },
  "method": "GET",
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}
'
