<h1>
  <img alt=Logo src='https://github.com/sharat87/httpbun/raw/main/assets/icon-32.png'>
  Httpbun
</h1>

This is an HTTP service with endpoints that are useful when testing any HTTP client, like a browser, a library, or any
API developer tool. It's inspired by [httpbin](https://httpbin.org), but is _so_ much more today.

Hosted at [httpbun.com](https://httpbun.com). Run your own with:

```sh
docker run -p 80:80 sharat87/httpbun
```

A project by [Shri](https://sharats.me).

:warning: If you are using this from your CI, please don't. Run a local version using the above Docker command, within your CI system, and use that "locally".

## Building

There's a `Taskfile.dist.yml` included in the project, which is a [Taskfile](https://taskfile.dev). Once you have `task` installed, running `task run` will start a local server from source. There's also:

1. `task build` to build the binary
2. `task test` to run tests
3. `task fmt` to format code
4. `task docker` to build binaries for building Docker image

We patch Go's standard lib a little. There's a line in `net/http/server.go` that delets the `Host` header in all incoming requests. We comment that line out during build, and uncomment it again to restore it.

So, if you're using the same Go installation for this and other projects _at the same time_, you may see unexpected behaviour.

The patching and unpatching is in `task patch` and `task unpatch` targets.

## Contributing

Bug fixes, yes. New features, less so please. Either way, consider opening an issue to discuss
it *before* you start working. This will ensure we are on the same page and
your work would be in the right place to be merged in. It'll also ensure we don't end up working on the same thing,
duplicating efforts. Thanks!

New features are added mostly as _I_ need them, and in the form that I want to use them.

## License

[Apache-2.0 License](https://github.com/sharat87/httpbun/blob/main/LICENSE). Project includes a
[NOTICE](https://github.com/sharat87/httpbun/blob/main/NOTICE) file.
