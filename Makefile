build:
	CGO_ENABLED=0 go build -o bin/ok ./cmd/ok

run:
	go run ./cmd/ok

test:
	go test ./tests/integration/... -v

clean:
	rm -rf bin/

.PHONY: build run test clean
