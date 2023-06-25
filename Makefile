LDFLAGS := -ldflags "-X github.com/sharat87/httpbun/info.Version=$(VERSION) -X github.com/sharat87/httpbun/info.Commit=$$(git rev-parse HEAD) -X github.com/sharat87/httpbun/info.Date=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

run:
	go mod tidy
	go fmt ./...
	go run $(LDFLAGS) main.go

build:
	@go build $(LDFLAGS) -v -o bin/httpbun

docker:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -a -installsuffix cgo -o bin/httpbun-docker .
	docker build -t httpbun .

test:
	@go test -count=1 -vet=all ./...

fmt:
	@go fmt ./...

.PHONY: run build docker test fmt
