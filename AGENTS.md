# Httpbun Agent Guide

## Project Purpose

Httpbun is an HTTP testing service that provides endpoints useful for testing HTTP clients, browsers, libraries, and API developer tools. It's heavily inspired by [httpbin](https://httpbin.org) and is designed to help developers test and debug HTTP interactions.

The service provides various endpoints for:
- Testing different HTTP methods (GET, POST, PUT, DELETE, etc.)
- Inspecting request headers, cookies, and query parameters
- Testing authentication mechanisms
- Simulating redirects, delays, and streaming responses
- Testing various response status codes

## Project Structure

```
httpbun/
├── main.go              # Entry point, starts the server
├── server/              # Server setup and HTTP request handling
│   ├── server.go        # Main server implementation
│   └── spec/            # Server configuration/specification
├── routes/              # Route handlers organized by feature
│   ├── routes.go        # Main route registration
│   ├── method/          # HTTP method handlers (GET, POST, etc.)
│   ├── headers/         # Header inspection/setting handlers
│   ├── cookies/         # Cookie manipulation handlers
│   ├── auth/            # Authentication handlers (Basic, Bearer, Digest)
│   ├── redirect/        # Redirect testing handlers
│   ├── mix/             # Mixed response handlers
│   └── ...              # Other route packages
├── ex/                  # Exchange object (request/response wrapper)
│   └── exchange.go      # Core Exchange type and methods
├── response/            # Response types and helpers
│   └── response.go      # Response struct and helper functions
└── util/                # Utility functions
```

## How Route Handlers Work

### Handler Function Signature

Route handlers are functions that take an `*ex.Exchange` and return a `response.Response`:

```go
func handleMyRoute(ex *ex.Exchange) response.Response {
    // Handler logic here
    return response.Response{Body: "Hello"}
}
```

### Registering Routes

Routes are registered using `ex.NewRoute()` which takes a regex pattern and a handler function:

```go
var RouteList = []ex.Route{
    ex.NewRoute("/my-route", handleMyRoute),
    ex.NewRoute("/users/(?P<id>\\d+)", handleUser),
}
```

### Route Patterns

Route patterns are regular expressions that can include named capture groups:

- `(?P<name>pattern)` - Named capture group accessible via `ex.Field("name")`
- Example: `/users/(?P<id>\\d+)` matches `/users/123` and captures `id` as `"123"`

### Route Registration Flow

1. Each route package exports a `RouteList` variable containing `[]ex.Route`
2. Routes are collected in `routes.GetRoutes()` which concatenates all route lists
3. The server matches incoming requests against routes in order
4. When a route matches, `MatchAndLoadFields()` extracts named groups into `ex.fields`
5. The handler function is called with the Exchange object
6. The handler returns a `response.Response` which is sent via `ex.Finish()`

### Example Route Handler

```go
package mypackage

import (
    "github.com/sharat87/httpbun/ex"
    "github.com/sharat87/httpbun/response"
)

var RouteList = []ex.Route{
    ex.NewRoute("/greet/(?P<name>\\w+)", handleGreet),
}

func handleGreet(ex *ex.Exchange) response.Response {
    name := ex.Field("name")
    return response.Response{
        Body: map[string]string{
            "message": "Hello, " + name,
        },
    }
}
```

## Exchange Object Methods

The `Exchange` object wraps the HTTP request and response, providing convenient methods for accessing request data and building responses.

### Accessing Route Parameters

- **`Field(name string) string`** - Get a captured route parameter by name
  ```go
  // Route: /users/(?P<id>\\d+)
  userId := ex.Field("id")
  ```

### Query Parameters

- **`QueryParamInt(name string, value int) (int, error)`** - Get query parameter as integer with default value
  ```go
  page, err := ex.QueryParamInt("page", 1) // Defaults to 1 if missing
  ```

- **`QueryParamSingle(name string) (string, error)`** - Get single query parameter (errors if missing or multiple values)
  ```go
  token, err := ex.QueryParamSingle("token")
  ```

### Form Parameters

- **`FormParamSingle(name string) (string, error)`** - Get single form parameter (errors if missing or multiple values)
  ```go
  email, err := ex.FormParamSingle("email")
  ```

### Headers

- **`HeaderValueLast(name string) string`** - Get the last value of a header (handles multiple values)
  ```go
  contentType := ex.HeaderValueLast("Content-Type")
  ```

- **`ExposableHeadersMap() map[string]any`** - Get all request headers as a map (excludes internal headers)
  ```go
  headers := ex.ExposableHeadersMap()
  ```

### Request Body

- **`BodyBytes() []byte`** - Get request body as bytes (capped at 10KB)
  ```go
  body := ex.BodyBytes()
  ```

- **`BodyString() string`** - Get request body as string
  ```go
  body := ex.BodyString()
  ```

### Request Information

- **`FindScheme() string`** - Get request scheme ("http" or "https")
  ```go
  scheme := ex.FindScheme()
  ```

- **`FullUrl() string`** - Get the full request URL including scheme
  ```go
  url := ex.FullUrl() // "https://example.com/path?query=value"
  ```

- **`FindIncomingIPAddress() string`** - Get the client's IP address
  ```go
  ip := ex.FindIncomingIPAddress()
  ```

### Direct Request Access

- **`Request *http.Request`** - Direct access to the underlying HTTP request
  ```go
  method := ex.Request.Method
  cookies := ex.Request.Cookies()
  ```

### Response Helpers

- **`RedirectResponse(target string) *response.Response`** - Create a redirect response
  ```go
  return *ex.RedirectResponse("/new-path")
  ```

- **`Finish(resp response.Response)`** - Send the response (called automatically by the server)
  ```go
  // Usually not called directly - handler returns Response instead
  ```

### Route Matching

- **`MatchAndLoadFields(routePat regexp.Regexp) bool`** - Match route pattern and load fields (used internally)
- **`RoutedPath string`** - The request path after removing the server's path prefix

### Server Configuration

- **`ServerSpec spec.Spec`** - Access server configuration/specification
  ```go
  prefix := ex.ServerSpec.PathPrefix
  ```

## Response Object

The `response.Response` struct has the following fields:

```go
type Response struct {
    Status  int              // HTTP status code (defaults to 200)
    Header  http.Header      // Response headers
    Cookies []http.Cookie    // Cookies to set
    Body    any              // Response body (string, []byte, or JSON-serializable)
    Writer  func(w BodyWriter) // Optional streaming writer function
}
```

### Response Helpers

- **`response.New(status int, header http.Header, body []byte) Response`** - Create a response with status, headers, and body
- **`response.BadRequest(message string, vars ...any) Response`** - Create a 400 Bad Request response
  ```go
  return response.BadRequest("Invalid parameter: %s", paramName)
  ```

### Response Body Types

The `Body` field accepts:
- `string` - Sent as-is
- `[]byte` - Sent as raw bytes
- `map[string]any` or other JSON-serializable types - Automatically serialized to JSON with `Content-Type: application/json`

### Streaming Responses

For streaming/chunked responses, use the `Writer` field:

```go
return response.Response{
    Status: 200,
    Writer: func(w response.BodyWriter) {
        w.Write("chunk1")
        w.Write("chunk2")
    },
}
```

## Example: Complete Route Handler

```go
package example

import (
    "net/http"
    "github.com/sharat87/httpbun/ex"
    "github.com/sharat87/httpbun/response"
)

var RouteList = []ex.Route{
    ex.NewRoute("/api/users/(?P<id>\\d+)", handleUser),
}

func handleUser(ex *ex.Exchange) response.Response {
    // Get route parameter
    userID := ex.Field("id")
    
    // Get query parameters
    includeDetails, _ := ex.QueryParamInt("details", 0)
    
    // Get header
    authToken := ex.HeaderValueLast("Authorization")
    
    // Validate
    if authToken == "" {
        return response.BadRequest("Missing Authorization header")
    }
    
    // Build response
    return response.Response{
        Status: http.StatusOK,
        Header: http.Header{
            "Content-Type": []string{"application/json"},
        },
        Body: map[string]any{
            "id": userID,
            "details": includeDetails == 1,
            "authenticated": authToken != "",
        },
    }
}
```
