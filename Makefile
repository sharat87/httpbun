LDFLAGS := -ldflags "-X github.com/sharat87/httpbun/info.Commit=$$(git rev-parse HEAD) -X github.com/sharat87/httpbun/info.Date=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

run:
	go mod tidy
	go fmt ./...
	go run $(LDFLAGS) main.go

build:
	@go build $(LDFLAGS) -v -o bin/httpbun

docker:
	make patch
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -a -installsuffix cgo -o bin/httpbun-docker .
	docker build -t httpbun .
	make unpatch

test:
	@go test -count=1 -vet=all ./...

fmt:
	@go fmt ./...

# We patch the Go stdlib to remove the code that removes the Host header from incoming requests.
patch:
	sed 's:\(delete(req.Header, "Host")\)$$://\1:' "$$(go env GOROOT)/src/net/http/server.go" > x
	mv x "$$(go env GOROOT)/src/net/http/server.go"

unpatch:
	sed 's://\(delete(req.Header, "Host")\)$$:\1:' "$$(go env GOROOT)/src/net/http/server.go" > x
	mv x "$$(go env GOROOT)/src/net/http/server.go"

.PHONY: run build docker test fmt
