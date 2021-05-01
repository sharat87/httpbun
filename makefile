run:
	go run main.go

httpbun:
	go build -o httpbun

test:
	go test


.PHONY: run test
