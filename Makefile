.PHONY: run test

run:
	go run ./cmd/flash-sale/main.go

test:
	go test -v ./test/concurrent_test.go
