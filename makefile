LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

run:
	@go run $(LDFLAGS) main.go

build:
	@mkdir -p bin
	@go build $(LDFLAGS) -v -o bin/httpbun

build-all:
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-darwin-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-linux-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-windows-amd64.exe

test:
	@go test ./...

fmt:
	@go fmt ./...

docker-image:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o httpbun-docker .
	docker build -t sharat87/httpbun:latest .


.PHONY: run build test fmt docker-image
