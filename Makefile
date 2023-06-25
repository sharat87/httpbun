LDFLAGS := -ldflags "-X info.Version=$(VERSION) -X info.Commit=$$(git rev-parse HEAD) -X info.Date=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

run:
	go mod tidy
	go fmt ./...
	@CGO_ENABLED=0 \
		go run $(LDFLAGS) main.go

build:
	@mkdir -p bin
	@go build $(LDFLAGS) -v -o bin/httpbun
	@cd bin && zip httpbun.zip httpbun

build-all:
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-darwin-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-linux-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-windows-amd64.exe

build-for-docker:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -a -installsuffix cgo -o bin/httpbun-docker .

build-for-prod:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-linux-amd64
	cd bin && tar -caf ../package.tar.gz httpbun-linux-amd64

test:
	@go test ./...

fmt:
	@go fmt ./...

.PHONY: run build test fmt docker-image
