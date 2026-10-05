# Httpbun Agent Guide

Httpbun is an HTTP testing service, like [httpbin](https://httpbin.org): endpoints for testing HTTP clients, like
echoing requests, auth, redirects, cookies, streaming, and mock LLM APIs. It's public at httpbun.com, and people
self-host it, often in CI.

## Checking your work

```sh
task check
```

This runs everything CI runs: formatting, `go vet`, staticcheck, the Go tests, and the LLM SDK tests. A change is done
when this passes. Run `task fmt` to fix formatting. For quicker loops, `task test`, or `go test ./api-tests/ -run Name`.

The SDK tests need [uv](https://docs.astral.sh/uv/). `task llm-sdk-tests` starts its own server.

## Layout

- `main.go`, `server/`: Startup and routing. `server/spec` has all configuration, from flags and env variables.
- `ex/`: `Exchange`, which wraps a request, with helpers to read it, and `Finish`, which writes a response.
- `response/`: The `Response` that handlers return.
- `routes/`: Handlers, grouped by feature. `routes/routes.go` lists all routes, in the order they're matched.
- `assets/`: The HTML pages, which are Go templates. `assets/index.html` is the documentation for all endpoints.
- `api-tests/`: Tests for endpoints, through real HTTP.
- `llm-sdk-tests/`: Tests for the `/llm` mocks, with the real OpenAI and Anthropic Python SDKs.

## Handlers

A handler takes an `*ex.Exchange`, and returns a `response.Response`. Routes are regexes, registered with
`ex.NewRoute`, and named groups are read with `ex.Field`. Look at an existing handler in `routes/` for the details.

- **Validate all input.** Invalid input gets a 400 that says what's wrong, never a panic. Use `ex.QueryInt` and
  `ex.FieldInt` for numbers, which check a range, and `util.ParseStatusCode` for status codes. 1xx status codes aren't
  allowed, since they can't be the final status of a response.
- **Respect the path prefix.** With `--path-prefix /mount`, every endpoint is under `/mount`. Never write a URL that
  starts with `/` in a response or a page, without the prefix: use `ex.ServerSpec.PathPrefix`, or `{{.pathPrefix}}` in
  templates. `ex.RedirectResponse` adds it for you.
- **A trailing slash is removed before routing.** So `/get/` is `/get`, and patterns shouldn't allow one. If a handler's
  output depends on it, like a relative redirect, check `ex.HadTrailingSlash`.
- **Stop streaming when the client goes away.** A handler that streams with `Writer` must stop when
  `ex.Request.Context()` is done.
- **Configuration goes in `spec.Spec`.** Don't read env variables anywhere else.

## Tests

- Test endpoints in `api-tests/`, through HTTP, with `ExecRequest`. That covers routing and `Finish`, which is where
  status codes, headers and cookies are actually written. Use `NewServer(t, spec.Spec{...})` for other configurations,
  like a path prefix.
- Unit tests next to the code are only for pure functions, like parsers.
- Use `github.com/stretchr/testify/assert`.
- Every route needs an example in `endpoints`, in `api-tests/routes_test.go`, with where it's documented. Tests use
  these to check that every route is documented, and that it works with a path prefix and a trailing slash.

## Documentation

`assets/index.html` is the user facing documentation. When an endpoint's behaviour changes, update its docs in the same
change. The `/mix` and `/run` endpoints are documented on their own pages, `assets/mixer-help.html` and
`assets/runner.html`.
