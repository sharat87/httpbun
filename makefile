run:
	@go run -ldflags "-X main.Version=WIP -X main.Commit=$(COMMIT) -X main.Date=$(DATE)" main.go

build:
	@go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)" -v -o httpbun

test:
	@go test ./...

fmt:
	@go fmt ./...

docker-image:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o httpbun-docker .
	docker build -t sharat87/httpbun:latest .


.PHONY: run build test fmt docker-image
