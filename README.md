<h1>
  <img alt=Logo src='https://github.com/sharat87/httpbun/raw/master/assets/icon-32.png'>
  Httpbun
</h1>

This is an HTTP service with endpoints that are useful when testing any HTTP client, like a browser, a library, or any
API developer tool. It's heavily inspired by [httpbin](https://httpbin.org).

Hosted at [httpbun.com](https://httpbun.com) and [httpbun.org](https://httpbun.org). Run your own with:

```sh
docker run -p 80:80 ghcr.io/sharat87/httpbun
```

A project by [Shri](https://sharats.me).

## Building

We patch Go's standard lib a little. There's a line in `net/http/server.go` that delets the `Host` header in all incoming requests. We comment that line out during build, and uncomment it again to restore it.

So, if you're using the same Go installation for this and other projects _at the same time_, you may see unexpected behaviour.

The patching and unpatching is in `make patch` and `make unpatch` targets.

## Contributing

Contributions to httpbun are welcome, for the most part. However, I strongly urge you to open an issue to discuss
whatever you're working to contribute *before* you start working on it. This will ensure we are on the same page and
your work would be in the right place to be merged in. It'll also ensure we don't end up working on the same thing,
duplicating efforts. Thanks!

## Plug

If you are interested in API testing and API development, you should check [Prestige](https://prestige.dev) out. It is a text based API testing tool with Javascript templating support. It's also open source at [sharat87/prestige](https://github.com/sharat87/prestige).

## License

[Apache-2.0 License](https://github.com/sharat87/httpbun/blob/master/LICENSE). Project includes a
[NOTICE](https://github.com/sharat87/httpbun/blob/master/NOTICE) file.
