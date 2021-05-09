LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

run:
	@HTTPBUN_ALLOW_HOSTS=localhost:$${PORT:-3090} go run $(LDFLAGS) main.go

build:
	@mkdir -p bin
	@go build $(LDFLAGS) -v -o bin/httpbun

build-all:
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-darwin-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-linux-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-windows-amd64.exe

build-for-docker:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/httpbun-docker .

test:
	@HTTPBUN_ALLOW_HOSTS=example.com go test ./...

fmt:
	@go fmt ./...

.PHONY: run build test fmt docker-image
