#!/usr/bin/env bash

assert-eq '/redirect-to' 'HTTP/1.1 400 Bad Request
X-Powered-By: httpbun
Content-Length: 19
Content-Type: text/plain; charset=utf-8

Need url parameter
'

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

assert-eq '/absolute-redirect/2' 'HTTP/1.1 302 Found
Location: /absolute-redirect/1
X-Powered-By: httpbun
Content-Length: 214
Content-Type: text/html; charset=utf-8

HTTP/1.1 302 Found
Location: /get
X-Powered-By: httpbun
Content-Length: 182
Content-Type: text/html; charset=utf-8

HTTP/1.1 200 OK
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

assert-eq '/redirect-to?url=http://'"$HTTPBUN_BIND"'/get' 'HTTP/1.1 302 Found
Location: http://'"$HTTPBUN_BIND"'/get
X-Powered-By: httpbun
Content-Length: 0

HTTP/1.1 200 OK
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
