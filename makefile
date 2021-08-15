LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

run:
	@HTTPBUN_INFO_ENABLED=1 \
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
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/httpbun-docker .

build-for-prod:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o bin/httpbun-linux-amd64
	cd bin && tar -caf ../package.tar.gz httpbun-linux-amd64

upload-package:
	aws s3 cp package.tar.gz s3://ssk-artifacts/httpbun-package.tar.gz

test:
	@HTTPBUN_ALLOW_HOSTS=example.com \
		HTTPBUN_INFO_ENABLED=1 \
		go test ./...

fmt:
	@go fmt ./...

.PHONY: run build test fmt docker-image
