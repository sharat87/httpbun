#!/usr/bin/env bash

assert-eq '/get' 'HTTP/1.1 200 OK
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
}
'

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
}
'

assert-eq '/get?first=Sherlock&last=Holmes' 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 228

{
  "args": {
    "first": "Sherlock",
    "last": "Holmes"
  },
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl"
  },
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get?first=Sherlock\u0026last=Holmes"
}
'

assert-eq '/get' -H x-custom:first-custom 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 179

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl",
    "X-Custom": "first-custom"
  },
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}
'

assert-eq '/get' -H x-first:first-custom -H x-second:second-custom 'HTTP/1.1 200 OK
Content-Type: application/json
X-Powered-By: httpbun
Content-Length: 211

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl",
    "X-First": "first-custom",
    "X-Second": "second-custom"
  },
  "origin": "127.0.0.1",
  "url": "http://'"$HTTPBUN_BIND"'/get"
}
'
